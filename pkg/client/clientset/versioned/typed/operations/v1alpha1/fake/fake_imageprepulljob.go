/*
Copyright The KubeEdge Authors.

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

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	v1alpha1 "github.com/kubeedge/kubeedge/pkg/apis/operations/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeImagePrePullJobs implements ImagePrePullJobInterface
type FakeImagePrePullJobs struct {
	Fake *FakeOperationsV1alpha1
}

var imageprepulljobsResource = v1alpha1.SchemeGroupVersion.WithResource("imageprepulljobs")

var imageprepulljobsKind = v1alpha1.SchemeGroupVersion.WithKind("ImagePrePullJob")

// Get takes name of the imagePrePullJob, and returns the corresponding imagePrePullJob object, and an error if there is any.
func (c *FakeImagePrePullJobs) Get(ctx context.Context, name string, options v1.GetOptions) (result *v1alpha1.ImagePrePullJob, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootGetAction(imageprepulljobsResource, name), &v1alpha1.ImagePrePullJob{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ImagePrePullJob), err
}

// List takes label and field selectors, and returns the list of ImagePrePullJobs that match those selectors.
func (c *FakeImagePrePullJobs) List(ctx context.Context, opts v1.ListOptions) (result *v1alpha1.ImagePrePullJobList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootListAction(imageprepulljobsResource, imageprepulljobsKind, opts), &v1alpha1.ImagePrePullJobList{})
	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &v1alpha1.ImagePrePullJobList{ListMeta: obj.(*v1alpha1.ImagePrePullJobList).ListMeta}
	for _, item := range obj.(*v1alpha1.ImagePrePullJobList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested imagePrePullJobs.
func (c *FakeImagePrePullJobs) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewRootWatchAction(imageprepulljobsResource, opts))
}

// Create takes the representation of a imagePrePullJob and creates it.  Returns the server's representation of the imagePrePullJob, and an error, if there is any.
func (c *FakeImagePrePullJobs) Create(ctx context.Context, imagePrePullJob *v1alpha1.ImagePrePullJob, opts v1.CreateOptions) (result *v1alpha1.ImagePrePullJob, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootCreateAction(imageprepulljobsResource, imagePrePullJob), &v1alpha1.ImagePrePullJob{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ImagePrePullJob), err
}

// Update takes the representation of a imagePrePullJob and updates it. Returns the server's representation of the imagePrePullJob, and an error, if there is any.
func (c *FakeImagePrePullJobs) Update(ctx context.Context, imagePrePullJob *v1alpha1.ImagePrePullJob, opts v1.UpdateOptions) (result *v1alpha1.ImagePrePullJob, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateAction(imageprepulljobsResource, imagePrePullJob), &v1alpha1.ImagePrePullJob{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ImagePrePullJob), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeImagePrePullJobs) UpdateStatus(ctx context.Context, imagePrePullJob *v1alpha1.ImagePrePullJob, opts v1.UpdateOptions) (*v1alpha1.ImagePrePullJob, error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootUpdateSubresourceAction(imageprepulljobsResource, "status", imagePrePullJob), &v1alpha1.ImagePrePullJob{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ImagePrePullJob), err
}

// Delete takes name of the imagePrePullJob and deletes it. Returns an error if one occurs.
func (c *FakeImagePrePullJobs) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewRootDeleteActionWithOptions(imageprepulljobsResource, name, opts), &v1alpha1.ImagePrePullJob{})
	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeImagePrePullJobs) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewRootDeleteCollectionAction(imageprepulljobsResource, listOpts)

	_, err := c.Fake.Invokes(action, &v1alpha1.ImagePrePullJobList{})
	return err
}

// Patch applies the patch and returns the patched imagePrePullJob.
func (c *FakeImagePrePullJobs) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *v1alpha1.ImagePrePullJob, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewRootPatchSubresourceAction(imageprepulljobsResource, name, pt, data, subresources...), &v1alpha1.ImagePrePullJob{})
	if obj == nil {
		return nil, err
	}
	return obj.(*v1alpha1.ImagePrePullJob), err
}