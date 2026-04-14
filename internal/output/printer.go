package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"gopkg.in/yaml.v3"

	"github.com/ssoriche/kubectl-ondemand/internal/analysis"
	"github.com/ssoriche/kubectl-ondemand/internal/karpenter"
)

type Printer struct {
	out          io.Writer
	noHeaders    bool
	outputFormat string
	capabilities *karpenter.ClusterCapabilities
	showLabels   bool
	labelColumns []string
}

func NewPrinter(capabilities *karpenter.ClusterCapabilities, outputFormat string, noHeaders bool, showLabels bool, labelColumns []string) *Printer {
	return &Printer{
		out:          os.Stdout,
		noHeaders:    noHeaders,
		outputFormat: outputFormat,
		capabilities: capabilities,
		showLabels:   showLabels,
		labelColumns: labelColumns,
	}
}

func (p *Printer) PrintNodes(nodes []analysis.NodeAnalysis) error {
	switch p.outputFormat {
	case "json":
		return p.printNodesJSON(nodes)
	case "yaml":
		return p.printNodesYAML(nodes)
	default:
		return p.printNodesTable(nodes)
	}
}

func (p *Printer) printNodesTable(nodes []analysis.NodeAnalysis) error {
	w := tabwriter.NewWriter(p.out, 0, 0, 2, ' ', 0)

	poolHeader := p.capabilities.DeterminePoolColumnHeader()

	if !p.noHeaders {
		header := fmt.Sprintf("NAME\tINSTANCE-TYPE\t%s\tAGE\tCPU-UTIL\tMEM-UTIL\tON-DEMAND-REASON\tSPOT-CAPABLE%%", poolHeader)
		for _, lc := range p.labelColumns {
			header += "\t" + labelColumnHeader(lc)
		}
		if p.showLabels {
			header += "\tLABELS"
		}
		if _, err := fmt.Fprintln(w, header); err != nil {
			return err
		}
	}

	for _, info := range nodes {
		node := info.Node
		age := analysis.FormatAge(node.CreationTimestamp.Time)

		poolName := info.PoolName
		if poolName == "" {
			poolName = "<none>"
		}
		instanceType := info.InstanceType
		if instanceType == "" {
			instanceType = "<none>"
		}

		cpuUtil := analysis.FormatUtilization(info.CPUUtilization)
		memUtil := analysis.FormatUtilization(info.MemoryUtilization)
		spotCapable := analysis.FormatUtilization(info.SpotCapablePercent)

		line := fmt.Sprintf("%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s",
			node.Name, instanceType, poolName, age,
			cpuUtil, memUtil, string(info.Reason), spotCapable)

		for _, lc := range p.labelColumns {
			val := node.Labels[lc]
			if val == "" {
				val = "<none>"
			}
			line += "\t" + val
		}
		if p.showLabels {
			line += "\t" + formatLabels(node.Labels)
		}

		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}

	return w.Flush()
}

// labelColumnHeader returns a short uppercase header for a label key.
// It uses the last segment after "/" if present.
func labelColumnHeader(key string) string {
	if idx := strings.LastIndex(key, "/"); idx >= 0 {
		key = key[idx+1:]
	}
	return strings.ToUpper(key)
}

// formatLabels returns a sorted comma-separated key=value string.
func formatLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return ""
	}
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = k + "=" + labels[k]
	}
	return strings.Join(parts, ",")
}

type nodeOutput struct {
	Name               string            `json:"name" yaml:"name"`
	InstanceType       string            `json:"instanceType" yaml:"instanceType"`
	PoolName           string            `json:"poolName" yaml:"poolName"`
	Age                string            `json:"age" yaml:"age"`
	CPUUtilization     string            `json:"cpuUtilization" yaml:"cpuUtilization"`
	MemoryUtilization  string            `json:"memoryUtilization" yaml:"memoryUtilization"`
	OnDemandReason     string            `json:"onDemandReason" yaml:"onDemandReason"`
	SpotCapablePercent string            `json:"spotCapablePercent" yaml:"spotCapablePercent"`
	Labels             map[string]string `json:"labels,omitempty" yaml:"labels,omitempty"`
	LabelColumns       map[string]string `json:"labelColumns,omitempty" yaml:"labelColumns,omitempty"`
}

func (p *Printer) nodesToOutput(nodes []analysis.NodeAnalysis) []nodeOutput {
	out := make([]nodeOutput, len(nodes))
	for i, info := range nodes {
		out[i] = nodeOutput{
			Name:               info.Node.Name,
			InstanceType:       info.InstanceType,
			PoolName:           info.PoolName,
			Age:                analysis.FormatAge(info.Node.CreationTimestamp.Time),
			CPUUtilization:     analysis.FormatUtilization(info.CPUUtilization),
			MemoryUtilization:  analysis.FormatUtilization(info.MemoryUtilization),
			OnDemandReason:     string(info.Reason),
			SpotCapablePercent: analysis.FormatUtilization(info.SpotCapablePercent),
		}
		if p.showLabels {
			out[i].Labels = info.Node.Labels
		}
		if len(p.labelColumns) > 0 {
			lc := make(map[string]string, len(p.labelColumns))
			for _, key := range p.labelColumns {
				val := info.Node.Labels[key]
				if val == "" {
					val = "<none>"
				}
				lc[key] = val
			}
			out[i].LabelColumns = lc
		}
	}
	return out
}

