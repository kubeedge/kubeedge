package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	oteltrace "go.opentelemetry.io/otel/trace"
	"k8s.io/apimachinery/pkg/api/errors"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/proxy"
	"k8s.io/apimachinery/pkg/watch"
	apirequest "k8s.io/apiserver/pkg/endpoints/request"
	genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/storage"
	runtimeapi "k8s.io/cri-api/pkg/apis/runtime/v1"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/kubelet/cri/remote"

	"github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/edge/pkg/edged/config"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/agent"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/common"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/storage/restful"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/storage/sqlite"
	"github.com/kubeedge/kubeedge/edge/pkg/metamanager/metaserver/kubernetes/storage/sqlite/imitator"
	"github.com/kubeedge/kubeedge/pkg/metaserver"
	"github.com/kubeedge/kubeedge/pkg/metaserver/util"
)

// REST implements a RESTStorage for all resource against imitator.
type REST struct {
	*genericregistry.Store
	*agent.Agent
}

type responder struct{}

func (r *responder) Error(w http.ResponseWriter, _ *http.Request, err error) {
	klog.ErrorS(err, "Error while proxying request")
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

// NewREST returns a RESTStorage object that will work against all resources
func NewREST() (*REST, error) {
	store := &genericregistry.Store{
		NewFunc:                  func() runtime.Object { return &unstructured.Unstructured{} },
		NewListFunc:              func() runtime.Object { return &unstructured.UnstructuredList{} },
		DefaultQualifiedResource: schema.GroupResource{},

		KeyFunc:     metaserver.KeyFuncReq,
		KeyRootFunc: metaserver.KeyRootFunc,

		CreateStrategy: nil,
		UpdateStrategy: nil,
		DeleteStrategy: nil,

		TableConvertor:   nil,
		StorageVersioner: nil,
		Storage:          genericregistry.DryRunnableStorage{},
	}
	store.PredicateFunc = func(label labels.Selector, field fields.Selector) storage.SelectionPredicate {
		return storage.SelectionPredicate{
			Label:    label,
			Field:    field,
			GetAttrs: util.UnstructuredAttr,
		}
	}

	store.Storage.Storage = sqlite.New()
	store.Storage.Codec = unstructured.UnstructuredJSONScheme

	return &REST{store, agent.DefaultAgent}, nil
}

// decorateList set list's gvk if it's gvk is empty
func decorateList(ctx context.Context, list runtime.Object) {
	info, ok := apirequest.RequestInfoFrom(ctx)
	if ok && list.GetObjectKind().GroupVersionKind().Empty() {
		gvk := schema.GroupVersionKind{
			Group:   info.APIGroup,
			Version: info.APIVersion,
			Kind:    util.UnsafeResourceToKind(info.Resource) + "List",
		}
		list.GetObjectKind().SetGroupVersionKind(gvk)
	}
}

func (r *REST) Get(ctx context.Context, _ string, options *metav1.GetOptions) (runtime.Object, error) {
	info, _ := apirequest.RequestInfoFrom(ctx)
	// First try to get the object from remote cloud
	obj, err := func() (runtime.Object, error) {
		app, err := r.Agent.Generate(ctx, metaserver.Get, *options, nil)
		if err != nil {
			klog.Errorf("[metaserver/reststorage] failed to generate application: %v", err)
			return nil, err
		}
		err = r.Agent.Apply(app)
		defer app.Close()
		if err != nil {
			klog.Errorf("[metaserver/reststorage] failed to get obj from cloud: %v", err)
			return nil, err
		}
		var obj = new(unstructured.Unstructured)
		err = json.Unmarshal(app.RespBody, obj)
		if err != nil {
			return nil, err
		}
		// save to local, ignore error
		if err := imitator.DefaultV2Client.InsertOrUpdateObj(context.TODO(), obj); err != nil {
			klog.V(3).Infof("failed to save obj to metav2, err: %v", err)
		}
		klog.Infof("[metaserver/reststorage] successfully process get req (%v) through cloud", info.Path)
		return obj, nil
	}()

	// If we get object from cloud failed, try to get the object from the local metaManager
	if err != nil {
		obj, err = r.Store.Get(ctx, "", options) // name is needless, we get all key information from ctx
		if err != nil {
			return nil, errors.NewNotFound(schema.GroupResource{Group: info.APIGroup, Resource: info.Resource}, info.Name)
		}
		klog.Infof("[metaserver/reststorage] successfully process get req (%v) at local", info.Path)
	}
	return obj, err
}

// PassThrough
// The request is routed to the dynamic controller via the metaServer.
// If the request is approved, the response will be saved to local storage.
// It will be acquired from local data storage if it fails.
func (r *REST) PassThrough(ctx context.Context, options *metav1.GetOptions) ([]byte, error) {
	info, _ := apirequest.RequestInfoFrom(ctx)
	resp, err := func() ([]byte, error) {
		app, err := r.Agent.Generate(ctx, metaserver.ApplicationVerb(info.Verb), *options, nil)
		if err != nil {
			klog.Errorf("[metaserver/passThrough] failed to generate application: %v", err)
			return nil, err
		}
		err = r.Agent.Apply(app)
		defer app.Close()
		if err != nil {
			klog.Errorf("[metaserver/passThrough] failed to request from cloud: %v", err)
			return nil, err
		}

		err = imitator.DefaultV2Client.InsertOrUpdatePassThroughObj(context.TODO(), app.RespBody, app.Key)
		if err != nil {
			klog.Warningf("[metaserver/passThrough] failed to insert version information into database: %v", err)
		}
		return app.RespBody, nil
	}()
	if err != nil {
		resp, err = imitator.DefaultV2Client.GetPassThroughObj(ctx, info.Path)
		if err != nil {
			klog.Errorf("[metaserver/reststorage] failed to get req at local: %v", err)
			return nil, errors.NewNotFound(schema.GroupResource{Group: info.APIGroup, Resource: info.Resource}, info.Name)
		}
		klog.Infof("[metaserver/reststorage] successfully process get req (%v) at local", info.Path)
	}

	klog.Infof("[metaserver/passThrough] successfully process request (%v)", info.Path)
	return resp, nil
}

func (r *REST) List(ctx context.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
	info, _ := apirequest.RequestInfoFrom(ctx)
	// First try to list the object from remote cloud
	list, err := func() (runtime.Object, error) {
		app, err := r.Agent.Generate(ctx, metaserver.List, *options, nil)
		if err != nil {
			klog.Errorf("[metaserver/reststorage] failed to generate application: %v", err)
			return nil, err
		}
		err = r.Agent.Apply(app)
		defer app.Close()
		if err != nil {
			klog.Errorf("[metaserver/reststorage] failed to list obj from cloud: %v", err)
			return nil, err
		}
		var list = new(unstructured.UnstructuredList)
		err = json.Unmarshal(app.RespBody, list)
		if err != nil {
			return nil, err
		}
		// imitator.DefaultV2Client.InsertOrUpdateObj(context.TODO(), list)
		klog.Infof("[metaserver/reststorage] successfully process list req (%v) through cloud", info.Path)
		return list, nil
	}()

	// If we list object from cloud failed, try to list the object from the local metaManager
	if err != nil {
		list, err = r.Store.List(ctx, options)
		if err != nil {
			return nil, err
		}
		klog.Infof("[metaserver/reststorage] successfully process list req (%v) at local", info.Path)
	}

	decorateList(ctx, list)
	return list, err
}

func (r *REST) Watch(ctx context.Context, options *metainternalversion.ListOptions) (watch.Interface, error) {
	info, _ := apirequest.RequestInfoFrom(ctx)

	// First try watch from remote cloud
	_, err := func() (runtime.Object, error) {
		app, err := r.Agent.Generate(ctx, metaserver.Watch, *options, nil)
		if err != nil {
			klog.Errorf("[metaserver/reststorage] failed to generate application: %v", err)
			return nil, err
		}
		err = r.Agent.Apply(app)
		// For watch long connection request, we close the application when the watch is closed.
		if err != nil {
			klog.Errorf("[metaserver/reststorage] failed to apply for a watch listener from cloud: %v", err)
			app.Close()
			return nil, errors.NewInternalError(err)
		}

		ctx = util.WithApplicationID(ctx, app.ID)
		klog.Infof("[metaserver/reststorage] successfully apply for a watch listener (%v) through cloud", info.Path)
		return nil, nil
	}()

	// If we watch object from cloud failed, try to watch the object from the local metaManager
	if err != nil {
		klog.Errorf("[metaserver/reststorage] failed to get a approved application for watch(%v) from cloud application center, %v", info.Path, err)
	}

	return r.Store.Watch(ctx, options)
}

func (r *REST) Create(ctx context.Context, obj runtime.Object, _ rest.ValidateObjectFunc, options *metav1.CreateOptions) (runtime.Object, error) {
	obj, err := func() (runtime.Object, error) {
		app, err := r.Agent.Generate(ctx, metaserver.Create, *options, obj)
		if err != nil {
			klog.Errorf("[metaserver/reststorage] failed to generate application: %v", err)
			return nil, err
		}
		err = r.Agent.Apply(app)
		defer app.Close()
		if err != nil {
			klog.Errorf("[metaserver/reststorage] failed to create obj: %v", err)
			return nil, err
		}

		retObj := new(unstructured.Unstructured)
		if err := json.Unmarshal(app.RespBody, retObj); err != nil {
			return nil, err
		}
		return retObj, nil
	}()

	if err != nil {
		klog.Errorf("[metaserver/reststorage] failed to create (%v)", metaserver.KeyFunc(obj))
		return nil, err
	}

	klog.Infof("[metaserver/reststorage] successfully create (%v)", metaserver.KeyFunc(obj))
	return obj, nil
}

func (r *REST) Delete(ctx context.Context, _ string, _ rest.ValidateObjectFunc, options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	key, _ := metaserver.KeyFuncReq(ctx, "")
	app, err := r.Agent.Generate(ctx, metaserver.Delete, options, nil)
	if err != nil {
		klog.Errorf("[metaserver/reststorage] failed to generate application: %v", err)
		return nil, false, err
	}
	err = r.Agent.Apply(app)
	defer app.Close()
	if err != nil {
		klog.Errorf("[metaserver/reststorage] failed to delete (%v) through cloud", key)
		return nil, false, err
	}
	klog.Infof("[metaserver/reststorage] successfully delete (%v) through cloud", key)
	return nil, true, nil
}

func (r *REST) Update(ctx context.Context, _ string, objInfo rest.UpdatedObjectInfo, _ rest.ValidateObjectFunc, _ rest.ValidateObjectUpdateFunc, _ bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	obj, err := objInfo.UpdatedObject(ctx, nil)
	if err != nil {
		return nil, false, errors.NewInternalError(err)
	}

	reqInfo, _ := apirequest.RequestInfoFrom(ctx)
	var app *metaserver.Application
	if reqInfo.Subresource == "status" {
		app, err = r.Agent.Generate(ctx, metaserver.UpdateStatus, options, obj)
	} else {
		app, err = r.Agent.Generate(ctx, metaserver.Update, options, obj)
	}
	if err != nil {
		klog.Errorf("[metaserver/reststorage] failed to generate application: %v", err)
		return nil, false, err
	}
	defer app.Close()
	if err := r.Agent.Apply(app); err != nil {
		return nil, false, err
	}
	retObj := new(unstructured.Unstructured)
	if err := json.Unmarshal(app.RespBody, retObj); err != nil {
		return nil, false, errors.NewInternalError(err)
	}
	return retObj, false, nil
}

func (r *REST) Patch(ctx context.Context, pi metaserver.PatchInfo) (runtime.Object, error) {
	app, err := r.Agent.Generate(ctx, metaserver.Patch, pi, nil)
	if err != nil {
		klog.Errorf("[metaserver/reststorage] failed to generate application: %v", err)
		return nil, err
	}
	defer app.Close()
	if err := r.Agent.Apply(app); err != nil {
		return nil, err
	}
	retObj := new(unstructured.Unstructured)
	if err := json.Unmarshal(app.RespBody, retObj); err != nil {
		return nil, errors.NewInternalError(err)
	}
	return retObj, nil
}

func (r *REST) Restart(ctx context.Context, restartInfo common.RestartInfo) *types.RestartResponse {
	namespace := restartInfo.Namespace
	podNames := restartInfo.PodNames
	restartResponse := &types.RestartResponse{
		ErrMessages: make([]string, 0),
		LogMessages: make([]string, 0),
	}

	endpoint := config.Config.Edged.TailoredKubeletConfig.ContainerRuntimeEndpoint
	remoteRuntimeService, err := remote.NewRemoteRuntimeService(endpoint, time.Second*10, oteltrace.NewNoopTracerProvider())
	if err != nil {
		errMessage := fmt.Sprintf("new remote runtimeservice with err: %v", err)
		klog.Errorf("[metaserver/restart] %v", errMessage)
		restartResponse.ErrMessages = append(restartResponse.ErrMessages, errMessage)
		return restartResponse
	}
	for _, podName := range podNames {
		labelSelector := map[string]string{
			"io.kubernetes.pod.name":      podName,
			"io.kubernetes.pod.namespace": namespace,
		}

		filter := &runtimeapi.ContainerFilter{
			LabelSelector: labelSelector,
		}
		containers, err := remoteRuntimeService.ListContainers(ctx, filter)
		if err != nil {
			errMessage := fmt.Sprintf("failed to list containers: %v", err)
			klog.Warningf("[metaserver/restart] %v", errMessage)
			restartResponse.ErrMessages = append(restartResponse.ErrMessages, errMessage)
			continue
		}

		if len(containers) == 0 {
			errMessage := fmt.Sprintf("not found pod:\"/%s/%s\"", namespace, podName)
			klog.Warningf("[metaserver/restart] %v", errMessage)
			restartResponse.ErrMessages = append(restartResponse.ErrMessages, errMessage)
			continue
		}

		count := 0
		var errMessage string
		for _, container := range containers {
			containerID := container.Id
			err := remoteRuntimeService.StopContainer(ctx, containerID, 3)
			if err != nil {
				errMessage += fmt.Sprintf("failed to stop container %s for pod \"/%s/%s\" with err:%v\n", container.Metadata.Name, namespace, podName, err)
			} else {
				count++
			}
		}

		if count == len(containers) {
			message := fmt.Sprintf("the pod \"%s/%s\" restart successful", namespace, podName)
			klog.V(4).Infof("[metaserver/restart] %v", message)
			restartResponse.LogMessages = append(restartResponse.LogMessages, message)
		} else {
			klog.Warningf("[metaserver/restart] %v", errMessage)
			restartResponse.ErrMessages = append(restartResponse.ErrMessages, errMessage)
		}
	}
	return restartResponse
}

// Get logs from container through edged API.
func (r *REST) Logs(ctx context.Context, logsInfo common.LogsInfo) (*types.LogsResponse, *http.Response) {
	nameSpace := logsInfo.Namespace
	podName := logsInfo.PodName

	logsResponse := &types.LogsResponse{
		ErrMessages: make([]string, 0),
		LogMessages: make([]string, 0),
	}

	if !config.Config.Edged.Enable {
		errMessage := "edged is not enabled"
		klog.Errorf("[metaserver/logs] %v", errMessage)
		logsResponse.ErrMessages = append(logsResponse.ErrMessages, errMessage)
		return logsResponse, nil
	}

	endpoint := config.Config.Edged.TailoredKubeletConfig.ContainerRuntimeEndpoint
	remoteRuntimeService, err := remote.NewRemoteRuntimeService(endpoint, time.Second*10, oteltrace.NewNoopTracerProvider())
	if err != nil {
		errMessage := fmt.Sprintf("new remote runtimeservice with err: %v", err)
		klog.Errorf("[metaserver/logs] %v", errMessage)
		logsResponse.ErrMessages = append(logsResponse.ErrMessages, errMessage)
		return logsResponse, nil
	}

	labelSelector := map[string]string{
		"io.kubernetes.pod.name":      podName,
		"io.kubernetes.pod.namespace": nameSpace,
	}

	filter := &runtimeapi.ContainerFilter{
		LabelSelector: labelSelector,
	}

	containers, err := remoteRuntimeService.ListContainers(ctx, filter)
	if err != nil {
		errMessage := fmt.Sprintf("failed to list containers: %v", err)
		klog.Warningf("[metaserver/logs] %v", errMessage)
		logsResponse.ErrMessages = append(logsResponse.ErrMessages, errMessage)
		return logsResponse, nil
	}

	if len(containers) == 0 {
		errMessage := fmt.Sprintf("not found pod:\"/%s/%s\"", nameSpace, podName)
		klog.Warningf("[metaserver/logs] %v", errMessage)
		logsResponse.ErrMessages = append(logsResponse.ErrMessages, errMessage)
		return logsResponse, nil
	}

	var container string

	if logsInfo.ContainerName != "" {
		for _, c := range containers {
			if c.Metadata.Name == logsInfo.ContainerName {
				container = c.Metadata.Name
				break
			}
		}
	} else {
		container = containers[0].Metadata.Name
	}

	req := restful.LogsRequest(nameSpace, podName, container, logsInfo)
	res, err := req.RestfulRequest()
	if err != nil {
		errMessage := fmt.Sprintf("failed to get logs for container %s for pod \"/%s/%s\" with err:%v", container, nameSpace, podName, err)
		klog.Warningf("[metaserver/logs] %v", errMessage)
		logsResponse.ErrMessages = append(logsResponse.ErrMessages, errMessage)
	}

	return logsResponse, res
}

// Exec command in container through edged API.
// Return http.Handler for exec when stdin is a tty.
func (r *REST) Exec(ctx context.Context, execInfo common.ExecInfo) (*types.ExecResponse, http.Handler) {
	nameSpace := execInfo.Namespace
	podName := execInfo.PodName
	container := execInfo.Container
	commands := execInfo.Commands
	stdin := execInfo.Stdin
	stdout := execInfo.Stdout
	stderr := execInfo.Stderr
	tty := execInfo.TTY

	execResponse := &types.ExecResponse{
		ErrMessages:    make([]string, 0),
		RunOutMessages: make([]string, 0),
		RunErrMessages: make([]string, 0),
	}

	if !config.Config.Edged.Enable {
		errMessage := "edged is not enabled"
		klog.Errorf("[metaserver/exec] %v", errMessage)
		execResponse.ErrMessages = append(execResponse.ErrMessages, errMessage)
		return execResponse, nil
	}

	if commands == nil {
		errMessage := "You must specify at least one command for the container"
		klog.Errorf("[metaserver/exec] %v", errMessage)
		execResponse.ErrMessages = append(execResponse.ErrMessages, errMessage)
		return execResponse, nil
	}

	endpoint := config.Config.Edged.TailoredKubeletConfig.ContainerRuntimeEndpoint
	remoteRuntimeService, err := remote.NewRemoteRuntimeService(endpoint, time.Second*10, oteltrace.NewNoopTracerProvider())
	if err != nil {
		errMessage := fmt.Sprintf("new remote runtimeservice with err: %v", err)
		klog.Errorf("[metaserver/exec] %v", errMessage)
		execResponse.ErrMessages = append(execResponse.ErrMessages, errMessage)
		return execResponse, nil
	}

	labelSelector := map[string]string{
		"io.kubernetes.pod.name":      podName,
		"io.kubernetes.pod.namespace": nameSpace,
	}

	filter := &runtimeapi.ContainerFilter{
		LabelSelector: labelSelector,
	}

	containers, err := remoteRuntimeService.ListContainers(ctx, filter)
	if err != nil {
		errMessage := fmt.Sprintf("failed to list containers: %v", err)
		klog.Warningf("[metaserver/exec] %v", errMessage)
		execResponse.ErrMessages = append(execResponse.ErrMessages, errMessage)
		return execResponse, nil
	}

	if len(containers) == 0 {
		errMessage := fmt.Sprintf("not found pod:\"/%s/%s\"", nameSpace, podName)
		klog.Warningf("[metaserver/exec] %v", errMessage)
		execResponse.ErrMessages = append(execResponse.ErrMessages, errMessage)
		return execResponse, nil
	}

	var execContainer *runtimeapi.Container
	var execContainerID string

	if container == "" {
		if len(containers) > 1 {
			errMessage := fmt.Sprintf("more than one container in pod:\"/%s/%s\"", nameSpace, podName)
			klog.Warningf("[metaserver/exec] %v", errMessage)
			execResponse.ErrMessages = append(execResponse.ErrMessages, errMessage)
			return execResponse, nil
		}
		execContainer = containers[0]
		execContainerID = execContainer.Id
	} else {
		for _, c := range containers {
			if c.Metadata.Name == container {
				execContainer = c
				execContainerID = execContainer.Id
				break
			}
		}
		if execContainer == nil {
			errMessage := fmt.Sprintf("not found container %s in pod:\"/%s/%s\"", container, nameSpace, podName)
			klog.Warningf("[metaserver/exec] %v", errMessage)
			execResponse.ErrMessages = append(execResponse.ErrMessages, errMessage)
			return execResponse, nil
		}
	}

	if !tty {
		stdout, stderr, err := remoteRuntimeService.ExecSync(ctx, execContainerID, commands, time.Second*10)
		if err != nil {
			errMessage := fmt.Sprintf("failed to exec command %s for container %s for pod \"/%s/%s\" with err:%v", commands, execContainer.Metadata.Name, nameSpace, podName, err)
			klog.Warningf("[metaserver/exec] %v", errMessage)
			execResponse.ErrMessages = append(execResponse.ErrMessages, errMessage)
			return execResponse, nil
		}

		outString := string(stdout)
		errString := string(stderr)

		if outString != "" {
			execResponse.RunOutMessages = append(execResponse.RunOutMessages, outString)
			execResponse.RunErrMessages = append(execResponse.RunErrMessages, errString)
			return execResponse, nil
		}
	} else {
		res, err := remoteRuntimeService.Exec(ctx, &runtimeapi.ExecRequest{
			ContainerId: execContainerID,
			Cmd:         commands,
			Stdin:       stdin,
			Stdout:      stdout,
			Stderr:      stderr,
			Tty:         true,
		})
		if err != nil {
			errMessage := fmt.Sprintf("failed to exec command %s for container %s for pod \"/%s/%s\" with err:%v", commands, execContainer.Metadata.Name, nameSpace, podName, err)
			klog.Warningf("[metaserver/exec] %v", errMessage)
			execResponse.ErrMessages = append(execResponse.ErrMessages, errMessage)
			return execResponse, nil
		}

		execURL, err := url.Parse(res.Url)
		if err != nil {
			errMessage := fmt.Sprintf("failed to parse exec url with err:%v", err)
			klog.Warningf("[metaserver/exec] %v", errMessage)
			execResponse.ErrMessages = append(execResponse.ErrMessages, errMessage)
			return execResponse, nil
		}

		handler := proxy.NewUpgradeAwareHandler(execURL, nil, false, true, &responder{})
		return execResponse, handler
	}
	return execResponse, nil
}
