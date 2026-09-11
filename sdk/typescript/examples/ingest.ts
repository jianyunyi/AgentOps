import { AgentOpsClient } from "../src/index.js";

const baseUrl = process.env.AGENTOPS_BASE_URL;
const apiKey = process.env.AGENT_API_KEY;
const signingSecret = process.env.AGENT_SIGNING_SECRET;
if (!baseUrl || !apiKey || !signingSecret) {
  throw new Error("AGENTOPS_BASE_URL, AGENT_API_KEY, and AGENT_SIGNING_SECRET are required");
}

const client = new AgentOpsClient({ baseUrl, apiKey, signingSecret });
const result = await client.ingest({
  event_id: `sdk_example_${Date.now()}`,
  trace_id: `trace_example_${Date.now()}`,
  span_id: `span_example_${Date.now()}`,
  event_type: "llm_call",
  occurred_at: new Date().toISOString(),
  payload: { message: "hello from the AgentOps TypeScript SDK" },
});

console.log(JSON.stringify({ duplicate: result.duplicate }));
