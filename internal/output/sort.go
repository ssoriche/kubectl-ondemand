package output

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"k8s.io/client-go/util/jsonpath"

	"github.com/ssoriche/kubectl-ondemand/internal/analysis"
)

// SortNodes sorts nodes by the given column name or JSONPath expression.
// An empty sortBy is a no-op (preserves existing order).
func SortNodes(nodes []analysis.NodeAnalysis, sortBy string) error {
	if sortBy == "" {
		return nil
	}

	sortBy = strings.Trim(sortBy, "'\"")

	if strings.HasPrefix(sortBy, "{") || strings.HasPrefix(sortBy, ".") {
		return sortByJSONPath(nodes, sortBy)
	}

	return sortByColumn(nodes, sortBy)
}

func sortByColumn(nodes []analysis.NodeAnalysis, column string) error {
	var less func(i, j int) bool

	switch column {
	case "name":
		less = func(i, j int) bool { return nodes[i].Node.Name < nodes[j].Node.Name }
	case "age":
		less = func(i, j int) bool {
			return nodes[i].Node.CreationTimestamp.Time.Before(nodes[j].Node.CreationTimestamp.Time)
		}
	case "instanceType":
		less = func(i, j int) bool { return nodes[i].InstanceType < nodes[j].InstanceType }
	case "poolName":
		less = func(i, j int) bool { return nodes[i].PoolName < nodes[j].PoolName }
	case "cpuUtilization":
		less = func(i, j int) bool { return nodes[i].CPUUtilization < nodes[j].CPUUtilization }
	case "memoryUtilization":
		less = func(i, j int) bool { return nodes[i].MemoryUtilization < nodes[j].MemoryUtilization }
	case "onDemandReason":
		less = func(i, j int) bool { return string(nodes[i].Reason) < string(nodes[j].Reason) }
	case "spotCapablePercent":
		less = func(i, j int) bool { return nodes[i].SpotCapablePercent < nodes[j].SpotCapablePercent }
	default:
		return fmt.Errorf("unknown sort column %q; valid columns: name, age, instanceType, poolName, cpuUtilization, memoryUtilization, onDemandReason, spotCapablePercent", column)
	}

	sort.SliceStable(nodes, less)
	return nil
}

func sortByJSONPath(nodes []analysis.NodeAnalysis, expr string) error {
	// Normalize: ensure wrapped in braces
	if strings.HasPrefix(expr, ".") {
		expr = "{" + expr + "}"
	}

	jp := jsonpath.New("sort")
	if err := jp.Parse(expr); err != nil {
		return fmt.Errorf("invalid JSONPath expression %q: %w", expr, err)
	}

	// Extract values for each node
	values := make([]string, len(nodes))
	for i, na := range nodes {
		var buf bytes.Buffer
		if err := jp.Execute(&buf, na.Node); err != nil {
			values[i] = ""
			continue
		}
		values[i] = buf.String()
	}

	sort.SliceStable(nodes, func(i, j int) bool {
		return values[i] < values[j]
	})
	return nil
}
