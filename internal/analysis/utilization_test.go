package analysis

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCalculateUtilization(t *testing.T) {
	tests := []struct {
		name           string
		node           *corev1.Node
		pods           []corev1.Pod
		expectedCPU    int
		expectedMemory int
	}{
		{
			name: "empty node",
			node: &corev1.Node{
				Status: corev1.NodeStatus{
					Allocatable: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("4"),
						corev1.ResourceMemory: resource.MustParse("16Gi"),
					},
				},
			},
			pods: nil, expectedCPU: 0, expectedMemory: 0,
		},
		{
			name: "50% utilization",
			node: &corev1.Node{
				Status: corev1.NodeStatus{
					Allocatable: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("4"),
						corev1.ResourceMemory: resource.MustParse("16Gi"),
					},
				},
			},
			pods: []corev1.Pod{{
				Status: corev1.PodStatus{Phase: corev1.PodRunning},
				Spec: corev1.PodSpec{Containers: []corev1.Container{{
					Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("2"),
						corev1.ResourceMemory: resource.MustParse("8Gi"),
					}},
				}}},
			}},
			expectedCPU: 50, expectedMemory: 50,
		},
		{
			name: "skips completed pods",
			node: &corev1.Node{
				Status: corev1.NodeStatus{
					Allocatable: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("4"),
						corev1.ResourceMemory: resource.MustParse("16Gi"),
					},
				},
			},
			pods: []corev1.Pod{{
				Status: corev1.PodStatus{Phase: corev1.PodSucceeded},
				Spec: corev1.PodSpec{Containers: []corev1.Container{{
					Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("4"),
					}},
				}}},
			}},
			expectedCPU: 0, expectedMemory: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cpu, mem := CalculateUtilization(tt.node, tt.pods)
			if cpu != tt.expectedCPU {
				t.Errorf("CPU = %d, want %d", cpu, tt.expectedCPU)
			}
			if mem != tt.expectedMemory {
				t.Errorf("Memory = %d, want %d", mem, tt.expectedMemory)
			}
		})
	}
}

func TestIsDaemonSetPod(t *testing.T) {
	tests := []struct {
		name     string
		pod      corev1.Pod
		expected bool
	}{
		{name: "regular pod", pod: corev1.Pod{}, expected: false},
		{
			name: "daemonset pod",
			pod: corev1.Pod{ObjectMeta: metav1.ObjectMeta{
				OwnerReferences: []metav1.OwnerReference{{Kind: "DaemonSet", Name: "fluentd"}},
			}},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsDaemonSetPod(&tt.pod)
			if got != tt.expected {
				t.Errorf("IsDaemonSetPod() = %v, want %v", got, tt.expected)
			}
		})
	}
}
