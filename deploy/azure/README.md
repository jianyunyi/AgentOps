# AKS production deployment

## Required Azure resources

- AKS cluster with a system node pool and Azure CNI or kubenet networking.
- Azure Container Registry (ACR) attached to the AKS cluster with `AcrPull`.
- Managed MySQL and Redis endpoints reachable from the cluster, preferably private endpoints.
- TLS certificate and an ingress controller or Application Gateway.
- Azure Monitor/Managed Prometheus with the rules in `deploy/prometheus/`.

## GitHub configuration

Configure a `production` environment and use federated Azure OIDC credentials. Add these repository/environment secrets:

- `AZURE_CLIENT_ID`
- `AZURE_TENANT_ID`
- `AZURE_SUBSCRIPTION_ID`

Add these variables:

- `ACR_LOGIN_SERVER`, for example `agentscope.azurecr.io`
- `AZURE_RESOURCE_GROUP`
- `AKS_CLUSTER_NAME`

Do not put database passwords, OIDC client secrets, or `SESSION_SECRET` in the repository. Create the `agentscope-secrets` Kubernetes Secret through an approved secret-management workflow such as Azure Key Vault CSI Driver.

## Release sequence

1. Apply `namespace.yaml`, `serviceaccount.yaml`, `configmap.yaml`, and the secret provider configuration.
2. Apply the stable API/Worker Deployments and Prometheus integration.
3. Run the `aks-release` workflow with `canary=true`.
4. Verify canary readiness, error rate, latency, outbox lag, and worker stream health.
5. Run the workflow with `canary=false` to promote the immutable image tag.
6. If smoke tests or SLO checks fail, use the workflow rollback or `kubectl rollout undo`.

The workflow intentionally does not create or alter an unknown Azure resource. Resource provisioning, private networking, DNS, TLS, Key Vault, and production secrets must be reviewed in the target subscription first.
