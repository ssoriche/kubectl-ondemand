package analysis

import (
	"context"
	"sort"
	"sync"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"

	"github.com/ssoriche/kubectl-ondemand/internal/karpenter"
)

type NodeAnalysis struct {
	Node               *corev1.Node
	PoolName           string
	PoolVersion        karpenter.APIVersion
	InstanceType       string
	CPUUtilization     int
	MemoryUtilization  int
	Reason             NodeReason
	SpotCapablePercent int
	PodClassifications []PodClassification
	Pods               []corev1.Pod
}

type PodDetail struct {
	Namespace string
	Name      string
	CPU       string
	Memory    string
	Category  PodCategory
	Reasons   []Reason
	IsDaemon  bool
}

type Collector struct {
	client            kubernetes.Interface
	dynClient         dynamic.Interface
	capabilities      *karpenter.ClusterCapabilities
	classifier        *Classifier
	includeDaemonSets bool
}

func NewCollector(client kubernetes.Interface, dynClient dynamic.Interface, capabilities *karpenter.ClusterCapabilities, spotTaint string, includeDaemonSets bool) *Collector {
	return &Collector{
		client:            client,
		dynClient:         dynClient,
		capabilities:      capabilities,
		classifier:        NewClassifier(spotTaint),
		includeDaemonSets: includeDaemonSets,
	}
}

func (c *Collector) Collect(ctx context.Context, nodeNames []string, selector string) ([]NodeAnalysis, error) {
	nodes, err := FetchOnDemandNodes(ctx, c.client, nodeNames, selector)
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, nil
	}

	var podsByNode map[string][]corev1.Pod
	var nodepoolConfigs map[string]bool
	var podErr, npErr error

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		podsByNode, podErr = FetchAllPods(ctx, c.client)
	}()
	go func() {
		defer wg.Done()
		nodepoolConfigs, npErr = FetchNodepoolConfigs(ctx, c.dynClient, c.capabilities)
	}()
	wg.Wait()

	if podErr != nil {
		return nil, podErr
	}
	if npErr != nil {
		nodepoolConfigs = make(map[string]bool)
	}

	return c.collectParallel(nodes, podsByNode, nodepoolConfigs)
}

const maxWorkers = 10

func (c *Collector) collectParallel(nodes []corev1.Node, podsByNode map[string][]corev1.Pod, nodepoolConfigs map[string]bool) ([]NodeAnalysis, error) {
	results := make([]NodeAnalysis, len(nodes))

	sem := make(chan struct{}, maxWorkers)
	var wg sync.WaitGroup

	for i := range nodes {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			nodeName := nodes[idx].Name
			results[idx] = c.analyzeNode(&nodes[idx], podsByNode[nodeName], nodepoolConfigs)
		}(i)
	}

	wg.Wait()

	sort.Slice(results, func(i, j int) bool {
		order := map[NodeReason]int{
			NodeReasonRequested:    0,
			NodeReasonSpotFallback: 1,
			NodeReasonInherited:    2,
		}
		if order[results[i].Reason] != order[results[j].Reason] {
			return order[results[i].Reason] < order[results[j].Reason]
		}
		return results[i].Node.Name < results[j].Node.Name
	})

	return results, nil
}

func (c *Collector) analyzeNode(node *corev1.Node, pods []corev1.Pod, nodepoolConfigs map[string]bool) NodeAnalysis {
	info := NodeAnalysis{Node: node}

	info.PoolName, info.PoolVersion = karpenter.GetPoolName(node)
	info.InstanceType = karpenter.GetInstanceType(node)
	info.CPUUtilization, info.MemoryUtilization = CalculateUtilization(node, pods)

	classifications := make([]PodClassification, len(pods))
	for i := range pods {
		if !c.includeDaemonSets && IsDaemonSetPod(&pods[i]) {
			classifications[i] = PodClassification{Category: CategorySystem, Reasons: []Reason{ReasonDaemonSet}}
		} else {
			classifications[i] = c.classifier.ClassifyPod(&pods[i])
		}
	}

	info.PodClassifications = classifications
	info.Pods = pods

	nodepoolAllowsSpot := nodepoolConfigs[info.PoolName]
	info.Reason = DetermineNodeReason(classifications, nodepoolAllowsSpot)
	info.SpotCapablePercent = CalculateSpotCapablePercent(classifications)

	return info
}

func GetPodDetails(analysis *NodeAnalysis) []PodDetail {
	details := make([]PodDetail, len(analysis.Pods))

	for i, pod := range analysis.Pods {
		cpu := "0"
		mem := "0"
		for _, container := range pod.Spec.Containers {
			if container.Resources.Requests != nil {
				if c, ok := container.Resources.Requests[corev1.ResourceCPU]; ok {
					cpu = c.String()
				}
				if m, ok := container.Resources.Requests[corev1.ResourceMemory]; ok {
					mem = m.String()
				}
			}
		}

		details[i] = PodDetail{
			Namespace: pod.Namespace,
			Name:      pod.Name,
			CPU:       cpu,
			Memory:    mem,
			Category:  analysis.PodClassifications[i].Category,
			Reasons:   analysis.PodClassifications[i].Reasons,
			IsDaemon:  IsDaemonSetPod(&pod),
		}
	}

	return details
}
