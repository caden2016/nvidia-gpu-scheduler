package server

import (
	"github.com/caden2016/nvidia-gpu-scheduler/api/jsonstruct"
	"github.com/caden2016/nvidia-gpu-scheduler/cmd/gpuserver-ds/app/options"
	"github.com/caden2016/nvidia-gpu-scheduler/pkg/gpuserver/apis"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/sets"
	"reflect"
	"testing"
)

func TestGetBusyDeviceSet(t *testing.T) {
	type args struct {
		prl []*jsonstruct.PodResourcesDetail
	}
	tests := []struct {
		name string
		args args
		want sets.String
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetBusyDeviceSet(tt.args.prl); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetBusyDeviceSet() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetPodRequestGpuNum(t *testing.T) {
	type args struct {
		pod *corev1.Pod
	}
	var tests = []struct {
		name string
		args args
		want int64
	}{
		{
			name: "pod one container gpu request 2",
			args: args{&corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									options.NVIDIAGPUResourceName: *resource.NewQuantity(2, resource.DecimalExponent),
								},
							},
						},
					},
				},
			}},
			want: 2,
		},
		{
			name: "pod two containers gpu request 2 and 3",
			args: args{&corev1.Pod{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									options.NVIDIAGPUResourceName: *resource.NewQuantity(2, resource.DecimalExponent),
								},
							},
						},
						{
							Resources: corev1.ResourceRequirements{
								Limits: corev1.ResourceList{
									options.NVIDIAGPUResourceName: *resource.NewQuantity(3, resource.DecimalExponent),
								},
							},
						},
					},
				},
			}},
			want: 5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetPodRequestGpuNum(tt.args.pod); got != tt.want {
				t.Errorf("GetPodRequestGpuNum() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMapSetToList(t *testing.T) {
	type args struct {
		mapset map[string]sets.String
	}
	set1 := sets.NewString("a", "b", "c")
	set2 := sets.NewString("d", "e", "f")
	var (
		tests = []struct {
			name string
			args args
			want map[string][]string
		}{
			{name: "test1",
				args: args{map[string]sets.String{"set1": set1, "set2": set2}},
				want: map[string][]string{
					"set1": {"a", "b", "c"},
					"set2": {"d", "e", "f"},
				},
			},
		}
	)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MapSetToList(tt.args.mapset); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MapSetToList() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPodResourcesDetailToPodResource(t *testing.T) {
	type args struct {
		prdList []*jsonstruct.PodResourcesDetail
	}
	tests := []struct {
		name string
		args args
		want []*apis.PodResource
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := PodResourcesDetailToPodResource(tt.args.prdList); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PodResourcesDetailToPodResource() = %v, want %v", got, tt.want)
			}
		})
	}
}
