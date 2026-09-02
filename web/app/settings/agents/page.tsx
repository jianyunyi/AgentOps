"use client";

import { useEffect, useState } from "react";
import { APIError } from "../../../lib/api/client";
import { getCurrentUser } from "../../../lib/api/auth";
import { createAgent, listAgents, revokeAgentKey, rotateAgentKey } from "../../../lib/api/agents";
import type { Agent, CurrentUser } from "../../../lib/api/types";
import { hasPermission } from "../../../lib/permissions";

const writePermission = "agent:write";
function errorMessage(error: unknown): string { return error instanceof APIError ? error.message : "Request failed. Please retry."; }

export default function AgentsPage() {
  const [agents, setAgents] = useState<Agent[]>([]); const [user, setUser] = useState<CurrentUser | null>(null);
  const [error, setError] = useState<string | null>(null); const [busy, setBusy] = useState<string | null>(null);
  const [newCredential, setNewCredential] = useState<{ api_key: string; signing_secret: string } | null>(null); const [name, setName] = useState("");
  const canWrite = hasPermission(user, writePermission);
  useEffect(() => { Promise.all([getCurrentUser(), listAgents()]).then(([currentUser, list]) => { setUser(currentUser); setAgents(list); }).catch((err) => setError(errorMessage(err))); }, []);
  async function create() { if (!canWrite || !name.trim()) return; setBusy("create"); setError(null); try { const result = await createAgent({ name, description: "", environment: "production" }); setAgents((current) => [result.agent, ...current]); setNewCredential({ api_key: result.api_key, signing_secret: result.signing_secret }); setName(""); } catch (err) { setError(errorMessage(err)); } finally { setBusy(null); } }
  async function rotate(agent: Agent) { if (!canWrite) return; setBusy(agent.id); setError(null); try { const result = await rotateAgentKey(agent.id); setNewCredential({ api_key: result.api_key, signing_secret: result.signing_secret }); } catch (err) { setError(errorMessage(err)); } finally { setBusy(null); } }
  async function revoke(agent: Agent) { if (!canWrite) return; setBusy(agent.id); setError(null); try { await revokeAgentKey(agent.id); setAgents((current) => current.map((item) => item.id === agent.id ? { ...item, status: "credential_revoked" } : item)); } catch (err) { setError(errorMessage(err)); } finally { setBusy(null); } }
  return <section><h1 className="page-title">Agents</h1><p className="page-description">Manage production Agent identities and credentials.</p>{user && <p role="status">Signed in as <strong>{user.role}</strong>. {canWrite ? "You can manage credentials." : "Read-only access."}</p>}{error && <div role="alert" className="error-banner">{error}</div>}{canWrite && <div className="panel agent-form"><input aria-label="Agent name" placeholder="Agent name" value={name} onChange={(e) => setName(e.target.value)} /><button disabled={busy === "create"} onClick={create}>{busy === "create" ? "Creating…" : "Create Agent"}</button></div>}{newCredential && <div role="alert" className="key-reveal"><strong>Save these credentials now. They will not be shown again:</strong><p>API key</p><code>{newCredential.api_key}</code><p>Signing secret</p><code>{newCredential.signing_secret}</code><button onClick={() => setNewCredential(null)}>Hide</button></div>}<div className="panel"><table className="trace-table"><thead><tr><th>Name</th><th>Environment</th><th>Status</th>{canWrite && <th>Actions</th>}</tr></thead><tbody>{agents.map((agent) => <tr key={agent.id}><td>{agent.name}</td><td>{agent.environment}</td><td>{agent.status}</td>{canWrite && <td><button disabled={busy === agent.id} onClick={() => rotate(agent)}>Rotate key</button> <button disabled={busy === agent.id} onClick={() => revoke(agent)}>Revoke key</button></td>}</tr>)}</tbody></table></div></section>;
}
