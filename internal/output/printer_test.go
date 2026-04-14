package output

import (
	"bytes"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/ssoriche/kubectl-ondemand/internal/analysis"
	"github.com/ssoriche/kubectl-ondemand/internal/karpenter"
)

func TestPrintNodesTable(t *testing.T) {
	nodes := []analysis.NodeAnalysis{
		{
			Node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node-1", CreationTimestamp: metav1.Now(),
					Labels: map[string]string{"node.kubernetes.io/instance-type": "m6i.8xlarge"},
				},
				Status: corev1.NodeStatus{
					Allocatable: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("32"),
						corev1.ResourceMemory: resource.MustParse("128Gi"),
					},
				},
			},
			PoolName: "default", InstanceType: "m6i.8xlarge",
			CPUUtilization: 82, MemoryUtilization: 71,
			Reason: analysis.NodeReasonSpotFallback, SpotCapablePercent: 75,
		},
	}

	var buf bytes.Buffer
	caps := &karpenter.ClusterCapabilities{HasNodePools: true}
	p := &Printer{out: &buf, capabilities: caps}

	err := p.PrintNodes(nodes)
	if err != nil {
		t.Fatalf("PrintNodes() error = %v", err)
	}

	output := buf.String()
	for _, want := range []string{"NODEPOOL", "ON-DEMAND-REASON", "SPOT-CAPABLE%", "spot-fallback", "75%"} {
		if !strings.Contains(output, want) {
			t.Errorf("expected %q in output", want)
		}
	}
}

func TestPrintNodesJSON(t *testing.T) {
	nodes := []analysis.NodeAnalysis{
		{
			Node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: "node-1", CreationTimestamp: metav1.Now()},
				Status: corev1.NodeStatus{Allocatable: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse("4"), corev1.ResourceMemory: resource.MustParse("16Gi"),
				}},
			},
			PoolName: "default", InstanceType: "m6i.xlarge",
			Reason: analysis.NodeReasonRequested, SpotCapablePercent: 0,
		},
	}

	var buf bytes.Buffer
	caps := &karpenter.ClusterCapabilities{}
	p := &Printer{out: &buf, outputFormat: "json", capabilities: caps}

	err := p.PrintNodes(nodes)
	if err != nil {
		t.Fatalf("PrintNodes() error = %v", err)
	}

	output := buf.String()
	for _, want := range []string{`"onDemandReason"`, `"requested"`} {
		if !strings.Contains(output, want) {
			t.Errorf("expected %q in JSON output", want)
		}
	}
}

func TestPrintNodesShowLabels(t *testing.T) {
	nodes := []analysis.NodeAnalysis{
		{
			Node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node-1", CreationTimestamp: metav1.Now(),
					Labels: map[string]string{
						"node.kubernetes.io/instance-type": "m6i.8xlarge",
						"topology.kubernetes.io/zone":      "us-east-1a",
						"team":                             "platform",
					},
				},
				Status: corev1.NodeStatus{
					Allocatable: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("32"),
						corev1.ResourceMemory: resource.MustParse("128Gi"),
					},
				},
			},
			PoolName: "default", InstanceType: "m6i.8xlarge",
			Reason: analysis.NodeReasonRequested,
		},
	}

	var buf bytes.Buffer
	caps := &karpenter.ClusterCapabilities{HasNodePools: true}
	p := &Printer{out: &buf, capabilities: caps, showLabels: true}

	err := p.PrintNodes(nodes)
	if err != nil {
		t.Fatalf("PrintNodes() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "LABELS") {
		t.Error("expected LABELS header in output")
	}
	// Labels are comma-separated key=value
	if !strings.Contains(output, "team=platform") {
		t.Errorf("expected label team=platform in output, got:\n%s", output)
	}
}

func TestPrintNodesLabelColumns(t *testing.T) {
	nodes := []analysis.NodeAnalysis{
		{
			Node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node-1", CreationTimestamp: metav1.Now(),
					Labels: map[string]string{
						"topology.kubernetes.io/zone": "us-east-1a",
						"team":                        "platform",
					},
				},
				Status: corev1.NodeStatus{
					Allocatable: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("4"),
						corev1.ResourceMemory: resource.MustParse("16Gi"),
					},
				},
			},
			PoolName: "default", InstanceType: "m6i.xlarge",
			Reason: analysis.NodeReasonRequested,
		},
	}

	var buf bytes.Buffer
	caps := &karpenter.ClusterCapabilities{HasNodePools: true}
	p := &Printer{out: &buf, capabilities: caps, labelColumns: []string{"topology.kubernetes.io/zone", "team"}}

	err := p.PrintNodes(nodes)
	if err != nil {
		t.Fatalf("PrintNodes() error = %v", err)
	}

	output := buf.String()
	for _, want := range []string{"ZONE", "TEAM", "us-east-1a", "platform"} {
		if !strings.Contains(output, want) {
			t.Errorf("expected %q in output, got:\n%s", want, output)
		}
	}
}

