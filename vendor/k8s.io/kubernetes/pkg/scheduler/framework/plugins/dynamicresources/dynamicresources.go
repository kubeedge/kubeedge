/*
Copyright 2022 The Kubernetes Authors.

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

package dynamicresources

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/google/go-cmp/cmp"

	v1 "k8s.io/api/core/v1"
	resourcev1alpha2 "k8s.io/api/resource/v1alpha2"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	resourcev1alpha2listers "k8s.io/client-go/listers/resource/v1alpha2"
	corev1helpers "k8s.io/component-helpers/scheduling/corev1"
	"k8s.io/component-helpers/scheduling/corev1/nodeaffinity"
	"k8s.io/dynamic-resource-allocation/resourceclaim"
	"k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/scheduler/framework"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/feature"
	"k8s.io/kubernetes/pkg/scheduler/framework/plugins/names"
	schedutil "k8s.io/kubernetes/pkg/scheduler/util"
)

const (
	// Name is the name of the plugin used in Registry and configurations.
	Name = names.DynamicResources

	stateKey framework.StateKey = Name
)

// The state is initialized in PreFilter phase. Because we save the pointer in
// framework.CycleState, in the later phases we don't need to call Write method
// to update the value
type stateData struct {
	// A copy of all claims for the Pod (i.e. 1:1 match with
	// pod.Spec.ResourceClaims), initially with the status from the start
	// of the scheduling cycle. Each claim instance is read-only because it
	// might come from the informer cache. The instances get replaced when
	// the plugin itself successfully does an Update.
	//
	// Empty if the Pod has no claims.
	claims []*resourcev1alpha2.ResourceClaim

	// The AvailableOnNodes node filters of the claims converted from the
	// v1 API to nodeaffinity.NodeSelector by PreFilter for repeated
	// evaluation in Filter. Nil for claims which don't have it.
	availableOnNodes []*nodeaffinity.NodeSelector

	// The indices of all claims that:
	// - are allocated
	// - use delayed allocation
	// - were not available on at least one node
	//
	// Set in parallel during Filter, so write access there must be
	// protected by the mutex. Used by PostFilter.
	unavailableClaims sets.Int

	// A pointer to the PodSchedulingContext object for the pod, if one exists.
	// Gets set on demand.
	//
	// Conceptually, this object belongs into the scheduler framework
	// where it might get shared by different plugins. But in practice,
	// it is currently only used by dynamic provisioning and thus
	// managed entirely here.
	schedulingCtx *resourcev1alpha2.PodSchedulingContext

	// podSchedulingDirty is true if the current copy was locally modified.
	podSchedulingDirty bool

	mutex sync.Mutex
}

func (d *stateData) Clone() framework.StateData {
	return d
}

func (d *stateData) updateClaimStatus(ctx context.Context, clientset kubernetes.Interface, index int, claim *resourcev1alpha2.ResourceClaim) error {
	// TODO (#113700): replace with patch operation. Beware that patching must only succeed if the
	// object has not been modified in parallel by someone else.
	claim, err := clientset.ResourceV1alpha2().ResourceClaims(claim.Namespace).UpdateStatus(ctx, claim, metav1.UpdateOptions{})
	// TODO: metric for update results, with the operation ("set selected
	// node", "set PotentialNodes", etc.) as one dimension.
	if err != nil {
		return fmt.Errorf("update resource claim: %w", err)
	}

	// Remember the new instance. This is relevant when the plugin must
	// update the same claim multiple times (for example, first reserve
	// the claim, then later remove the reservation), because otherwise the second
	// update would fail with a "was modified" error.
	d.claims[index] = claim

	return nil
}

// initializePodSchedulingContext can be called concurrently. It returns an existing PodSchedulingContext
// object if there is one already, retrieves one if not, or as a last resort creates
// one from scratch.
func (d *stateData) initializePodSchedulingContexts(ctx context.Context, pod *v1.Pod, podSchedulingContextLister resourcev1alpha2listers.PodSchedulingContextLister) (*resourcev1alpha2.PodSchedulingContext, error) {
	// TODO (#113701): check if this mutex locking can be avoided by calling initializePodSchedulingContext during PreFilter.
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if d.schedulingCtx != nil {
		return d.schedulingCtx, nil
	}

	schedulingCtx, err := podSchedulingContextLister.PodSchedulingContexts(pod.Namespace).Get(pod.Name)
	switch {
	case apierrors.IsNotFound(err):
		controller := true
		schedulingCtx = &resourcev1alpha2.PodSchedulingContext{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pod.Name,
				Namespace: pod.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: "v1",
						Kind:       "Pod",
						Name:       pod.Name,
						UID:        pod.UID,
						Controller: &controller,
					},
				},
			},
		}
		err = nil
	case err != nil:
		return nil, err
	default:
		// We have an object, but it might be obsolete.
		if !metav1.IsControlledBy(schedulingCtx, pod) {
			return nil, fmt.Errorf("PodSchedulingContext object with UID %s is not owned by Pod %s/%s", schedulingCtx.UID, pod.Namespace, pod.Name)
		}
	}
	d.schedulingCtx = schedulingCtx
	return schedulingCtx, err
}

// publishPodSchedulingContext creates or updates the PodSchedulingContext object.
func (d *stateData) publishPodSchedulingContexts(ctx context.Context, clientset kubernetes.Interface, schedulingCtx *resourcev1alpha2.PodSchedulingContext) error {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	var err error
	logger := klog.FromContext(ctx)
	msg := "Updating PodSchedulingContext"
	if schedulingCtx.UID == "" {
		msg = "Creating PodSchedulingContext"
	}
	if loggerV := logger.V(6); loggerV.Enabled() {
		// At a high enough log level, dump the entire object.
		loggerV.Info(msg, "podSchedulingCtxDump", klog.Format(schedulingCtx))
	} else {
		logger.V(5).Info(msg, "podSchedulingCtx", klog.KObj(schedulingCtx))
	}
	if schedulingCtx.UID == "" {
		schedulingCtx, err = clientset.ResourceV1alpha2().PodSchedulingContexts(schedulingCtx.Namespace).Create(ctx, schedulingCtx, metav1.CreateOptions{})
	} else {
		// TODO (#113700): patch here to avoid racing with drivers which update the status.
		schedulingCtx, err = clientset.ResourceV1alpha2().PodSchedulingContexts(schedulingCtx.Namespace).Update(ctx, schedulingCtx, metav1.UpdateOptions{})
	}
	if err != nil {
		return err
	}
	d.schedulingCtx = schedulingCtx
	d.podSchedulingDirty = false
	return nil
}

// storePodSchedulingContext replaces the pod schedulingCtx object in the state.
func (d *stateData) storePodSchedulingContexts(schedulingCtx *resourcev1alpha2.PodSchedulingContext) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	d.schedulingCtx = schedulingCtx
	d.podSchedulingDirty = true
}

func statusForClaim(schedulingCtx *resourcev1alpha2.PodSchedulingContext, podClaimName string) *resourcev1alpha2.ResourceClaimSchedulingStatus {
	for _, status := range schedulingCtx.Status.ResourceClaims {
		if status.Name == podClaimName {
			return &status
		}
	}
	return nil
}

// dynamicResources is a plugin that ensures that ResourceClaims are allocated.
type dynamicResources struct {
	enabled                    bool
	fh                         framework.Handle
	clientset                  kubernetes.Interface
	claimLister                resourcev1alpha2listers.ResourceClaimLister
	classLister                resourcev1alpha2listers.ResourceClassLister
	podSchedulingContextLister resourcev1alpha2listers.PodSchedulingContextLister
}

// New initializes a new plugin and returns it.
func New(plArgs runtime.Object, fh framework.Handle, fts feature.Features) (framework.Plugin, error) {
	if !fts.EnableDynamicResourceAllocation {
		// Disabled, won't do anything.
		return &dynamicResources{}, nil
	}

	return &dynamicResources{
		enabled:                    true,
		fh:                         fh,
		clientset:                  fh.ClientSet(),
		claimLister:                fh.SharedInformerFactory().Resource().V1alpha2().ResourceClaims().Lister(),
		classLister:                fh.SharedInformerFactory().Resource().V1alpha2().ResourceClasses().Lister(),
		podSchedulingContextLister: fh.SharedInformerFactory().Resource().V1alpha2().PodSchedulingContexts().Lister(),
	}, nil
}

var _ framework.PreEnqueuePlugin = &dynamicResources{}
var _ framework.PreFilterPlugin = &dynamicResources{}
var _ framework.FilterPlugin = &dynamicResources{}
var _ framework.PostFilterPlugin = &dynamicResources{}
var _ framework.PreScorePlugin = &dynamicResources{}
var _ framework.ReservePlugin = &dynamicResources{}
var _ framework.EnqueueExtensions = &dynamicResources{}
var _ framework.PostBindPlugin = &dynamicResources{}

// Name returns name of the plugin. It is used in logs, etc.
func (pl *dynamicResources) Name() string {
	return Name
}

// EventsToRegister returns the possible events that may make a Pod
// failed by this plugin schedulable.
func (pl *dynamicResources) EventsToRegister() []framework.ClusterEventWithHint {
	if !pl.enabled {
		return nil
	}
	events := []framework.ClusterEventWithHint{
		// Allocation is tracked in ResourceClaims, so any changes may make the pods schedulable.
		{Event: framework.ClusterEvent{Resource: framework.ResourceClaim, ActionType: framework.Add | framework.Update}, QueueingHintFn: pl.isSchedulableAfterClaimChange},
		// When a driver has provided additional information, a pod waiting for that information
		// may be schedulable.
		{Event: framework.ClusterEvent{Resource: framework.PodSchedulingContext, ActionType: framework.Add | framework.Update}, QueueingHintFn: pl.isSchedulableAfterPodSchedulingContextChange},
		// A resource might depend on node labels for topology filtering.
		// A new or updated node may make pods schedulable.
		{Event: framework.ClusterEvent{Resource: framework.Node, ActionType: framework.Add | framework.UpdateNodeLabel}},
	}
	return events
}

// PreEnqueue checks if there are known reasons why a pod currently cannot be
// scheduled. When this fails, one of the registered events can trigger another
// attempt.
func (pl *dynamicResources) PreEnqueue(ctx context.Context, pod *v1.Pod) (status *framework.Status) {
	if err := pl.foreachPodResourceClaim(pod, nil); err != nil {
		return statusUnschedulable(klog.FromContext(ctx), err.Error())
	}
	return nil
}

// isSchedulableAfterClaimChange is invoked for all claim events reported by
// an informer. It checks whether that change made a previously unschedulable
// pod schedulable. It errs on the side of letting a pod scheduling attempt
// happen.
func (pl *dynamicResources) isSchedulableAfterClaimChange(logger klog.Logger, pod *v1.Pod, oldObj, newObj interface{}) framework.QueueingHint {
	if newObj == nil {
		// Deletes don't make a pod schedulable.
		return framework.QueueSkip
	}

	_, modifiedClaim, err := schedutil.As[*resourcev1alpha2.ResourceClaim](nil, newObj)
	if err != nil {
		// Shouldn't happen.
		logger.Error(err, "unexpected new object in isSchedulableAfterClaimChange")
		return framework.QueueAfterBackoff
	}

	usesClaim := false
	if err := pl.foreachPodResourceClaim(pod, func(_ string, claim *resourcev1alpha2.ResourceClaim) {
		if claim.UID == modifiedClaim.UID {
			usesClaim = true
		}
	}); err != nil {
		// This is not an unexpected error: we know that
		// foreachPodResourceClaim only returns errors for "not
		// schedulable".
		logger.V(4).Info("pod is not schedulable", "pod", klog.KObj(pod), "claim", klog.KObj(modifiedClaim), "reason", err.Error())
		return framework.QueueSkip
	}

	if !usesClaim {
		// This was not the claim the pod was waiting for.
		logger.V(6).Info("unrelated claim got modified", "pod", klog.KObj(pod), "claim", klog.KObj(modifiedClaim))
		return framework.QueueSkip
	}

	if oldObj == nil {
		logger.V(4).Info("claim for pod got created", "pod", klog.KObj(pod), "claim", klog.KObj(modifiedClaim))
		return framework.QueueImmediately
	}

	// Modifications may or may not be relevant. If the entire
	// status is as before, then something else must have changed
	// and we don't care. What happens in practice is that the
	// resource driver adds the finalizer.
	originalClaim, ok := oldObj.(*resourcev1alpha2.ResourceClaim)
	if !ok {
		// Shouldn't happen.
		logger.Error(nil, "unexpected old object in isSchedulableAfterClaimAddOrUpdate", "obj", oldObj)
		return framework.QueueAfterBackoff
	}
	if apiequality.Semantic.DeepEqual(&originalClaim.Status, &modifiedClaim.Status) {
		if loggerV := logger.V(7); loggerV.Enabled() {
			// Log more information.
			loggerV.Info("claim for pod got modified where the pod doesn't care", "pod", klog.KObj(pod), "claim", klog.KObj(modifiedClaim), "diff", cmp.Diff(originalClaim, modifiedClaim))
		} else {
			logger.V(6).Info("claim for pod got modified where the pod doesn't care", "pod", klog.KObj(pod), "claim", klog.KObj(modifiedClaim))
		}
		return framework.QueueSkip
	}

	logger.V(4).Info("status of claim for pod got updated", "pod", klog.KObj(pod), "claim", klog.KObj(modifiedClaim))
	return framework.QueueImmediately
}

// isSchedulableAfterPodSchedulingContextChange is invoked for all
// PodSchedulingContext events reported by an informer. It checks whether that
// change made a previously unschedulable pod schedulable (updated) or a new
// attempt is needed to re-create the object (deleted). It errs on the side of
// letting a pod scheduling attempt happen.
func (pl *dynamicResources) isSchedulableAfterPodSchedulingContextChange(logger klog.Logger, pod *v1.Pod, oldObj, newObj interface{}) framework.QueueingHint {
	// Deleted? That can happen because we ourselves delete the PodSchedulingContext while
	// working on the pod. This can be ignored.
	if oldObj != nil && newObj == nil {
		logger.V(4).Info("PodSchedulingContext got deleted")
		return framework.QueueSkip
	}

	oldPodScheduling, newPodScheduling, err := schedutil.As[*resourcev1alpha2.PodSchedulingContext](oldObj, newObj)
	if err != nil {
		// Shouldn't happen.
		logger.Error(nil, "isSchedulableAfterPodSchedulingChange")
		return framework.QueueAfterBackoff
	}
	podScheduling := newPodScheduling // Never nil because deletes are handled above.

	if podScheduling.Name != pod.Name || podScheduling.Namespace != pod.Namespace {
		logger.V(7).Info("PodSchedulingContext for unrelated pod got modified", "pod", klog.KObj(pod), "podScheduling", klog.KObj(podScheduling))
		return framework.QueueSkip
	}

	// If the drivers have provided information about all
	// unallocated claims with delayed allocation, then the next
	// scheduling attempt is able to pick a node, so we let it run
	// immediately if this occurred for the first time, otherwise
	// we allow backoff.
	pendingDelayedClaims := 0
	if err := pl.foreachPodResourceClaim(pod, func(podResourceName string, claim *resourcev1alpha2.ResourceClaim) {
		if claim.Spec.AllocationMode == resourcev1alpha2.AllocationModeWaitForFirstConsumer &&
			claim.Status.Allocation == nil &&
			!podSchedulingHasClaimInfo(podScheduling, podResourceName) {
			pendingDelayedClaims++
		}
	}); err != nil {
		// This is not an unexpected error: we know that
		// foreachPodResourceClaim only returns errors for "not
		// schedulable".
		logger.V(4).Info("pod is not schedulable, keep waiting", "pod", klog.KObj(pod), "reason", err.Error())
		return framework.QueueSkip
	}

	// Some driver responses missing?
	if pendingDelayedClaims > 0 {
		// We could start a pod scheduling attempt to refresh the
		// potential nodes list.  But pod scheduling attempts are
		// expensive and doing them too often causes the pod to enter
		// backoff. Let's wait instead for all drivers to reply.
		if loggerV := logger.V(6); loggerV.Enabled() {
			loggerV.Info("PodSchedulingContext with missing resource claim information, keep waiting", "pod", klog.KObj(pod), "podSchedulingDiff", cmp.Diff(oldPodScheduling, podScheduling))
		} else {
			logger.V(5).Info("PodSchedulingContext with missing resource claim information, keep waiting", "pod", klog.KObj(pod))
		}
		return framework.QueueSkip
	}

	if oldPodScheduling == nil /* create */ ||
		len(oldPodScheduling.Status.ResourceClaims) < len(podScheduling.Status.ResourceClaims) /* new information and not incomplete (checked above) */ {
		// This definitely is new information for the scheduler. Try again immediately.
		logger.V(4).Info("PodSchedulingContext for pod has all required information, schedule immediately", "pod", klog.KObj(pod))
		return framework.QueueImmediately
	}

	// The other situation where the scheduler needs to do
	// something immediately is when the selected node doesn't
	// work: waiting in the backoff queue only helps eventually
	// resources on the selected node become available again. It's
	// much more likely, in particular when trying to fill up the
	// cluster, that the choice simply didn't work out. The risk
	// here is that in a situation where the cluster really is
	// full, backoff won't be used because the scheduler keeps
	// trying different nodes. This should not happen when it has
	// full knowledge about resource availability (=
	// PodSchedulingContext.*.UnsuitableNodes is complete) but may happen
	// when it doesn't (= PodSchedulingContext.*.UnsuitableNodes had to be
	// truncated).
	//
	// Truncation only happens for very large clusters and then may slow
	// down scheduling, but should not break it completely. This is
	// acceptable while DRA is alpha and will be investigated further
	// before moving DRA to beta.
	if podScheduling.Spec.SelectedNode != "" {
		for _, claimStatus := range podScheduling.Status.ResourceClaims {
			if sliceContains(claimStatus.UnsuitableNodes, podScheduling.Spec.SelectedNode) {
				logger.V(5).Info("PodSchedulingContext has unsuitable selected node, schedule immediately", "pod", klog.KObj(pod), "selectedNode", podScheduling.Spec.SelectedNode, "podResourceName", claimStatus.Name)
				return framework.QueueImmediately
			}
		}
	}

	// Update with only the spec modified?
	if oldPodScheduling != nil &&
		!apiequality.Semantic.DeepEqual(&oldPodScheduling.Spec, &podScheduling.Spec) &&
		apiequality.Semantic.DeepEqual(&oldPodScheduling.Status, &podScheduling.Status) {
		logger.V(5).Info("PodSchedulingContext has only the scheduler spec changes, ignore the update", "pod", klog.KObj(pod))
		return framework.QueueSkip
	}

	// Once we get here, all changes which are known to require special responses
	// have been checked for. Whatever the change was, we don't know exactly how
	// to handle it and thus return QueueAfterBackoff. This will cause the
	// scheduler to treat the event as if no event hint callback had been provided.
	// Developers who want to investigate this can enable a diff at log level 6.
	if loggerV := logger.V(6); loggerV.Enabled() {
		loggerV.Info("PodSchedulingContext for pod with unknown changes, maybe schedule", "pod", klog.KObj(pod), "podSchedulingDiff", cmp.Diff(oldPodScheduling, podScheduling))
	} else {
		logger.V(5).Info("PodSchedulingContext for pod with unknown changes, maybe schedule", "pod", klog.KObj(pod))
	}
	return framework.QueueAfterBackoff

}

