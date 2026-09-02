# AgentScope Go SDK

The Go SDK signs and ingests AgentScope events using the server's HMAC v1 protocol. It has no third-party runtime dependencies.

## Install

```bash
go get github.com/jianyunyi/AgentOps/sdk/go
```

## Minimal usage

Load both credentials from a secret manager or environment variables. Do not commit them, print them, or place them in request logs.

```go
client, err := agentops.NewClient(agentops.Config{
    BaseURL:       os.Getenv("AGENTOPS_BASE_URL"),
    APIKey:        os.Getenv("AGENT_API_KEY"),
    SigningSecret: os.Getenv("AGENT_SIGNING_SECRET"),
})
if err != nil {
    return err
}

result, err := client.Ingest(ctx, agentops.Event{
    EventID:    "stable-business-event-id",
    TraceID:    "trace-001",
    SpanID:     "span-001",
    EventType:  "llm_call",
    OccurredAt: time.Now().UTC(),
    Payload:    json.RawMessage(`{"input":"hello"}`),
})
```

`EventID` must remain stable when the caller retries an operation. The SDK serializes the Body once, signs the exact bytes, and automatically generates a fresh Timestamp, Nonce, and HMAC signature for each retry. By default it makes at most three attempts and retries only transport failures, 408, 429, 500, 502, 503, and 504. It never retries 401 or 409 replay responses.

## Credential and rollout rules

The Agent management response returns `api_key` and `signing_secret` once. Store both in a secret manager immediately. The SDK cannot recover either value later.

For a zero-downtime rollout, keep `AGENT_SIGNATURE_REQUIRED=false` while legacy Agents are rotated. After every active Agent has a new signing secret and all callers have deployed the SDK, set `AGENT_SIGNATURE_REQUIRED=true`.

The SDK sends:

```text
Authorization: Bearer <api_key>
X-Agent-Timestamp: <unix seconds>
X-Agent-Nonce: <fresh printable nonce>
X-Agent-Signature: v1=<lowercase HMAC-SHA256 hex>
```

The signed canonical request is:

```text
v1\nPOST\n/api/v1/ingest/events\n<timestamp>\n<nonce>\n<body_sha256>
```

Use HTTPS in production. HMAC does not replace TLS.
