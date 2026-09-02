# Agent 凭证迁移与 HMAC 强制上线

本流程用于把历史 Agent 凭证迁移到 API Key + HMAC Signing Secret。系统不会批量导出明文凭证；每个 Agent 必须通过控制台或 `POST /api/v1/agents/:id/rotate-key` 单独轮换，响应中的新凭证只展示一次。

## 发布顺序

1. 先部署签名加密密钥，并保持兼容模式：`AGENT_SIGNING_ENCRYPTION_KEY` 使用 32 字节随机密钥，`AGENT_SIGNATURE_REQUIRED=false`。
2. 登录租户控制台的 Agent 管理页，查看“Credential migration”状态。也可以调用 `GET /api/v1/agents/migration-status?page=1&page_size=100`，接口只返回当前租户仍使用历史凭证的 Agent 元数据。
3. 对列表中的每个 Agent 执行 `Rotate key`，把 API Key 和 Signing Secret 安全交给对应调用方。不要把响应写入日志、工单、前端持久化存储或提交到 Git。
4. 先升级调用方到支持 HMAC 的 SDK/集成，并在灰度环境验证签名请求、时间窗和 nonce 防重放。
5. 重复检查迁移状态，直到 `legacy_agents=0`。新创建的 Agent 默认已经是 HMAC-ready。
6. 将所有 API/Worker 实例配置为 `AGENT_SIGNATURE_REQUIRED=true` 并重新部署。API 在监听端口前会做全局预检：仍有活跃历史凭证或数据库检查失败时会拒绝启动。
7. 通过健康检查、Agent SDK 冒烟请求和错误率/延迟指标确认发布成功。

## 回退与故障处理

- 如果强制模式启动预检失败，保持或切回 `AGENT_SIGNATURE_REQUIRED=false`，完成剩余 Agent 轮换后再次发布。
- 如果灰度调用方失败，先回退调用方或暂时切回兼容模式，再排查签名版本、方法、路径、原始 Body、时间戳和 nonce；不要恢复已经撤销的旧凭证。
- `AGENT_SIGNING_ENCRYPTION_KEY` 一旦用于生产凭证加密，必须纳入密钥管理系统并备份。丢失该密钥会使已存储的 Signing Secret 无法解密，届时需要逐 Agent 重新轮换。

## 预检示例

```text
AGENT_SIGNING_ENCRYPTION_KEY=<secret-manager:agentscope/signing-encryption-key>
AGENT_SIGNATURE_REQUIRED=false

# 迁移完成后
AGENT_SIGNATURE_REQUIRED=true
```

预检错误只包含数量和处理建议，不包含 key hash、密文或任何明文凭证。

