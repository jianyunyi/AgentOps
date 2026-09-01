# AgentScope P1-1 安全与数据边界设计

**日期：** 2026-09-01  
**范围：** Prompt/Trace 数据脱敏、Agent 状态校验、摄入请求防重放  
**目标：** 在不改变 Trace 业务模型和租户授权语义的前提下，阻止敏感内容落库、禁用 Agent 继续写入，以及同一摄入请求被重复提交。

## 背景与问题

当前 Trace 摄入会把事件 Payload 直接写入 `spans.input_snapshot`，并把同一份原文放入 Outbox 消息。风险分析虽然会生成脱敏文本，但脱敏发生在异步分析阶段，不能保护数据库、Outbox 或调试日志中的原始 Prompt。

当前 Agent Key 查询只校验 Credential，未进一步确认关联 Agent 仍存在且处于 active 状态。这样 Agent 被 suspended 后，旧的 active Credential 仍可能用于摄入。

事件 ID 幂等只能阻止完全相同的事件重复落库，不能阻止攻击者在有效时间内重放同一请求或用同一 Key 构造新的事件。因此摄入接口增加时间戳和一次性 Nonce 校验，Nonce 使用 Redis `SETNX` 保存并设置 TTL。

## 设计决策

### 1. 单一脱敏边界

在 `trace.Service.Ingest` 完成基础字段校验后，立即对 Payload 执行版本化脱敏，后续所有持久化和异步消息只使用脱敏结果：

- JSON 对象和数组递归处理所有字符串值；敏感字段名（`password`、`secret`、`token`、`api_key`、`authorization`、`cookie` 等）直接替换为 `[REDACTED]`。
- 普通字符串使用现有风险规则处理 Prompt 注入、API Key、邮箱和手机号，并保持 JSON 可解析。
- Payload 必须是合法 JSON；非法 JSON 直接返回 `400 INVALID_EVENT`，不使用字符串兜底，避免写入 `type:json` 列时产生不可查询的数据。
- 脱敏规则有固定版本号 `v1`，便于后续规则升级和历史数据解释。
- `Span.InputSnapshot`、Outbox `payload.input` 和传入分析 Worker 的内容必须来自同一份脱敏结果。
- 原始 Payload 仅允许在当前请求内存中短暂存在，不写日志、不写错误信息、不进入审计快照。

脱敏失败采用 fail-closed：请求返回 `400 INVALID_EVENT`，不创建 Trace、Span 或 Outbox 记录。

### 2. Agent 状态是认证条件

Agent API Key 认证流程调整为：

1. 通过 `KeyHash` 查找 active 且未过期 Credential。
2. 按 Credential 的 `tenant_id + agent_id` 查询 Agent。
3. Agent 不存在、租户不一致或状态不是 `active` 时统一返回 `UNAUTHORIZED_AGENT`。
4. 只有通过以上校验才更新 Credential 的 `last_used_at` 并返回服务端生成的身份。

Agent 管理接口的 rotate/revoke 也必须先确认目标 Agent 属于当前租户；允许对 suspended Agent 执行 rotate/revoke 以便恢复或彻底封禁，但不能通过该操作恢复 Agent 的摄入权限。

### 3. 摄入请求防重放

仅对写入接口 `POST /api/v1/ingest/events` 强制以下请求头：

- `X-Agent-Timestamp`：Unix 秒时间戳，服务端允许与当前时间相差不超过 5 分钟。
- `X-Agent-Nonce`：1 到 128 个 ASCII 字符的一次性随机值。

认证器在 API Key 校验成功后，通过注入的 `NonceStore` 执行：

```text
SET agentscope:agent:nonce:{agent_id}:{nonce} 1 NX EX 600
```

Redis 返回已存在时返回 `409 REPLAY_DETECTED`；时间戳或 Nonce 格式不合法时返回 `400 INVALID_AGENT_REQUEST`。Redis 不可用时 fail-closed，返回 `503 AGENT_AUTH_UNAVAILABLE`，不允许绕过防重放继续写入。

本阶段不引入 HMAC 签名，因为当前 Credential 只保存单向 KeyHash，无法在不改变凭证存储模型的情况下验证签名。HMAC 凭证格式、密钥加密存储和 SDK 迁移单独作为后续 P1 子项目。

查询接口继续支持 Bearer Agent Key，但不强制 Nonce；查询没有写入副作用，仍受 P0 的 Agent 自身 Trace 作用域限制。

## 组件与接口

- `internal/risk/redaction.go`：提供 `RedactPayload(json.RawMessage) ([]byte, error)` 和固定规则版本；不暴露原文到日志。
- `internal/trace/service.go`：只调用一次脱敏函数，并把结果传给 Span 和 Outbox。
- `internal/agent/repository.go`：增加按租户查询 Agent 的能力。
- `internal/agent/service.go`：增加 Agent active 校验和 `AuthenticationMetadata` 校验流程。
- `internal/agent/nonce.go`：定义 `NonceStore`；提供内存实现用于单测和 Redis 实现用于 API 主进程。
- `internal/trace/handler.go`：摄入接口读取时间戳和 Nonce，并将认证错误映射为稳定 HTTP 错误码。
- `cmd/api/main.go`：构造 Redis NonceStore 并注入 Agent Service。
- `internal/platform/config/config.go`：提供 `AGENT_REPLAY_WINDOW_SECONDS` 和 `AGENT_NONCE_TTL_SECONDS`，默认分别为 300 和 600，且限制在安全范围内。

## 错误与恢复

- 脱敏失败：400，不产生任何业务副作用。
- Agent 不存在或 suspended：401，避免泄露 Agent 是否存在。
- 时间戳/Nonce 非法：400。
- Nonce 重复：409，客户端生成新 Nonce 后可重试；事件 ID 幂等仍作为第二道保护。
- Redis 不可用：503，服务端不降级为无防重放模式。
- Trace 和 Outbox 必须继续沿用已有原子事务；事务失败时脱敏后的内容也不应部分落库。

## 测试验收标准

- 规则测试覆盖嵌套 JSON、敏感字段、Prompt 注入、邮箱、手机号、API Key；Trace 摄入测试覆盖非法 JSON 拒绝。
- Trace Service 测试证明原始敏感字段不会出现在 Span 或 Outbox Payload 中。
- Agent Service 测试覆盖不存在 Agent、suspended Agent、跨租户 Credential 和 active Agent。
- Nonce Store 测试覆盖首次写入成功、重复 Nonce 拒绝、TTL 和 Redis 错误 fail-closed。
- Handler 测试覆盖缺失/过期时间戳、缺失/重复 Nonce，以及稳定错误码。
- Go 全量测试、`go vet`、真实 MySQL/Redis 集成测试和 CI 必须通过。

## 非目标

- 本阶段不实现 HMAC 签名协议。
- 本阶段不改变现有 Trace 表结构和历史数据迁移策略。
- 本阶段不实现 Worker Pending 恢复、风险策略 fail-closed 或前端功能；这些属于后续 P1 子项目。
