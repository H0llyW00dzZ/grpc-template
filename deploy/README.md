# Deployment Templates

This directory contains production-ready deployment templates for the gRPC server.

## Available Platforms

| Platform | Status | Path |
|----------|--------|------|
| [Kubernetes](kubernetes/) | ✅ Available | `deploy/kubernetes/` |

> [!NOTE]
> **Why only Kubernetes?** gRPC relies on HTTP/2 with long-lived multiplexed connections and bidirectional streaming. Most PaaS platforms (e.g., Railway, Heroku, Render) either lack full HTTP/2 support, impose request-level timeouts that kill streams, or don't expose raw TCP ports — making them unsuitable for production gRPC workloads. Kubernetes gives you full control over networking, load balancing (via headless Services or L4 LBs), and health checking, which gRPC requires to operate correctly.

Additional deployment targets (e.g., Docker Compose, Nomad) may be added in the future.

> [!TIP]
> All references to `grpc-template` in the manifests below (namespace, labels, image name) are **automatically rewritten** by `make init` to match your project name. The values shown here are the template defaults.

## Kubernetes

Production-ready manifests using [Kustomize](https://kustomize.io/):

```bash
# Apply all resources
kubectl apply -k deploy/kubernetes

# Or build and apply with custom image
cd deploy/kubernetes
kustomize edit set image grpc-template=your-registry/grpc-template:v1.0.0
kubectl apply -k .
```

### What's Included

| Manifest | Description |
|----------|-------------|
| `namespace.yaml` | Dedicated namespace (named after your project) |
| `deployment.yaml` | Multi-replica Deployment with resource limits and security context |
| `service.yaml` | ClusterIP Service on port 50051 (internal) |
| `service-lb.yaml` | LoadBalancer Service for external access (opt-in, see below) |
| `hpa.yaml` | HorizontalPodAutoscaler (CPU + memory metrics) |
| `networkpolicy.yaml` | Ingress/egress control with DNS-only egress |
| `pdb.yaml` | PodDisruptionBudget (`minAvailable: 2`) |
| `kustomization.yaml` | Kustomize entrypoint with common labels |

### Highlights

- **Native gRPC health probes** — liveness, readiness, and startup probes using Kubernetes-native `grpc` probe type (no sidecar binary needed)
- **Topology spread constraints** — pods are distributed evenly across nodes
- **Hardened security** — non-root user, read-only root filesystem, all capabilities dropped
- **Auto-scaling** — HPA scales from 3 to 10 replicas based on CPU/memory utilization

### External Access (LoadBalancer)

By default, the gRPC server is only accessible within the cluster (`ClusterIP`). To expose it externally, uncomment `service-lb.yaml` in `kustomization.yaml`:

```yaml
resources:
  # ...
  - service-lb.yaml  # ← uncomment this line
```

This provisions an **L4 (TCP) LoadBalancer** — required for gRPC since it uses HTTP/2. The manifest includes the following gRPC-optimized settings:

| Field | Value | Purpose |
|-------|-------|---------|
| `appProtocol` | `grpc` | Informs Kubernetes, Gateway API, and service meshes that this port carries gRPC traffic |
| `externalTrafficPolicy` | `Local` | Preserves client source IP and avoids extra network hops — recommended for long-lived gRPC connections |
| `sessionAffinity` | `None` | gRPC clients typically handle their own load balancing via client-side balancing |

#### Cloud-Provider Annotations

The manifest includes commented annotations for major cloud providers. Uncomment the relevant block in `service-lb.yaml`:

| Provider | Annotation | Effect |
|----------|------------|--------|
| GKE | `networking.gke.io/load-balancer-type: "Internal"` | Provisions an internal (VPC-only) load balancer |
| AWS/EKS | `service.beta.kubernetes.io/aws-load-balancer-type: "nlb"` | Uses a Network Load Balancer for TCP/gRPC |
| Azure/AKS | `service.beta.kubernetes.io/azure-load-balancer-internal: "true"` | Provisions an internal load balancer |

> [!WARNING]
> When using a LoadBalancer, ensure your `NetworkPolicy` allows ingress from external sources — the default `networkpolicy.yaml` restricts ingress to in-cluster pods only.

### Customization

Adjust the following for your environment:

- **Image**: `kustomize edit set image grpc-template=your-registry/image:tag`
- **TLS**: Add TLS Secrets and mount them as volumes in the Deployment
- **ConfigMaps**: Add environment-specific configuration via ConfigMaps or Secrets
