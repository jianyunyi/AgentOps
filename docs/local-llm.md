# Local open-source model deployment

AgentScope uses the OpenAI-compatible chat completion protocol. The same worker can call Ollama, vLLM, or LM Studio; raw prompt content is still redacted and bounded by the tenant policy before it leaves the worker process.

## Ollama with Docker Compose

Start the production-shaped stack with the local LLM override:

```powershell
$env:MYSQL_PASSWORD = "..."
$env:MYSQL_ROOT_PASSWORD = "..."
$env:SESSION_SECRET = "replace-with-at-least-32-random-characters"
$env:LLM_MODEL = "qwen2.5:7b"
docker compose -f docker-compose.prod.yml -f docker-compose.ollama.yml --profile local-llm up -d
docker compose -f docker-compose.prod.yml -f docker-compose.ollama.yml exec ollama ollama pull $env:LLM_MODEL
```

The worker calls `http://ollama:11434/v1/chat/completions`. Ollama does not require a real API key; `ollama` is used as a non-secret compatibility value. For larger models, allocate GPU nodes and use vLLM with an OpenAI-compatible endpoint instead.

## Production safeguards

- Keep `LLM_ENABLED=false` until the local model health check is green.
- Use `MaxInputBytes` policies to cap prompt size and cost.
- Monitor model latency, timeout rate, and fallback-to-rules count.
- Keep the deterministic rules enabled; an unavailable or malformed local model response must never block ingestion.
- Do not expose Ollama's port publicly; only the worker network should reach it.
