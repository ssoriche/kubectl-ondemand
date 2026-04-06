package analysis

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func FetchAllPods(ctx context.Context, client kubernetes.Interface) (map[string][]corev1.Pod, error) {
	podList, err := client.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	podsByNode := make(map[string][]corev1.Pod)
	for _, pod := range podList.Items {
		if nodeName := pod.Spec.NodeName; nodeName != "" {
			podsByNode[nodeName] = append(podsByNode[nodeName], pod)
		}
	}
	return podsByNode, nil
}
