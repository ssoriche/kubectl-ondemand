package analysis

import (
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/ssoriche/kubectl-ondemand/internal/karpenter"
)

// PodCategory represents why a pod is on an on-demand node
type PodCategory string

const (
	CategoryRequested PodCategory = "requested"
	CategoryInherited PodCategory = "inherited"
	CategorySpotOK    PodCategory = "spot-ok"
	CategorySystem    PodCategory = "system"
)

// Reason is a specific reason a pod is categorized a certain way
type Reason string

const (
	ReasonExplicitOnDemandSelector Reason = "explicit-ondemand-selector"
	ReasonOnDemandAffinity         Reason = "ondemand-affinity"
	ReasonMissingSpotToleration    Reason = "missing-spot-toleration"
	ReasonZonePinned               Reason = "zone-pinned"
	ReasonRestrictiveAntiAffinity  Reason = "restrictive-anti-affinity"
	ReasonRestrictiveAffinity      Reason = "restrictive-affinity"
	ReasonLocalStorage             Reason = "local-storage"
	ReasonDaemonSet                Reason = "daemonset"
)

// PodClassification contains the category and reasons for a pod's classification
type PodClassification struct {
	Category PodCategory
	Reasons  []Reason
}

// SpotTaint represents a parsed spot taint configuration
type SpotTaint struct {
	Key    string
	Value  string
	Effect corev1.TaintEffect
}

// Classifier classifies pods based on their scheduling constraints
type Classifier struct {
	spotTaint *SpotTaint
}

// NewClassifier creates a new Classifier with the given spot taint configuration.
// spotTaint should be in the format "key=value:Effect" (e.g., "karpenter.sh/disruption=spot:NoSchedule").
// An empty string skips the toleration check.
func NewClassifier(spotTaint string) *Classifier {
	var st *SpotTaint
	if spotTaint != "" {
		st = parseSpotTaint(spotTaint)
	}
	return &Classifier{
		spotTaint: st,
	}
}

// parseSpotTaint parses a spot taint string in the format "key=value:Effect"
func parseSpotTaint(taint string) *SpotTaint {
	parts := strings.Split(taint, ":")
	if len(parts) != 2 {
		return nil
	}

	effectStr := parts[1]
	var effect corev1.TaintEffect
	switch effectStr {
	case "NoSchedule":
		effect = corev1.TaintEffectNoSchedule
	case "NoExecute":
		effect = corev1.TaintEffectNoExecute
	case "PreferNoSchedule":
		effect = corev1.TaintEffectPreferNoSchedule
	default:
		return nil
	}

	keyValue := strings.Split(parts[0], "=")
	if len(keyValue) != 2 {
		return nil
	}

	return &SpotTaint{
		Key:    keyValue[0],
		Value:  keyValue[1],
		Effect: effect,
	}
}

// ClassifyPod classifies a pod based on its scheduling constraints
func (c *Classifier) ClassifyPod(pod *corev1.Pod) PodClassification {
	var reasons []Reason

	// Check for explicit on-demand selector
	if c.hasExplicitOnDemandSelector(pod) {
		reasons = append(reasons, ReasonExplicitOnDemandSelector)
	}

	// Check for on-demand node affinity
	if c.hasOnDemandAffinity(pod) {
		reasons = append(reasons, ReasonOnDemandAffinity)
	}

	// Check for spot toleration (if spot taint is configured)
	if c.spotTaint != nil && !c.hasSpotToleration(pod) {
		reasons = append(reasons, ReasonMissingSpotToleration)
	}

	// Check for zone pinning
	if c.isZonePinned(pod) {
		reasons = append(reasons, ReasonZonePinned)
	}

	// Check for restrictive pod anti-affinity
	if c.hasRestrictiveAntiAffinity(pod) {
		reasons = append(reasons, ReasonRestrictiveAntiAffinity)
	}

	// Check for restrictive pod affinity
	if c.hasRestrictiveAffinity(pod) {
		reasons = append(reasons, ReasonRestrictiveAffinity)
	}

	// Check for local storage
	if c.hasLocalStorage(pod) {
		reasons = append(reasons, ReasonLocalStorage)
	}

	// Determine category based on reasons
	category := c.determineCategory(reasons)

	return PodClassification{
		Category: category,
		Reasons:  reasons,
	}
}

