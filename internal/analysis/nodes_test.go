package analysis

import (
	"testing"
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
		{
			name: "system pods ignored — only spot-ok remains with spot-allowing nodepool means fallback",
			classifications: []PodClassification{
				{Category: CategorySystem, Reasons: []Reason{ReasonDaemonSet}},
				{Category: CategorySpotOK},
			},
			nodepoolAllowsSpot: true,
			expected:           NodeReasonSpotFallback,
		},
		{
			name: "system pods ignored — inherited from system pod would have been inherited but is excluded",
			classifications: []PodClassification{
				{Category: CategorySystem, Reasons: []Reason{ReasonDaemonSet}},
				{Category: CategorySystem, Reasons: []Reason{ReasonDaemonSet}},
				{Category: CategorySpotOK},
			},
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
		expected        int
	}{
		{name: "empty", classifications: nil, expected: 0},
		{
			name:            "all spot-ok workloads",
			classifications: []PodClassification{{Category: CategorySpotOK}, {Category: CategorySpotOK}},
			expected:        100,
		},
		{
			name:            "50% spot-ok",
			classifications: []PodClassification{{Category: CategorySpotOK}, {Category: CategoryInherited, Reasons: []Reason{ReasonZonePinned}}},
			expected:        50,
		},
		{
			name: "system pods excluded from count",
			classifications: []PodClassification{
				{Category: CategorySpotOK},
				{Category: CategorySystem, Reasons: []Reason{ReasonDaemonSet}},
			},
			expected: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateSpotCapablePercent(tt.classifications)
			if got != tt.expected {
				t.Errorf("CalculateSpotCapablePercent() = %d, want %d", got, tt.expected)
			}
		})
	}
}
