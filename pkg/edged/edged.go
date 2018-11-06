package edged

import (
	"encoding/json"
	"fmt"
	"time"

	"kubeedge/beehive/pkg/common/config"
	"kubeedge/beehive/pkg/common/log"
	"kubeedge/beehive/pkg/core"
	"kubeedge/beehive/pkg/core/context"
	"kubeedge/beehive/pkg/core/model"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/apimachinery/pkg/util/sets"
	utilwait "k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/flowcontrol"
	"k8s.io/client-go/util/workqueue"
	kubecontainer "k8s.io/kubernetes/pkg/kubelet/container"
	"k8s.io/kubernetes/pkg/kubelet/gpu"
	"k8s.io/kubernetes/pkg/kubelet/gpu/nvidia"
	"k8s.io/kubernetes/pkg/kubelet/images"
	"k8s.io/kubernetes/pkg/kubelet/lifecycle"
	"k8s.io/kubernetes/pkg/kubelet/pleg"
	"k8s.io/kubernetes/pkg/kubelet/prober"
	proberesults "k8s.io/kubernetes/pkg/kubelet/prober/results"
	"k8s.io/kubernetes/pkg/kubelet/util/format"
	"k8s.io/kubernetes/pkg/kubelet/util/queue"
	"k8s.io/kubernetes/pkg/kubelet/volumemanager"
	"k8s.io/kubernetes/pkg/scheduler/schedulercache"
	kubeio "k8s.io/kubernetes/pkg/util/io"
	"k8s.io/kubernetes/pkg/util/mount"
	"k8s.io/kubernetes/pkg/volume"
	"k8s.io/kubernetes/pkg/volume/configmap"
	"k8s.io/kubernetes/pkg/volume/empty_dir"
	"k8s.io/kubernetes/pkg/volume/host_path"
	secretvolume "k8s.io/kubernetes/pkg/volume/secret"

	"kubeedge/beehive/pkg/common/util"
	"kubeedge/pkg/edged/apis"
	"kubeedge/pkg/edged/containers"
	"kubeedge/pkg/edged/dockertools"
	edgeImages "kubeedge/pkg/edged/images"
	"kubeedge/pkg/edged/metaclient"
	edgepleg "kubeedge/pkg/edged/pleg"
	"kubeedge/pkg/edged/podmanager"
	"kubeedge/pkg/edged/rainerruntime"
	"kubeedge/pkg/edged/server"
	"kubeedge/pkg/edged/store"
	edgedutil "kubeedge/pkg/edged/util"
	utilpod "kubeedge/pkg/edged/util/pod"
	"kubeedge/pkg/edged/util/record"
	"kubeedge/pkg/metamanager"
	"k8s.io/kubernetes/pkg/kubelet/status"
)

const (
	plegChannelCapacity = 1000
	plegRelistPeriod    = time.Second * 1
	concurrentConsumers = 5
	backOffPeriod       = 10 * time.Second
	MaxContainerBackOff = 300 * time.Second
	enqueueDuration     = 10 * time.Second
	ImageGCPeriod       = 5 * time.Second
	ContainerGCPeriod   = 60 * time.Second
	// Period for performing global cleanup tasks.
	housekeepingPeriod               = time.Second * 2
	syncWorkQueuePeriod              = time.Second * 2
	minAge                           = 60 * time.Second
	imageGcHighThreshold             = "edged.image-gc-high-threshold"
	syncMsgRespTimeout               = 1 * time.Minute
	DefaultRootDir                   = "/var/lib/edged"
	workerResyncIntervalJitterFactor = 0.5
	EdgeController                   = "controller"
)

