/*
Copyright © 2021 The nvidia-gpu-scheduler Authors.
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
// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	"context"

	gpupodv1 "github.com/caden2016/nvidia-gpu-scheduler/api/gpupod/v1"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	schema "k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	testing "k8s.io/client-go/testing"
)

// FakeGpuPods implements GpuPodInterface
type FakeGpuPods struct {
	Fake *FakeGpupodV1
	ns   string
}

var gpupodsResource = schema.GroupVersionResource{Group: "gpupod", Version: "v1", Resource: "gpupods"}

var gpupodsKind = schema.GroupVersionKind{Group: "gpupod", Version: "v1", Kind: "GpuPod"}

// Get takes name of the gpuPod, and returns the corresponding gpuPod object, and an error if there is any.
func (c *FakeGpuPods) Get(ctx context.Context, name string, options v1.GetOptions) (result *gpupodv1.GpuPod, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewGetAction(gpupodsResource, c.ns, name), &gpupodv1.GpuPod{})

	if obj == nil {
		return nil, err
	}
	return obj.(*gpupodv1.GpuPod), err
}

// List takes label and field selectors, and returns the list of GpuPods that match those selectors.
func (c *FakeGpuPods) List(ctx context.Context, opts v1.ListOptions) (result *gpupodv1.GpuPodList, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewListAction(gpupodsResource, gpupodsKind, c.ns, opts), &gpupodv1.GpuPodList{})

	if obj == nil {
		return nil, err
	}

	label, _, _ := testing.ExtractFromListOptions(opts)
	if label == nil {
		label = labels.Everything()
	}
	list := &gpupodv1.GpuPodList{ListMeta: obj.(*gpupodv1.GpuPodList).ListMeta}
	for _, item := range obj.(*gpupodv1.GpuPodList).Items {
		if label.Matches(labels.Set(item.Labels)) {
			list.Items = append(list.Items, item)
		}
	}
	return list, err
}

// Watch returns a watch.Interface that watches the requested gpuPods.
func (c *FakeGpuPods) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	return c.Fake.
		InvokesWatch(testing.NewWatchAction(gpupodsResource, c.ns, opts))

}

// Create takes the representation of a gpuPod and creates it.  Returns the server's representation of the gpuPod, and an error, if there is any.
func (c *FakeGpuPods) Create(ctx context.Context, gpuPod *gpupodv1.GpuPod, opts v1.CreateOptions) (result *gpupodv1.GpuPod, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewCreateAction(gpupodsResource, c.ns, gpuPod), &gpupodv1.GpuPod{})

	if obj == nil {
		return nil, err
	}
	return obj.(*gpupodv1.GpuPod), err
}

// Update takes the representation of a gpuPod and updates it. Returns the server's representation of the gpuPod, and an error, if there is any.
func (c *FakeGpuPods) Update(ctx context.Context, gpuPod *gpupodv1.GpuPod, opts v1.UpdateOptions) (result *gpupodv1.GpuPod, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateAction(gpupodsResource, c.ns, gpuPod), &gpupodv1.GpuPod{})

	if obj == nil {
		return nil, err
	}
	return obj.(*gpupodv1.GpuPod), err
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *FakeGpuPods) UpdateStatus(ctx context.Context, gpuPod *gpupodv1.GpuPod, opts v1.UpdateOptions) (*gpupodv1.GpuPod, error) {
	obj, err := c.Fake.
		Invokes(testing.NewUpdateSubresourceAction(gpupodsResource, "status", c.ns, gpuPod), &gpupodv1.GpuPod{})

	if obj == nil {
		return nil, err
	}
	return obj.(*gpupodv1.GpuPod), err
}

// Delete takes name of the gpuPod and deletes it. Returns an error if one occurs.
func (c *FakeGpuPods) Delete(ctx context.Context, name string, opts v1.DeleteOptions) error {
	_, err := c.Fake.
		Invokes(testing.NewDeleteAction(gpupodsResource, c.ns, name), &gpupodv1.GpuPod{})

	return err
}

// DeleteCollection deletes a collection of objects.
func (c *FakeGpuPods) DeleteCollection(ctx context.Context, opts v1.DeleteOptions, listOpts v1.ListOptions) error {
	action := testing.NewDeleteCollectionAction(gpupodsResource, c.ns, listOpts)

	_, err := c.Fake.Invokes(action, &gpupodv1.GpuPodList{})
	return err
}

// Patch applies the patch and returns the patched gpuPod.
func (c *FakeGpuPods) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *gpupodv1.GpuPod, err error) {
	obj, err := c.Fake.
		Invokes(testing.NewPatchSubresourceAction(gpupodsResource, c.ns, name, pt, data, subresources...), &gpupodv1.GpuPod{})

	if obj == nil {
		return nil, err
	}
	return obj.(*gpupodv1.GpuPod), err
}
