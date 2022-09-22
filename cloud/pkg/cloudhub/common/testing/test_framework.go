/*
Copyright 2022 The KubeEdge Authors.

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

package testing

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"sync"
	"time"

	"k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/diff"
	core "k8s.io/client-go/testing"
	"k8s.io/klog/v2"

	beehivemodel "github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/cloud/pkg/cloudhub/common"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/messagelayer"
	"github.com/kubeedge/kubeedge/cloud/pkg/common/modules"
	"github.com/kubeedge/kubeedge/pkg/apis/reliablesyncs/v1alpha1"
	"github.com/kubeedge/kubeedge/pkg/client/clientset/versioned/fake"
)

var (
	NoErrors      []ReactorError
	NoObjectSyncs []*v1alpha1.ObjectSync
)

type TestCase struct {
	// Name of the test, for logging
	Name string

	// Initial content of ObjectSync cache.
	InitialObjectSyncs []*v1alpha1.ObjectSync

	// Expected content of ObjectSync cache at the end of the test.
	ExpectedObjectSyncs []*v1alpha1.ObjectSync

	// Expected message store in node message pool
	ExpectedStoreMessage *beehivemodel.Message

	// reactorErrors to produce on matching ObjectSync action
	ReactorErrors []ReactorError

	// injectedConnErrTimes the connection write err times, used for node session test
	InjectedConnErrTimes int

	// Initial content of message cache.
	InitialMessages []*beehivemodel.Message

	// CurrentArriveMessage current arrive message
	CurrentArriveMessage *beehivemodel.Message

	// Simulate downstream message from edgeController or syncController
	SimulateMessageFunc func(pool *common.NodeMessagePool, messages []*beehivemodel.Message)
}

// ErrVersionConflict is the error returned when resource version of requested
// object conflicts with the object in storage.
var ErrVersionConflict = errors.New("VersionError")

// ObjectSyncReactor is a core.Reactor that simulates etcd and API server. It
// stores:
// - Latest version of objectSyncs saved by the session.
// - Optionally, list of error that should be returned by reactor, simulating
//   etcd / API server failures. These errors are evaluated in order and every
//   error is returned only once. I.e. when the reactor finds matching
//   ReactorError, it return appropriate error and removes the ReactorError from
//   the list.
type ObjectSyncReactor struct {
	objectSyncs map[string]*v1alpha1.ObjectSync
	lock        sync.RWMutex
	errors      []ReactorError
}

// ReactorError is an error that is returned by test reactor (=simulated
// etcd+/API server) when an action performed by the reactor matches given verb
// ("get", "update", "create", "delete" or "*"") on given resource
// ("objectsyncs" or "*").
type ReactorError struct {
	Verb     string
	Resource string
	Error    error
}

// React is a callback called by fake kubeClient from the controller.
// In other words, every objectSync change performed by the session ends
// here.
// This callback checks versions of the updated objects and refuse those that
// are too old (simulating real etcd).
// All updated objects are stored locally to keep track of object versions and
// to evaluate test results.
func (r *ObjectSyncReactor) React(action core.Action) (handled bool, ret runtime.Object, err error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	klog.V(4).Infof("reactor got operation %q on %q", action.GetVerb(), action.GetResource())

	// Inject error when requested
	err = r.injectReactError(action)
	if err != nil {
		return true, nil, err
	}

	// Test did not request to inject an error, continue simulating API server.
	switch {
	case action.Matches("create", "objectsyncs"):
		obj := action.(core.UpdateAction).GetObject()
		objectSync := obj.(*v1alpha1.ObjectSync)

		// check the objectSync does not exist
		_, found := r.objectSyncs[objectSync.Name]
		if found {
			return true, nil, fmt.Errorf("cannot create objectSync %s: objectSync already exists", objectSync.Name)
		}

		// Store the updated object to appropriate places.
		r.objectSyncs[objectSync.Name] = objectSync

		klog.V(4).Infof("created objectSync %s", objectSync.Name)
		return true, objectSync, nil

	case action.Matches("update", "objectsyncs"):
		obj := action.(core.UpdateAction).GetObject()
		objectSync := obj.(*v1alpha1.ObjectSync)

		// Check and bump object version
		storedObjectSync, found := r.objectSyncs[objectSync.Name]
		if found {
			storedVer, _ := strconv.Atoi(storedObjectSync.ResourceVersion)
			requestedVer, _ := strconv.Atoi(objectSync.ResourceVersion)
			if storedVer != requestedVer {
				return true, obj, ErrVersionConflict
			}
			if reflect.DeepEqual(storedObjectSync, objectSync) {
				klog.V(4).Infof("nothing updated objectSync %s", objectSync.Name)
				return true, objectSync, nil
			}
			// Don't modify the existing object
			objectSync = objectSync.DeepCopy()
			objectSync.ResourceVersion = strconv.Itoa(storedVer + 1)
		} else {
			return true, nil, fmt.Errorf("cannot update objectSync %s: objectSync not found", objectSync.Name)
		}

		r.objectSyncs[objectSync.Name] = objectSync
		klog.V(4).Infof("saved updated objectSync %s", objectSync.Name)
		return true, objectSync, nil

	case action.Matches("get", "objectsyncs"):
		name := action.(core.GetAction).GetName()
		objectSync, found := r.objectSyncs[name]
		if found {
			klog.V(4).Infof("GetObjectSync: found %s", objectSync.Name)
			return true, objectSync.DeepCopy(), nil
		}
		klog.V(4).Infof("GetObjectSync: objectSync %s not found", name)
		return true, nil, apierrors.NewNotFound(action.GetResource().GroupResource(), name)

	case action.Matches("delete", "objectsyncs"):
		name := action.(core.DeleteAction).GetName()
		klog.V(4).Infof("deleted objectSync %s", name)
		_, found := r.objectSyncs[name]
		if found {
			delete(r.objectSyncs, name)
			return true, nil, nil
		}
		return true, nil, fmt.Errorf("cannot delete objectSync %s: not found", name)
	}

	return false, nil, nil
}

// injectReactError returns an error when the test requested given action to
// fail. nil is returned otherwise.
func (r *ObjectSyncReactor) injectReactError(action core.Action) error {
	if len(r.errors) == 0 {
		// No more errors to inject, everything should succeed.
		return nil
	}

	for i, expected := range r.errors {
		klog.V(4).Infof("trying to match %q %q with %q %q", expected.Verb, expected.Resource, action.GetVerb(), action.GetResource())
		if action.Matches(expected.Verb, expected.Resource) {
			// That's the action we're waiting for, remove it from injectedErrors
			r.errors = append(r.errors[:i], r.errors[i+1:]...)
			klog.V(4).Infof("reactor found matching error at index %d: %q %q, returning %v", i, expected.Verb, expected.Resource, expected.Error)
			return expected.Error
		}
	}
	return nil
}

// AddObjectSync adds a objectSync into ObjectSyncReactor.
func (r *ObjectSyncReactor) AddObjectSync(objectSync *v1alpha1.ObjectSync) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.objectSyncs[objectSync.Name] = objectSync
}

// DeleteObjectSync deletes a objectSync by name.
func (r *ObjectSyncReactor) DeleteObjectSync(name string) {
	r.lock.Lock()
	defer r.lock.Unlock()
	delete(r.objectSyncs, name)
}

// AddObjectSyncs adds objectSyncs into ObjectSyncReactor.
func (r *ObjectSyncReactor) AddObjectSyncs(objectSyncs []*v1alpha1.ObjectSync) {
	r.lock.Lock()
	defer r.lock.Unlock()
	for _, objectSync := range objectSyncs {
		r.objectSyncs[objectSync.Name] = objectSync
	}
}

// CheckObjectSyncs compares all expectedObjectSyncs with set of objectSyncs at the end of the
// test and reports differences.
func (r *ObjectSyncReactor) CheckObjectSyncs(expectedObjectSyncs []*v1alpha1.ObjectSync) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	expectedMap := make(map[string]*v1alpha1.ObjectSync)
	gotMap := make(map[string]*v1alpha1.ObjectSync)
	for _, c := range expectedObjectSyncs {
		// Don't modify the existing object
		c = c.DeepCopy()
		c.ResourceVersion = ""
		expectedMap[c.Name] = c
	}
	for _, c := range r.objectSyncs {
		// We must clone the ObjectSync because of golang race check - it was
		// written by the controller without any locks on it.
		c = c.DeepCopy()
		c.ResourceVersion = ""
		gotMap[c.Name] = c
	}
	if !reflect.DeepEqual(expectedMap, gotMap) {
		// Print ugly but useful diff of expected and received objects for
		// easier debugging.
		return fmt.Errorf("ObjectSync check failed [A-expected, B-got result]: %s", diff.ObjectDiff(expectedMap, gotMap))
	}
	return nil
}

const (
	TestNodeID                    = "foo-node"
	TestProjectID                 = "foo-project"
	TestNamespace                 = "foo-ns"
	TestPodName                   = "foo-pod"
	TestConfigMapName             = "foo-config"
	TestPodUID                    = "c7399134-f4fb-43e6-8088-c273bfffe7af"
	TestDiffPodUID                = "c7399134-f4fb-43e6-8088-c273bfffe999"
	TestConfigMapUID              = "22b0074a-8c07-4fc2-adad-1589f7f6f8b1"
	KeepaliveInterval             = 2 * time.Second
	NormalSendKeepaliveInterval   = 1 * time.Second
	AbnormalSendKeepaliveInterval = 3 * time.Second
)

// NewObjectSyncReactor creates a ObjectSync reactor.
func NewObjectSyncReactor(client *fake.Clientset, errors []ReactorError) *ObjectSyncReactor {
	reactor := &ObjectSyncReactor{
		objectSyncs: make(map[string]*v1alpha1.ObjectSync),
		errors:      errors,
	}
	client.AddReactor("create", "objectsyncs", reactor.React)
	client.AddReactor("update", "objectsyncs", reactor.React)
	client.AddReactor("get", "objectsyncs", reactor.React)
	client.AddReactor("delete", "objectsyncs", reactor.React)
	return reactor
}

func NewTestPodResource(name, UID, resourceVersion string) *v1.Pod {
	return &v1.Pod{
		TypeMeta: v12.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: v12.ObjectMeta{
			Name:            name,
			Namespace:       TestNamespace,
			ResourceVersion: resourceVersion,
			UID:             types.UID(UID),
			Labels: map[string]string{
				"version": resourceVersion,
			},
		},
		Spec: v1.PodSpec{
			NodeName:   TestNodeID,
			Containers: []v1.Container{{Name: "ctr", Image: "image"}},
		},
	}
}

func NewObjectSync(object v12.Object, kind string) *v1alpha1.ObjectSync {
	return &v1alpha1.ObjectSync{
		ObjectMeta: v12.ObjectMeta{
			Name:      fmt.Sprintf("%s.%s", TestNodeID, object.GetUID()),
			Namespace: object.GetNamespace(),
		},
		Spec: v1alpha1.ObjectSyncSpec{
			ObjectAPIVersion: "v1",
			ObjectKind:       kind,
			ObjectName:       object.GetName(),
		},
		Status: v1alpha1.ObjectSyncStatus{
			ObjectResourceVersion: object.GetResourceVersion(),
		},
	}
}

func NewPodMessage(pod *v1.Pod, operation string) *beehivemodel.Message {
	resource, err := messagelayer.BuildResource(pod.Spec.NodeName, pod.Namespace, "pod", pod.Name)
	if err != nil {
		klog.Warningf("built message resource failed with error: %s", err)
		return nil
	}
	message := beehivemodel.NewMessage("").
		SetResourceVersion(pod.ResourceVersion).
		FillBody(pod)

	return message.BuildRouter("edgecontroller", "resource", resource, operation)
}

func NewTestConfigMapResource(name, UID, resourceVersion string) *v1.ConfigMap {
	return &v1.ConfigMap{
		TypeMeta: v12.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: v12.ObjectMeta{
			Name:            name,
			Namespace:       TestNamespace,
			ResourceVersion: resourceVersion,
			UID:             types.UID(UID),
			Labels: map[string]string{
				"version": resourceVersion,
			},
		},
		Data: map[string]string{"foo": "bar"},
	}
}

func NewConfigMapMessage(configMap *v1.ConfigMap, operation string) *beehivemodel.Message {
	resource, err := messagelayer.BuildResource(TestNodeID, configMap.Namespace, "configmap", configMap.Name)
	if err != nil {
		klog.Warningf("build message resource failed with error: %s", err)
		return nil
	}
	return beehivemodel.NewMessage("").
		SetResourceVersion(configMap.ResourceVersion).
		BuildRouter(modules.EdgeControllerModuleName, "resource", resource, operation).
		FillBody(configMap)
}
