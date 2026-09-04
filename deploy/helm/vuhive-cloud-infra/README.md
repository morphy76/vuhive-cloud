# vuhive-cloud-infra

Helm chart to deploy backing infrastructure services (PostgreSQL and MinIO) for `vuhive-cloud`.

## Quickstart

```bash
# Add dependency chart repositories
helm repo add groundhog2k https://groundhog2k.github.io/helm-charts/
helm repo add minio https://charts.min.io/
helm repo update

# Build chart dependencies
helm dependency build deploy/helm/vuhive-cloud-infra

# Install infrastructure chart
helm install vuhive-infra deploy/helm/vuhive-cloud-infra --namespace vuhive-system --create-namespace
```
