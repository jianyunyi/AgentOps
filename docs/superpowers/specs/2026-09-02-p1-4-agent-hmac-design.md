# AgentScope P1-4 Agent 请求签名与 HMAC 凭证升级设计

**日期：** 2026-09-02  
**范围：** Agent 凭证签名密钥、HMAC 请求完整性、版本化迁移、旧凭证兼容  
**前置：** P1-1 重放保护已完成，P1-2 Worker 可靠性已完成，P1-3 可观测性已完成  
**目标：** 在保留 Agent 身份认证和时间戳/Nonce 防重放的基础上，验证 Trace 摄入请求没有被篡改，并让签名密钥只在创建/轮换时返回一次、数据库中只保存加密密文。

## 背景与风险

当前请求使用 Bearer Agent API Key、Unix 时间戳和 Nonce。时间戳/Nonce 可以阻止相同请求被重复接受，但不能证明请求 Body、Path 或 Method 在传输过程中未被修改。当前凭证只保存不可逆 `KeyHash`，因此不能直接把现有 KeyHash 当作 HMAC 密钥，也不能在服务端恢复 API Key。

如果直接把 HMAC 设为全量强制，历史 Agent 没有签名密钥，会在部署后全部无法摄入。因此本阶段采用“新凭证强制签名、旧凭证兼容迁移、可配置全量强制”的滚动策略。

## 设计决策

### 1. 凭证双密钥模型

每次创建或轮换 Agent Credential 时生成两份独立随机值：

- `api_key`：现有 Bearer 身份密钥，只保存 SHA-256 `KeyHash`。
- `signing_secret`：HMAC-SHA256 请求签名密钥，只在创建/轮换响应中返回一次。

Credential 新增 `SigningSecretCiphertext` 字段，使用 AES-256-GCM 加密后保存。明文签名密钥不写入数据库、日志、审计快照、Prometheus 指标或错误响应。

Agent 管理接口的创建和轮换响应新增 `signing_secret`。前端和调用方必须提示用户仅此一次保存；后续列表接口永远不返回签名密钥。

### 2. 加密密钥与启动边界

新增配置：

```env
AGENT_SIGNING_ENCRYPTION_KEY=
AGENT_SIGNATURE_REQUIRED=false
```

`AGENT_SIGNING_ENCRYPTION_KEY` 必须是 32 字节密钥，支持 64 位十六进制或标准 Base64。生产环境启用 HMAC 新凭证前必须配置；配置值不出现在日志。

`AGENT_SIGNATURE_REQUIRED=false` 时：

- 新 Credential（存在签名密文）必须带合法 HMAC 签名。
- 历史 Credential（签名密文为空）继续使用 API Key + 时间戳 + Nonce 兼容模式。

`AGENT_SIGNATURE_REQUIRED=true` 时，所有摄入请求必须走 HMAC；历史 Credential 没有签名密钥时返回稳定的迁移错误，不能静默降级。启用该配置前必须完成历史 Credential 轮换。

配置解析失败或启用签名强制但缺少加密密钥时，API 启动失败，避免以错误安全假设运行。

### 3. 签名协议 v1

客户端先计算请求 Body 的 SHA-256 十六进制小写值，然后构造 canonical string：

```text
v1\nPOST\n/api/v1/ingest/events\n<unix_timestamp>\n<nonce>\n<body_sha256>
```

使用 HMAC-SHA256：

```text
signature = hex_lower(HMAC-SHA256(signing_secret, canonical_string))
```

请求头：

```text
Authorization: Bearer <api_key>
X-Agent-Timestamp: <unix seconds>
X-Agent-Nonce: <unique printable nonce>
X-Agent-Signature: v1=<hex signature>
```

服务端对 Method、Path、Timestamp、Nonce 和原始 Body Hash 使用同一 canonical 规则。签名比较使用 constant-time compare；缺少、版本未知、长度错误或签名不匹配统一返回 `401 UNAUTHORIZED_AGENT`，不泄露失败原因。

签名验证顺序为：请求体读取和大小限制 → Header 格式检查 → API Key/Agent 状态认证 → 时间戳/Nonce 校验 → 解密签名密钥 → HMAC 校验 → 业务 JSON 解析和持久化。Nonce 只有在 API Key、时间戳和签名全部通过后才 Claim，避免攻击者消耗有效 Nonce。

