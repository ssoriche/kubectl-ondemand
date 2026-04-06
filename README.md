# kubectl-ondemand

A kubectl plugin that analyzes why Karpenter nodes are on-demand and whether workloads are configured correctly for spot.

## Features

- Classifies on-demand nodes as **requested**, **spot-fallback**, or **inherited**
- Shows per-pod analysis of why workloads are on on-demand nodes
- Detects misconfigured workloads that could run on spot
- Calculates spot-capable percentage per node
- Automatically detects Karpenter API version (v1alpha5, v1beta1, v1)
- Supports table, JSON, and YAML output formats

## Usage

```bash
# Show all on-demand nodes with summary
kubectl ondemand

# Show workloads on a specific node
kubectl ondemand ip-10-0-1-100.ec2.internal

# Filter by label
kubectl ondemand -l karpenter.sh/nodepool=default

# Show pod details for all nodes
kubectl ondemand --pods

# Validate spot taint configuration
kubectl ondemand --spot-taint core.zr.org/dedicated=spot:NoSchedule

# Output as JSON
kubectl ondemand -o json
```

## Output Example

```
NAME                          INSTANCE-TYPE   NODEPOOL   AGE   CPU-UTIL   MEM-UTIL   ON-DEMAND-REASON   SPOT-CAPABLE%
ip-10-0-1-100.ec2.internal    r6g.4xlarge     default    5d    45%        62%        requested          0%
ip-10-0-1-101.ec2.internal    m6i.8xlarge     default    3d    82%        71%        spot-fallback      75%
ip-10-0-1-102.ec2.internal    c6g.2xlarge     batch      1d    55%        48%        inherited          100%
```

## Node Classification

| Reason | Meaning |
|--------|---------|
| `requested` | A workload explicitly asks for on-demand (nodeSelector, affinity) |
| `spot-fallback` | Nodepool supports spot but Karpenter fell back to on-demand |
| `inherited` | Workload constraints prevent spot, but didn't explicitly request on-demand |

## Pod Categories

| Category | Meaning |
|----------|---------|
| `requested` | Pod explicitly requires on-demand |
| `inherited` | Pod has constraints preventing spot |
| `spot-ok` | Pod could run on spot |

## Detection Rules

| Reason | Description |
|--------|-------------|
| `explicit-ondemand-selector` | nodeSelector has `karpenter.sh/capacity-type: on-demand` |
| `ondemand-affinity` | Node affinity requires on-demand capacity type |
| `missing-spot-toleration` | Pod lacks toleration for configured spot taint |
| `zone-pinned` | nodeSelector or affinity pins to specific AZ |
| `restrictive-anti-affinity` | Required pod anti-affinity limits scheduling |
| `restrictive-affinity` | Required pod affinity forces co-location |
| `local-storage` | Uses hostPath or emptyDir with memory medium |

## Development

```bash
# Build
make build

# Run tests
make test

# Run linter
make lint

# Run all checks
make check
```

## License

MIT
