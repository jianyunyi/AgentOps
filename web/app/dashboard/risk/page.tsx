"use client";

import { useEffect, useState } from "react";
import { APIError } from "../../../lib/api/client";
import { listRiskEvents, reviewRiskEvent } from "../../../lib/api/governance";
import type { RiskEvent } from "../../../lib/api/types";

export default function RiskPage() {
  const [events, setEvents] = useState<RiskEvent[]>([]); const [error, setError] = useState<string | null>(null); const [busy, setBusy] = useState<string | null>(null);
  useEffect(() => { listRiskEvents().then((result) => setEvents(result.data)).catch((err) => setError(err instanceof APIError ? err.message : "Unable to load risk events")); }, []);
  async function review(event: RiskEvent, status: string) { setBusy(event.id); setError(null); try { await reviewRiskEvent(event.id, status); setEvents((current) => current.map((item) => item.id === event.id ? { ...item, status } : item)); } catch (err) { setError(err instanceof APIError ? err.message : "Unable to review risk event"); } finally { setBusy(null); } }
  return <section><h1 className="page-title">Risk events</h1><p className="page-description">Review deterministic and LLM-assisted safety findings.</p>{error && <div role="alert" className="error-state">{error}</div>}<div className="panel"><table className="trace-table"><thead><tr><th>Level</th><th>Rule</th><th>Reason</th><th>Status</th><th>Action</th></tr></thead><tbody>{events.map((event) => <tr key={event.id}><td>{event.risk_level}</td><td>{event.rule_code}</td><td>{event.reason}</td><td>{event.status}</td><td><button disabled={busy === event.id} onClick={() => review(event, "acknowledged")}>Acknowledge</button> <button disabled={busy === event.id} onClick={() => review(event, "resolved")}>Resolve</button></td></tr>)}</tbody></table></div></section>;
}