//Define edged
type edged struct {
	context                   *context.Context
	initPodReady              bool
	hostname                  string
	namespace                 string
	nodeName                  string
	uid                       types.UID
	nodeStatusUpdateFrequency time.Duration
	registrationCompleted     bool
	runtime                   rainerruntime.Runtime
	podManager                podmanager.Manager
	pleg                      pleg.PodLifecycleEventGenerator
	store                     store.BackendStore
	statusManager             status.Manager
	probeManager              prober.Manager
	livenessManager           proberesults.Manager
	server                    *server.Server
	podAdditionQueue          *workqueue.Type
	podAdditionBackoff        *flowcontrol.Backoff
	podDeletionQueue          *workqueue.Type
	podDeletionBackoff        *flowcontrol.Backoff
	imageGCManager            images.ImageGCManager
	containerGCManager        kubecontainer.ContainerGC
	gpuManager                gpu.GPUManager
	metaClient                metaclient.CoreInterface
	volumePluginMgr           *volume.VolumePluginMgr
	mounter                   mount.Interface
	writer                    kubeio.Writer
	volumeManager             volumemanager.VolumeManager
	rootDirectory             string
	gpuPluginEnabled          bool
	version                   string
	// cache for secret
	secretStore    cache.Store
	configMapStore cache.Store
	workQueue      queue.WorkQueue
}

type Config struct {
	nodeName                 string
	nodeNamespace            string
	interfaceName            string
	nodeStatusUpdateInterval time.Duration
	devicePluginEnabled      bool
	gpuPluginEnabled         bool
	imageGCHighThreshold     int
	imageGCLowThreshold      int
	MaxPerPodContainerCount  int
	version                  string
}

func init() {
	edged, err := NewEdged()
	if err != nil {
		log.LOGGER.Errorf("init new edged error, %v", err)
		return
	}
	core.Register(edged)
}

func (e *edged) Name() string {
	return "edged"
}

func (e *edged) Group() string {
	return core.EdgedGroup
}

func (e *edged) Start(c *context.Context) {
	e.context = c
	if err := e.initializeModules(); err != nil {
		log.LOGGER.Errorf("initialize module error: %v", err)
		return
	}
	err := e.makePodDir()
	if err != nil {
		log.LOGGER.Errorf("create pod dir [%s] failed: %v", e.getPodsDir(), err)
		return
	}
	e.metaClient = metaclient.New(c)
	go e.volumeManager.Run(edgedutil.NewSourcesReady(), utilwait.NeverStop)
	go utilwait.Until(e.syncNodeStatus, e.nodeStatusUpdateFrequency, utilwait.NeverStop)

	e.probeManager = prober.NewManager(e.statusManager, e.livenessManager, containers.NewContainerRunner(), kubecontainer.NewRefManager(), record.NewEventRecorder())
	e.pleg = edgepleg.NewGenericLifecycle(e.runtime.(*dockertools.DockerManager).ContainerManager, e.probeManager, plegChannelCapacity, plegRelistPeriod, e.podManager, e.statusManager)
	e.statusManager.Start()
	e.pleg.Start()

	e.podAddWorkerRun(concurrentConsumers)
	e.podRemoveWorkerRun(concurrentConsumers)

	housekeepingTicker := time.NewTicker(housekeepingPeriod)
	syncWorkQueueCh := time.NewTicker(syncWorkQueuePeriod)
	e.probeManager.Start()
	go e.syncLoopIteration(e.pleg.Watch(), housekeepingTicker.C, syncWorkQueueCh.C)
	go e.server.ListenAndServe()

	e.imageGCManager.Start()
	e.StartGarbageCollection()
	e.syncPod()
}

func (e *edged) Cleanup() {
	if err := recover(); err != nil {
		log.LOGGER.Errorf("edged exit with error: %v", err)
	}
	e.context.Cleanup(e.Name())
	log.LOGGER.Info("edged exit!")
}