func podSchedulingHasClaimInfo(podScheduling *resourcev1alpha2.PodSchedulingContext, podResourceName string) bool {
	for _, claimStatus := range podScheduling.Status.ResourceClaims {
		if claimStatus.Name == podResourceName {
			return true
		}
	}
	return false
}

func sliceContains(hay []string, needle string) bool {
	for _, item := range hay {
		if item == needle {
			return true
		}
	}
	return false
}

// podResourceClaims returns the ResourceClaims for all pod.Spec.PodResourceClaims.
func (pl *dynamicResources) podResourceClaims(pod *v1.Pod) ([]*resourcev1alpha2.ResourceClaim, error) {
	claims := make([]*resourcev1alpha2.ResourceClaim, 0, len(pod.Spec.ResourceClaims))
	if err := pl.foreachPodResourceClaim(pod, func(_ string, claim *resourcev1alpha2.ResourceClaim) {
		// We store the pointer as returned by the lister. The
		// assumption is that if a claim gets modified while our code
		// runs, the cache will store a new pointer, not mutate the
		// existing object that we point to here.
		claims = append(claims, claim)
	}); err != nil {
		return nil, err
	}
	return claims, nil
}

// foreachPodResourceClaim checks that each ResourceClaim for the pod exists.
// It calls an optional handler for those claims that it finds.
func (pl *dynamicResources) foreachPodResourceClaim(pod *v1.Pod, cb func(podResourceName string, claim *resourcev1alpha2.ResourceClaim)) error {
	for _, resource := range pod.Spec.ResourceClaims {
		claimName, mustCheckOwner, err := resourceclaim.Name(pod, &resource)
		if err != nil {
			return err
		}
		// The claim name might be nil if no underlying resource claim
		// was generated for the referenced claim. There are valid use
		// cases when this might happen, so we simply skip it.
		if claimName == nil {
			continue
		}
		claim, err := pl.claimLister.ResourceClaims(pod.Namespace).Get(*claimName)
		if err != nil {
			return err
		}

		if claim.DeletionTimestamp != nil {
			return fmt.Errorf("resourceclaim %q is being deleted", claim.Name)
		}

		if mustCheckOwner {
			if err := resourceclaim.IsForPod(pod, claim); err != nil {
				return err
			}
		}
		if cb != nil {
			cb(resource.Name, claim)
		}
	}
	return nil
}

