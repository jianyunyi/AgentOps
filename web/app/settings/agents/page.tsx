"use client";
import { useEffect, useState } from "react";
import { createAgent, listAgents, revokeAgentKey, rotateAgentKey } from "../../../lib/api/agents";
import type { Agent } from "../../../lib/api/types";

export default function AgentsPage() {
  const [agents, setAgents] = useState<Agent[]>([]); const [newKey, setNewKey] = useState<string | null>(null); const [name, setName] = useState("");
  useEffect(() => { listAgents().then(setAgents).catch(() => setAgents([])); }, []);
  async function create() { if (!name.trim()) return; const result = await createAgent({ name, description: "", environment: "production" }); setAgents((current) => [result.agent, ...current]); setNewKey(result.api_key); setName(""); }
  async function rotate(agent: Agent) { const result = await rotateAgentKey(agent.id); setNewKey(result.api_key); }
  async function revoke(agent: Agent) { await revokeAgentKey(agent.id); setAgents((current) => current.map((item) => item.id === agent.id ? { ...item, status: "credential_revoked" } : item)); }
  return <section><h1 className="page-title">Agents</h1><p className="page-description">Manage production Agent identities and credentials.</p><div className="panel agent-form"><input aria-label="Agent name" placeholder="Agent name" value={name} onChange={(e) => setName(e.target.value)} /><button onClick={create}>Create Agent</button></div>{newKey && <div role="alert" className="key-reveal"><strong>Save this API key now:</strong><code>{newKey}</code><button onClick={() => setNewKey(null)}>Hide</button></div>}<div className="panel"><table className="trace-table"><thead><tr><th>Name</th><th>Environment</th><th>Status</th><th>Actions</th></tr></thead><tbody>{agents.map((agent) => <tr key={agent.id}><td>{agent.name}</td><td>{agent.environment}</td><td>{agent.status}</td><td><button onClick={() => rotate(agent)}>Rotate key</button> <button onClick={() => revoke(agent)}>Revoke key</button></td></tr>)}</tbody></table></div></section>;
}
