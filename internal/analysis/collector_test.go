package analysis

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAnalyzeNode_DaemonSetClassification(t *testing.T) {
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: "test-node"},
		Status: corev1.NodeStatus{
			Allocatable: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("4"),
				corev1.ResourceMemory: resource.MustParse("8Gi"),
			},
		},
	}

	daemonSetPod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kube-proxy-abc",
			Namespace: "kube-system",
			OwnerReferences: []metav1.OwnerReference{
				{Kind: "DaemonSet", Name: "kube-proxy"},
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("100m"),
						corev1.ResourceMemory: resource.MustParse("128Mi"),
					},
				},
			}},
		},
	}

	regularPod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "web-server-abc",
			Namespace: "default",
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("500m"),
						corev1.ResourceMemory: resource.MustParse("512Mi"),
					},
				},
			}},
		},
	}

	pods := []corev1.Pod{daemonSetPod, regularPod}
	nodepoolConfigs := map[string]bool{}

	t.Run("daemonset pods classified as system by default", func(t *testing.T) {
		c := &Collector{
			classifier:        NewClassifier(""),
			includeDaemonSets: false,
		}
		result := c.analyzeNode(node, pods, nodepoolConfigs)

		if result.PodClassifications[0].Category != CategorySystem {
			t.Errorf("DaemonSet pod category = %v, want %v", result.PodClassifications[0].Category, CategorySystem)
		}
		if len(result.PodClassifications[0].Reasons) != 1 || result.PodClassifications[0].Reasons[0] != ReasonDaemonSet {
			t.Errorf("DaemonSet pod reasons = %v, want [%v]", result.PodClassifications[0].Reasons, ReasonDaemonSet)
		}
		if result.PodClassifications[1].Category != CategorySpotOK {
			t.Errorf("Regular pod category = %v, want %v", result.PodClassifications[1].Category, CategorySpotOK)
		}
	})

	t.Run("daemonset pods classified normally when includeDaemonSets is true", func(t *testing.T) {
		c := &Collector{
			classifier:        NewClassifier(""),
			includeDaemonSets: true,
		}
		result := c.analyzeNode(node, pods, nodepoolConfigs)

		if result.PodClassifications[0].Category == CategorySystem {
			t.Errorf("DaemonSet pod should not be system when includeDaemonSets=true, got %v", result.PodClassifications[0].Category)
		}
	})
}
