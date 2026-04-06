package analysis

import (
	"context"
	"sort"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/ssoriche/kubectl-ondemand/internal/karpenter"
)

type NodeReason string

const (
	NodeReasonRequested    NodeReason = "requested"
	NodeReasonSpotFallback NodeReason = "spot-fallback"
	NodeReasonInherited    NodeReason = "inherited"
)

func FetchOnDemandNodes(ctx context.Context, client kubernetes.Interface, nodeNames []string, selector string) ([]corev1.Node, error) {
	onDemandSelector := karpenter.LabelCapacityType + "=" + karpenter.CapacityTypeOnDemand
	if selector != "" {
		onDemandSelector = selector + "," + onDemandSelector
	}

	listOpts := metav1.ListOptions{LabelSelector: onDemandSelector}

	nodeList, err := client.CoreV1().Nodes().List(ctx, listOpts)
	if err != nil {
		return nil, err
	}

	nodes := nodeList.Items

	if len(nodeNames) > 0 {
		nameSet := make(map[string]bool)
		for _, name := range nodeNames {
			nameSet[name] = true
		}
		filtered := make([]corev1.Node, 0, len(nodeNames))
		for _, node := range nodes {
			if nameSet[node.Name] {
				filtered = append(filtered, node)
			}
		}
		nodes = filtered
	}

	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].CreationTimestamp.Before(&nodes[j].CreationTimestamp)
	})

	return nodes, nil
}

func DetermineNodeReason(classifications []PodClassification, nodepoolAllowsSpot bool) NodeReason {
	hasRequested := false
	hasInherited := false

	for _, c := range classifications {
		switch c.Category {
		case CategoryRequested:
			hasRequested = true
		case CategoryInherited:
			hasInherited = true
		}
	}

	if hasRequested {
		return NodeReasonRequested
	}

	if nodepoolAllowsSpot && !hasInherited {
		return NodeReasonSpotFallback
	}

	if hasInherited {
		return NodeReasonInherited
	}

	return NodeReasonInherited
}

func CalculateSpotCapablePercent(classifications []PodClassification, pods []corev1.Pod) int {
	if len(classifications) == 0 {
		return 0
	}

	workloadCount := 0
	spotOKCount := 0

	for i, c := range classifications {
		if IsDaemonSetPod(&pods[i]) {
			continue
		}
		workloadCount++
		if c.Category == CategorySpotOK {
			spotOKCount++
		}
	}

	if workloadCount == 0 {
		return 0
	}

	return (spotOKCount * 100) / workloadCount
}

func GetNodeStatus(node *corev1.Node) string {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			if condition.Status == corev1.ConditionTrue {
				return "Ready"
			}
			return "NotReady"
		}
	}
	return "Unknown"
}
