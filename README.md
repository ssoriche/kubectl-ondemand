# kubectl-ondemand

A kubectl plugin that analyzes why Karpenter nodes are on-demand and whether workloads are configured correctly for spot.

## Features

- Classifies on-demand nodes as **requested**, **spot-fallback**, or **inherited**
- Shows per-pod analysis of why workloads are on on-demand nodes
- Detects misconfigured workloads that could run on spot
- Calculates spot-capable percentage per node
- Configurable spot taint validation
- Automatically detects Karpenter API version (v1alpha5, v1beta1, v1)
- Supports table, JSON, and YAML output formats

## Usage

### List all on-demand nodes

```bash
kubectl ondemand
```

```text
NAME                          INSTANCE-TYPE   NODEPOOL   AGE   CPU-UTIL   MEM-UTIL   ON-DEMAND-REASON   SPOT-CAPABLE%
ip-10-0-1-100.ec2.internal    r6g.4xlarge     default    5d    45%        62%        requested          0%
ip-10-0-1-101.ec2.internal    m6i.8xlarge     default    3d    82%        71%        spot-fallback      75%
ip-10-0-1-102.ec2.internal    c6g.2xlarge     batch      1d    55%        48%        inherited          100%
```

### Inspect workloads on a specific node

Passing a node name (or `--pods`) shows per-pod classification with the reasons each pod landed on on-demand:

```bash
kubectl ondemand ip-10-0-1-101.ec2.internal
```

```text
NODE: ip-10-0-1-101.ec2.internal (m6i.8xlarge, nodepool: default, age: 3d)
REASON: spot-fallback
CPU: 82%    MEM: 71%

NAMESPACE     POD                        CPU     MEM      CATEGORY     REASONS
spark         driver-abc-123             4000m   8Gi      requested    explicit-ondemand-selector
batch         job-xyz-456                2000m   4Gi      inherited    missing-spot-toleration
ml            training-pod-789           8000m   32Gi     inherited    zone-pinned
web           api-server-abc             500m    1Gi      spot-ok      —
```

### Filter by label

```bash
kubectl ondemand -l karpenter.sh/nodepool=default
```

### Show pod details for all on-demand nodes

```bash
kubectl ondemand --pods
```

### Validate spot taint configuration

When your cluster uses a taint to mark spot nodes, pass `--spot-taint` to detect pods that are missing the corresponding toleration:

```bash
kubectl ondemand --spot-taint core.zr.org/dedicated=spot:NoSchedule
```

The format is `key=value:Effect` where Effect is `NoSchedule`, `NoExecute`, or `PreferNoSchedule`. When this flag is omitted, the missing-spot-toleration check is skipped.

### Output formats

```bash
# JSON output (useful for scripting)
kubectl ondemand -o json

# YAML output
kubectl ondemand -o yaml

# Suppress headers (useful for piping)
kubectl ondemand --no-headers
```

## How Classification Works

The plugin answers the question: **why is this node on-demand?** It does this at two levels — per-pod and per-node.

### Pod Categories

Each pod on an on-demand node is classified into one of three categories based on its scheduling constraints:

| Category | Meaning | Action |
|----------|---------|--------|
| `requested` | Pod **explicitly asks** for on-demand via nodeSelector or node affinity | Intentional — verify this is actually needed |
| `inherited` | Pod has constraints that **prevent spot placement**, but never explicitly asked for on-demand | Candidate for reconfiguration |
| `spot-ok` | Pod has **no constraints** preventing spot — it's on this node because the node existed | Safe to move to spot |

**The key distinction between `requested` and `inherited`:** A `requested` pod said "I want on-demand." An `inherited` pod never said that — its other constraints (tolerations, zone pinning, affinity rules) just happen to prevent it from running on spot. Inherited pods are the best candidates for reconfiguration since they may not actually need to avoid spot.

### Pod Detection Rules

Rules are checked in order. A pod can trigger multiple reasons. The highest-priority category wins (requested > inherited > spot-ok).

#### Reasons that produce `requested` (pod explicitly wants on-demand)

