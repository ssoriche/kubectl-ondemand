package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/ssoriche/kubectl-ondemand/internal/analysis"
	"github.com/ssoriche/kubectl-ondemand/internal/karpenter"
	"github.com/ssoriche/kubectl-ondemand/internal/kube"
	"github.com/ssoriche/kubectl-ondemand/internal/output"
)

var version = "dev"

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	var opts options

	cmd := &cobra.Command{
		Use:   "kubectl-ondemand [flags] [NODE...]",
		Short: "Analyze why Karpenter nodes are on-demand",
		Long: `Analyzes workloads on Karpenter on-demand nodes to determine why each node
is on-demand and whether workloads are configured correctly for spot.

Classifies each node as:
  requested     - workload explicitly asks for on-demand
  spot-fallback - nodepool supports spot but Karpenter fell back to on-demand
  inherited     - workload constraints prevent spot, but didn't ask for on-demand

Shows per-pod classification with --pods flag or when specific nodes are given.`,
		Example: `  # Show all on-demand nodes with summary
  kubectl ondemand

  # Show workloads on a specific node
  kubectl ondemand ip-10-0-1-100.ec2.internal

  # Filter by label
  kubectl ondemand -l karpenter.sh/nodepool=default

  # Show pod details for all on-demand nodes
  kubectl ondemand --pods

  # Check with spot taint validation
  kubectl ondemand --spot-taint core.zr.org/dedicated=spot:NoSchedule

  # Output as JSON
  kubectl ondemand -o json`,
		Version:      version,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(cmd.Context(), args, opts)
		},
	}

	cmd.Flags().BoolVar(&opts.pods, "pods", false, "Show detailed pod-level classification")
	cmd.Flags().StringVarP(&opts.selector, "selector", "l", "", "Label selector for nodes")
	cmd.Flags().StringVarP(&opts.output, "output", "o", "", "Output format (json, yaml)")
	cmd.Flags().BoolVar(&opts.noHeaders, "no-headers", false, "Don't print headers")
	cmd.Flags().StringVar(&opts.spotTaint, "spot-taint", "", "Spot taint to check for (key=value:Effect)")

	return cmd
}

type options struct {
	pods      bool
	selector  string
	output    string
	noHeaders bool
	spotTaint string
}

func run(ctx context.Context, args []string, opts options) error {
	client, err := kube.NewClient()
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	discoveryClient, err := kube.NewDiscoveryClient()
	if err != nil {
		return fmt.Errorf("failed to create discovery client: %w", err)
	}

	dynClient, err := kube.NewDynamicClient()
	if err != nil {
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}

	capabilities, err := karpenter.DetectCapabilities(ctx, discoveryClient)
	if err != nil {
		capabilities = &karpenter.ClusterCapabilities{}
	}

	collector := analysis.NewCollector(client, dynClient, capabilities, opts.spotTaint)
	printer := output.NewPrinter(capabilities, opts.output, opts.noHeaders)

	nodes, err := collector.Collect(ctx, args, opts.selector)
	if err != nil {
		return fmt.Errorf("failed to collect node information: %w", err)
	}

	if opts.pods || len(args) > 0 {
		return printer.PrintPodDetails(nodes)
	}

	return printer.PrintNodes(nodes)
}