// PreFilter invoked at the prefilter extension point to check if pod has all
// immediate claims bound. UnschedulableAndUnresolvable is returned if
// the pod cannot be scheduled at the moment on any node.
func (pl *dynamicResources) PreFilter(ctx context.Context, state *framework.CycleState, pod *v1.Pod) (*framework.PreFilterResult, *framework.Status) {
	if !pl.enabled {
		return nil, framework.NewStatus(framework.Skip)
	}
	logger := klog.FromContext(ctx)

	// If the pod does not reference any claim, we don't need to do
	// anything for it. We just initialize an empty state to record that
	// observation for the other functions. This gets updated below
	// if we get that far.
	s := &stateData{}
	state.Write(stateKey, s)

	claims, err := pl.podResourceClaims(pod)
	if err != nil {
		return nil, statusUnschedulable(logger, err.Error())
	}
	logger.V(5).Info("pod resource claims", "pod", klog.KObj(pod), "resourceclaims", klog.KObjSlice(claims))
	// If the pod does not reference any claim,
	// DynamicResources Filter has nothing to do with the Pod.
	if len(claims) == 0 {
		return nil, framework.NewStatus(framework.Skip)
	}

	s.availableOnNodes = make([]*nodeaffinity.NodeSelector, len(claims))
	for index, claim := range claims {
		if claim.Spec.AllocationMode == resourcev1alpha2.AllocationModeImmediate &&
			claim.Status.Allocation == nil {
			// This will get resolved by the resource driver.
			return nil, statusUnschedulable(logger, "unallocated immediate resourceclaim", "pod", klog.KObj(pod), "resourceclaim", klog.KObj(claim))
		}
		if claim.Status.DeallocationRequested {
			// This will get resolved by the resource driver.
			return nil, statusUnschedulable(logger, "resourceclaim must be reallocated", "pod", klog.KObj(pod), "resourceclaim", klog.KObj(claim))
		}
		if claim.Status.Allocation != nil &&
			!resourceclaim.CanBeReserved(claim) &&
			!resourceclaim.IsReservedForPod(pod, claim) {
			// Resource is in use. The pod has to wait.
			return nil, statusUnschedulable(logger, "resourceclaim in use", "pod", klog.KObj(pod), "resourceclaim", klog.KObj(claim))
		}
		if claim.Status.Allocation != nil &&
			claim.Status.Allocation.AvailableOnNodes != nil {
			nodeSelector, err := nodeaffinity.NewNodeSelector(claim.Status.Allocation.AvailableOnNodes)
			if err != nil {
				return nil, statusError(logger, err)
			}
			s.availableOnNodes[index] = nodeSelector
		}
	}

	s.claims = claims
	state.Write(stateKey, s)
	return nil, nil
}

