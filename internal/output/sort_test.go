package output

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/ssoriche/kubectl-ondemand/internal/analysis"
)

func makeNode(name string, createdAt time.Time, labels map[string]string) *corev1.Node {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			CreationTimestamp: metav1.NewTime(createdAt),
			Labels:            labels,
		},
	}
}

func TestSortNodesByName(t *testing.T) {
	now := time.Now()
	nodes := []analysis.NodeAnalysis{
		{Node: makeNode("charlie", now, nil)},
		{Node: makeNode("alpha", now, nil)},
		{Node: makeNode("bravo", now, nil)},
	}

	if err := SortNodes(nodes, "name"); err != nil {
		t.Fatalf("SortNodes() error = %v", err)
	}

	want := []string{"alpha", "bravo", "charlie"}
	for i, n := range nodes {
		if n.Node.Name != want[i] {
			t.Errorf("index %d: got %s, want %s", i, n.Node.Name, want[i])
		}
	}
}

func TestSortNodesByAge(t *testing.T) {
	t1 := time.Now().Add(-3 * time.Hour)
	t2 := time.Now().Add(-1 * time.Hour)
	t3 := time.Now().Add(-2 * time.Hour)
	nodes := []analysis.NodeAnalysis{
		{Node: makeNode("old", t1, nil)},
		{Node: makeNode("new", t2, nil)},
		{Node: makeNode("mid", t3, nil)},
	}

	if err := SortNodes(nodes, "age"); err != nil {
		t.Fatalf("SortNodes() error = %v", err)
	}

	// age sorts by creation timestamp ascending (oldest first)
	want := []string{"old", "mid", "new"}
	for i, n := range nodes {
		if n.Node.Name != want[i] {
			t.Errorf("index %d: got %s, want %s", i, n.Node.Name, want[i])
		}
	}
}

func TestSortNodesByCPUUtilization(t *testing.T) {
	now := time.Now()
	nodes := []analysis.NodeAnalysis{
		{Node: makeNode("med", now, nil), CPUUtilization: 50},
		{Node: makeNode("high", now, nil), CPUUtilization: 90},
		{Node: makeNode("low", now, nil), CPUUtilization: 10},
	}

	if err := SortNodes(nodes, "cpuUtilization"); err != nil {
		t.Fatalf("SortNodes() error = %v", err)
	}

	want := []string{"low", "med", "high"}
	for i, n := range nodes {
		if n.Node.Name != want[i] {
			t.Errorf("index %d: got %s, want %s", i, n.Node.Name, want[i])
		}
	}
}

func TestSortNodesByMemoryUtilization(t *testing.T) {
	now := time.Now()
	nodes := []analysis.NodeAnalysis{
		{Node: makeNode("high", now, nil), MemoryUtilization: 90},
		{Node: makeNode("low", now, nil), MemoryUtilization: 10},
	}

	if err := SortNodes(nodes, "memoryUtilization"); err != nil {
		t.Fatalf("SortNodes() error = %v", err)
	}

	want := []string{"low", "high"}
	for i, n := range nodes {
		if n.Node.Name != want[i] {
			t.Errorf("index %d: got %s, want %s", i, n.Node.Name, want[i])
		}
	}
}

func TestSortNodesBySpotCapablePercent(t *testing.T) {
	now := time.Now()
	nodes := []analysis.NodeAnalysis{
		{Node: makeNode("high", now, nil), SpotCapablePercent: 90},
		{Node: makeNode("low", now, nil), SpotCapablePercent: 10},
	}

	if err := SortNodes(nodes, "spotCapablePercent"); err != nil {
		t.Fatalf("SortNodes() error = %v", err)
	}

	want := []string{"low", "high"}
	for i, n := range nodes {
		if n.Node.Name != want[i] {
			t.Errorf("index %d: got %s, want %s", i, n.Node.Name, want[i])
		}
	}
}

func TestSortNodesByPoolName(t *testing.T) {
	now := time.Now()
	nodes := []analysis.NodeAnalysis{
		{Node: makeNode("n1", now, nil), PoolName: "zeta"},
		{Node: makeNode("n2", now, nil), PoolName: "alpha"},
	}

	if err := SortNodes(nodes, "poolName"); err != nil {
		t.Fatalf("SortNodes() error = %v", err)
	}

	want := []string{"n2", "n1"}
	for i, n := range nodes {
		if n.Node.Name != want[i] {
			t.Errorf("index %d: got %s, want %s", i, n.Node.Name, want[i])
		}
	}
}