func getConfig() *Config {
	var conf Config
	conf.nodeName = config.CONFIG.GetConfigurationByKey("edged.hostname-override").(string)
	conf.nodeNamespace = config.CONFIG.GetConfigurationByKey("edged.register-node-namespace").(string)
	conf.interfaceName = config.CONFIG.GetConfigurationByKey("edged.interface-name").(string)
	nodeStatusUpdateInterval := config.CONFIG.GetConfigurationByKey("edged.node-status-update-frequency").(int)
	conf.nodeStatusUpdateInterval = time.Duration(nodeStatusUpdateInterval) * time.Second
	conf.devicePluginEnabled = config.CONFIG.GetConfigurationByKey("edged.device-plugin-enabled").(bool)
	conf.gpuPluginEnabled = config.CONFIG.GetConfigurationByKey("edged.gpu-plugin-enabled").(bool)
	conf.imageGCHighThreshold = config.CONFIG.GetConfigurationByKey("edged.image-gc-high-threshold").(int)
	conf.imageGCLowThreshold = config.CONFIG.GetConfigurationByKey("edged.image-gc-low-threshold").(int)
	conf.MaxPerPodContainerCount = config.CONFIG.GetConfigurationByKey("edged.maximum-dead-containers-per-container").(int)
	conf.version = config.CONFIG.GetConfigurationByKey("edged.version").(string)
	return &conf
}

func NewEdged() (*edged, error) {
	var gpuManager gpu.GPUManager
	conf := getConfig()
	backoff := flowcontrol.NewBackOff(backOffPeriod, MaxContainerBackOff)

	podManager := podmanager.NewPodManager()
	policy := images.ImageGCPolicy{
		HighThresholdPercent: conf.imageGCHighThreshold,
		LowThresholdPercent:  conf.imageGCLowThreshold,
		MinAge:               minAge,
	}
	statusManager := status.NewManager(nil, podManager, utilpod.NewPodDeleteSafety()) // TODO

	ed := &edged{
		nodeName:                  conf.nodeName,
		namespace:                 conf.nodeNamespace,
		gpuPluginEnabled:          conf.gpuPluginEnabled,
		podManager:                podManager,
		podAdditionQueue:          workqueue.New(),
		podAdditionBackoff:        backoff,
		podDeletionQueue:          workqueue.New(),
		podDeletionBackoff:        backoff,
		statusManager:             statusManager,
		nodeStatusUpdateFrequency: conf.nodeStatusUpdateInterval,
		mounter:                   mount.New(""),
		writer:                    &kubeio.StdWriter{},
		uid:                       types.UID("38796d14-1df3-11e8-8e5a-286ed488f209"),
		version:                   conf.version,
		rootDirectory:             DefaultRootDir,
		secretStore:               cache.NewStore(cache.MetaNamespaceKeyFunc),
		configMapStore:            cache.NewStore(cache.MetaNamespaceKeyFunc),
		workQueue:                 queue.NewBasicWorkQueue(clock.RealClock{}),
	}

	if conf.gpuPluginEnabled {
		gpuManager, _ = nvidia.NewNvidiaGPUManager(ed, dockertools.NewDockerConfig())
	} else {
		gpuManager = gpu.NewGPUManagerStub()
	}
	ed.gpuManager = gpuManager
	ed.livenessManager = proberesults.NewManager()
	runtime, err := dockertools.NewDockerManager(ed.livenessManager, 0, 0, backoff, true, conf.devicePluginEnabled, gpuManager, conf.interfaceName)
	if err != nil {
		return nil, fmt.Errorf("get docker manager failed, err: %s", err.Error())
	}

	ed.runtime = runtime

	// build new object to match interface
	recorder := record.NewEventRecorder()
	nodeRef := &v1.ObjectReference{
		Kind:      "Node",
		Name:      string(ed.nodeName),
		UID:       types.UID(ed.nodeName),
		Namespace: "",
	}
	statsProvider := edgeImages.NewStatsProvider()

	ed.imageGCManager, err = images.NewImageGCManager(runtime, statsProvider, recorder, nodeRef, policy, "")
	if err != nil {
		return nil, fmt.Errorf("init Image GC Manager failed with error %s", err.Error())
	}
	containerGCPolicy := kubecontainer.ContainerGCPolicy{
		MinAge:             minAge,
		MaxContainers:      -1,
		MaxPerPodContainer: conf.MaxPerPodContainerCount,
	}
	ed.containerGCManager, err = kubecontainer.NewContainerGC(runtime, containerGCPolicy, &containers.KubeSourcesReady{})
	if err != nil {
		return nil, fmt.Errorf("init Container GC Manager failed with error %s", err.Error())
	}

	ed.store = store.NewStore("")
	ed.server = server.NewServer(ed.podManager)
	ed.volumePluginMgr, err = NewInitializedVolumePluginMgr(ed, ProbeVolumePlugins(""))
	ed.volumeManager = volumemanager.NewVolumeManager(
		false,
		types.NodeName(ed.nodeName),
		podManager,
		ed.statusManager,
		nil, // TODO
		ed.volumePluginMgr,
		runtime,
		ed.mounter,
		ed.getPodsDir(),
		recorder,
		false,
		false,
	)
	return ed, nil
}