// PreFilterExtensions returns prefilter extensions, pod add and remove.
func (pl *dynamicResources) PreFilterExtensions() framework.PreFilterExtensions {
	return nil
}

func getStateData(cs *framework.CycleState) (*stateData, error) {
	state, err := cs.Read(stateKey)
	if err != nil {
		return nil, err
	}
	s, ok := state.(*stateData)
	if !ok {
		return nil, errors.New("unable to convert state into stateData")
	}
	return s, nil
}

// Filter invoked at the filter extension point.
// It evaluates if a pod can fit due to the resources it requests,
// for both allocated and unallocated claims.
//
// For claims that are bound, then it checks that the node affinity is
// satisfied by the given node.
//
// For claims that are unbound, it checks whether the claim might get allocated
// for the node.
func (pl *dynamicResources) Filter(ctx context.Context, cs *framework.CycleState, pod *v1.Pod, nodeInfo *framework.NodeInfo) *framework.Status {
	if !pl.enabled {
		return nil
	}
	state, err := getStateData(cs)
	if err != nil {
		return statusError(klog.FromContext(ctx), err)
	}
	if len(state.claims) == 0 {
		return nil
	}

	logger := klog.FromContext(ctx)
	node := nodeInfo.Node()

	var unavailableClaims []int
	for index, claim := range state.claims {
		logger.V(10).Info("filtering based on resource claims of the pod", "pod", klog.KObj(pod), "node", klog.KObj(node), "resourceclaim", klog.KObj(claim))
		switch {
		case claim.Status.Allocation != nil:
			if nodeSelector := state.availableOnNodes[index]; nodeSelector != nil {
				if !nodeSelector.Match(node) {
					logger.V(5).Info("AvailableOnNodes does not match", "pod", klog.KObj(pod), "node", klog.KObj(node), "resourceclaim", klog.KObj(claim))
					unavailableClaims = append(unavailableClaims, index)
				}
			}
		case claim.Status.DeallocationRequested:
			// We shouldn't get here. PreFilter already checked this.
			return statusUnschedulable(logger, "resourceclaim must be reallocated", "pod", klog.KObj(pod), "node", klog.KObj(node), "resourceclaim", klog.KObj(claim))
		case claim.Spec.AllocationMode == resourcev1alpha2.AllocationModeWaitForFirstConsumer:
			// The ResourceClass might have a node filter. This is
			// useful for trimming the initial set of potential
			// nodes before we ask the driver(s) for information
			// about the specific pod.
			class, err := pl.classLister.Get(claim.Spec.ResourceClassName)
			if err != nil {
				// If the class does not exist, then allocation cannot proceed.
				return statusError(logger, fmt.Errorf("look up resource class: %v", err))
			}
			if class.SuitableNodes != nil {
				// TODO (#113700): parse class.SuitableNodes once in PreFilter, reuse result.
				matches, err := corev1helpers.MatchNodeSelectorTerms(node, class.SuitableNodes)
				if err != nil {
					return statusError(logger, fmt.Errorf("potential node filter: %v", err))
				}
				if !matches {
					return statusUnschedulable(logger, "excluded by resource class node filter", "pod", klog.KObj(pod), "node", klog.KObj(node), "resourceclass", klog.KObj(class))
				}
			}

			// Now we need information from drivers.
			schedulingCtx, err := state.initializePodSchedulingContexts(ctx, pod, pl.podSchedulingContextLister)
			if err != nil {
				return statusError(logger, err)
			}
			status := statusForClaim(schedulingCtx, pod.Spec.ResourceClaims[index].Name)
			if status != nil {
				for _, unsuitableNode := range status.UnsuitableNodes {
					if node.Name == unsuitableNode {
						return statusUnschedulable(logger, "resourceclaim cannot be allocated for the node (unsuitable)", "pod", klog.KObj(pod), "node", klog.KObj(node), "resourceclaim", klog.KObj(claim), "unsuitablenodes", status.UnsuitableNodes)
					}
				}
			}
		default:
			// This should have been delayed allocation. Immediate
			// allocation was already checked for in PreFilter.
			return statusError(logger, fmt.Errorf("internal error, unexpected allocation mode %v", claim.Spec.AllocationMode))
		}
	}

	if len(unavailableClaims) > 0 {
		state.mutex.Lock()
		defer state.mutex.Unlock()
		if state.unavailableClaims == nil {
			state.unavailableClaims = sets.NewInt()
		}

		for index := range unavailableClaims {
			claim := state.claims[index]
			// Deallocation makes more sense for claims with
			// delayed allocation. Claims with immediate allocation
			// would just get allocated again for a random node,
			// which is unlikely to help the pod.
			if claim.Spec.AllocationMode == resourcev1alpha2.AllocationModeWaitForFirstConsumer {
				state.unavailableClaims.Insert(unavailableClaims...)
			}
		}
		return statusUnschedulable(logger, "resourceclaim not available on the node", "pod", klog.KObj(pod))
	}

	return nil
}

