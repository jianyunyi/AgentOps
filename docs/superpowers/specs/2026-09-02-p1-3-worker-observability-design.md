# AgentScope P1-3 Worker 可观测性与告警设计

**日期：** 2026-09-02  
**范围：** Worker 指标端点、Outbox/Redis Stream 指标、Prometheus 抓取和生产告警  
**前置：** P1-2 Worker 可靠性已完成，提交 `ce76f00`  
**目标：** 让 Worker 的可靠性状态可被 Prometheus 观测、告警和容量评估，避免消息堆积、死信增长或 Redis 故障只能通过日志发现。

## 为什么下一阶段做这个

P1-2 已经保证了 Pending 接管、重试、死信和故障退避，但当前这些状态只存在于 Redis、MySQL 和日志中。Worker 是独立进程，没有 `/metrics` 端点；现有 Prometheus 配置只抓取 API，无法形成生产值守闭环。

## 设计决策

### 1. Worker 独立 Metrics HTTP 服务

Worker 启动一个独立的只读 HTTP 服务，默认监听 `WORKER_METRICS_ADDR=:9091`，提供：

- `GET /metrics`：Prometheus text exposition format。
- `GET /health/live`：进程存活检查，不访问 MySQL/Redis。

Metrics 服务使用独立的 `http.Server` 和 5 秒读写超时；收到 SIGTERM 时和 Worker 主循环一起优雅退出。Metrics 端口不暴露业务 API，不接受写请求，也不需要 Session 或 Agent Key。

如果 Metrics 服务启动失败，Worker 进程启动失败并交给编排系统重启，避免出现“业务运行但监控失效”的隐性状态。运行中 Metrics 请求失败不影响消息消费。

### 2. 指标名称与标签

使用固定、低基数标签；禁止使用 tenant ID、message ID、trace ID、consumer ID 作为 Prometheus label。

Worker 指标：

- `agentscope_worker_messages_recovered_total`：`XAUTOCLAIM` 接管成功的消息数。
- `agentscope_worker_messages_acked_total`：成功 ACK 的消息数。
- `agentscope_worker_messages_retried_total`：成功写回重试消息的次数。
- `agentscope_worker_messages_dead_lettered_total`：成功写入死信的次数。
- `agentscope_worker_redis_errors_total`：Redis 运行期错误数，标签 `operation=claim|read|ack|requeue|dead_letter`。
- `agentscope_worker_processing_errors_total`：分析器返回错误数。
- `agentscope_worker_outbox_published_total`：Outbox 成功发布数。
- `agentscope_worker_outbox_publish_errors_total`：Outbox 发布或状态更新失败数。
- `agentscope_worker_outbox_pending_age_seconds`：最近一次成功 Claim 的 Outbox 事件年龄；无事件时为 0。
- `agentscope_worker_info`：常量 Gauge，包含 `consumer_id` 不作为 label，只提供 `version` 等静态信息，避免实例高基数；Consumer ID 继续只写结构化日志。

HTTP 指标复用现有格式和命名风格。所有指标使用 `counter` 或 `gauge`，进程重启后允许归零，Prometheus 使用 `rate()` 和 `increase()` 计算趋势。

### 3. 业务代码埋点边界

- `RunWithOptions` 接收可选的 Worker Metrics 注入；未注入时使用 no-op，不改变单元测试和库调用行为。
- 只有成功的 ACK、Requeue、DLQ、Pending 接管才增加对应成功指标。
- Redis 操作失败按操作类型增加错误指标；日志仍只记录安全摘要。
- Processor 分析错误单独计数，达到最大尝试后成功写 DLQ 仍计入 dead-lettered。
- Outbox Publisher 暴露明确的发布结果（无事件、成功、发送失败、状态更新失败），Runner 据此更新指标，不能把“本轮无事件”计为成功发布。

### 4. Prometheus 抓取

- 本地/Compose Prometheus 增加 Worker target `worker:9091`。
- Kubernetes 为 Worker 增加 `containerPort: 9091` 及 Prometheus scrape annotations；不新增业务 Service，避免把 Metrics 端口暴露到集群外。
- `WORKER_METRICS_ADDR` 进入 `.env.example`、生产 Compose 和 Kubernetes ConfigMap。

### 5. 告警规则

增加以下初始告警，阈值是上线前的保守基线，后续基于实际基线调整：

- `AgentScopeWorkerDown`：Worker target `up == 0` 持续 2 分钟。
- `AgentScopeWorkerRedisErrors`：Redis 错误速率持续 5 分钟大于 0.1/s。
- `AgentScopeWorkerDeadLetters`：死信增长速率大于 0，持续 5 分钟，严重级别 warning。
- `AgentScopeWorkerOutboxPublishErrors`：Outbox 发布错误持续 5 分钟，严重级别 critical。
- `AgentScopeWorkerOutboxPendingAge`：Pending age 超过 60 秒持续 5 分钟，严重级别 warning。
- `AgentScopeWorkerProcessingStalled`：消费 ACK 速率为 0 且存在 recovered/retried 活动，持续 10 分钟。

告警表达式必须只使用已定义指标；不依赖日志内容，不使用高基数标签。告警 annotation 提供排查方向，但不写入 Secret、Payload 或消息正文。

## 配置

```env
WORKER_METRICS_ADDR=:9091
```

空值使用默认值；地址格式由 HTTP Server 解析。Metrics 服务只绑定容器端口，生产环境通过 NetworkPolicy 限制仅允许 Prometheus 访问。

## 测试验收标准

- Worker Metrics handler 输出合法 Prometheus 文本和预期 counter/gauge。
- ACK、Pending 接管、重试、死信、Redis 错误和 Outbox 结果各自只在成功/失败边界增加一次。
- Metrics 端点不泄露 Payload、Prompt、tenant ID、trace ID、API Key、Authorization 或 Consumer ID。
- Worker Metrics HTTP 服务能启动、响应 `/health/live`，并在 Context 取消时退出。
- Prometheus 配置包含 API 和 Worker target；告警规则引用的指标均存在。
- 全量 Go 测试、`go vet`、真实 Redis 集成测试和 YAML 静态检查通过。

## 非目标

- 本阶段不引入 Prometheus client 第三方依赖；沿用当前轻量 exposition 实现。
- 本阶段不实现日志平台、Trace 系统或跨区域监控。
- 本阶段不改变 Worker 消息处理语义，不更换 Redis Streams，也不实现 HMAC 凭证协议。
