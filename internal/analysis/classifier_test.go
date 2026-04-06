package analysis

import (
    "testing"

    corev1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

    "github.com/ssoriche/kubectl-ondemand/internal/karpenter"
)

func TestNewClassifier(t *testing.T) {
    tests := []struct {
        name        string
        spotTaint   string
        wantKey     string
        wantValue   string
        wantEffect  corev1.TaintEffect
        wantNil     bool
    }{
        {
            name:       "valid spot taint",
            spotTaint:  "karpenter.sh/disruption=spot:NoSchedule",
            wantKey:    "karpenter.sh/disruption",
            wantValue:  "spot",
            wantEffect: corev1.TaintEffectNoSchedule,
            wantNil:    false,
        },
        {
            name:       "NoExecute effect",
            spotTaint:  "spot=true:NoExecute",
            wantKey:    "spot",
            wantValue:  "true",
            wantEffect: corev1.TaintEffectNoExecute,
            wantNil:    false,
        },
        {
            name:       "PreferNoSchedule effect",
            spotTaint:  "spot=true:PreferNoSchedule",
            wantKey:    "spot",
            wantValue:  "true",
            wantEffect: corev1.TaintEffectPreferNoSchedule,
            wantNil:    false,
        },
        {
            name:      "empty spot taint",
            spotTaint: "",
            wantNil:   true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            c := NewClassifier(tt.spotTaint)
            if tt.wantNil {
                if c.spotTaint != nil {
                    t.Errorf("expected nil spotTaint, got %+v", c.spotTaint)
                }
            } else {
                if c.spotTaint == nil {
                    t.Fatal("expected non-nil spotTaint")
                }
                if c.spotTaint.Key != tt.wantKey {
                    t.Errorf("Key = %q, want %q", c.spotTaint.Key, tt.wantKey)
                }
                if c.spotTaint.Value != tt.wantValue {
                    t.Errorf("Value = %q, want %q", c.spotTaint.Value, tt.wantValue)
                }
                if c.spotTaint.Effect != tt.wantEffect {
                    t.Errorf("Effect = %q, want %q", c.spotTaint.Effect, tt.wantEffect)
                }
            }
        })
    }
}

func TestClassifyPod_ExplicitOnDemandSelector(t *testing.T) {
    c := NewClassifier("")
    pod := &corev1.Pod{
        Spec: corev1.PodSpec{
            NodeSelector: map[string]string{
                karpenter.LabelCapacityType: karpenter.CapacityTypeOnDemand,
            },
        },
    }

    result := c.ClassifyPod(pod)

    if result.Category != CategoryRequested {
        t.Errorf("Category = %q, want %q", result.Category, CategoryRequested)
    }
    if len(result.Reasons) != 1 {
        t.Fatalf("len(Reasons) = %d, want 1", len(result.Reasons))
    }
    if result.Reasons[0] != ReasonExplicitOnDemandSelector {
        t.Errorf("Reasons[0] = %q, want %q", result.Reasons[0], ReasonExplicitOnDemandSelector)
    }
}

func TestClassifyPod_OnDemandAffinity(t *testing.T) {
    c := NewClassifier("")
    pod := &corev1.Pod{
        Spec: corev1.PodSpec{
            Affinity: &corev1.Affinity{
                NodeAffinity: &corev1.NodeAffinity{
                    RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
                        NodeSelectorTerms: []corev1.NodeSelectorTerm{
                            {
                                MatchExpressions: []corev1.NodeSelectorRequirement{
                                    {
                                        Key:      karpenter.LabelCapacityType,
                                        Operator: corev1.NodeSelectorOpIn,
                                        Values:   []string{karpenter.CapacityTypeOnDemand},
                                    },
                                },
                            },
                        },
                    },
                },
            },
        },
    }

    result := c.ClassifyPod(pod)

    if result.Category != CategoryRequested {
        t.Errorf("Category = %q, want %q", result.Category, CategoryRequested)
    }
    if len(result.Reasons) != 1 {
        t.Fatalf("len(Reasons) = %d, want 1", len(result.Reasons))
    }
    if result.Reasons[0] != ReasonOnDemandAffinity {
        t.Errorf("Reasons[0] = %q, want %q", result.Reasons[0], ReasonOnDemandAffinity)
    }
}

