# P1-2 Worker 可靠性实施计划

> 本计划基于 `docs/superpowers/specs/2026-09-02-p1-2-worker-reliability-design.md`，按测试先行执行。

## 目标

让分析 Worker 和 Outbox 发布循环在进程重启、多实例、Redis 短暂故障及下游失败时保持至少一次处理语义：Pending 可接管，目标写入成功前不 ACK，运行期故障自动退避恢复。

## 实施任务

### 1. 配置边界与消费抽象

- 在 `internal/platform/config/config.go` 增加 `WORKER_CONSUMER_ID`、`WORKER_PENDING_IDLE_SECONDS`、`WORKER_MAX_ATTEMPTS`。
- 保留严格范围校验：Pending idle 为 30–3600 秒，最大尝试次数为 1–20。
- 在 `internal/worker/stream.go` 把 Redis 命令封装到可测试的 Stream 客户端接口，避免单元测试依赖外部 Redis。
- 增加 Consumer ID 自动生成函数，显式配置优先，默认包含 hostname、PID 和随机片段。

### 2. 测试先行覆盖消息状态机

- 新增 Worker fake Stream client，按调用顺序模拟 Pending、New message、ACK、Requeue、DLQ 和 Redis 错误。
- 先写失败测试，覆盖 Pending 优先、`XAUTOCLAIM`、动态 Consumer、成功 ACK、重试、死信、目标写入失败不 ACK、ACK 失败不退出、Context 取消。
- 新增配置测试，覆盖默认值、显式值和越界拒绝。
- 新增 Outbox Runner 测试，证明单次发布失败不会结束循环，Context 取消能够退出。

### 3. Stream Consumer 与 Runner 实现

- 使用 `XAUTOCLAIM` 从 `0-0` 扫描超过 idle 阈值的 Pending 消息，并沿用原始 Stream ID。
- 每轮先处理 recovered messages，再读取 `XREADGROUP ... >` 的新消息。
- 对 Decode、分析决策、Requeue、DLQ、ACK 严格执行目标成功后确认原消息。
- 对运行期 Redis 错误记录不含 Payload 的结构化摘要并指数退避；消息失败时保留 Pending，后续由接管流程恢复。
- 将运行时错误与 Context 取消区分，只有 Context 取消才结束消费循环；启动时创建 Group 失败仍返回。

### 4. Outbox Worker 与主进程接入

- 修改 `RunOutbox`：发布/数据库单次错误不退出，使用可取消退避后继续 Claim。
- 在 `cmd/worker/main.go` 注入配置的 Consumer ID、Pending idle 和最大尝试次数。
- 将 Outbox 与 Analysis 两个循环的退出语义统一为：Context 取消正常收敛，启动错误和不可恢复错误显式退出并由编排系统处理。

### 5. 集成与回归验证

- 扩展 `internal/integration`：真实 Redis Stream Group 测试 Pending 接管、ACK、重试消息和 DLQ。
- 运行 `gofmt`、`go test ./...`、`go vet ./...`，并在可用环境运行 MySQL/Redis 集成测试。
- 检查日志和错误路径不包含 Payload、Prompt、API Key 或 Authorization。
- 完成代码审查后提交 P1-2 变更。

## 验收标准

- 多实例默认 Consumer ID 不重复，显式 Consumer ID 可复现。
- Pending 消息优先于新消息处理，XAUTOCLAIM 接管后使用原消息 ID ACK。
- Requeue/DLQ/ACK 任一失败时原消息不 ACK，Worker 不因单次运行期 Redis 错误退出。
- 达到最大尝试次数进入 DLQ；Outbox 单次失败不停止发布循环。
- Context 取消可终止阻塞读取和退避等待。
- 全量 Go 测试、`go vet` 和真实 Redis 集成测试通过。