// determineCategory determines the category based on the reasons
func (c *Classifier) determineCategory(reasons []Reason) PodCategory {
	if len(reasons) == 0 {
		return CategorySpotOK
	}

	// Check if any reason is "requested"
	for _, r := range reasons {
		if r == ReasonExplicitOnDemandSelector || r == ReasonOnDemandAffinity {
			return CategoryRequested
		}
	}

	// All other reasons are "inherited"
	return CategoryInherited
}

// hasExplicitOnDemandSelector checks if the pod has an explicit on-demand node selector
func (c *Classifier) hasExplicitOnDemandSelector(pod *corev1.Pod) bool {
	if pod.Spec.NodeSelector == nil {
		return false
	}
	capacityType, ok := pod.Spec.NodeSelector[karpenter.LabelCapacityType]
	return ok && capacityType == karpenter.CapacityTypeOnDemand
}

// hasOnDemandAffinity checks if the pod has required node affinity for on-demand capacity
func (c *Classifier) hasOnDemandAffinity(pod *corev1.Pod) bool {
	if pod.Spec.Affinity == nil || pod.Spec.Affinity.NodeAffinity == nil {
		return false
	}

	required := pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution
	if required == nil {
		return false
	}

	for _, term := range required.NodeSelectorTerms {
		for _, expr := range term.MatchExpressions {
			if expr.Key == karpenter.LabelCapacityType {
				for _, value := range expr.Values {
					if value == karpenter.CapacityTypeOnDemand {
						return true
					}
				}
			}
		}
	}

	return false
}

// hasSpotToleration checks if the pod has a toleration for the configured spot taint
func (c *Classifier) hasSpotToleration(pod *corev1.Pod) bool {
	if c.spotTaint == nil {
		return true // No spot taint configured, so we consider it tolerated
	}

	for _, toleration := range pod.Spec.Tolerations {
		// Check for exact match with Equal operator
		if toleration.Operator == corev1.TolerationOpEqual {
			if toleration.Key == c.spotTaint.Key &&
				toleration.Value == c.spotTaint.Value &&
				toleration.Effect == c.spotTaint.Effect {
				return true
			}
		}

		// Check for match with Exists operator
		if toleration.Operator == corev1.TolerationOpExists {
			if toleration.Key == c.spotTaint.Key &&
				toleration.Effect == c.spotTaint.Effect {
				return true
			}
		}
	}

	return false
}

// isZonePinned checks if the pod is pinned to a specific zone
func (c *Classifier) isZonePinned(pod *corev1.Pod) bool {
	// Check node selector
	if pod.Spec.NodeSelector != nil {
		if _, ok := pod.Spec.NodeSelector[karpenter.LabelTopologyZone]; ok {
			return true
		}
	}

	// Check node affinity
	if pod.Spec.Affinity == nil || pod.Spec.Affinity.NodeAffinity == nil {
		return false
	}

	required := pod.Spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution
	if required == nil {
		return false
	}

	for _, term := range required.NodeSelectorTerms {
		for _, expr := range term.MatchExpressions {
			if expr.Key == karpenter.LabelTopologyZone {
				return true
			}
		}
	}

	return false
}

// hasRestrictiveAntiAffinity checks if the pod has required pod anti-affinity
func (c *Classifier) hasRestrictiveAntiAffinity(pod *corev1.Pod) bool {
	if pod.Spec.Affinity == nil || pod.Spec.Affinity.PodAntiAffinity == nil {
		return false
	}

	return len(pod.Spec.Affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution) > 0
}

// hasRestrictiveAffinity checks if the pod has required pod affinity
func (c *Classifier) hasRestrictiveAffinity(pod *corev1.Pod) bool {
	if pod.Spec.Affinity == nil || pod.Spec.Affinity.PodAffinity == nil {
		return false
	}

	return len(pod.Spec.Affinity.PodAffinity.RequiredDuringSchedulingIgnoredDuringExecution) > 0
}

// hasLocalStorage checks if the pod uses local storage (hostPath or emptyDir with Memory medium)
func (c *Classifier) hasLocalStorage(pod *corev1.Pod) bool {
	for _, volume := range pod.Spec.Volumes {
		// Check for hostPath
		if volume.HostPath != nil {
			return true
		}

		// Check for emptyDir with Memory medium
		if volume.EmptyDir != nil && volume.EmptyDir.Medium == corev1.StorageMediumMemory {
			return true
		}
	}

	return false
}