func TestClassifyPod_MissingSpotToleration(t *testing.T) {
    c := NewClassifier("karpenter.sh/disruption=spot:NoSchedule")
    pod := &corev1.Pod{
        Spec: corev1.PodSpec{
            Tolerations: []corev1.Toleration{
                {
                    Key:      "some-other-taint",
                    Operator: corev1.TolerationOpEqual,
                    Value:    "value",
                    Effect:   corev1.TaintEffectNoSchedule,
                },
            },
        },
    }

    result := c.ClassifyPod(pod)

    if result.Category != CategoryInherited {
        t.Errorf("Category = %q, want %q", result.Category, CategoryInherited)
    }
    if len(result.Reasons) != 1 {
        t.Fatalf("len(Reasons) = %d, want 1", len(result.Reasons))
    }
    if result.Reasons[0] != ReasonMissingSpotToleration {
        t.Errorf("Reasons[0] = %q, want %q", result.Reasons[0], ReasonMissingSpotToleration)
    }
}

func TestClassifyPod_HasSpotToleration(t *testing.T) {
    tests := []struct {
        name        string
        tolerations []corev1.Toleration
    }{
        {
            name: "exact match with Equal operator",
            tolerations: []corev1.Toleration{
                {
                    Key:      "karpenter.sh/disruption",
                    Operator: corev1.TolerationOpEqual,
                    Value:    "spot",
                    Effect:   corev1.TaintEffectNoSchedule,
                },
            },
        },
        {
            name: "match with Exists operator",
            tolerations: []corev1.Toleration{
                {
                    Key:      "karpenter.sh/disruption",
                    Operator: corev1.TolerationOpExists,
                    Effect:   corev1.TaintEffectNoSchedule,
                },
            },
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            c := NewClassifier("karpenter.sh/disruption=spot:NoSchedule")
            pod := &corev1.Pod{
                Spec: corev1.PodSpec{
                    Tolerations: tt.tolerations,
                },
            }

            result := c.ClassifyPod(pod)

            if result.Category != CategorySpotOK {
                t.Errorf("Category = %q, want %q", result.Category, CategorySpotOK)
            }
            if len(result.Reasons) != 0 {
                t.Errorf("len(Reasons) = %d, want 0", len(result.Reasons))
            }
        })
    }
}

func TestClassifyPod_ZonePinnedNodeSelector(t *testing.T) {
    c := NewClassifier("")
    pod := &corev1.Pod{
        Spec: corev1.PodSpec{
            NodeSelector: map[string]string{
                "topology.kubernetes.io/zone": "us-east-1a",
            },
        },
    }

    result := c.ClassifyPod(pod)

    if result.Category != CategoryInherited {
        t.Errorf("Category = %q, want %q", result.Category, CategoryInherited)
    }
    if len(result.Reasons) != 1 {
        t.Fatalf("len(Reasons) = %d, want 1", len(result.Reasons))
    }
    if result.Reasons[0] != ReasonZonePinned {
        t.Errorf("Reasons[0] = %q, want %q", result.Reasons[0], ReasonZonePinned)
    }
}

func TestClassifyPod_ZonePinnedAffinity(t *testing.T) {
    c := NewClassifier("")
    pod := &corev1.Pod{
        Spec: corev1.PodSpec{
            Affinity: &corev1.Affinity{
                NodeAffinity: &corev1.NodeAffinity{
                    RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
                        NodeSelectorTerms: []corev1.NodeSelectorTerm{
                            {
                                MatchExpressions: []corev1.NodeSelectorRequirement{
                                    {
                                        Key:      "topology.kubernetes.io/zone",
                                        Operator: corev1.NodeSelectorOpIn,
                                        Values:   []string{"us-east-1a"},
                                    },
                                },
                            },
                        },
                    },
                },
            },
        },
    }

    result := c.ClassifyPod(pod)

    if result.Category != CategoryInherited {
        t.Errorf("Category = %q, want %q", result.Category, CategoryInherited)
    }
    if len(result.Reasons) != 1 {
        t.Fatalf("len(Reasons) = %d, want 1", len(result.Reasons))
    }
    if result.Reasons[0] != ReasonZonePinned {
        t.Errorf("Reasons[0] = %q, want %q", result.Reasons[0], ReasonZonePinned)
    }
}