func TestSortNodesByInstanceType(t *testing.T) {
	now := time.Now()
	nodes := []analysis.NodeAnalysis{
		{Node: makeNode("n1", now, nil), InstanceType: "m6i.xlarge"},
		{Node: makeNode("n2", now, nil), InstanceType: "c5.large"},
	}

	if err := SortNodes(nodes, "instanceType"); err != nil {
		t.Fatalf("SortNodes() error = %v", err)
	}

	want := []string{"n2", "n1"}
	for i, n := range nodes {
		if n.Node.Name != want[i] {
			t.Errorf("index %d: got %s, want %s", i, n.Node.Name, want[i])
		}
	}
}

func TestSortNodesByOnDemandReason(t *testing.T) {
	now := time.Now()
	nodes := []analysis.NodeAnalysis{
		{Node: makeNode("n1", now, nil), Reason: analysis.NodeReasonSpotFallback},
		{Node: makeNode("n2", now, nil), Reason: analysis.NodeReasonInherited},
		{Node: makeNode("n3", now, nil), Reason: analysis.NodeReasonRequested},
	}

	if err := SortNodes(nodes, "onDemandReason"); err != nil {
		t.Fatalf("SortNodes() error = %v", err)
	}

	// alphabetical: inherited < requested < spot-fallback
	want := []string{"n2", "n3", "n1"}
	for i, n := range nodes {
		if n.Node.Name != want[i] {
			t.Errorf("index %d: got %s, want %s", i, n.Node.Name, want[i])
		}
	}
}

func TestSortNodesUnknownColumn(t *testing.T) {
	nodes := []analysis.NodeAnalysis{
		{Node: makeNode("n1", time.Now(), nil)},
	}

	err := SortNodes(nodes, "bogus")
	if err == nil {
		t.Fatal("expected error for unknown column")
	}
}

func TestSortNodesEmpty(t *testing.T) {
	nodes := []analysis.NodeAnalysis{
		{Node: makeNode("n1", time.Now(), nil)},
	}

	// Empty sortBy is a no-op
	if err := SortNodes(nodes, ""); err != nil {
		t.Fatalf("SortNodes() error = %v", err)
	}
}

func TestSortNodesByJSONPath(t *testing.T) {
	t1 := time.Now().Add(-3 * time.Hour)
	t2 := time.Now().Add(-1 * time.Hour)
	nodes := []analysis.NodeAnalysis{
		{Node: makeNode("new", t2, nil)},
		{Node: makeNode("old", t1, nil)},
	}

	if err := SortNodes(nodes, "{.metadata.creationTimestamp}"); err != nil {
		t.Fatalf("SortNodes() error = %v", err)
	}

	want := []string{"old", "new"}
	for i, n := range nodes {
		if n.Node.Name != want[i] {
			t.Errorf("index %d: got %s, want %s", i, n.Node.Name, want[i])
		}
	}
}

func TestSortNodesByJSONPathLabel(t *testing.T) {
	now := time.Now()
	nodes := []analysis.NodeAnalysis{
		{Node: makeNode("n1", now, map[string]string{"zone": "us-east-1c"})},
		{Node: makeNode("n2", now, map[string]string{"zone": "us-east-1a"})},
	}

	if err := SortNodes(nodes, "{.metadata.labels.zone}"); err != nil {
		t.Fatalf("SortNodes() error = %v", err)
	}

	want := []string{"n2", "n1"}
	for i, n := range nodes {
		if n.Node.Name != want[i] {
			t.Errorf("index %d: got %s, want %s", i, n.Node.Name, want[i])
		}
	}
}

func TestSortNodesByJSONPathWithDotPrefix(t *testing.T) {
	now := time.Now()
	nodes := []analysis.NodeAnalysis{
		{Node: makeNode("n1", now, map[string]string{"zone": "us-east-1c"})},
		{Node: makeNode("n2", now, map[string]string{"zone": "us-east-1a"})},
	}

	// .metadata.labels.zone (dot prefix, no braces)
	if err := SortNodes(nodes, ".metadata.labels.zone"); err != nil {
		t.Fatalf("SortNodes() error = %v", err)
	}

	want := []string{"n2", "n1"}
	for i, n := range nodes {
		if n.Node.Name != want[i] {
			t.Errorf("index %d: got %s, want %s", i, n.Node.Name, want[i])
		}
	}
}
