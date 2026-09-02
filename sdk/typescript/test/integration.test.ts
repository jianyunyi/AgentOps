import { describe, expect, it } from "vitest";
import { AgentOpsClient } from "../src/client.js";

const enabled = process.env.AGENTSCOPE_INTEGRATION === "1";
const baseUrl = process.env.AGENTOPS_SDK_BASE_URL;
const apiKey = process.env.AGENT_API_KEY;
const signingSecret = process.env.AGENT_SIGNING_SECRET;

describe("AgentOps TypeScript SDK integration", () => {
  it.skipIf(!enabled || !baseUrl || !apiKey || !signingSecret)("ingests a signed event through the configured API", async () => {
    const client = new AgentOpsClient({ baseUrl: baseUrl!, apiKey: apiKey!, signingSecret: signingSecret! });
    const stamp = `${Date.now()}_${Math.random().toString(16).slice(2)}`;
    await expect(client.ingest({ event_id: `sdk_it_${stamp}`, trace_id: `trace_it_${stamp}`, span_id: `span_it_${stamp}`, event_type: "llm_call", occurred_at: new Date().toISOString(), payload: { source: "typescript-sdk-integration" } })).resolves.toEqual({ duplicate: false });
  });
});
