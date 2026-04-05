package karpenter

import (
	corev1 "k8s.io/api/core/v1"
)

type APIVersion string

const (
	APIVersionV1Alpha5 APIVersion = "v1alpha5"
	APIVersionV1Beta1  APIVersion = "v1beta1"
	APIVersionV1       APIVersion = "v1"
	APIVersionUnknown  APIVersion = "unknown"
)

type ClusterCapabilities struct {
	HasNodeClaims   bool
	HasMachines     bool
	HasNodePools    bool
	HasProvisioners bool
	PrimaryVersion  APIVersion
}

func DetectNodeVersion(node *corev1.Node) APIVersion {
	if node == nil || node.Labels == nil {
		return APIVersionUnknown
	}
	if _, ok := node.Labels[LabelNodePool]; ok {
		return APIVersionV1Beta1
	}
	if _, ok := node.Labels[LabelProvisionerName]; ok {
		return APIVersionV1Alpha5
	}
	return APIVersionUnknown
}

func GetPoolName(node *corev1.Node) (name string, version APIVersion) {
	if node == nil || node.Labels == nil {
		return "", APIVersionUnknown
	}
	if name, ok := node.Labels[LabelNodePool]; ok {
		return name, APIVersionV1Beta1
	}
	if name, ok := node.Labels[LabelProvisionerName]; ok {
		return name, APIVersionV1Alpha5
	}
	return "", APIVersionUnknown
}

func GetCapacityType(node *corev1.Node) string {
	if node == nil || node.Labels == nil {
		return ""
	}
	return node.Labels[LabelCapacityType]
}

func GetInstanceType(node *corev1.Node) string {
	if node == nil || node.Labels == nil {
		return ""
	}
	return node.Labels[LabelInstanceType]
}

func (c *ClusterCapabilities) DeterminePoolColumnHeader() string {
	if c.HasNodePools || c.HasNodeClaims {
		return "NODEPOOL"
	}
	if c.HasProvisioners || c.HasMachines {
		return "PROVISIONER"
	}
	return "NODEPOOL"
}

func (c *ClusterCapabilities) HasKarpenter() bool {
	return c.HasNodeClaims || c.HasMachines || c.HasNodePools || c.HasProvisioners
}
