"use client";

import { useEffect, useState, type FormEvent } from "react";
import { APIError } from "../../../lib/api/client";
import { activatePolicy, createPolicy, listPolicies } from "../../../lib/api/policies";
import type { Policy } from "../../../lib/api/types";

export default function PoliciesPage() {
  const [policies, setPolicies] = useState<Policy[]>([]); const [error, setError] = useState<string | null>(null); const [busy, setBusy] = useState(false); const [name, setName] = useState(""); const [llm, setLlm] = useState(false); const [rules, setRules] = useState(true); const [maxInput, setMaxInput] = useState(65536);
  const load = () => listPolicies().then(setPolicies).catch((err) => setError(err instanceof APIError ? err.message : "Unable to load policies"));
  useEffect(() => { load(); }, []);
  async function submit(event: FormEvent) { event.preventDefault(); setBusy(true); setError(null); try { await createPolicy({ name, rules_enabled: rules, llm_enabled: llm, max_input_bytes: maxInput }); setName(""); await load(); } catch (err) { setError(err instanceof APIError ? err.message : "Unable to create policy"); } finally { setBusy(false); } }
  async function activate(id: string) { setBusy(true); setError(null); try { await activatePolicy(id); await load(); } catch (err) { setError(err instanceof APIError ? err.message : "Unable to activate policy"); } finally { setBusy(false); } }
  return <section><h1 className="page-title">AI policies</h1><p className="page-description">Versioned tenant controls for deterministic rules, LLM analysis, and input boundaries.</p>{error && <div role="alert" className="error-state">{error}</div>}<div className="panel"><form className="agent-form" onSubmit={submit}><input aria-label="Policy name" placeholder="Policy name" value={name} onChange={(event) => setName(event.target.value)} required /><label><input type="checkbox" checked={rules} onChange={(event) => setRules(event.target.checked)} /> Rules</label><label><input type="checkbox" checked={llm} onChange={(event) => setLlm(event.target.checked)} /> LLM analysis</label><label>Max bytes<input type="number" min={1024} max={1048576} value={maxInput} onChange={(event) => setMaxInput(Number(event.target.value))} /></label><button disabled={busy} type="submit">Create version</button></form><table className="trace-table"><thead><tr><th>Version</th><th>Name</th><th>Rules</th><th>LLM</th><th>Input limit</th><th>Status</th><th /></tr></thead><tbody>{policies.map((policy) => <tr key={policy.id}><td>v{policy.version}</td><td>{policy.name}</td><td>{policy.rules_enabled ? "on" : "off"}</td><td>{policy.llm_enabled ? "on" : "off"}</td><td>{policy.max_input_bytes}</td><td>{policy.enabled ? "active" : "draft"}</td><td>{!policy.enabled && <button disabled={busy} onClick={() => activate(policy.id)}>Activate</button>}</td></tr>)}</tbody></table></div></section>;
}