func TestClassifyPod_RequiredPodAntiAffinity(t *testing.T) {
    c := NewClassifier("")
    pod := &corev1.Pod{
        Spec: corev1.PodSpec{
            Affinity: &corev1.Affinity{
                PodAntiAffinity: &corev1.PodAntiAffinity{
                    RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
                        {
                            LabelSelector: &metav1.LabelSelector{
                                MatchLabels: map[string]string{
                                    "app": "myapp",
                                },
                            },
                            TopologyKey: "kubernetes.io/hostname",
                        },
                    },
                },
            },
        },
    }

    result := c.ClassifyPod(pod)

    if result.Category != CategoryInherited {
        t.Errorf("Category = %q, want %q", result.Category, CategoryInherited)
    }
    if len(result.Reasons) != 1 {
        t.Fatalf("len(Reasons) = %d, want 1", len(result.Reasons))
    }
    if result.Reasons[0] != ReasonRestrictiveAntiAffinity {
        t.Errorf("Reasons[0] = %q, want %q", result.Reasons[0], ReasonRestrictiveAntiAffinity)
    }
}

func TestClassifyPod_PreferredPodAntiAffinity(t *testing.T) {
    c := NewClassifier("")
    pod := &corev1.Pod{
        Spec: corev1.PodSpec{
            Affinity: &corev1.Affinity{
                PodAntiAffinity: &corev1.PodAntiAffinity{
                    PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
                        {
                            Weight: 100,
                            PodAffinityTerm: corev1.PodAffinityTerm{
                                LabelSelector: &metav1.LabelSelector{
                                    MatchLabels: map[string]string{
                                        "app": "myapp",
                                    },
                                },
                                TopologyKey: "kubernetes.io/hostname",
                            },
                        },
                    },
                },
            },
        },
    }

    result := c.ClassifyPod(pod)

    if result.Category != CategorySpotOK {
        t.Errorf("Category = %q, want %q", result.Category, CategorySpotOK)
    }
    if len(result.Reasons) != 0 {
        t.Errorf("len(Reasons) = %d, want 0", len(result.Reasons))
    }
}

func TestClassifyPod_RequiredPodAffinity(t *testing.T) {
    c := NewClassifier("")
    pod := &corev1.Pod{
        Spec: corev1.PodSpec{
            Affinity: &corev1.Affinity{
                PodAffinity: &corev1.PodAffinity{
                    RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
                        {
                            LabelSelector: &metav1.LabelSelector{
                                MatchLabels: map[string]string{
                                    "app": "myapp",
                                },
                            },
                            TopologyKey: "kubernetes.io/hostname",
                        },
                    },
                },
            },
        },
    }

    result := c.ClassifyPod(pod)

    if result.Category != CategoryInherited {
        t.Errorf("Category = %q, want %q", result.Category, CategoryInherited)
    }
    if len(result.Reasons) != 1 {
        t.Fatalf("len(Reasons) = %d, want 1", len(result.Reasons))
    }
    if result.Reasons[0] != ReasonRestrictiveAffinity {
        t.Errorf("Reasons[0] = %q, want %q", result.Reasons[0], ReasonRestrictiveAffinity)
    }
}

func TestClassifyPod_LocalStorageHostPath(t *testing.T) {
    c := NewClassifier("")
    pod := &corev1.Pod{
        Spec: corev1.PodSpec{
            Volumes: []corev1.Volume{
                {
                    Name: "host-volume",
                    VolumeSource: corev1.VolumeSource{
                        HostPath: &corev1.HostPathVolumeSource{
                            Path: "/var/data",
                        },
                    },
                },
            },
        },
    }

    result := c.ClassifyPod(pod)

    if result.Category != CategoryInherited {
        t.Errorf("Category = %q, want %q", result.Category, CategoryInherited)
    }
    if len(result.Reasons) != 1 {
        t.Fatalf("len(Reasons) = %d, want 1", len(result.Reasons))
    }
    if result.Reasons[0] != ReasonLocalStorage {
        t.Errorf("Reasons[0] = %q, want %q", result.Reasons[0], ReasonLocalStorage)
    }
}

