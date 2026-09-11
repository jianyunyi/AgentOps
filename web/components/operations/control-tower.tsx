"use client";

import Link from "next/link";
import { useEffect, useMemo, useState } from "react";
import { getCurrentUser } from "../../lib/api/auth";
import { getOverviewData, type OverviewData } from "../../lib/api/overview";
import { listPolicies } from "../../lib/api/policies";
import { hasPermission } from "../../lib/permissions";
import { StateView } from "../ui/state-view";
import { buildControlTowerModel } from "./control-tower-model";
import { buildPolicyPosture } from "./policy-posture-model";
import { PolicyPosture, type PolicyPostureState } from "./policy-posture";
import { SectorTopology } from "./sector-topology";

type LoadState =
  | { kind: "loading" }
  | { kind: "forbidden" }
  | { kind: "error" }
  | { kind: "ready"; data: OverviewData };

function isEmpty(data: OverviewData) {
  return data.agents.length === 0 && data.traces.length === 0 && data.risks.length === 0;
}

function formatDuration(durationMs: number) {
  return `${durationMs} ms`;
}

export function ControlTower() {
  const [requestVersion, setRequestVersion] = useState(0);
  const [policyRequestVersion, setPolicyRequestVersion] = useState(0);
  const [loadState, setLoadState] = useState<LoadState>({ kind: "loading" });
  const [canReadPolicies, setCanReadPolicies] = useState(false);
  const [policyState, setPolicyState] = useState<PolicyPostureState>({ kind: "loading" });
  const [selectedId, setSelectedId] = useState<string>();

  useEffect(() => {
    let active = true;
    setLoadState({ kind: "loading" });
    setCanReadPolicies(false);
    setSelectedId(undefined);

    async function load() {
      try {
        const user = await getCurrentUser();
        if (!active) return;

        if (!hasPermission(user, "agent:read") || !hasPermission(user, "risk:read")) {
          setLoadState({ kind: "forbidden" });
          return;
        }

        setCanReadPolicies(hasPermission(user, "policy:read"));
        const data = await getOverviewData();
        if (active) setLoadState({ kind: "ready", data });
      } catch {
        if (active) setLoadState({ kind: "error" });
      }
    }

    void load();
    return () => { active = false; };
  }, [requestVersion]);

  useEffect(() => {
    if (!canReadPolicies) return;

    let active = true;
    setPolicyState({ kind: "loading" });

    void listPolicies()
      .then((policies) => {
        if (active) setPolicyState({ kind: "ready", posture: buildPolicyPosture(policies) });
      })
      .catch(() => {
        if (active) setPolicyState({ kind: "error", onRetry: () => setPolicyRequestVersion((version) => version + 1) });
      });

    return () => { active = false; };
  }, [canReadPolicies, policyRequestVersion]);

  const model = useMemo(
    () => loadState.kind === "ready" ? buildControlTowerModel(loadState.data) : undefined,
    [loadState],
  );

  if (loadState.kind === "loading") {
    return <StateView kind="loading" title="Reading live sector" />;
  }

  if (loadState.kind === "forbidden") {
    return <StateView kind="forbidden" title="Overview access is restricted" />;
  }

  if (loadState.kind === "error") {
    return <StateView kind="error" title="The console could not load this workspace snapshot." action={{ label: "Retry", onClick: () => setRequestVersion((version) => version + 1) }} />;
  }

  if (isEmpty(loadState.data)) {
    return <StateView kind="empty" title="No live signals yet" />;
  }

  if (!model) {
    return null;
  }

  const currentSelectedId = selectedId ?? model.nodes[0]?.id;
  const selectedAgent = model.nodes.find((node) => node.id === currentSelectedId);

  return (
    <section className="control-tower" aria-label="Agent operations control tower">
      <header className="tower-status">
        <div>
          <span className="tower-live">Live workspace data</span>
          <h1>Autonomous traffic control</h1>
        </div>
        <div className="tower-kpis" aria-label="Workspace summary">
          <div className="tower-kpi"><span>Registry status</span><strong>{model.registry.active} / {model.registry.total} active</strong><small>Loaded Agent records</small></div>
          <div className="tower-kpi tower-kpi--risk"><span>Active Risk</span><strong>{model.activeRisk.critical} critical</strong><small>{model.activeRisk.total} open in loaded results</small></div>
        </div>
      </header>

      <div className="tower-grid">
        <section className="tower-sector" aria-labelledby="tower-sector-title">
          <div className="tower-sector-header">
            <div><span className="eyebrow">Live topology</span><h2 id="tower-sector-title">Agent sector</h2></div>
            <p>Up to 12 registered Agents</p>
          </div>
          <SectorTopology nodes={model.nodes} routes={model.routes} selectedId={currentSelectedId} onSelect={setSelectedId} />
        </section>

        <aside className="tower-risk-queue" aria-labelledby="tower-risk-title">
          <div className="tower-panel-heading"><div><span className="eyebrow">Needs attention</span><h2 id="tower-risk-title">Active Risk</h2></div><Link href="/dashboard/risk">View all</Link></div>
          {model.risks.length === 0 ? <p className="tower-empty">No open risk events in loaded results.</p> : (
            <ul>
              {model.risks.slice(0, 3).map((risk) => (
                <li key={risk.id}><Link href="/dashboard/risk">{risk.reason}</Link><span className="tower-risk-meta"><span className="risk-severity">{risk.risk_level}</span><span>{risk.detector}</span></span></li>
              ))}
            </ul>
          )}
        </aside>
      </div>

      <div className="tower-bottom">
        <section className="tower-traces" aria-labelledby="tower-traces-title">
          <div className="tower-panel-heading"><div><span className="eyebrow">Evidence stream</span><h2 id="tower-traces-title">Recent Traces</h2></div><Link href="/dashboard/traces">Explore traces</Link></div>
          {model.traces.length === 0 ? <p className="tower-empty">No recent traces in loaded results.</p> : (
            <table className="trace-table">
              <thead><tr><th scope="col">Trace id</th><th scope="col">Agent</th><th scope="col">Status</th><th scope="col">Duration</th><th scope="col">Risk level</th></tr></thead>
              <tbody>{model.traces.map((trace) => <tr key={trace.traceId}><td><Link href={`/dashboard/traces/${encodeURIComponent(trace.traceId)}`}>{trace.traceId}</Link></td><td>{trace.agentName}</td><td>{trace.status}</td><td>{formatDuration(trace.durationMs)}</td><td>{trace.riskLevel}</td></tr>)}</tbody>
            </table>
          )}
        </section>

        <section className="tower-evidence" aria-labelledby="tower-evidence-title">
          <span className="eyebrow">Selected signal</span><h2 id="tower-evidence-title">Sector Evidence</h2>
          {selectedAgent ? <dl><dt>Agent</dt><dd>{selectedAgent.label}</dd><dt>Callsign</dt><dd>{selectedAgent.callsign}</dd><dt>State</dt><dd>{selectedAgent.tone}</dd></dl> : <p className="tower-empty">No Agent node is available in loaded results.</p>}
          <p>Routes visualize loaded Trace activity and do not assert Agent-to-Agent calls.</p>
        </section>
        {canReadPolicies ? <PolicyPosture state={policyState} /> : null}
      </div>
    </section>
  );
}
