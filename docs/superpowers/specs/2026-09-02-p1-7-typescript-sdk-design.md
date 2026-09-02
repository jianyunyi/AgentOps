# P1-7 TypeScript/Node Agent SDK 设计

## 背景与目标

AgentScope 已经提供 Go SDK，但 TypeScript/Next.js 生态的服务端 Agent 仍需要自行实现 HMAC 签名、Nonce、防重放请求头、重试和错误解析。本阶段新增一个独立的 TypeScript SDK，让 Node.js 服务端能够以与 Go SDK 相同的协议安全摄入 Agent 事件。

SDK 只支持服务端运行时。API Key 和 Signing Secret 不允许进入浏览器 bundle、React Client Component、`localStorage` 或 URL 参数。

## 范围

### 包结构

在 `sdk/typescript` 创建独立 npm 包，源码不依赖服务端 `internal` 包，也不依赖 Next.js。公共入口只导出客户端、事件类型、错误类型和重试配置。

### 客户端行为

`AgentOpsClient` 构造时校验绝对 `http`/`https` Base URL、API Key、Base64 Signing Secret 和重试策略。`ingest(event)` 将事件序列化为一次固定的 UTF-8 Body；每次尝试生成新的时间戳和随机 Nonce，按已冻结的 HMAC v1 canonical 规则签名，然后请求 `/api/v1/ingest/events`。

默认最多 3 次尝试。仅网络错误、429、502、503、504 和 5xx 可重试；400/401/403/404、结构化协议错误和参数错误立即返回。等待使用有上限的指数退避，并支持 `AbortSignal` 取消。

### 错误与安全

提供可判定的 `AgentOpsAPIError`，包含 HTTP 状态、服务端错误码和脱敏后的 Request ID，不包含 API Key、Signing Secret 或原始 Body。响应体大小受上限约束，非 JSON 响应转换成稳定错误。SDK 内部不记录日志，也不接受 logger 注入，避免凭证通过默认日志泄露。

SDK 的签名实现必须与 Go SDK 使用同一组测试向量：版本、HTTP method、请求路径、Unix 时间戳、Nonce 和 Body SHA-256 的 canonical 顺序及大小写保持一致。

## 目录与职责

```text
sdk/typescript/
├── src/client.ts       # AgentOpsClient 与一次请求/重试流程
├── src/signing.ts      # HMAC v1、Body hash、Nonce
├── src/retry.ts        # 可取消退避与重试分类
├── src/errors.ts       # 结构化错误与敏感值脱敏
├── src/types.ts        # Event、IngestResult、配置类型
├── src/index.ts        # 公共导出
├── test/               # 向量、HTTP、错误和配置测试
├── examples/           # Node 服务端最小示例，不含真实凭证
├── package.json
├── tsconfig.json
└── README.md
```

包默认使用 Node 18+ 原生 `fetch`、Web Crypto 和 `AbortController`，不增加运行时第三方依赖。Node 版本和 TypeScript 编译目标在 package metadata 中明确声明。

## API 契约

```ts
export type AgentEvent = {
  event_id: string;
  trace_id: string;
  span_id: string;
  parent_span_id?: string;
  event_type: string;
  occurred_at: string;
  sequence?: number;
  payload: unknown;
};

export type AgentOpsClientOptions = {
  baseUrl: string;
  apiKey: string;
  signingSecret: string;
  maxAttempts?: number;
  baseRetryDelayMs?: number;
  maxRetryDelayMs?: number;
  fetch?: typeof globalThis.fetch;
};

export class AgentOpsClient {
  ingest(event: AgentEvent, options?: { signal?: AbortSignal }): Promise<{ duplicate: boolean }>;
}
```

实际字段命名必须与服务端现有 `Event` JSON 契约和 Go SDK 保持一致；公共 API 不返回或缓存任何新凭证。

## 测试与验收

- 配置测试覆盖 Base URL、空凭证、非法 Base64 secret、重试边界和默认值。
- 签名测试复用服务端/Go SDK 的固定向量，并验证不同 Body、Path、Method、Nonce 会产生不同签名。
- HTTP 测试验证请求头、固定 Body、每次重试的新 Nonce、可重试状态、不可重试状态、响应大小限制和 AbortSignal。
- 错误测试验证结构化错误、安全脱敏和非法响应处理。
- 可选真实集成测试使用环境变量中的临时凭证；未配置时跳过，不阻塞普通 CI。
- CI 执行 TypeScript 类型检查、单元测试、包构建和依赖审计。

本阶段不实现浏览器 SDK、Python SDK、凭证自动刷新、服务端查询 API 或 OpenTelemetry 集成。

