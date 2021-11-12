/*
Copyright 2018 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package nodeleasecontroller

import (
	"context"
	"fmt"
	"time"

	coordinationv1 "k8s.io/api/coordination/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
	coordclientset "k8s.io/client-go/kubernetes/typed/coordination/v1"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"
	"k8s.io/utils/pointer"

	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/client"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/common/constants"
	"github.com/kubeedge/kubeedge/pkg/apis/componentconfig/cloudcore/v1alpha1"
)

const (
	// renewIntervalFraction is the fraction of lease duration to renew the lease
	renewIntervalFraction = 0.25
	// maxUpdateRetries is the number of retries the nodelease controller will attempt to
	// update the nodelease after receiving the heartbeat messages from edge nodes.
	maxUpdateRetries = 5
	// maxBackoff is the maximum sleep time during backoff (e.g. in backoffEnsureLease)
	maxBackoff = 7 * time.Second
)

type HeartbeatMsg struct {
	Timestamp string
	NodeName  string
}

type nodeHeartbeat struct {
	updateRetryTimes int
	HeartbeatMsg
}

// parse will parse the HeartbeatMsg to get nodeName and timestamp.
func (n *nodeHeartbeat) parse(timeLayout string) (string, *time.Time, error) {
	nodeName := n.NodeName
	heartbeatTimestamp, err := time.Parse(timeLayout, string(n.Timestamp))
	if err != nil {
		return nodeName, nil, fmt.Errorf("failed to parse heartbeatMsg: %s, node: %s", n.Timestamp, nodeName)
	}
	return nodeName, &heartbeatTimestamp, nil
}

type NodeLeaseController struct {
	enable        bool
	client        clientset.Interface
	leaseClient   coordclientset.LeaseInterface
	heartbeatCh   chan *nodeHeartbeat
	ensureRetryCh chan *nodeHeartbeat

	// leaseDurationSeconds       int32
	renewInterval       time.Duration
	retryHeartbeatQueue workqueue.DelayingInterface
	retryEnsureQueue    workqueue.RateLimitingInterface
	timeLayout          string

	// latestLease is a map contains the latest node lease of each edge node
	// which nodelease controller updated or created.
	latestLease map[string]*coordinationv1.Lease
}

func Register(c *v1alpha1.NodeLeaseController) {
	core.Register(NewNodeLeaseController(c.Enable))
}

func (c *NodeLeaseController) Name() string {
	return modules.NodeLeaseControllerModuleName
}

func (c *NodeLeaseController) Group() string {
	return modules.NodeLeaseControllerModuleGroup
}

func (c *NodeLeaseController) Enable() bool {
	return c.enable
}

// NewNodeLeaseController constructs and returns a nodelease controller
func NewNodeLeaseController(enable bool) *NodeLeaseController {
	client := client.GetKubeClient()
	leaseClient := client.CoordinationV1().Leases(corev1.NamespaceNodeLease)
	nodeLimit := v1alpha1.NewDefaultCloudCoreConfig().Modules.CloudHub.NodeLimit
	return &NodeLeaseController{
		enable:        enable,
		client:        client,
		leaseClient:   leaseClient,
		heartbeatCh:   make(chan *nodeHeartbeat, nodeLimit),
		ensureRetryCh: make(chan *nodeHeartbeat, nodeLimit),
		timeLayout:    constants.DefaultTimestampLayout, // edgehub also reports its heartbeat timestamp in DefaultTimestampLayout
		// TODO: set renewInterval according to heartbeat time
		renewInterval:       time.Duration(renewIntervalFraction * float64(10*time.Second)),
		retryHeartbeatQueue: workqueue.NewDelayingQueue(),
		retryEnsureQueue:    workqueue.NewRateLimitingQueue(workqueue.NewItemExponentialFailureRateLimiter(100*time.Millisecond, maxBackoff)),
		latestLease:         make(map[string]*coordinationv1.Lease),
	}
}

// Start runs the controller
func (c *NodeLeaseController) Start() {
	if c.leaseClient == nil {
		klog.Infof("node lease controller has nil lease client, will not claim or renew leases")
		return
	}

	klog.Info("Starting NodeLease Controller")

	go c.processMessage()
	go c.sync()
	// retry updatelease
	go func() {
		for {
			retryCase, shutdown := c.retryHeartbeatQueue.Get()
			if shutdown {
				klog.Infof("stop retry updatelease for nodelease controller")
				return
			}
			c.heartbeatCh <- retryCase.(*nodeHeartbeat)
		}
	}()

	// retry ensurelease
	go func() {
		for {
			retryCase, shutdown := c.retryEnsureQueue.Get()
			if shutdown {
				klog.Infof("stop retry ensurelease for nodelease controller")
				return
			}
			c.ensureRetryCh <- retryCase.(*nodeHeartbeat)
		}
	}()
}

func (c *NodeLeaseController) processMessage() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Info("stop processMessage of nodelease controller")
			return
		default:
		}
		msg, err := beehiveContext.Receive(modules.NodeLeaseControllerModuleName)
		if err != nil {
			klog.Warningf("nodelease controller processMessage reveive message failed, %s", err)
			continue
		}
		heartbeat := msg.GetContent().(HeartbeatMsg)
		klog.V(4).Infof("nodelease controller get heartbeat message: %v", heartbeat)
		c.heartbeatCh <- &nodeHeartbeat{
			updateRetryTimes: 0,
			HeartbeatMsg:     heartbeat,
		}
	}
}

func (c *NodeLeaseController) sync() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Infof("node lease controller stopped")
			c.retryHeartbeatQueue.ShutDown()
			c.retryEnsureQueue.ShutDown()
			return

		case nodeHeartbeat := <-c.heartbeatCh:
			// try to update nodelease
			nodeName, heartbeatTimestamp, err := nodeHeartbeat.parse(c.timeLayout)
			if err != nil {
				klog.Errorf("failed to parse heartbeat, %v, ignore the msg", err)
				continue
			}

			// filter out nodeHearbeat which has reached its maxRetryTimes.
			if !c.needUpdate(nodeHeartbeat) {
				continue
			}

			latestLease, ok := c.latestLease[nodeName]
			if ok && latestLease != nil {
				// As long as node lease is not (or very rarely) updated by any other agent than the nodelease controller,
				// we can optimistically assume it didn't change since our last update and try updating
				// based on the version from that time. Thanks to it we avoid GET call and reduce load
				// on etcd and kube-apiserver.
				// If at some point other agents will also be frequently updating the Lease object, this
				// can result in performance degradation, because we will end up with calling additional
				// GET/PUT - at this point this whole "if" should be removed.
				err := c.updateLease(nodeName, latestLease, heartbeatTimestamp)
				if err == nil {
					continue
				}
				klog.Errorf("failed to update lease using latest lease, fallback to ensure lease, err: %v", err)
			}

			c.ensureLeaseWithRetry(nodeHeartbeat)

		case nodeHeartbeat := <-c.ensureRetryCh:
			// ensureRetry workqueue has different rate from heartbeatRetry workqueue.
			// Thus, we process it in another case.
			c.ensureLeaseWithRetry(nodeHeartbeat)
		}
	}
}

// This function will ensure that the nodelease exists in APIServer and the local latestLease
// has the same resourceVersion as the corresponding nodeLease in APIServer. When it finds
// both conditions are satisfied, it will try to update the nodelease once. If any error occurs,
// it will retry operations with different strategies:
// 1. ensureLease failed: exponentially increasing waits
// 2. updateLease failed: fixed interval waits with max retry times
//
// When it comes to ensureLeaseWithRetry, there're two possible cases:
// 1) the corresponding nodelease does not exsit in APIServer
// 2) the local lastestLease has different resourceVersion from the nodelease in APIServer, which
// means the nodelease was changed by someone else. It's optimisticLockError and requires getting
// the newer version of lease to proceed.
func (c *NodeLeaseController) ensureLeaseWithRetry(heartbeatCase *nodeHeartbeat) {
	nodeName, heartbeatTimestamp, err := heartbeatCase.parse(c.timeLayout)
	if err != nil {
		klog.Errorf("unknown format of nodeHeartbeat message, drop it: %v", err)
		return
	}

	lease, created, err := c.backoffEnsureLease(nodeName, heartbeatTimestamp)
	if err != nil {
		klog.Errorf("sync heartbeat for node: %s failed, timestamp: %s, retry ensure at next time", nodeName, heartbeatTimestamp)
		c.retryEnsureQueue.AddRateLimited(heartbeatCase)
		return
	}

	// Now, we ensure that the nodelease of the node exists and make lastestLease up to date.
	c.latestLease[nodeName] = lease
	// we don't need to update the lease if we just created it
	if !created && lease != nil {
		if err := c.updateLease(nodeName, lease, heartbeatTimestamp); err != nil {
			// Here we've ensured the nodelease and tried to update again, and finally failed.
			// Now there's nothing we can do for it, so just retry at the next time.
			klog.Errorf("failed to update nodelease for %s, %v, will retry after %v", nodeName, err, c.renewInterval)
			heartbeatCase.updateRetryTimes++
			c.retryHeartbeatQueue.AddAfter(heartbeatCase, c.renewInterval)
			return
		}
	}
}

func (c *NodeLeaseController) needUpdate(updateCase *nodeHeartbeat) bool {
	// TODO： consider that if we need to filter out expired heartbeat msg
	return updateCase.updateRetryTimes <= maxUpdateRetries
}

// backoffEnsureLease attempts to create the lease if it does not exist.
// Returns the lease, and true if this call created the lease,
// false otherwise.
func (c *NodeLeaseController) backoffEnsureLease(nodeName string, heartbeatTimestamp *time.Time) (*coordinationv1.Lease, bool, error) {
	var (
		lease   *coordinationv1.Lease
		created bool
		err     error
	)

	lease, created, err = c.ensureLease(nodeName, heartbeatTimestamp)
	if err == nil {
		return lease, created, nil
	}

	return lease, created, fmt.Errorf("failed to ensure node lease exists, will retry at next time, error: %v", err)
}

// ensureLease creates the lease if it does not exist. Returns the lease and
// a bool (true if this call created the lease), or any error that occurs.
func (c *NodeLeaseController) ensureLease(nodeName string, heartbeatTimestamp *time.Time) (*coordinationv1.Lease, bool, error) {
	lease, err := c.leaseClient.Get(context.TODO(), nodeName, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		// lease does not exist, create it.
		leaseToCreate := c.newLease(nodeName, nil, heartbeatTimestamp)
		if leaseToCreate == nil {
			return nil, false, fmt.Errorf("failed to get new lease, ensureLease failed for node: %s", nodeName)
		}

		if len(leaseToCreate.OwnerReferences) == 0 {
			// We want to ensure that a lease will always have OwnerReferences set.
			// Thus, given that we weren't able to set it correctly, we simply
			// not create it this time - we will retry in the next iteration.
			return nil, false, nil
		}
		lease, err := c.leaseClient.Create(context.TODO(), leaseToCreate, metav1.CreateOptions{})
		if err != nil {
			return nil, false, err
		}
		return lease, true, nil
	} else if err != nil {
		// unexpected error getting lease
		return nil, false, err
	}
	// lease already existed
	return lease, false, nil
}

// updateLease attempts to update the lease
// call this once you're sure the lease has been created
func (c *NodeLeaseController) updateLease(nodeName string, base *coordinationv1.Lease, heartbeatTimestamp *time.Time) error {
	lease, err := c.leaseClient.Update(context.TODO(), c.newLease(nodeName, base, heartbeatTimestamp), metav1.UpdateOptions{})
	if err == nil {
		c.latestLease[nodeName] = lease
		return nil
	}
	klog.Errorf("failed to update node lease, error: %v", err)

	return fmt.Errorf("failed %d attempts to update node lease", maxUpdateRetries)
}

// newLease constructs a new lease if base is nil, or returns a copy of base
// with desired state asserted on the copy.
func (c *NodeLeaseController) newLease(nodeName string, base *coordinationv1.Lease, heartbeatTimestamp *time.Time) *coordinationv1.Lease {
	// Use the bare minimum set of fields; other fields exist for debugging/legacy,
	// but we don't need to make node heartbeats more complicated by using them.
	var lease *coordinationv1.Lease
	if base == nil {
		lease = &coordinationv1.Lease{
			ObjectMeta: metav1.ObjectMeta{
				Name:      nodeName,
				Namespace: corev1.NamespaceNodeLease,
			},
			Spec: coordinationv1.LeaseSpec{
				HolderIdentity: pointer.StringPtr(nodeName),
			},
		}
	} else {
		lease = base.DeepCopy()
	}

	lease.Spec.RenewTime = &metav1.MicroTime{Time: *heartbeatTimestamp}

	// Setting owner reference needs node's UID.
	if len(lease.OwnerReferences) == 0 {
		if node, err := c.client.CoreV1().Nodes().Get(context.TODO(), nodeName, metav1.GetOptions{}); err == nil {
			lease.OwnerReferences = []metav1.OwnerReference{
				{
					APIVersion: corev1.SchemeGroupVersion.WithKind("Node").Version,
					Kind:       corev1.SchemeGroupVersion.WithKind("Node").Kind,
					Name:       nodeName,
					UID:        node.UID,
				},
			}
		} else {
			klog.Errorf("failed to get node %q when trying to set owner ref to the node lease: %v", nodeName, err)
		}
	}

	return lease
}