func (p *Printer) printNodesJSON(nodes []analysis.NodeAnalysis) error {
	out := p.nodesToOutput(nodes)
	encoder := json.NewEncoder(p.out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(out)
}

func (p *Printer) printNodesYAML(nodes []analysis.NodeAnalysis) error {
	out := p.nodesToOutput(nodes)
	encoder := yaml.NewEncoder(p.out)
	encoder.SetIndent(2)
	return encoder.Encode(out)
}

func (p *Printer) PrintPodDetails(nodeAnalysis []analysis.NodeAnalysis) error {
	switch p.outputFormat {
	case "json":
		return p.printPodDetailsJSON(nodeAnalysis)
	case "yaml":
		return p.printPodDetailsYAML(nodeAnalysis)
	default:
		return p.printPodDetailsTable(nodeAnalysis)
	}
}

func (p *Printer) printPodDetailsTable(nodeAnalyses []analysis.NodeAnalysis) error {
	w := tabwriter.NewWriter(p.out, 0, 0, 2, ' ', 0)

	for idx, na := range nodeAnalyses {
		if idx > 0 {
			if _, err := fmt.Fprintln(w); err != nil {
				return err
			}
		}

		node := na.Node
		instanceType := na.InstanceType
		if instanceType == "" {
			instanceType = "<unknown>"
		}
		poolName := na.PoolName
		if poolName == "" {
			poolName = "<none>"
		}

		if _, err := fmt.Fprintf(w, "NODE: %s (%s, nodepool: %s, age: %s)\n",
			node.Name, instanceType, poolName, analysis.FormatAge(node.CreationTimestamp.Time)); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "REASON: %s\n", na.Reason); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "CPU: %s\tMEM: %s\n", analysis.FormatUtilization(na.CPUUtilization), analysis.FormatUtilization(na.MemoryUtilization)); err != nil {
			return err
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}

		if !p.noHeaders {
			if _, err := fmt.Fprintln(w, "NAMESPACE\tPOD\tCPU\tMEM\tCATEGORY\tREASONS"); err != nil {
				return err
			}
		}

		details := analysis.GetPodDetails(&na)
		for _, d := range details {
			reasons := "—"
			if len(d.Reasons) > 0 {
				reasons = formatReasons(d.Reasons)
			}
			if _, err := fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
				d.Namespace, d.Name, d.CPU, d.Memory, string(d.Category), reasons); err != nil {
				return err
			}
		}
	}

	return w.Flush()
}

func formatReasons(reasons []analysis.Reason) string {
	if len(reasons) == 0 {
		return "—"
	}
	result := string(reasons[0])
	for i := 1; i < len(reasons); i++ {
		result += ", " + string(reasons[i])
	}
	return result
}

type podDetailOutput struct {
	Namespace string   `json:"namespace" yaml:"namespace"`
	Name      string   `json:"name" yaml:"name"`
	CPU       string   `json:"cpu" yaml:"cpu"`
	Memory    string   `json:"memory" yaml:"memory"`
	Category  string   `json:"category" yaml:"category"`
	Reasons   []string `json:"reasons" yaml:"reasons"`
	IsDaemon  bool     `json:"isDaemonSet" yaml:"isDaemonSet"`
}

type nodeDetailOutput struct {
	Name               string            `json:"name" yaml:"name"`
	InstanceType       string            `json:"instanceType" yaml:"instanceType"`
	PoolName           string            `json:"poolName" yaml:"poolName"`
	OnDemandReason     string            `json:"onDemandReason" yaml:"onDemandReason"`
	CPUUtilization     string            `json:"cpuUtilization" yaml:"cpuUtilization"`
	MemoryUtilization  string            `json:"memoryUtilization" yaml:"memoryUtilization"`
	SpotCapablePercent string            `json:"spotCapablePercent" yaml:"spotCapablePercent"`
	Pods               []podDetailOutput `json:"pods" yaml:"pods"`
}

func nodeAnalysesToDetailOutput(analyses []analysis.NodeAnalysis) []nodeDetailOutput {
	out := make([]nodeDetailOutput, len(analyses))
	for i, na := range analyses {
		details := analysis.GetPodDetails(&na)
		pods := make([]podDetailOutput, len(details))
		for j, d := range details {
			reasons := make([]string, len(d.Reasons))
			for k, r := range d.Reasons {
				reasons[k] = string(r)
			}
			pods[j] = podDetailOutput{
				Namespace: d.Namespace,
				Name:      d.Name,
				CPU:       d.CPU,
				Memory:    d.Memory,
				Category:  string(d.Category),
				Reasons:   reasons,
				IsDaemon:  d.IsDaemon,
			}
		}
		out[i] = nodeDetailOutput{
			Name:               na.Node.Name,
			InstanceType:       na.InstanceType,
			PoolName:           na.PoolName,
			OnDemandReason:     string(na.Reason),
			CPUUtilization:     analysis.FormatUtilization(na.CPUUtilization),
			MemoryUtilization:  analysis.FormatUtilization(na.MemoryUtilization),
			SpotCapablePercent: analysis.FormatUtilization(na.SpotCapablePercent),
			Pods:               pods,
		}
	}
	return out
}

func (p *Printer) printPodDetailsJSON(analyses []analysis.NodeAnalysis) error {
	out := nodeAnalysesToDetailOutput(analyses)
	encoder := json.NewEncoder(p.out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(out)
}

func (p *Printer) printPodDetailsYAML(analyses []analysis.NodeAnalysis) error {
	out := nodeAnalysesToDetailOutput(analyses)
	encoder := yaml.NewEncoder(p.out)
	encoder.SetIndent(2)
	return encoder.Encode(out)
}