func TestPrintNodesLabelColumnsMissing(t *testing.T) {
	nodes := []analysis.NodeAnalysis{
		{
			Node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node-1", CreationTimestamp: metav1.Now(),
					Labels: map[string]string{},
				},
				Status: corev1.NodeStatus{
					Allocatable: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("4"),
						corev1.ResourceMemory: resource.MustParse("16Gi"),
					},
				},
			},
			Reason: analysis.NodeReasonRequested,
		},
	}

	var buf bytes.Buffer
	caps := &karpenter.ClusterCapabilities{HasNodePools: true}
	p := &Printer{out: &buf, capabilities: caps, labelColumns: []string{"team"}}

	err := p.PrintNodes(nodes)
	if err != nil {
		t.Fatalf("PrintNodes() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "TEAM") {
		t.Errorf("expected TEAM header in output, got:\n%s", output)
	}
	// Missing label should show <none>
	if !strings.Contains(output, "<none>") {
		t.Errorf("expected <none> for missing label, got:\n%s", output)
	}
}

func TestPrintNodesShowLabelsJSON(t *testing.T) {
	nodes := []analysis.NodeAnalysis{
		{
			Node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node-1", CreationTimestamp: metav1.Now(),
					Labels: map[string]string{
						"team": "platform",
						"zone": "us-east-1a",
					},
				},
				Status: corev1.NodeStatus{
					Allocatable: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("4"), corev1.ResourceMemory: resource.MustParse("16Gi"),
					},
				},
			},
			PoolName: "default", InstanceType: "m6i.xlarge",
			Reason: analysis.NodeReasonRequested,
		},
	}

	var buf bytes.Buffer
	caps := &karpenter.ClusterCapabilities{}
	p := &Printer{out: &buf, outputFormat: "json", capabilities: caps, showLabels: true}

	err := p.PrintNodes(nodes)
	if err != nil {
		t.Fatalf("PrintNodes() error = %v", err)
	}

	output := buf.String()
	for _, want := range []string{`"labels"`, `"team"`, `"platform"`} {
		if !strings.Contains(output, want) {
			t.Errorf("expected %q in JSON output, got:\n%s", want, output)
		}
	}
}

func TestPrintNodesLabelColumnsJSON(t *testing.T) {
	nodes := []analysis.NodeAnalysis{
		{
			Node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node-1", CreationTimestamp: metav1.Now(),
					Labels: map[string]string{
						"team": "platform",
						"zone": "us-east-1a",
					},
				},
				Status: corev1.NodeStatus{
					Allocatable: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("4"), corev1.ResourceMemory: resource.MustParse("16Gi"),
					},
				},
			},
			PoolName: "default", InstanceType: "m6i.xlarge",
			Reason: analysis.NodeReasonRequested,
		},
	}

	var buf bytes.Buffer
	caps := &karpenter.ClusterCapabilities{}
	p := &Printer{out: &buf, outputFormat: "json", capabilities: caps, labelColumns: []string{"team"}}

	err := p.PrintNodes(nodes)
	if err != nil {
		t.Fatalf("PrintNodes() error = %v", err)
	}

	output := buf.String()
	for _, want := range []string{`"labelColumns"`, `"team"`, `"platform"`} {
		if !strings.Contains(output, want) {
			t.Errorf("expected %q in JSON output, got:\n%s", want, output)
		}
	}
}

func TestPrintNodesNoLabelsInJSONByDefault(t *testing.T) {
	nodes := []analysis.NodeAnalysis{
		{
			Node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name: "node-1", CreationTimestamp: metav1.Now(),
					Labels: map[string]string{"team": "platform"},
				},
				Status: corev1.NodeStatus{
					Allocatable: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("4"), corev1.ResourceMemory: resource.MustParse("16Gi"),
					},
				},
			},
			Reason: analysis.NodeReasonRequested,
		},
	}

	var buf bytes.Buffer
	caps := &karpenter.ClusterCapabilities{}
	p := &Printer{out: &buf, outputFormat: "json", capabilities: caps}

	err := p.PrintNodes(nodes)
	if err != nil {
		t.Fatalf("PrintNodes() error = %v", err)
	}

	output := buf.String()
	if strings.Contains(output, `"labels"`) {
		t.Errorf("expected no labels in JSON output when not requested, got:\n%s", output)
	}
}

func TestPrintNodesNoHeaders(t *testing.T) {
	nodes := []analysis.NodeAnalysis{
		{
			Node: &corev1.Node{
				ObjectMeta: metav1.ObjectMeta{Name: "node-1", CreationTimestamp: metav1.Now()},
				Status: corev1.NodeStatus{Allocatable: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse("4"), corev1.ResourceMemory: resource.MustParse("16Gi"),
				}},
			},
			Reason: analysis.NodeReasonInherited,
		},
	}

	var buf bytes.Buffer
	caps := &karpenter.ClusterCapabilities{}
	p := &Printer{out: &buf, noHeaders: true, capabilities: caps}

	err := p.PrintNodes(nodes)
	if err != nil {
		t.Fatalf("PrintNodes() error = %v", err)
	}

	if strings.Contains(buf.String(), "NAME") {
		t.Error("expected no headers")
	}
}