func (e *edged) initializeModules() error {
	if err := e.gpuManager.Start(); err != nil {
		log.LOGGER.Errorf("Failed to start gpuManager %v", err)
		return err
	}

	if err := e.runtime.Start(e.GetActivePods); err != nil {
		log.LOGGER.Errorf("Failed to start device plugin manager %v", err)
		return err
	}

	return nil
}

func (e *edged) StartGarbageCollection() {
	go utilwait.Until(func() {
		err := e.imageGCManager.GarbageCollect()
		if err != nil {
			log.LOGGER.Errorf("Image garbage collection failed")
		}
	}, ImageGCPeriod, utilwait.NeverStop)

	go utilwait.Until(func() {
		if e.initPodReady {
			err := e.containerGCManager.GarbageCollect()
			if err != nil {
				log.LOGGER.Errorf("Container garbage collection failed: %v", err)
			}
		}
	}, ContainerGCPeriod, utilwait.NeverStop)
}

func (e *edged) syncLoopIteration(plegCh <-chan *pleg.PodLifecycleEvent, housekeepingCh <-chan time.Time, syncWorkQueueCh <-chan time.Time) {

	for {
		select {
		case update := <-e.livenessManager.Updates():
			if update.Result == proberesults.Failure {
				pod, ok := e.podManager.GetPodByUID(update.PodUID)
				if !ok {
					log.LOGGER.Infof("SyncLoop (container unhealthy): ignore irrelevant update: %#v", update)
					break
				}
				log.LOGGER.Infof("SyncLoop (container unhealthy): %q", format.Pod(pod))
				if pod.Spec.RestartPolicy == v1.RestartPolicyNever {
					break
				}
				var containerCompleted bool
				if pod.Spec.RestartPolicy == v1.RestartPolicyOnFailure {
					for _, containerStatus := range pod.Status.ContainerStatuses {
						if containerStatus.State.Terminated != nil && containerStatus.State.Terminated.ExitCode == 0 {
							containerCompleted = true
							break
						}
					}
					if containerCompleted {
						break
					}
				}
				log.LOGGER.Infof("Will restart pod [%s]", pod.Name)
				key := types.NamespacedName{
					pod.Namespace,
					pod.Name,
				}
				e.podAdditionQueue.Add(key.String())
			}
		case plegEvent := <-plegCh:
			if pod, ok := e.podManager.GetPodByUID(plegEvent.ID); ok {
				if plegEvent.Type == pleg.ContainerDied {
					if pod.Spec.RestartPolicy == v1.RestartPolicyNever {
						break
					}
					var containerCompleted bool
					if pod.Spec.RestartPolicy == v1.RestartPolicyOnFailure {
						for _, containerStatus := range pod.Status.ContainerStatuses {
							if containerStatus.State.Terminated != nil && containerStatus.State.Terminated.ExitCode == 0 {
								containerCompleted = true
								break
							}
						}
						if containerCompleted {
							break
						}
					}
					log.LOGGER.Infof("sync loop get event container died, restart pod [%s]", pod.Name)
					key := types.NamespacedName{
						pod.Namespace,
						pod.Name,
					}
					e.podAdditionQueue.Add(key.String())
				} else {
					log.LOGGER.Infof("sync loop get event [%s], ignore it now.", plegEvent.Type)
				}
			} else {
				log.LOGGER.Infof("sync loop ignore event: [%s], with pod [%s] not found", plegEvent.Type, plegEvent.ID)
			}
		case <-housekeepingCh:
			err := e.HandlePodCleanups()
			if err != nil {
				log.LOGGER.Errorf("Handle Pod Cleanup Failed: %v", err)
			}
		case <-syncWorkQueueCh:
			podsToSync := e.getPodsToSync()
			if len(podsToSync) == 0 {
				break
			}
			for _, pod := range podsToSync {
				if !e.podIsTerminated(pod) {
					key := types.NamespacedName{
						pod.Namespace,
						pod.Name,
					}
					e.podAdditionQueue.Add(key.String())
				}
			}
		}
	}
}