// PostFilter checks whether there are allocated claims that could get
// deallocated to help get the Pod schedulable. If yes, it picks one and
// requests its deallocation.  This only gets called when filtering found no
// suitable node.
func (pl *dynamicResources) PostFilter(ctx context.Context, cs *framework.CycleState, pod *v1.Pod, filteredNodeStatusMap framework.NodeToStatusMap) (*framework.PostFilterResult, *framework.Status) {
	if !pl.enabled {
		return nil, framework.NewStatus(framework.Unschedulable, "plugin disabled")
	}
	logger := klog.FromContext(ctx)
	state, err := getStateData(cs)
	if err != nil {
		return nil, statusError(logger, err)
	}
	if len(state.claims) == 0 {
		return nil, framework.NewStatus(framework.Unschedulable, "no new claims to deallocate")
	}

	// Iterating over a map is random. This is intentional here, we want to
	// pick one claim randomly because there is no better heuristic.
	for index := range state.unavailableClaims {
		claim := state.claims[index]
		if len(claim.Status.ReservedFor) == 0 ||
			len(claim.Status.ReservedFor) == 1 && claim.Status.ReservedFor[0].UID == pod.UID {
			claim := state.claims[index].DeepCopy()
			claim.Status.DeallocationRequested = true
			claim.Status.ReservedFor = nil
			logger.V(5).Info("Requesting deallocation of ResourceClaim", "pod", klog.KObj(pod), "resourceclaim", klog.KObj(claim))
			if err := state.updateClaimStatus(ctx, pl.clientset, index, claim); err != nil {
				return nil, statusError(logger, err)
			}
			return nil, nil
		}
	}
	return nil, framework.NewStatus(framework.Unschedulable, "still not schedulable")
}

