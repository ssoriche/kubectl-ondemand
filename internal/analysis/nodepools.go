package analysis

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/ssoriche/kubectl-ondemand/internal/karpenter"
)

var (
	nodePoolGVR = schema.GroupVersionResource{
		Group: "karpenter.sh", Version: "v1", Resource: "nodepools",
	}
	nodePoolV1Beta1GVR = schema.GroupVersionResource{
		Group: "karpenter.sh", Version: "v1beta1", Resource: "nodepools",
	}
	provisionerGVR = schema.GroupVersionResource{
		Group: "karpenter.sh", Version: "v1alpha5", Resource: "provisioners",
	}
)

func FetchNodepoolConfigs(ctx context.Context, dynClient dynamic.Interface, caps *karpenter.ClusterCapabilities) (map[string]bool, error) {
	configs := make(map[string]bool)

	if caps.HasNodePools {
		if err := fetchAndMap(ctx, dynClient, nodePoolGVR, configs); err != nil {
			if err := fetchAndMap(ctx, dynClient, nodePoolV1Beta1GVR, configs); err != nil {
				return nil, err
			}
		}
	}

	if caps.HasProvisioners {
		if err := fetchAndMap(ctx, dynClient, provisionerGVR, configs); err != nil {
			if len(configs) == 0 {
				return nil, err
			}
		}
	}

	return configs, nil
}

func fetchAndMap(ctx context.Context, dynClient dynamic.Interface, gvr schema.GroupVersionResource, configs map[string]bool) error {
	list, err := dynClient.Resource(gvr).List(ctx, metav1.ListOptions{})
	if err != nil {
		return err
	}

	for i := range list.Items {
		name := list.Items[i].GetName()
		configs[name] = NodepoolAllowsSpot(&list.Items[i])
	}
	return nil
}

func NodepoolAllowsSpot(obj *unstructured.Unstructured) bool {
	requirements := extractRequirements(obj)
	if requirements == nil {
		return true
	}

	for _, req := range requirements {
		reqMap, ok := req.(map[string]any)
		if !ok {
			continue
		}
		key, _ := reqMap["key"].(string)
		if key != karpenter.LabelCapacityType {
			continue
		}
		values, ok := reqMap["values"].([]any)
		if !ok {
			continue
		}
		for _, v := range values {
			if s, ok := v.(string); ok && s == karpenter.CapacityTypeSpot {
				return true
			}
		}
		return false
	}

	return true
}

func extractRequirements(obj *unstructured.Unstructured) []any {
	if tmpl, ok, _ := unstructured.NestedMap(obj.Object, "spec", "template", "spec"); ok {
		if reqs, ok := tmpl["requirements"].([]any); ok {
			return reqs
		}
	}

	if reqs, ok, _ := unstructured.NestedSlice(obj.Object, "spec", "requirements"); ok {
		return reqs
	}

	return nil
}
