package karpenter

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDetectNodeVersion(t *testing.T) {
	tests := []struct {
		name     string
		node     *corev1.Node
		expected APIVersion
	}{
		{name: "nil node", node: nil, expected: APIVersionUnknown},
		{name: "no labels", node: &corev1.Node{}, expected: APIVersionUnknown},
		{
			name: "v1beta1 nodepool label",
			node: &corev1.Node{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{LabelNodePool: "default"}}},
			expected: APIVersionV1Beta1,
		},
		{
			name: "v1alpha5 provisioner label",
			node: &corev1.Node{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{LabelProvisionerName: "default"}}},
			expected: APIVersionV1Alpha5,
		},
		{
			name: "both labels prefers v1beta1",
			node: &corev1.Node{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{LabelNodePool: "default", LabelProvisionerName: "default"}}},
			expected: APIVersionV1Beta1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectNodeVersion(tt.node)
			if got != tt.expected {
				t.Errorf("DetectNodeVersion() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetPoolName(t *testing.T) {
	tests := []struct {
		name            string
		node            *corev1.Node
		expectedName    string
		expectedVersion APIVersion
	}{
		{name: "nil node", node: nil, expectedName: "", expectedVersion: APIVersionUnknown},
		{
			name: "v1beta1 nodepool",
			node: &corev1.Node{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{LabelNodePool: "gpu-pool"}}},
			expectedName: "gpu-pool", expectedVersion: APIVersionV1Beta1,
		},
		{
			name: "v1alpha5 provisioner",
			node: &corev1.Node{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{LabelProvisionerName: "batch"}}},
			expectedName: "batch", expectedVersion: APIVersionV1Alpha5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			name, version := GetPoolName(tt.node)
			if name != tt.expectedName {
				t.Errorf("GetPoolName() name = %v, want %v", name, tt.expectedName)
			}
			if version != tt.expectedVersion {
				t.Errorf("GetPoolName() version = %v, want %v", version, tt.expectedVersion)
			}
		})
	}
}

func TestGetCapacityType(t *testing.T) {
	tests := []struct {
		name     string
		node     *corev1.Node
		expected string
	}{
		{name: "nil node", node: nil, expected: ""},
		{
			name: "on-demand",
			node: &corev1.Node{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{LabelCapacityType: "on-demand"}}},
			expected: "on-demand",
		},
		{
			name: "spot",
			node: &corev1.Node{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{LabelCapacityType: "spot"}}},
			expected: "spot",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetCapacityType(tt.node)
			if got != tt.expected {
				t.Errorf("GetCapacityType() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetInstanceType(t *testing.T) {
	node := &corev1.Node{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{LabelInstanceType: "m6i.8xlarge"}}}
	got := GetInstanceType(node)
	if got != "m6i.8xlarge" {
		t.Errorf("GetInstanceType() = %v, want m6i.8xlarge", got)
	}
}

func TestDeterminePoolColumnHeader(t *testing.T) {
	tests := []struct {
		name     string
		caps     ClusterCapabilities
		expected string
	}{
		{name: "has nodepools", caps: ClusterCapabilities{HasNodePools: true}, expected: "NODEPOOL"},
		{name: "has provisioners only", caps: ClusterCapabilities{HasProvisioners: true}, expected: "PROVISIONER"},
		{name: "empty", caps: ClusterCapabilities{}, expected: "NODEPOOL"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.caps.DeterminePoolColumnHeader()
			if got != tt.expected {
				t.Errorf("DeterminePoolColumnHeader() = %v, want %v", got, tt.expected)
			}
		})
	}
}
