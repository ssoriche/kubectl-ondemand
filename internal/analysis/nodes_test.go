package analysis

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDetermineNodeReason(t *testing.T) {
	tests := []struct {
		name               string
		classifications    []PodClassification
		nodepoolAllowsSpot bool
		expected           NodeReason
	}{
		{
			name: "any requested pod means node is requested",
			classifications: []PodClassification{
				{Category: CategoryRequested, Reasons: []Reason{ReasonExplicitOnDemandSelector}},
				{Category: CategorySpotOK},
			},
			nodepoolAllowsSpot: true,
			expected:           NodeReasonRequested,
		},
		{
			name: "all spot-ok with nodepool allowing spot means fallback",
			classifications: []PodClassification{
				{Category: CategorySpotOK},
				{Category: CategorySpotOK},
			},
			nodepoolAllowsSpot: true,
			expected:           NodeReasonSpotFallback,
		},
		{
			name: "all spot-ok but nodepool doesn't allow spot means inherited",
			classifications: []PodClassification{
				{Category: CategorySpotOK},
			},
			nodepoolAllowsSpot: false,
			expected:           NodeReasonInherited,
		},
		{
			name: "inherited pods means inherited",
			classifications: []PodClassification{
				{Category: CategoryInherited, Reasons: []Reason{ReasonMissingSpotToleration}},
				{Category: CategorySpotOK},
			},
			nodepoolAllowsSpot: true,
			expected:           NodeReasonInherited,
		},
		{
			name:               "empty classifications with spot allowed means fallback",
			classifications:    nil,
			nodepoolAllowsSpot: true,
			expected:           NodeReasonSpotFallback,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetermineNodeReason(tt.classifications, tt.nodepoolAllowsSpot)
			if got != tt.expected {
				t.Errorf("DetermineNodeReason() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestCalculateSpotCapablePercent(t *testing.T) {
	tests := []struct {
		name            string
		classifications []PodClassification
		pods            []corev1.Pod
		expected        int
	}{
		{name: "empty", classifications: nil, pods: nil, expected: 0},
		{
			name:            "all spot-ok workloads",
			classifications: []PodClassification{{Category: CategorySpotOK}, {Category: CategorySpotOK}},
			pods:            []corev1.Pod{{}, {}},
			expected:        100,
		},
		{
			name:            "50% spot-ok",
			classifications: []PodClassification{{Category: CategorySpotOK}, {Category: CategoryInherited, Reasons: []Reason{ReasonZonePinned}}},
			pods:            []corev1.Pod{{}, {}},
			expected:        50,
		},
		{
			name: "daemonset pods excluded from count",
			classifications: []PodClassification{
				{Category: CategorySpotOK},
				{Category: CategoryInherited, Reasons: []Reason{ReasonLocalStorage}},
			},
			pods: []corev1.Pod{
				{},
				{ObjectMeta: metav1.ObjectMeta{OwnerReferences: []metav1.OwnerReference{{Kind: "DaemonSet", Name: "fluentd"}}}},
			},
			expected: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateSpotCapablePercent(tt.classifications, tt.pods)
			if got != tt.expected {
				t.Errorf("CalculateSpotCapablePercent() = %d, want %d", got, tt.expected)
			}
		})
	}
}
