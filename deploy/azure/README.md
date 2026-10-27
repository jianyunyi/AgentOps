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
2. Apply the stable API/Worker/Web Deployments (`api.yaml`, `worker.yaml`, `web.yaml`) and Prometheus integration.
3. Run the `aks-release` workflow with `canary=true`.
4. Verify canary readiness, error rate, latency, outbox lag, and worker stream health.
5. Run the workflow with `canary=false` to promote the immutable image tag.
6. If smoke tests or SLO checks fail, use the workflow rollback for ordinary releases; during the first HMAC Session cutover, repair forward rather than restoring the old API image.

The workflow intentionally does not create or alter an unknown Azure resource. Resource provisioning, private networking, DNS, TLS, Key Vault, and production secrets must be reviewed in the target subscription first.

## Web、TLS 与数据服务边界

`deploy/k8s/web.yaml` 发布两个 Web 副本；Web 默认使用空的 `NEXT_PUBLIC_API_BASE_URL`，因此浏览器请求保持同源 `/api/...`。Ingress 或 Application Gateway 应将 `/` 转发到 `agentscope-web`、`/api` 转发到 `agentscope-api`，并在入口终止 TLS；不要把 API、Web、MySQL 或 Redis Service 直接绑定公网 IP。仅在确认 TLS、认证、CORS/`WEB_ORIGIN` 与健康检查均已配置后才开放公网 DNS。`TRUSTED_PROXY_CIDRS` 默认留空，客户端提供的 `X-Forwarded-For` 会被忽略；只有确定了 ingress 的真实源地址网段后，才把这些 CIDR 写入 ConfigMap。

Redis 应使用私有端点或集群内部 Service，配置网络 ACL/NSG 只允许 API 和 Worker 访问，并保留持久化（托管 Redis 持久化或 `appendonly`/受保护卷）。API 和 Worker 支持 `REDIS_USERNAME`、`REDIS_PASSWORD`、`REDIS_TLS_ENABLED=true` 与可选 `REDIS_TLS_SERVER_NAME`；TLS 校验证书链与主机名，最低 TLS 1.2，不提供跳过证书校验的开关。AKS 将密码放在 `agentscope-secrets`，将地址与 TLS/SNI 配置放在 ConfigMap。不要以未认证 Redis 暴露公网替代私有网络与 ACL。

## 首次 HMAC Session 安全切换

旧版本把原始 Session cookie 值作为 `ses_...` 主键保存。首次发布本修复时先运行 Canary（稳定 Service 只选择 `version=stable`，Canary 不会混入生产 Service），确认后再次运行发布工作流选择 `canary=false` 和 `legacy_session_cutover=true`。工作流会记录稳定 API 副本数、先将旧 API 缩到 0 并等待旧 Pod 消失，再启用 `SESSION_LEGACY_SESSION_PURGE`、发布 HMAC 版本并恢复副本。启动时仅清除旧 `ses_` 会话，HMAC 会话保留；旧 cookie 用户需要重新登录，切换期间 API 会有短暂不可用。该切换失败时工作流不会回滚到会重新写入明文 token 的旧 API 镜像，应修复并向前发布 HMAC 兼容版本。首次切换完成后，后续发布使用默认 `legacy_session_cutover=false`，不需要再次停机。
