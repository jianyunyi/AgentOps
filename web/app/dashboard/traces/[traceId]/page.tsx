"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { APIError, getTrace } from "../../../../lib/api/client";
import { getCurrentUser } from "../../../../lib/api/auth";
import { listRiskEvents, reviewRiskEvent } from "../../../../lib/api/governance";
import type { RiskEvent, TraceDetail } from "../../../../lib/api/types";
import { hasPermission } from "../../../../lib/permissions";
import { SpanTree } from "../../../../components/traces/span-tree";
import { TraceRiskEvidence, type TraceRiskState } from "../../../../components/traces/trace-risk-evidence";
import { risksForTrace } from "../../../../components/traces/trace-risk-model";

export default function TraceDetailPage() {
  const params = useParams<{ traceId: string }>();
  const [trace, setTrace] = useState<TraceDetail | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [canReadRisks, setCanReadRisks] = useState(false);
  const [canReviewRisks, setCanReviewRisks] = useState(false);
  const [riskState, setRiskState] = useState<TraceRiskState>({ kind: "loading" });
  const [riskRequestVersion, setRiskRequestVersion] = useState(0);

  useEffect(() => {
    getTrace(params.traceId)
      .then(setTrace)
      .catch((cause: unknown) => setError(cause instanceof APIError ? cause.message : "Unable to load trace"));
  }, [params.traceId]);

  useEffect(() => {
    let cancelled = false;
    setCanReadRisks(false);
    setCanReviewRisks(false);
    setRiskState({ kind: "loading" });

    getCurrentUser()
      .then((user) => {
        if (cancelled) return;
        const canRead = hasPermission(user, "risk:read");
        setCanReadRisks(canRead);
        setCanReviewRisks(hasPermission(user, "risk:review"));
        if (!canRead) return;
        listRiskEvents()
          .then((page) => {
            if (!cancelled) setRiskState({ kind: "ready", risks: risksForTrace(page.data, params.traceId) });
          })
          .catch(() => {
            if (!cancelled) setRiskState({ kind: "error", onRetry: () => setRiskRequestVersion((value) => value + 1) });
          });
      })
      .catch(() => {
        if (!cancelled) setCanReadRisks(false);
      });

    return () => { cancelled = true; };
  }, [params.traceId, riskRequestVersion]);

  const reviewRisk = async (id: string, status: "acknowledged" | "resolved") => {
    try {
      await reviewRiskEvent(id, status);
      setRiskState((current) => current.kind === "ready"
        ? { kind: "ready", risks: current.risks.map((risk): RiskEvent => risk.id === id ? { ...risk, status } : risk) }
        : current);
    } catch {
      throw new Error("Unable to update risk event");
    }
  };

  if (error) return <div role="alert" className="error-state">{error}</div>;
  if (!trace) return <div className="panel">Loading trace…</div>;

  return (
    <section>
      <h1 className="page-title">Trace {trace.traceId}</h1>
      <p className="page-description">{trace.agentName} · {trace.status} · {trace.riskLevel} · {trace.durationMs} ms</p>
      <div className="trace-investigation">
        <div className="panel trace-timeline">
          <h2>Execution timeline</h2>
          <SpanTree spans={trace.spans} />
        </div>
        {canReadRisks ? <TraceRiskEvidence state={riskState} canReview={canReviewRisks} onReview={reviewRisk} /> : null}
      </div>
    </section>
  );
}