// PreScore is passed a list of all nodes that would fit the pod. Not all
// claims are necessarily allocated yet, so here we can set the SuitableNodes
// field for those which are pending.
func (pl *dynamicResources) PreScore(ctx context.Context, cs *framework.CycleState, pod *v1.Pod, nodes []*v1.Node) *framework.Status {
	if !pl.enabled {
		return nil
	}
	state, err := getStateData(cs)
	if err != nil {
		return statusError(klog.FromContext(ctx), err)
	}
	if len(state.claims) == 0 {
		return nil
	}

	logger := klog.FromContext(ctx)
	schedulingCtx, err := state.initializePodSchedulingContexts(ctx, pod, pl.podSchedulingContextLister)
	if err != nil {
		return statusError(logger, err)
	}
	pending := false
	for _, claim := range state.claims {
		if claim.Status.Allocation == nil {
			pending = true
		}
	}
	if pending && !haveAllNodes(schedulingCtx.Spec.PotentialNodes, nodes) {
		// Remember the potential nodes. The object will get created or
		// updated in Reserve. This is both an optimization and
		// covers the case that PreScore doesn't get called when there
		// is only a single node.
		logger.V(5).Info("remembering potential nodes", "pod", klog.KObj(pod), "potentialnodes", klog.KObjSlice(nodes))
		schedulingCtx = schedulingCtx.DeepCopy()
		numNodes := len(nodes)
		if numNodes > resourcev1alpha2.PodSchedulingNodeListMaxSize {
			numNodes = resourcev1alpha2.PodSchedulingNodeListMaxSize
		}
		schedulingCtx.Spec.PotentialNodes = make([]string, 0, numNodes)
		if numNodes == len(nodes) {
			// Copy all node names.
			for _, node := range nodes {
				schedulingCtx.Spec.PotentialNodes = append(schedulingCtx.Spec.PotentialNodes, node.Name)
			}
		} else {
			// Select a random subset of the nodes to comply with
			// the PotentialNodes length limit. Randomization is
			// done for us by Go which iterates over map entries
			// randomly.
			nodeNames := map[string]struct{}{}
			for _, node := range nodes {
				nodeNames[node.Name] = struct{}{}
			}
			for nodeName := range nodeNames {
				if len(schedulingCtx.Spec.PotentialNodes) >= resourcev1alpha2.PodSchedulingNodeListMaxSize {
					break
				}
				schedulingCtx.Spec.PotentialNodes = append(schedulingCtx.Spec.PotentialNodes, nodeName)
			}
		}
		sort.Strings(schedulingCtx.Spec.PotentialNodes)
		state.storePodSchedulingContexts(schedulingCtx)
	}
	logger.V(5).Info("all potential nodes already set", "pod", klog.KObj(pod), "potentialnodes", klog.KObjSlice(nodes))
	return nil
}