| Reason | What it detects | Example |
|--------|----------------|---------|
| `explicit-ondemand-selector` | `nodeSelector` contains `karpenter.sh/capacity-type: on-demand` | `nodeSelector: {"karpenter.sh/capacity-type": "on-demand"}` |
| `ondemand-affinity` | Required node affinity matches `karpenter.sh/capacity-type` with value `on-demand` | `requiredDuringSchedulingIgnoredDuringExecution` with `matchExpressions` on `karpenter.sh/capacity-type In [on-demand]` |

#### Reasons that produce `inherited` (constraints prevent spot)

| Reason | What it detects | Example | Why it matters |
|--------|----------------|---------|----------------|
| `missing-spot-toleration` | Pod lacks a toleration for the configured spot taint (requires `--spot-taint` flag) | Pod doesn't tolerate `core.zr.org/dedicated=spot:NoSchedule` | Pod can't be scheduled on spot nodes that have this taint |
| `zone-pinned` | `nodeSelector` or required node affinity pins to a specific `topology.kubernetes.io/zone` | `nodeSelector: {"topology.kubernetes.io/zone": "us-east-1a"}` | Limits scheduling to one AZ, reducing spot capacity options |
| `restrictive-anti-affinity` | Pod has `requiredDuringSchedulingIgnoredDuringExecution` pod anti-affinity rules | Must not run on same host as another pod | Forces spreading across nodes, may prevent bin-packing onto spot |
| `restrictive-affinity` | Pod has `requiredDuringSchedulingIgnoredDuringExecution` pod affinity rules | Must run on same host as another specific pod | Forces co-location with pods that may themselves be on on-demand |
| `local-storage` | Pod uses `hostPath` volumes or `emptyDir` with `medium: Memory` | `volumes: [{hostPath: {path: "/data"}}]` | Data tied to the node — spot interruption would lose it |

**Note:** Only *required* affinity/anti-affinity rules trigger classification. Preferred (soft) rules do not — they're hints, not constraints.

### Node Classification

The node-level reason rolls up from its pods:

| Node Reason | Logic | What it means |
|-------------|-------|---------------|
| `requested` | At least one pod explicitly requests on-demand | Someone intentionally asked for on-demand |
| `spot-fallback` | Nodepool config allows spot, no pods request on-demand, and no pods have inherited constraints | Karpenter tried spot but fell back — likely a capacity issue |
| `inherited` | Pods have constraints preventing spot, but none explicitly requested on-demand | Workload config is driving on-demand usage unintentionally |

The **SPOT-CAPABLE%** column shows what percentage of non-DaemonSet workload pods on the node are classified as `spot-ok`. A node showing `inherited` with high spot-capable% is a strong signal that a few misconfigured pods are keeping the whole node on-demand.

### Nodepool Spot Detection

To determine `spot-fallback`, the plugin fetches NodePool (v1beta1/v1) or Provisioner (v1alpha5) CRDs and checks whether the `karpenter.sh/capacity-type` requirement includes `spot`. If the nodepool allows spot but the node ended up on-demand, and no pod explicitly requested on-demand, it's classified as a spot fallback.

## Flags Reference

| Flag | Short | Description |
|------|-------|-------------|
| `--pods` | | Show per-pod classification details for all nodes |
| `--selector` | `-l` | Label selector to filter nodes (e.g., `karpenter.sh/nodepool=default`) |
| `--output` | `-o` | Output format: `json`, `yaml`, or table (default) |
| `--no-headers` | | Suppress table headers |
| `--spot-taint` | | Spot taint to validate (format: `key=value:Effect`) |
| `--version` | `-v` | Show version |

## Karpenter Version Support

The plugin automatically detects which Karpenter version is installed:

| Version | Node Labels | Column Header |
|---------|-------------|---------------|
| v1alpha5 | `karpenter.sh/provisioner-name` | PROVISIONER |
| v1beta1/v1 | `karpenter.sh/nodepool` | NODEPOOL |

Mixed-version clusters are supported during migrations.

## Development

```bash
# Build
make build

# Run tests
make test

# Run linter
make lint

# Run all checks (fmt, vet, lint, test)
make check

# Cross-platform build
make build-all

# Install to GOPATH/bin
make install
```

## License

MIT
