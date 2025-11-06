# NetBird CoreDNS Helm Chart

This Helm chart deploys NetBird CoreDNS on a Kubernetes cluster.

## Prerequisites

- Kubernetes 1.19+
- Helm 3.0+
- A storage class for persistent volume (or use default)
- **Network Requirements**:
  - The chart automatically configures the `NET_ADMIN` capability and mounts `/dev/net/tun` device (required for NetBird WireGuard interface)
  - These are Linux kernel requirements and apply to all Kubernetes distributions (k3s, RKE2, EKS, GKE, etc.)
  - For managed Kubernetes services, ensure your Pod Security Standards allow `NET_ADMIN` capability

## Installation

### Add the chart repository (if published)

```bash
# If the chart is published to a repository
helm repo add netbird-coredns https://your-repo-url
helm repo update
```

### Install from local chart

```bash
# Clone the repository
git clone https://github.com/christian-deleon/netbird-coredns.git
cd netbird-coredns/chart

# Install the chart
helm install netbird-coredns . --namespace netbird-coredns --create-namespace
```

### Install with custom values

```bash
# Create a custom values file
cat > my-values.yaml <<EOF
config:
  domains: "mydomain.com"
  forwardTo: "8.8.8.8"
  managementURL: "https://management.example.com:443"
  setupKey:
    value: "your-setup-key-here"
EOF

# Install with custom values
helm install netbird-coredns . -f my-values.yaml --namespace netbird-coredns --create-namespace
```

## Configuration

### Core Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of replicas | `1` |
| `image.repository` | Image repository | `ghcr.io/christian-deleon/netbird-coredns` |
| `image.tag` | Image tag | `v0.1.3` |
| `image.pullPolicy` | Image pull policy | `IfNotPresent` |

### DNS Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `config.domains` | Comma-separated domains for DNS resolution | `"mydomain.com"` |
| `config.forwardTo` | Forward server for unresolved queries | `"8.8.8.8"` |
| `config.dnsPort` | DNS server port | `5053` |
| `config.apiPort` | API server port | `8080` |
| `config.refreshInterval` | Refresh interval in seconds | `15` |
| `config.recordsFile` | Path to DNS records file | `"/etc/nb-dns/records/records.json"` |
| `config.logLevel` | Log level (debug, info, warn, error) | `"info"` |

### NetBird Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `config.managementURL` | NetBird Management server URL (optional, for self-hosted) | `""` |
| `config.setupKey.value` | NetBird setup key (creates secret automatically) | `""` |
| `config.setupKey.secret.name` | Name of existing secret containing setup key | `""` |
| `config.setupKey.secret.key` | Key in secret containing setup key | `""` |

### Storage Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `persistence.storageClass` | Storage class for PVC (empty = default) | `""` |
| `persistence.accessMode` | Access mode for PVC | `ReadWriteOnce` |
| `persistence.size` | Size of PVC | `1Gi` |

### Probe Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `probes.liveness.enabled` | Enable liveness probe | `true` |
| `probes.liveness.path` | Liveness probe path | `/health` |
| `probes.liveness.initialDelaySeconds` | Initial delay for liveness probe | `60` |
| `probes.liveness.periodSeconds` | Period for liveness probe | `30` |
| `probes.readiness.enabled` | Enable readiness probe | `true` |
| `probes.readiness.path` | Readiness probe path | `/health` |
| `probes.readiness.initialDelaySeconds` | Initial delay for readiness probe | `10` |
| `probes.readiness.periodSeconds` | Period for readiness probe | `10` |

## Examples

### Basic Installation

```bash
helm install netbird-coredns . \
  --set config.domains="example.com" \
  --namespace netbird-coredns \
  --create-namespace
```

### With NetBird Setup Key

Using a direct value (creates a Kubernetes secret automatically):

```bash
helm install netbird-coredns . \
  --set config.domains="example.com" \
  --set config.setupKey.value="your-setup-key-here" \
  --namespace netbird-coredns \
  --create-namespace
```

Using an existing secret:

```bash
# Create the secret first
kubectl create secret generic netbird-setup-key \
  --from-literal=setup-key="your-setup-key-here" \
  --namespace netbird-coredns

# Install the chart
helm install netbird-coredns . \
  --set config.domains="example.com" \
  --set config.setupKey.secret.name="netbird-setup-key" \
  --set config.setupKey.secret.key="setup-key" \
  --namespace netbird-coredns \
  --create-namespace
```

### Self-Hosted NetBird Management

```bash
helm install netbird-coredns . \
  --set config.domains="example.com" \
  --set config.managementURL="https://management.example.com:443" \
  --set config.setupKey.value="your-setup-key-here" \
  --namespace netbird-coredns \
  --create-namespace
```

### Custom Storage Class

```bash
helm install netbird-coredns . \
  --set config.domains="example.com" \
  --set persistence.storageClass="fast-ssd" \
  --set persistence.size="5Gi" \
  --namespace netbird-coredns \
  --create-namespace
```

### With Custom Values File

Create a `values.yaml` file:

```yaml
replicaCount: 1

config:
  domains: "example.com,staging.example.com"
  forwardTo: "1.1.1.1"
  dnsPort: 5053
  apiPort: 8080
  refreshInterval: 30
  logLevel: "debug"
  managementURL: "https://management.example.com:443"
  setupKey:
    value: "your-setup-key-here"

persistence:
  storageClass: "fast-ssd"
  size: 2Gi

resources:
  limits:
    cpu: 200m
    memory: 256Mi
  requests:
    cpu: 100m
    memory: 128Mi
```

Install with the custom values:

```bash
helm install netbird-coredns . -f values.yaml --namespace netbird-coredns --create-namespace
```

## Upgrading

```bash
# Upgrade with new values
helm upgrade netbird-coredns . \
  --set config.domains="newdomain.com" \
  --namespace netbird-coredns

# Upgrade with values file
helm upgrade netbird-coredns . -f values.yaml --namespace netbird-coredns
```

## Uninstallation

```bash
helm uninstall netbird-coredns --namespace netbird-coredns
```

**Note**: The PersistentVolumeClaim is not automatically deleted. To remove it:

```bash
kubectl delete pvc -l app.kubernetes.io/name=netbird-coredns --namespace netbird-coredns
```

## Development

### Linting the Chart

```bash
helm lint .
```

### Dry Run Installation

```bash
helm install netbird-coredns . --dry-run --debug --namespace netbird-coredns
```

### Template Rendering

```bash
helm template netbird-coredns . --namespace netbird-coredns
```

## Troubleshooting

### Check Pod Status

```bash
kubectl get pods -n netbird-coredns
kubectl logs -f -n netbird-coredns <pod-name>
```

### Check PVC Status

```bash
kubectl get pvc -n netbird-coredns
```

### Verify Configuration

```bash
kubectl describe deployment netbird-coredns -n netbird-coredns
```

### Access API

If you need to test the API from within the cluster:

```bash
kubectl run -it --rm debug --image=curlimages/curl --restart=Never -- \
  curl http://netbird-coredns:8080/health
```

## Chart Files

- `Chart.yaml` - Chart metadata
- `values.yaml` - Default configuration values
- `templates/` - Kubernetes manifest templates
  - `deployment.yaml` - Main deployment
  - `secret.yaml` - Auto-generated secret for setup key
  - `persistentvolumeclaim.yaml` - PVC for DNS records storage
  - `_helpers.tpl` - Template helpers
  - `NOTES.txt` - Post-installation notes

## Support

For issues specific to the Helm chart, please open an issue on [GitHub](https://github.com/christian-deleon/netbird-coredns/issues).

For general application documentation, see the [main README](../README.md).
