package analysis

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestNodepoolAllowsSpot(t *testing.T) {
	tests := []struct {
		name     string
		obj      *unstructured.Unstructured
		expected bool
	}{
		{
			name: "nodepool with spot in requirements",
			obj: &unstructured.Unstructured{
				Object: map[string]any{
					"spec": map[string]any{
						"template": map[string]any{
							"spec": map[string]any{
								"requirements": []any{
									map[string]any{
										"key":      "karpenter.sh/capacity-type",
										"operator": "In",
										"values":   []any{"spot", "on-demand"},
									},
								},
							},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "nodepool with only on-demand",
			obj: &unstructured.Unstructured{
				Object: map[string]any{
					"spec": map[string]any{
						"template": map[string]any{
							"spec": map[string]any{
								"requirements": []any{
									map[string]any{
										"key":      "karpenter.sh/capacity-type",
										"operator": "In",
										"values":   []any{"on-demand"},
									},
								},
							},
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "provisioner v1alpha5 with spot",
			obj: &unstructured.Unstructured{
				Object: map[string]any{
					"spec": map[string]any{
						"requirements": []any{
							map[string]any{
								"key":      "karpenter.sh/capacity-type",
								"operator": "In",
								"values":   []any{"spot", "on-demand"},
							},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "no capacity-type requirement assumes spot allowed",
			obj: &unstructured.Unstructured{
				Object: map[string]any{
					"spec": map[string]any{
						"template": map[string]any{
							"spec": map[string]any{
								"requirements": []any{
									map[string]any{
										"key":      "node.kubernetes.io/instance-type",
										"operator": "In",
										"values":   []any{"m6i.xlarge"},
									},
								},
							},
						},
					},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NodepoolAllowsSpot(tt.obj)
			if got != tt.expected {
				t.Errorf("NodepoolAllowsSpot() = %v, want %v", got, tt.expected)
			}
		})
	}
}