func (e *edged) podAddWorkerRun(consumers int) {
	for i := 0; i < consumers; i++ {
		log.LOGGER.Infof("start pod addition queue work %d", i)
		go func(i int) {
			for {
				item, quit := e.podAdditionQueue.Get()
				if quit {
					log.LOGGER.Errorf("consumer: [%d], worker addition queue is shutting down!", i)
					return
				}
				namespacedName := types.NewNamespacedNameFromString(item.(string))
				podName := namespacedName.Name
				log.LOGGER.Infof("worker [%d] get pod addition item [%s]", i, podName)
				backOffKey := fmt.Sprintf("pod_addition_worker_%s", podName)
				if e.podAdditionBackoff.IsInBackOffSinceUpdate(backOffKey, e.podAdditionBackoff.Clock.Now()) {
					log.LOGGER.Errorf("consume pod addition backoff: Back-off consume pod [%s] addition  error, backoff: [%v]", podName, e.podAdditionBackoff.Get(backOffKey))
					go func() {
						log.LOGGER.Infof("worker [%d] backoff pod addition item [%s] failed, re-add to queue", i, podName)
						time.Sleep(e.podAdditionBackoff.Get(backOffKey))
						e.podAdditionQueue.Add(item)
					}()
					e.podAdditionQueue.Done(item)
					continue
				}
				err := e.consumePodAddition(&namespacedName)
				if err != nil {
					if err == apis.ErrPodNotFound {
						log.LOGGER.Infof("worker [%d] handle pod addition item [%s] failed with not found error.", podName)
						e.podAdditionBackoff.Reset(backOffKey)
					} else {
						go func() {
							log.LOGGER.Errorf("worker [%d] handle pod addition item [%s] failed: %v, re-add to queue", i, podName, err)
							e.podAdditionBackoff.Next(backOffKey, e.podAdditionBackoff.Clock.Now())
							time.Sleep(enqueueDuration)
							e.podAdditionQueue.Add(item)
						}()
					}
				} else {
					e.podAdditionBackoff.Reset(backOffKey)
				}
				e.podAdditionQueue.Done(item)
			}
		}(i)
	}
}

func (e *edged) podRemoveWorkerRun(consumers int) {
	for i := 0; i < consumers; i++ {
		go func(i int) {
			for {
				item, quit := e.podDeletionQueue.Get()
				if quit {
					log.LOGGER.Errorf("consumer: [%d], worker addition queue is shutting down!", i)
					return
				}
				namespacedName := types.NewNamespacedNameFromString(item.(string))
				podName := namespacedName.Name
				log.LOGGER.Infof("consumer: [%d], worker get removed pod [%s]\n", i, podName)
				err := e.consumePodDeletion(&namespacedName)
				if err != nil {
					if err == apis.ErrContainerNotFound {
						log.LOGGER.Infof("pod [%s] is not exist, with container not found error", podName)
					} else {
						go func(item interface{}) {
							log.LOGGER.Errorf("worker remove pod [%s] failed: %v", podName, err)
							time.Sleep(2 * time.Second)
							e.podDeletionQueue.Add(item)
						}(item)
					}

				}
				e.podDeletionQueue.Done(item)
			}
		}(i)
	}
}

