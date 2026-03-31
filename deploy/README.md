# Deployment Templates

This directory contains production-ready deployment templates for the gRPC server.

## Available Platforms

| Platform | Status | Path |
|----------|--------|------|
| [Kubernetes](kubernetes/) | ✅ Available | `deploy/kubernetes/` |

> [!NOTE]
> **Why only Kubernetes?** gRPC relies on HTTP/2 with long-lived multiplexed connections and bidirectional streaming. Most PaaS platforms (e.g., Railway, Heroku, Render) either lack full HTTP/2 support, impose request-level timeouts that kill streams, or don't expose raw TCP ports — making them unsuitable for production gRPC workloads. Kubernetes gives you full control over networking, load balancing (via headless Services or L4 LBs), and health checking, which gRPC requires to operate correctly.

Additional deployment targets (e.g., Docker Compose, Nomad) may be added in the future.

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
| `namespace.yaml` | Dedicated `grpc-template` namespace |
| `deployment.yaml` | Multi-replica Deployment with resource limits and security context |
| `service.yaml` | ClusterIP Service on port 50051 |
| `hpa.yaml` | HorizontalPodAutoscaler (CPU + memory metrics) |
| `networkpolicy.yaml` | Ingress/egress control with DNS-only egress |
| `pdb.yaml` | PodDisruptionBudget (`minAvailable: 2`) |
| `kustomization.yaml` | Kustomize entrypoint with common labels |

### Highlights

- **Native gRPC health probes** — liveness, readiness, and startup probes using Kubernetes-native `grpc` probe type (no sidecar binary needed)
- **Topology spread constraints** — pods are distributed evenly across nodes
- **Hardened security** — non-root user, read-only root filesystem, all capabilities dropped
- **Auto-scaling** — HPA scales from 3 to 10 replicas based on CPU/memory utilization

### Customization

Adjust the following for your environment:

- **Image**: `kustomize edit set image grpc-template=your-registry/image:tag`
- **TLS**: Add TLS Secrets and mount them as volumes in the Deployment
- **ConfigMaps**: Add environment-specific configuration via ConfigMaps or Secrets
- **Ingress**: Add a gRPC-compatible Ingress (e.g., NGINX with `grpc` backend protocol, or an L4 LoadBalancer Service)