### 4. HTTP Body 与认证边界

Trace Handler 在绑定 JSON 前读取受 BodyLimit 约束的原始 Body，计算 Body Hash，再恢复 `Request.Body` 供 `ShouldBindJSON` 使用。不得在日志中输出原始 Body 或 Body Hash 与租户/消息 ID的组合高基数信息。

HMAC 当前只保护 `POST /api/v1/ingest/events`。查询接口继续使用 Session 或 Agent Key 的既有认证链路，不强制摄入签名 Header，避免扩大协议影响面。

### 5. 版本化数据库迁移

新增版本化迁移 `5_agent_signing_secret`：

- 为 `agent_credentials` 增加可空密文字段。
- 历史行保持 NULL，不回填、不猜测、不把 KeyHash 当作签名密钥。
- 创建/轮换事务同时写 Credential 加密密文和审计记录。
- 迁移失败时事务回滚；重复运行只执行一次。

在迁移完成前新二进制不得认为 HMAC 已启用。迁移版本和配置状态在启动日志中只输出非敏感摘要。

### 6. 服务接口与兼容性

保留现有 `AuthenticateAPIKey` 查询认证接口；扩展摄入认证元数据，增加 `Signature`、`Method`、`Path`、`BodyHash` 字段，避免让查询请求必须携带签名。

Service 通过注入的 `SigningSecretProtector` 完成加密和解密，测试使用内存实现或固定测试密钥，生产使用 AES-GCM 实现。已有不带签名密钥的构造函数保留兼容行为，但生产主进程必须使用显式 Protector 和配置状态。

错误类型至少包括：

- `ErrInvalidAgentSignature`：格式、版本、Body Hash 或签名不匹配。
- `ErrSigningSecretUnavailable`：服务端无法解密或密钥配置不可用，返回 `503 AGENT_AUTH_UNAVAILABLE`。
- `ErrSignatureRequired`：全量强制模式下历史 Credential 未迁移，返回 `401 UNAUTHORIZED_AGENT`。

错误响应不得返回解密错误、签名期望值、canonical string 或 API Key。

### 7. 审计与安全要求

- 创建/轮换审计只记录 `key_prefix`、Credential ID、签名协议版本和操作人，不记录 `api_key` 或 `signing_secret`。
- 签名密钥解密失败计入现有安全日志和 Metrics 的认证失败计数，但日志只包含 request ID、Credential ID 的安全摘要，不包含原文。
- HMAC 不替代 TLS；生产仍必须使用 HTTPS，避免 Bearer Key 和签名密钥在链路中泄露。
- API Key 轮换同时生成新签名密钥，旧 Credential 的 API Key 和签名密钥一起失效。

## 测试验收标准

- 创建/轮换只返回一次签名密钥，数据库和审计快照只保存密文/前缀。
- 正确签名请求成功；篡改 Body、Method、Path、Timestamp、Nonce 或 Signature 均失败。
- 签名比较为 constant-time，未知版本和错误长度不会触发 panic。
- 新 Credential 缺少签名时被拒绝；历史无签名 Credential 在兼容模式下仍可用。
- `AGENT_SIGNATURE_REQUIRED=true` 时历史 Credential 被拒绝，缺少加密配置时启动失败。
- 签名密钥解密失败 fail-closed，返回稳定 503，不消费 Nonce，不写 Trace/Span/Outbox。
- 轮换事务同时保存 API Key Hash、签名密文和审计记录；任一失败全部回滚。
- 真实 MySQL 迁移、真实 API 摄入、Redis Nonce Store 和旧 Credential 兼容测试通过。
- 全量 Go 测试、`go vet`、安全扫描和前端 Agent Key 一次性展示回归通过。

## 非目标

- 本阶段不实现 Python/Node SDK，只冻结协议并提供 Go 客户端辅助函数的接口设计；SDK 迁移作为后续 P1 子任务。
- 本阶段不修改查询认证授权，不把 HMAC Header 加到 Trace 查询接口。
- 本阶段不支持多版本同时签名验证；协议版本 v2 需单独设计和迁移。