func haveAllNodes(nodeNames []string, nodes []*v1.Node) bool {
	for _, node := range nodes {
		if !haveNode(nodeNames, node.Name) {
			return false
		}
	}
	return true
}

func haveNode(nodeNames []string, nodeName string) bool {
	for _, n := range nodeNames {
		if n == nodeName {
			return true
		}
	}
	return false
}

// Reserve reserves claims for the pod.
func (pl *dynamicResources) Reserve(ctx context.Context, cs *framework.CycleState, pod *v1.Pod, nodeName string) *framework.Status {
	if !pl.enabled {
		return nil
	}
	state, err := getStateData(cs)
	if err != nil {
		return statusError(klog.FromContext(ctx), err)
	}
	if len(state.claims) == 0 {
		return nil
	}

	numDelayedAllocationPending := 0
	numClaimsWithStatusInfo := 0
	logger := klog.FromContext(ctx)
	schedulingCtx, err := state.initializePodSchedulingContexts(ctx, pod, pl.podSchedulingContextLister)
	if err != nil {
		return statusError(logger, err)
	}
	for index, claim := range state.claims {
		if claim.Status.Allocation != nil {
			// Allocated, but perhaps not reserved yet.
			if resourceclaim.IsReservedForPod(pod, claim) {
				logger.V(5).Info("is reserved", "pod", klog.KObj(pod), "node", klog.ObjectRef{Name: nodeName}, "resourceclaim", klog.KObj(claim))
				continue
			}
			claim := claim.DeepCopy()
			claim.Status.ReservedFor = append(claim.Status.ReservedFor,
				resourcev1alpha2.ResourceClaimConsumerReference{
					Resource: "pods",
					Name:     pod.Name,
					UID:      pod.UID,
				})
			logger.V(5).Info("reserve", "pod", klog.KObj(pod), "node", klog.ObjectRef{Name: nodeName}, "resourceclaim", klog.KObj(claim))
			_, err := pl.clientset.ResourceV1alpha2().ResourceClaims(claim.Namespace).UpdateStatus(ctx, claim, metav1.UpdateOptions{})
			// TODO: metric for update errors.
			if err != nil {
				return statusError(logger, err)
			}
			// If we get here, we know that reserving the claim for
			// the pod worked and we can proceed with schedulingCtx
			// it.
		} else {
			// Must be delayed allocation.
			numDelayedAllocationPending++

			// Did the driver provide information that steered node
			// selection towards a node that it can support?
			if statusForClaim(schedulingCtx, pod.Spec.ResourceClaims[index].Name) != nil {
				numClaimsWithStatusInfo++
			}
		}
	}

	if numDelayedAllocationPending == 0 {
		// Nothing left to do.
		return nil
	}

	podSchedulingDirty := state.podSchedulingDirty
	if len(schedulingCtx.Spec.PotentialNodes) == 0 {
		// PreScore was not called, probably because there was
		// only one candidate. We need to ask whether that
		// node is suitable, otherwise the scheduler will pick
		// it forever even when it cannot satisfy the claim.
		schedulingCtx = schedulingCtx.DeepCopy()
		schedulingCtx.Spec.PotentialNodes = []string{nodeName}
		logger.V(5).Info("asking for information about single potential node", "pod", klog.KObj(pod), "node", klog.ObjectRef{Name: nodeName})
		podSchedulingDirty = true
	}

	// When there is only one pending resource, we can go ahead with
	// requesting allocation even when we don't have the information from
	// the driver yet. Otherwise we wait for information before blindly
	// making a decision that might have to be reversed later.
	if numDelayedAllocationPending == 1 || numClaimsWithStatusInfo == numDelayedAllocationPending {
		schedulingCtx = schedulingCtx.DeepCopy()
		// TODO: can we increase the chance that the scheduler picks
		// the same node as before when allocation is on-going,
		// assuming that that node still fits the pod?  Picking a
		// different node may lead to some claims being allocated for
		// one node and others for another, which then would have to be
		// resolved with deallocation.
		schedulingCtx.Spec.SelectedNode = nodeName
		logger.V(5).Info("start allocation", "pod", klog.KObj(pod), "node", klog.ObjectRef{Name: nodeName})
		if err := state.publishPodSchedulingContexts(ctx, pl.clientset, schedulingCtx); err != nil {
			return statusError(logger, err)
		}
		return statusUnschedulable(logger, "waiting for resource driver to allocate resource", "pod", klog.KObj(pod), "node", klog.ObjectRef{Name: nodeName})
	}

	// May have been modified earlier in PreScore or above.
	if podSchedulingDirty {
		if err := state.publishPodSchedulingContexts(ctx, pl.clientset, schedulingCtx); err != nil {
			return statusError(logger, err)
		}
	}

	// More than one pending claim and not enough information about all of them.
	//
	// TODO: can or should we ensure that schedulingCtx gets aborted while
	// waiting for resources *before* triggering delayed volume
	// provisioning?  On the one hand, volume provisioning is currently
	// irreversible, so it better should come last. On the other hand,
	// triggering both in parallel might be faster.
	return statusUnschedulable(logger, "waiting for resource driver to provide information", "pod", klog.KObj(pod))
}