func (e *edged) consumePodAddition(namespacedName *types.NamespacedName) error {
	podName := namespacedName.Name
	log.LOGGER.Infof("start to consume added pod [%s]", podName)
	pod, ok := e.podManager.GetPodByName(namespacedName.Namespace, podName)
	if !ok || pod.DeletionTimestamp != nil {
		return apis.ErrPodNotFound
	}

	if err := e.makePodDataDirs(pod); err != nil {
		log.LOGGER.Errorf("Unable to make pod data directories for pod %q: %v", format.Pod(pod), err)
		return err
	}

	if err := e.volumeManager.WaitForAttachAndMount(pod); err != nil {
		log.LOGGER.Errorf("Unable to mount volumes for pod %q: %v; skipping pod", format.Pod(pod), err)
		return err
	}

	secrets, err := e.getSecretsFromMetaManager(pod)
	if err != nil {
		return err
	}
	err = e.runtime.EnsureImageExists(pod, secrets)
	if err != nil {
		return fmt.Errorf("consume added pod [%s] ensure image exist failed, %v", podName, err)
	}
	opt, err := e.GenerateContainerOptions(pod)
	if err != nil {
		return err
	}
	err = e.runtime.StartPod(pod, opt)
	if err != nil {
		return fmt.Errorf("consume added pod [%s] start pod failed, %v", podName, err)
	}

	e.workQueue.Enqueue(pod.UID, utilwait.Jitter(time.Minute, workerResyncIntervalJitterFactor))
	log.LOGGER.Infof("consume added pod [%s] successfully\n", podName)
	return nil
}

func (e *edged) consumePodDeletion(namespacedName *types.NamespacedName) error {
	podName := namespacedName.Name
	log.LOGGER.Infof("start to consume removed pod [%s]", podName)
	pod, ok := e.podManager.GetPodByName(namespacedName.Namespace, podName)
	if !ok {
		return apis.ErrPodNotFound
	}
	err := e.runtime.TerminatePod(pod.UID)
	if err != nil {
		if err == apis.ErrContainerNotFound {
			return err
		}
		return fmt.Errorf("consume removed pod [%s] failed, %v", podName, err)
	}
	log.LOGGER.Infof("consume removed pod [%s] successfully\n", podName)
	return nil
}

func handleErrorInfo(err error, info string, request model.Message, c *context.Context) {
	log.LOGGER.Infof(info)
	errInfo := "Something wrong"
	if err != nil {
		errInfo = fmt.Sprintf(info+": %v", err)
	}
	errResponse := model.NewErrorMessage(&request, errInfo)
	c.Send(request.GetSource(), *errResponse)
}

func (e *edged) syncPod() {
	//read containers from host
	e.runtime.InitPodContainer()
	time.Sleep(10 * time.Second)

	//send msg to metamanager to get existing pods
	info := model.NewMessage("").BuildRouter(e.Name(), e.Group(), e.namespace+"/"+model.ResourceTypePod,
		model.QueryOperation)
	e.context.Send(metamanager.MetaManagerModuleName, *info)
	for {
		if request, err := e.context.Receive(e.Name()); err == nil {
			_, resType, resID, err := util.ParseResourceEdge(request.GetResource(), request.GetOperation())
			op := request.GetOperation()
			if err != nil {
				log.LOGGER.Errorf("failed to parse the Resource: %v", err)
				continue
			}

			content, err := json.Marshal(request.Content)
			if err != nil {
				log.LOGGER.Errorf("marshal message content failed: %v", err)
				continue
			}
			log.LOGGER.Infof("request content is %s", string(content))
			switch resType {
			case model.ResourceTypePod:
				if op == model.ResponseOperation && resID == "" && request.GetSource() == metamanager.MetaManagerModuleName {
					err := e.handlePodListFromMetaManager(content)
					if err != nil {
						log.LOGGER.Errorf("handle podList failed: %v", err)
						continue
					}
					e.initPodReady = true
				} else if op == model.ResponseOperation && resID == "" && request.GetSource() == EdgeController {
					err := e.handlePodListFromEdgeController(content)
					if err != nil {
						log.LOGGER.Errorf("handle controllerPodList failed: %v", err)
						continue
					}
					e.initPodReady = true
				} else {
					err := e.handlePod(op, content)
					if err != nil {
						log.LOGGER.Errorf("handle pod failed: %v", err)
						continue
					}
				}
			case model.ResourceTypeConfigmap:
				if op != model.ResponseOperation {
					err := e.handleConfigMap(op, content)
					if err != nil {
						log.LOGGER.Errorf("handle configMap failed: %v", err)
					}
				} else {
					log.LOGGER.Infof("skip to handle configMap with type response")
					continue
				}
			case model.ResourceTypeSecret:
				if op != model.ResponseOperation {
					err := e.handleSecret(op, content)
					if err != nil {
						log.LOGGER.Errorf("handle secret failed: %v", err)
					}
				} else {
					log.LOGGER.Infof("skip to handle secret with type response")
					continue
				}
			default:
				log.LOGGER.Errorf("resType is not pod or configmap or secret: esType is %s", resType)
				continue
			}

		} else {
			log.LOGGER.Errorf("failed to get pod")
		}
	}
}

