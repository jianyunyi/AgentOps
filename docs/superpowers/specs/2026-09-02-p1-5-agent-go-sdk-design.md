# AgentScope P1-5 Go Agent SDK 设计

**日期：** 2026-09-02  
**范围：** Go Agent SDK、HMAC 请求签名、可靠摄入、接入文档和集成测试  
**前置：** P1-4 已完成服务端 HMAC v1、AES-GCM 密钥保护和一次性凭证返回

## 目标

为企业业务 Agent 提供可直接引入的 Go SDK，使业务方不需要自行实现签名、Nonce、请求重试和错误解析，即可安全地向 AgentScope 上报 Trace 事件。

## 方案

SDK 放在同一仓库的独立模块 `sdk/go`，模块路径为 `github.com/jianyunyi/AgentOps/sdk/go`。SDK 不导入服务端 `internal` 包，独立实现已经冻结的 HMAC v1 协议，使用固定协议测试向量保证跨模块兼容。

公开 API 保持小而稳定：

```go
client, err := agentops.NewClient(agentops.Config{
    BaseURL:      "https://agentscope.example.com",
    APIKey:       os.Getenv("AGENT_API_KEY"),
    SigningSecret: os.Getenv("AGENT_SIGNING_SECRET"),
})
result, err := client.Ingest(ctx, agentops.Event{...})
```

`Event` 在 SDK 中定义为公开类型，字段 JSON 名称与服务端一致。`Payload` 使用 `json.RawMessage`，序列化请求体一次后，每次重试都复用完全相同的字节。

## 签名与请求流程

每次尝试都生成新的 Unix 秒 Timestamp 和 printable ASCII Nonce，计算同一请求体的 SHA-256 小写十六进制摘要，再按以下规则计算签名：

```text
v1\nPOST\n/api/v1/ingest/events\n<timestamp>\n<nonce>\n<body_sha256>
```

请求头为 `Authorization`、`X-Agent-Timestamp`、`X-Agent-Nonce` 和 `X-Agent-Signature: v1=<lowercase hex>`。Signing Secret 只驻留在 Client 内存，不写入错误、日志、Request ID 或 User-Agent。

## 重试语义

摄入接口依赖 `event_id` 幂等。默认最多 3 次尝试，允许配置 1–5 次。仅重试网络错误、上下文允许时的超时、408、429、500、502、503、504；401、409 和其他 4xx 直接返回。每次重试重新生成 Timestamp、Nonce 和签名，不能复用已发送的 Nonce。

退避使用 100ms、300ms、900ms 的指数序列；429 优先使用合法的 `Retry-After` 秒数，并限制单次等待不超过 10 秒。等待通过 Context-aware Timer 实现，Context 取消立即返回。

## 错误和边界

`APIError` 暴露 HTTP 状态码、服务端 code、脱敏 message、Request ID 和 Retry-After；不会拼接请求体、Authorization、签名、Canonical String 或底层响应中的敏感头。配置错误、签名协议错误、认证错误、重放错误、限流错误和服务端错误提供可判断的错误类型/辅助函数。

构造 Client 时校验绝对 HTTP(S) BaseURL、API Key、Signing Secret 和重试次数。生产默认使用 10 秒 HTTP 超时，并允许注入自定义 `http.Client`；不修改调用方传入的 Client 或 Transport。

## 测试和文档

- 单元测试覆盖配置校验、Body Hash、Canonical String、签名向量、Nonce 生成、错误解析和重试分类。
- `httptest.Server` 测试验证服务端收到的签名、原始 Body 字节、请求头和重试 Nonce 唯一性。
- 集成测试在 `AGENTSCOPE_INTEGRATION=1` 时使用真实 API 依赖的 MySQL 3307、Redis 3509，验证 SDK 创建的请求可被服务端接受。
- `sdk/go/README.md` 提供安装、凭证安全、最小摄入示例、重试说明和生产配置建议；示例不包含真实密钥。

## 非目标

- 本阶段不实现 Python/Node SDK。
- 本阶段不修改服务端 HMAC 协议、不提供服务端查询 SDK、不自动保存或刷新 Agent 凭证。
- 本阶段不把日志、Tracing exporter 或 OpenTelemetry 作为 SDK 依赖，保持零第三方运行时依赖。
