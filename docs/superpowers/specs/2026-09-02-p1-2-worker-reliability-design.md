# AgentScope P1-2 Worker 可靠性设计

**日期：** 2026-09-02  
**范围：** Redis Stream Pending 恢复、Consumer 隔离、Redis 故障自愈、重试和死信  
**目标：** 在 Worker 进程崩溃、实例扩缩容、Redis 短暂不可用和下游分析失败时，保证消息不被静默丢失，并让系统保持可恢复的至少一次处理语义。

## 背景与问题

当前分析 Worker 只用 `XREADGROUP ... >` 读取新消息。Worker 崩溃或网络中断后，已经进入 Pending Entries List（PEL）但未 ACK 的消息不会被新的 Consumer 自动接管。

当前所有实例使用固定 Consumer 名称 `agentscope-worker`，无法区分实例，也不利于故障接管。Redis 读取、ACK、重试或死信写入失败时，Worker 直接退出，容器虽然可以重启，但缺少进程内退避和明确的消息状态保护。

Outbox 发布循环也会在一次数据库或 Redis 错误后退出，导致短暂依赖故障扩大为持续停止。

## 设计决策

### 1. Pending 优先恢复

分析 Worker 每轮按以下顺序读取：

1. `XAUTOCLAIM` 接管本 Consumer Group 中空闲超过 `WORKER_PENDING_IDLE_SECONDS` 的 Pending 消息。
2. 处理本轮接管的消息。
3. `XREADGROUP ... >` 读取新消息。
4. 若没有消息，按阻塞时间等待下一轮。

`XAUTOCLAIM` 使用 `start = "0-0"`、配置的 `MinIdle` 和批量大小。每轮从头扫描是有意设计：已接管消息的 idle 时间会被重置，下一轮不会重复接管；这样不需要把恢复游标持久化到进程内，也能在 Worker 重启后从头恢复。

接管消息沿用原始 Stream ID 和 Payload。ACK、重试、死信都必须针对原始 ID 执行。

### 2. 每个实例使用独立 Consumer ID

Consumer ID 生成规则：

```text
agentscope-worker-{hostname}-{pid}-{random}
```

支持 `WORKER_CONSUMER_ID` 显式覆盖，便于 Kubernetes 排障。未配置时每次进程启动生成新 ID；旧实例留下的 Pending 消息由其他实例通过 `XAUTOCLAIM` 接管。

`StreamConsumer` 保存 Consumer ID，但不保存业务状态。Consumer ID 只用于 Redis Group 消费者隔离，不作为租户、Agent 或业务身份。

### 3. 严格 ACK 顺序

每条消息的状态转换如下：

```text
读取/接管
  ├─ Decode 失败 → 写 Dead Letter 成功 → ACK 原消息
  ├─ 分析成功 → ACK 原消息
  ├─ 临时失败 → Requeue 新消息成功 → ACK 原消息
  └─ 超过最大次数 → 写 Dead Letter 成功 → ACK 原消息
```

任何 Dead Letter、Requeue 或 ACK 失败，都不确认原消息。原消息留在 PEL，后续由 `XAUTOCLAIM` 恢复。Requeue 成功但 ACK 失败时可能产生重复消息，这是允许的至少一次语义；风险事件已有租户/Span/规则唯一约束，分析持久化必须保持幂等。

非法消息不把原始 Payload 写入日志或死信错误信息；死信只保留消息 ID、解析错误分类、attempt 和可安全解析的消息字段。

### 4. Redis 故障进程内自愈

Worker 的 Redis 运行期错误分为两类：

- 可恢复错误：读取、接管、ACK、Requeue、Dead Letter 的网络错误或 Redis 暂时错误。记录结构化错误摘要后指数退避，继续运行；未完成 ACK 的消息保持 Pending。
- 不可恢复启动错误：创建 Stream Group 失败、配置不完整或 Redis 明确拒绝命令。启动阶段返回错误，由容器编排系统重启。

退避使用有上限的指数策略，避免 Redis 恢复后雪崩；Context 取消立即中止等待。运行期不因一次 Redis 错误直接返回。

Outbox Publisher 使用同样的原则：`PublishOne` 错误不退出主循环，等待退避后重试；只有 Context 取消才正常返回。Outbox 事件只有 Redis Stream 写入成功后才标记 delivered；失败事件由数据库状态和 `available_at` 控制下一次 Claim。

### 5. 重试和死信边界

`attempt` 是消息字段，首次消息默认为 1。分析器返回错误时：

- `attempt < WORKER_MAX_ATTEMPTS`：按指数退避后写回 Stream，attempt 加 1。
- `attempt >= WORKER_MAX_ATTEMPTS`：写入死信 Stream，不再回队。

业务分析错误、LLM 超时和数据库暂时错误由 Processor 统一视为可重试；消息格式错误、缺少 EventID 等不可解析消息直接死信。无论哪一种分支，都必须在目标写入成功后 ACK 原消息。

### 6. 配置

新增配置：

```env
WORKER_CONSUMER_ID=
WORKER_PENDING_IDLE_SECONDS=120
WORKER_MAX_ATTEMPTS=3
```

`WORKER_PENDING_IDLE_SECONDS` 限制在 30–3600 秒，`WORKER_MAX_ATTEMPTS` 限制在 1–20。Consumer ID 为空时自动生成。重试延迟沿用现有有上限指数策略，避免增加不必要的配置维度。

## 组件与接口

- `internal/worker/stream.go`：Consumer ID、`XAUTOCLAIM` Pending 恢复和新消息读取；保留 ACK/Requeue/Dead Letter 方法。
- `internal/worker/runner.go`：Pending 优先消费、运行期 Redis 错误退避、ACK 顺序和 Context 停止。
- `internal/worker/analysis.go`：保留幂等处理和最大尝试次数语义，确保错误分类不会吞掉消息。
- `internal/outbox/publisher.go` 或 `internal/worker/runner.go`：Outbox 运行期故障退避，不因单次错误退出。
- `internal/platform/config/config.go`：解析 Worker 配置并做范围校验。
- `cmd/worker/main.go`：注入 Consumer ID、Pending idle 和最大尝试次数。

## 观测与日志

每次接管、重试、死信和 Redis 运行期错误记录结构化摘要：Stream、消息 ID、attempt、Consumer ID、错误分类和退避时长。禁止记录完整 Payload、Prompt、API Key 或 Authorization。

本阶段至少提供可测试的计数边界，后续接入 Prometheus 指标：`worker_messages_recovered_total`、`worker_messages_retried_total`、`worker_messages_dead_lettered_total`、`worker_redis_errors_total`。

## 测试验收标准

- Consumer ID 自动生成时不同实例不相同，显式配置值保持不变。
- `XAUTOCLAIM` 可以返回 idle Pending 消息，读取顺序是 Pending 优先、再读取新消息。
- Redis 读取、接管、ACK、Requeue、Dead Letter 错误不会让运行中的 Worker 直接退出。
- Requeue 或 Dead Letter 失败时原消息不 ACK；目标成功后才 ACK。
- 成功消息 ACK，临时错误按 attempt 回队，达到上限进入死信。
- Context 取消能够终止阻塞读取和退避等待。
- Outbox Publisher 的单次失败不会停止发布循环。
- 不记录原始消息 Payload。
- Go 全量测试、`go vet`、真实 Redis Stream 集成测试和 CI 必须通过。

## 非目标

- 本阶段不更换 Redis Streams，不引入 Kafka/RabbitMQ。
- 本阶段不实现跨区域消息复制、精确一次语义或分布式事务。
- 本阶段不改变风险事件业务规则；只保证 Worker 投递和消费可靠性。
