# P1-3 Worker 可观测性与告警实施计划

> 本计划基于 `docs/superpowers/specs/2026-09-02-p1-3-worker-observability-design.md`，按测试先行执行。

## 目标

让 Worker 的关键可靠性状态通过低基数 Prometheus 指标暴露，并为独立 Worker 提供可探活的 Metrics HTTP 服务和真实告警基线。

## 实施任务

### 1. Worker Metrics 数据模型

- 新增线程安全的 Worker Metrics，固定输出 ACK、接管、重试、死信、处理错误、Redis 错误、Outbox 成功/失败和 Pending age 指标。
- Redis 错误只允许固定 operation 标签：`claim`、`read`、`ack`、`requeue`、`dead_letter`。
- 不将 tenant ID、message ID、trace ID、consumer ID、Payload 放入指标或日志。

### 2. 测试先行与埋点

- 先新增 Metrics handler 测试，验证 Prometheus 文本、计数边界、低基数和脱敏。
- 新增 Outbox outcome 测试，区分无事件、发布成功、发送失败和状态更新失败。
- 扩展 Worker Runner 测试，验证成功 ACK/重试/DLQ、Redis 错误和分析器错误只在对应边界计数。

### 3. Outbox 与 Stream 接入

- 为 Outbox Publisher 增加兼容现有 `PublishOne` 的 outcome API，保留发送失败与状态更新失败的可观测结果。
- RunOutbox 主循环使用 outcome 更新 Metrics，单次失败仍按 P1-2 规则退避并继续。
- RunWithOptions 注入可选 Metrics，在 Pending 接管、ACK、重试、DLQ、Redis 错误和分析失败处埋点。

### 4. Worker Metrics HTTP 服务

- 新增独立 `:9091` 默认监听的 Metrics Server，提供 `/metrics` 和 `/health/live`。
- 使用独立 HTTP Server、超时、可取消优雅关闭；监听失败作为启动错误返回。
- Worker 主进程接入信号取消、Metrics Server、Outbox 和 Analysis 三个生命周期。

### 5. 配置与部署

- 增加 `WORKER_METRICS_ADDR` 配置，默认 `:9091`。
- 更新 `.env.example`、生产 Compose、Dockerfile、Kubernetes Worker Deployment 和 Prometheus 抓取配置。
- 增加 Worker Down、Redis 错误、死信、Outbox 错误、Pending age 和处理停滞告警。

### 6. 验证与交付

- 运行 `gofmt`、`go test ./...`、`go vet ./...`。
- 用真实 MySQL/Redis 运行集成测试，并检查 Metrics 端点和 Redis Stream 行为。
- 完成 diff 安全审查后提交并推送 P1-3。

## 验收标准

- `/metrics` 输出合法 Prometheus 文本，指标名称和固定标签完整。
- Worker 关键状态只在正确成功/失败边界增加一次，Outbox 无事件不计为发布成功。
- Metrics 端点不泄露 Payload、Prompt、tenant ID、trace ID、API Key、Authorization 或 Consumer ID。
- Metrics Server 启动失败可被主进程感知，Context 取消可优雅退出。
- Prometheus 配置和告警规则引用的指标均已实现。