// Unreserve clears the ReservedFor field for all claims.
// It's idempotent, and does nothing if no state found for the given pod.
func (pl *dynamicResources) Unreserve(ctx context.Context, cs *framework.CycleState, pod *v1.Pod, nodeName string) {
	if !pl.enabled {
		return
	}
	state, err := getStateData(cs)
	if err != nil {
		return
	}
	if len(state.claims) == 0 {
		return
	}

	logger := klog.FromContext(ctx)
	for index, claim := range state.claims {
		if claim.Status.Allocation != nil &&
			resourceclaim.IsReservedForPod(pod, claim) {
			// Remove pod from ReservedFor.
			claim := claim.DeepCopy()
			reservedFor := make([]resourcev1alpha2.ResourceClaimConsumerReference, 0, len(claim.Status.ReservedFor)-1)
			for _, reserved := range claim.Status.ReservedFor {
				// TODO: can UID be assumed to be unique all resources or do we also need to compare Group/Version/Resource?
				if reserved.UID != pod.UID {
					reservedFor = append(reservedFor, reserved)
				}
			}
			claim.Status.ReservedFor = reservedFor
			logger.V(5).Info("unreserve", "resourceclaim", klog.KObj(claim))
			if err := state.updateClaimStatus(ctx, pl.clientset, index, claim); err != nil {
				// We will get here again when pod schedulingCtx
				// is retried.
				logger.Error(err, "unreserve", "resourceclaim", klog.KObj(claim))
			}
		}
	}
}

// PostBind is called after a pod is successfully bound to a node. Now we are
// sure that a PodSchedulingContext object, if it exists, is definitely not going to
// be needed anymore and can delete it. This is a one-shot thing, there won't
// be any retries.  This is okay because it should usually work and in those
// cases where it doesn't, the garbage collector will eventually clean up.
func (pl *dynamicResources) PostBind(ctx context.Context, cs *framework.CycleState, pod *v1.Pod, nodeName string) {
	if !pl.enabled {
		return
	}
	state, err := getStateData(cs)
	if err != nil {
		return
	}
	if len(state.claims) == 0 {
		return
	}

	// We cannot know for sure whether the PodSchedulingContext object exists. We
	// might have created it in the previous pod schedulingCtx cycle and not
	// have it in our informer cache yet. Let's try to delete, just to be
	// on the safe side.
	logger := klog.FromContext(ctx)
	err = pl.clientset.ResourceV1alpha2().PodSchedulingContexts(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{})
	switch {
	case apierrors.IsNotFound(err):
		logger.V(5).Info("no PodSchedulingContext object to delete")
	case err != nil:
		logger.Error(err, "delete PodSchedulingContext")
	default:
		logger.V(5).Info("PodSchedulingContext object deleted")
	}
}

// statusUnschedulable ensures that there is a log message associated with the
// line where the status originated.
func statusUnschedulable(logger klog.Logger, reason string, kv ...interface{}) *framework.Status {
	if loggerV := logger.V(5); loggerV.Enabled() {
		helper, loggerV := loggerV.WithCallStackHelper()
		helper()
		kv = append(kv, "reason", reason)
		// nolint: logcheck // warns because it cannot check key/values
		loggerV.Info("pod unschedulable", kv...)
	}
	return framework.NewStatus(framework.UnschedulableAndUnresolvable, reason)
}

// statusError ensures that there is a log message associated with the
// line where the error originated.
func statusError(logger klog.Logger, err error, kv ...interface{}) *framework.Status {
	if loggerV := logger.V(5); loggerV.Enabled() {
		helper, loggerV := loggerV.WithCallStackHelper()
		helper()
		// nolint: logcheck // warns because it cannot check key/values
		loggerV.Error(err, "dynamic resource plugin failed", kv...)
	}
	return framework.AsStatus(err)
}