func (e *edged) handlePod(op string, content []byte) (err error) {
	var pod v1.Pod
	err = json.Unmarshal(content, &pod)
	if err != nil {
		return err
	}

	switch op {
	case model.InsertOperation:
		e.addPod(&pod)
	case model.UpdateOperation:
		e.updatePod(&pod)
	case model.DeleteOperation:
		e.deletePod(&pod)
	}
	return nil
}

func (e *edged) handlePodListFromMetaManager(content []byte) (err error) {
	var lists []string
	err = json.Unmarshal([]byte(content), &lists)
	if err != nil {
		return err
	}

	for _, list := range lists {
		var pod v1.Pod
		err = json.Unmarshal([]byte(list), &pod)
		if err != nil {
			return err
		}
		e.addPod(&pod)
	}

	return nil
}

func (e *edged) handlePodListFromEdgeController(content []byte) (err error) {
	var lists []v1.Pod

	for _, list := range lists {
		err = json.Unmarshal(content, &list)
		if err != nil {
			return err
		}
		e.addPod(&list)
	}

	return nil
}

func (e *edged) addPod(obj interface{}) {
	pod := obj.(*v1.Pod)
	log.LOGGER.Infof("start sync addition for pod [%s]", pod.Name)
	attrs := &lifecycle.PodAdmitAttributes{}
	attrs.Pod = pod
	otherpods := e.podManager.GetPods()
	attrs.OtherPods = otherpods
	nodeInfo := schedulercache.NewNodeInfo(pod)
	e.runtime.UpdatePluginResources(nodeInfo, attrs)
	key := types.NamespacedName{
		pod.Namespace,
		pod.Name,
	}
	e.podManager.AddPod(pod)
	e.probeManager.AddPod(pod)
	e.podAdditionQueue.Add(key.String())
	log.LOGGER.Infof("success sync addition for pod [%s]", pod.Name)
}

func (e *edged) updatePod(obj interface{}) {
	newPod := obj.(*v1.Pod)
	log.LOGGER.Infof("start update pod [%s]", newPod.Name)
	key := types.NamespacedName{
		newPod.Namespace,
		newPod.Name,
	}
	e.podManager.UpdatePod(newPod)
	e.probeManager.AddPod(newPod)
	if newPod.DeletionTimestamp == nil {
		e.podAdditionQueue.Add(key.String())
	} else {
		e.podDeletionQueue.Add(key.String())
	}
	log.LOGGER.Infof("success update pod is %+v\n", newPod)
}

func (e *edged) deletePod(obj interface{}) {
	pod := obj.(*v1.Pod)
	log.LOGGER.Infof("start remove pod [%s]", pod.Name)
	e.podManager.DeletePod(pod)
	e.statusManager.TerminatePod(pod)
	e.probeManager.RemovePod(pod)
	log.LOGGER.Infof("success remove pod [%s]", pod.Name)
}