func TestClassifyPod_LocalStorageEmptyDirMemory(t *testing.T) {
    c := NewClassifier("")
    medium := corev1.StorageMediumMemory
    pod := &corev1.Pod{
        Spec: corev1.PodSpec{
            Volumes: []corev1.Volume{
                {
                    Name: "memory-volume",
                    VolumeSource: corev1.VolumeSource{
                        EmptyDir: &corev1.EmptyDirVolumeSource{
                            Medium: medium,
                        },
                    },
                },
            },
        },
    }

    result := c.ClassifyPod(pod)

    if result.Category != CategoryInherited {
        t.Errorf("Category = %q, want %q", result.Category, CategoryInherited)
    }
    if len(result.Reasons) != 1 {
        t.Fatalf("len(Reasons) = %d, want 1", len(result.Reasons))
    }
    if result.Reasons[0] != ReasonLocalStorage {
        t.Errorf("Reasons[0] = %q, want %q", result.Reasons[0], ReasonLocalStorage)
    }
}

func TestClassifyPod_EmptyDirWithoutMemory(t *testing.T) {
    c := NewClassifier("")
    pod := &corev1.Pod{
        Spec: corev1.PodSpec{
            Volumes: []corev1.Volume{
                {
                    Name: "regular-volume",
                    VolumeSource: corev1.VolumeSource{
                        EmptyDir: &corev1.EmptyDirVolumeSource{},
                    },
                },
            },
        },
    }

    result := c.ClassifyPod(pod)

    if result.Category != CategorySpotOK {
        t.Errorf("Category = %q, want %q", result.Category, CategorySpotOK)
    }
    if len(result.Reasons) != 0 {
        t.Errorf("len(Reasons) = %d, want 0", len(result.Reasons))
    }
}

func TestClassifyPod_NoConstraints(t *testing.T) {
    c := NewClassifier("")
    pod := &corev1.Pod{
        Spec: corev1.PodSpec{},
    }

    result := c.ClassifyPod(pod)

    if result.Category != CategorySpotOK {
        t.Errorf("Category = %q, want %q", result.Category, CategorySpotOK)
    }
    if len(result.Reasons) != 0 {
        t.Errorf("len(Reasons) = %d, want 0", len(result.Reasons))
    }
}

func TestClassifyPod_MultipleReasonsRequestedWins(t *testing.T) {
    c := NewClassifier("karpenter.sh/disruption=spot:NoSchedule")
    pod := &corev1.Pod{
        Spec: corev1.PodSpec{
            NodeSelector: map[string]string{
                karpenter.LabelCapacityType:        karpenter.CapacityTypeOnDemand,
                "topology.kubernetes.io/zone":      "us-east-1a",
            },
        },
    }

    result := c.ClassifyPod(pod)

    if result.Category != CategoryRequested {
        t.Errorf("Category = %q, want %q", result.Category, CategoryRequested)
    }
    // Should have both reasons
    if len(result.Reasons) != 3 {
        t.Errorf("len(Reasons) = %d, want 3", len(result.Reasons))
    }
    // But category should be requested due to priority
    foundRequested := false
    foundInherited := false
    for _, r := range result.Reasons {
        if r == ReasonExplicitOnDemandSelector {
            foundRequested = true
        }
        if r == ReasonZonePinned || r == ReasonMissingSpotToleration {
            foundInherited = true
        }
    }
    if !foundRequested {
        t.Error("expected to find requested reason")
    }
    if !foundInherited {
        t.Error("expected to find inherited reason")
    }
}

func TestClassifyPod_NoSpotTaintConfigured(t *testing.T) {
    c := NewClassifier("")
    pod := &corev1.Pod{
        Spec: corev1.PodSpec{
            Tolerations: []corev1.Toleration{
                {
                    Key:      "some-taint",
                    Operator: corev1.TolerationOpEqual,
                    Value:    "value",
                    Effect:   corev1.TaintEffectNoSchedule,
                },
            },
        },
    }

    result := c.ClassifyPod(pod)

    // Should skip toleration check and be spot-ok
    if result.Category != CategorySpotOK {
        t.Errorf("Category = %q, want %q", result.Category, CategorySpotOK)
    }
    if len(result.Reasons) != 0 {
        t.Errorf("len(Reasons) = %d, want 0", len(result.Reasons))
    }
}
