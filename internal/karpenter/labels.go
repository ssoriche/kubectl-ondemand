package karpenter

const (
	LabelProvisionerName = "karpenter.sh/provisioner-name"
)

const (
	LabelNodePool = "karpenter.sh/nodepool"
)

const (
	LabelCapacityType = "karpenter.sh/capacity-type"
	LabelInstanceType = "node.kubernetes.io/instance-type"
)

const (
	LabelTopologyZone = "topology.kubernetes.io/zone"
)

const (
	CapacityTypeOnDemand = "on-demand"
	CapacityTypeSpot     = "spot"
)