func (e *edged) getSecretsFromMetaManager(pod *v1.Pod) ([]v1.Secret, error) {
	var secrets []v1.Secret
	for _, imagePullSecret := range pod.Spec.ImagePullSecrets {
		secret, err := e.metaClient.Secrets(e.namespace).Get(imagePullSecret.Name)
		if err != nil {
			return nil, err
		}
		secrets = append(secrets, *secret)
	}

	return secrets, nil
}

// Get pods which should be resynchronized. Currently, the following pod should be resynchronized:
//   * pod whose work is ready.
//   * internal modules that request sync of a pod.
func (e *edged) getPodsToSync() []*v1.Pod {
	allPods := e.podManager.GetPods()
	podUIDs := e.workQueue.GetWork()
	podUIDSet := sets.NewString()
	for _, podUID := range podUIDs {
		podUIDSet.Insert(string(podUID))
	}
	var podsToSync []*v1.Pod
	for _, pod := range allPods {
		if podUIDSet.Has(string(pod.UID)) {
			// The work of the pod is ready
			podsToSync = append(podsToSync, pod)
		}
	}
	return podsToSync
}

func (e *edged) handleConfigMap(op string, content []byte) (err error) {
	var configMap v1.ConfigMap
	err = json.Unmarshal(content, &configMap)
	if err != nil {
		return
	}
	_, exists, _ := e.configMapStore.Get(&configMap)
	switch op {
	case model.InsertOperation:
		err = e.configMapStore.Add(&configMap)
	case model.UpdateOperation:
		if exists {
			err = e.configMapStore.Update(&configMap)
		}
	case model.DeleteOperation:
		if exists {
			err = e.configMapStore.Delete(&configMap)
		}
	}
	if err == nil {
		log.LOGGER.Infof("%s configMap [%s] for cache success.", op, configMap.Name)
	}
	return
}

func (e *edged) handleSecret(op string, content []byte) (err error) {
	var podSecret v1.Secret
	err = json.Unmarshal(content, &podSecret)
	if err != nil {
		return
	}
	_, exists, _ := e.secretStore.Get(&podSecret)
	switch op {
	case model.InsertOperation:
		err = e.secretStore.Add(&podSecret)
	case model.UpdateOperation:
		if exists {
			err = e.secretStore.Update(&podSecret)
		}
	case model.DeleteOperation:
		if exists {
			err = e.secretStore.Delete(&podSecret)
		}
	}
	if err == nil {
		log.LOGGER.Infof("%s secret [%s] for cache success.", op, podSecret.Name)
	}
	return
}

// ProbeVolumePlugins collects all volume plugins into an easy to use list.
// PluginDir specifies the directory to search for additional third party
// volume plugins.
func ProbeVolumePlugins(pluginDir string) []volume.VolumePlugin {
	allPlugins := []volume.VolumePlugin{}
	hostPathConfig := volume.VolumeConfig{}
	allPlugins = append(allPlugins, configmap.ProbeVolumePlugins()...)
	allPlugins = append(allPlugins, empty_dir.ProbeVolumePlugins()...)
	allPlugins = append(allPlugins, secretvolume.ProbeVolumePlugins()...)
	allPlugins = append(allPlugins, host_path.ProbeVolumePlugins(hostPathConfig)...)
	return allPlugins
}

func (e *edged) HandlePodCleanups() error {
	if !e.initPodReady {
		return nil
	}
	pods := e.podManager.GetPods()
	containerRunningPods, err := e.runtime.GetPods(true)
	if err != nil {
		return err
	}

	e.removeOrphanedPodStatuses(pods)
	e.runtime.CleanupOrphanedPod(pods)

	err = e.cleanupOrphanedPodDirs(pods, containerRunningPods)
	if err != nil {
		return fmt.Errorf("Failed cleaning up orphaned pod directories: %s", err.Error())
	}
	return nil
}
