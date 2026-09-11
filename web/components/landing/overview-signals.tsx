"use client";

import { useEffect, useState } from "react";
import { getCurrentUser } from "../../lib/api/auth";
import { getOverviewData, type OverviewData } from "../../lib/api/overview";
import { hasPermission } from "../../lib/permissions";
import { StateView } from "../ui/state-view";
import { StatusBadge } from "../ui/status-badge";

function errorText(error: unknown) {
  return error instanceof Error ? error.message : "Unable to load live signals.";
}

export function OverviewSignals() {
  const [data, setData] = useState<OverviewData | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let active = true;
    getCurrentUser()
      .then((user) => {
        if (!active) return;
        if (!hasPermission(user, "agent:read") || !hasPermission(user, "risk:read")) {
          setError("Overview access is restricted");
          return;
        }
        return getOverviewData().then((overview) => { if (active) setData(overview); });
      })
      .catch((cause: unknown) => { if (active) setError(errorText(cause)); })
      .finally(() => { if (active) setLoading(false); });
    return () => { active = false; };
  }, []);

  if (loading) return <StateView kind="loading" title="Reading live signals" description="Loading the latest Agent, Trace, and Risk state." />;
  if (error) return <StateView kind="error" title={error} description="The console could not load this workspace snapshot." />;
  if (!data || (data.agents.length === 0 && data.traces.length === 0 && data.risks.length === 0)) {
    return <StateView kind="empty" title="No live signals yet" description="Connect an Agent to start building an operational picture." />;
  }

  const activeAgents = data.agents.filter((agent) => agent.status === "active").length;
  return (
    <section className="overview-signals" aria-labelledby="signals-title">
      <div className="section-heading"><div><span className="eyebrow">Operational pulse</span><h2 id="signals-title">What is happening now</h2></div><span className="section-note">Backed by live workspace data</span></div>
      <div className="signal-grid">
        <div className="signal-card"><span>Active Agents</span><strong>{activeAgents} active Agent{activeAgents === 1 ? "" : "s"}</strong><StatusBadge domain="agent" status={activeAgents > 0 ? "active" : "disabled"} /></div>
        <div className="signal-card"><span>Open Risk</span><strong>{data.risks.length} open risk{data.risks.length === 1 ? "" : "s"}</strong><StatusBadge domain="risk" status={data.risks.length > 0 ? "open" : "none"} /></div>
        <div className="signal-card"><span>Recent Traces</span><strong>{data.traces.length} recent trace{data.traces.length === 1 ? "" : "s"}</strong><StatusBadge domain="trace" status={data.traces[0]?.status ?? "pending"} /></div>
      </div>
      <div className="signal-list">
        {data.traces.slice(0, 3).map((trace) => <div className="signal-row" key={trace.traceId}><span><strong>{trace.agentName}</strong><small>{new Date(trace.startedAt).toLocaleString()}</small></span><StatusBadge domain="trace" status={trace.status} /><span className="signal-value">{trace.durationMs} ms</span></div>)}
      </div>
    </section>
  );
}
