# AgentScope 企业 AI Agent 可观测、审计与成本治理平台设计规格

## 1. 项目目标

AgentScope 面向已经在企业内部运行的 AI Agent，提供调用链追踪、风险审计、Token/成本统计和只读回放能力。首个业务场景是企业 IT 运维 Agent：它能够查询日志、读取监控、创建故障工单和发送通知。

第一版目标不是实现 Agent 编排，而是完成一条可上线验证的治理闭环：

```text
Agent 接入 → 事件采集 → Trace/Span 持久化 → 异步分析 → 控制台审计
```

## 2. 目标用户与角色

- Owner：管理租户、成员、Agent 和全部配置。
- Admin：管理 Agent、查看 Trace 和风险。
- Developer：查看所负责 Agent 的调用链和错误。
- Auditor：查看并审核风险事件、查看审计日志。
- Viewer：只读查看 Trace 和统计。

普通员工只使用外部 IT 运维 Agent，不直接操作 AgentScope 控制台。

## 3. 第一版范围

### 必须实现

- 企业用户登录。
- 租户和 RBAC 权限。
- 创建 Agent、吊销和轮换 Agent Key。
- Agent 通过 HTTP 上报统一事件。
- Trace/Span/LLM Call/Tool Call 持久化。
- 事件幂等处理。
- Redis Stream 异步任务。
- 基础风险检测：Prompt 注入、敏感信息、高危工具、Agent 循环。
- Token 与估算成本统计。
- Trace 列表、详情和调用树。
- 风险列表和人工审核状态。
- 只读调用回放。
- 审计日志。
- 跨租户访问隔离。

### 明确不在第一版

- Agent 编排器。
- 自研大模型或向量数据库。
- 完整多语言 SDK。
- 真实工具重新执行。
- 自动阻断所有高风险操作。
- 企业 SSO、账单系统、多云部署。

## 4. 技术架构

第一版采用模块化单体加异步 Worker：

```text
Next.js/React 控制台
          ↓
Go + Gin API
          ↓
MySQL + GORM
          ↓
Redis Stream
          ↓
Risk/Usage Worker
          ↓
大模型风险分析与统计
```

后端模块边界：

- auth：用户认证、Session、权限。
- tenant：租户和成员关系。
- agent：Agent 生命周期和凭证。
- trace：事件接收、Trace/Span 查询。
- risk：规则检测、大模型分析、风险审核。
- usage：Token 和成本计算。
- worker：异步消费、重试和死信处理。

代码依赖方向：

```text
Handler → Application Service → Domain Service/Repository
```

领域逻辑不依赖 Gin 和 GORM；Handler 不直接操作数据库。

## 5. 统一事件协议

所有 Agent 通过统一事件上报：

```json
{
  "event_id": "evt_01JABC",
  "trace_id": "trace_001",
  "span_id": "span_002",
  "parent_span_id": "span_001",
  "event_type": "llm_call",
  "sequence": 2,
  "occurred_at": "2026-08-17T10:00:01.120Z",
  "payload": {}
}
```

服务端从 Agent Key 认证结果中补充 `tenant_id` 和 `agent_id`，客户端不能指定这两个字段。

第一版事件类型：

- `trace_start`
- `llm_call`
- `tool_call`
- `risk_check`
- `agent_output`
- `trace_end`

事件通过 `event_id` 做幂等，数据库使用 `(tenant_id, event_id)` 唯一约束。

## 6. 核心数据模型

主要表：

- `tenants`
- `users`
- `roles`
- `permissions`
- `agents`
- `agent_credentials`
- `traces`
- `spans`
- `llm_calls`
- `tool_calls`
- `risk_events`
- `usage_records`
- `audit_logs`

所有业务表都必须包含 `tenant_id` 或通过带租户的关联关系实现隔离。关键索引以 `tenant_id` 开头，Trace 和 Span 使用租户内唯一标识。

`traces` 表示一次完整 Agent 请求；`spans` 表示请求中的步骤，并通过 `parent_span_id` 形成调用树。稳定且需要统计的 LLM/Tool 字段单独建列，变化快的请求和响应内容保存为 JSON 快照，并受脱敏策略保护。

## 7. 事件接收链路

```text
POST /api/v1/ingest/events
  → Agent Key 认证
  → 事件格式、大小、时间校验
  → 幂等检查
  → MySQL 事务保存 Trace/Span 明细
  → Redis Stream 投递
  → 返回 202 Accepted
```

同步链路不执行大模型分析。Redis 投递失败时，数据保持 `analysis_status=pending`，由补偿任务重试；后续可升级为 Outbox Pattern。

## 8. 异步处理

Worker 消费事件后执行：

- 风险规则检测。
- 必要时调用大模型进行语义分析。
- Token 和成本统计。
- Trace 风险等级合并。
- 更新分析状态。

失败任务采用退避重试，超过次数进入死信队列。Worker 必须幂等，重复消费不得重复创建风险事件或统计数据。

## 9. 认证与安全

用户使用邮箱/密码登录，密码使用 bcrypt 或 Argon2 保存，登录态使用 HttpOnly Cookie Session。Agent 使用独立 API Key，数据库只保存 Hash，原始 Key 创建时只返回一次。

所有业务查询必须带当前 `tenant_id`。前端隐藏按钮不等于权限控制，后端必须校验身份、租户归属和 RBAC 权限。

第一版风险内容支持敏感信息脱敏；普通应用日志禁止输出 API Key、Authorization、完整 Prompt 和密钥。风险审核、Agent Key 轮换、成员和权限变更必须写入追加式审计日志。

事件接口需要按租户/Agent/IP 限流，并限制请求体大小。第一版通过 API Key、事件幂等和时间窗口校验降低重放风险，后续再增加 HMAC 签名。

## 10. 前端控制台

Next.js App Router 页面：

- `/dashboard/traces`：Trace 列表、筛选和分页。
- `/dashboard/traces/[traceId]`：Trace 概览、时间线、Span 树和详情。
- `/dashboard/risks`：风险列表、详情和审核。
- `/settings/agents`：Agent 管理和 Key 轮换。
- `/settings/members`：成员和角色管理。

Trace 执行中的增量更新通过 SSE 推送。服务端状态、页面筛选状态和全局身份状态分离管理。前端必须覆盖加载、空数据、失败、无权限和连接断开状态。

## 11. 首个垂直切片

第一阶段打通：

```text
管理员创建 Agent
  → 生成 API Key
  → Agent 上报 trace_start/llm_call/tool_call/trace_end
  → 保存 Trace/Span
  → Redis Worker 消费
  → 前端分页查询并展示调用树
```

验收要求：

- API Key 只展示一次且只保存 Hash。
- 事件自动绑定认证得到的租户和 Agent。
- 重复事件不会重复创建数据。
- Trace/Span 可查询，状态和耗时正确。
- Worker 可消费、失败可重试。
- 跨租户查询无法读取数据。
- 核心链路有集成测试。

## 12. 测试策略

- 单元测试：认证、幂等、状态机、风险等级、成本计算。
- Repository 测试：查询条件、租户隔离、分页、唯一约束和事务回滚。
- 集成测试：创建 Agent、上报事件、查询 Trace 的完整链路。
- Worker 测试：消费、重复消费、重试和死信。
- 前端测试：列表筛选、详情加载、风险审核和错误状态。

## 13. 后续扩展

在第一版稳定后，再考虑：

- Outbox Pattern。
- Python/Node/Go SDK。
- HMAC 事件签名。
- 沙箱回放。
- 风险策略阻断和人工审批。
- 多模型评测与自动路由。
- SSO、数据归档和企业合规报告。
