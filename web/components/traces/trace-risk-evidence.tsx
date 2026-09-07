"use client";

import { useState } from "react";
import type { RiskEvent } from "../../lib/api/types";
import { StatusBadge } from "../ui/status-badge";

export type TraceRiskState =
  | { kind: "loading" }
  | { kind: "error"; onRetry: () => void }
  | { kind: "ready"; risks: RiskEvent[] };

export type TraceRiskEvidenceProps = {
  state: TraceRiskState;
  canReview: boolean;
  onReview: (id: string, status: "acknowledged" | "resolved") => Promise<void>;
};

const formattedCreatedAt = (value: string) => {
  const date = new Date(value);
  return Number.isNaN(date.getTime()) ? "Unavailable" : date.toISOString();
};

export const TraceRiskEvidence = ({ state, canReview, onReview }: TraceRiskEvidenceProps) => {
  const [busyId, setBusyId] = useState<string | null>(null);
  const [reviewError, setReviewError] = useState(false);

  const review = async (id: string, status: "acknowledged" | "resolved") => {
    setBusyId(id);
    setReviewError(false);

    try {
      await onReview(id, status);
    } catch {
      setReviewError(true);
    } finally {
      setBusyId(null);
    }
  };

  return (
    <aside aria-labelledby="trace-risk-heading">
      <h2 id="trace-risk-heading">Risk evidence</h2>

      {state.kind === "loading" ? <p role="status">Loading risk evidence</p> : null}

      {state.kind === "error" ? (
        <div role="alert">
          <p>Risk evidence is temporarily unavailable.</p>
          <button type="button" onClick={state.onRetry}>Retry risk evidence</button>
        </div>
      ) : null}

      {state.kind === "ready" && state.risks.length === 0 ? <p>No associated risk events in loaded results.</p> : null}

      {state.kind === "ready" ? (
        <div>
          {reviewError ? <p role="alert">Unable to update risk event</p> : null}
          {state.risks.map((risk) => {
            const isBusy = busyId === risk.id;

            return (
              <article key={risk.id}>
                <h3>{risk.id}</h3>
                <p><StatusBadge domain="risk" status={risk.risk_level} /> <StatusBadge domain="risk" status={risk.status} /></p>
                <dl>
                  <div><dt>Rule</dt><dd>{risk.rule_code}</dd></div>
                  <div><dt>Detector</dt><dd>{risk.detector}</dd></div>
                  <div><dt>Reason</dt><dd>{risk.reason}</dd></div>
                  <div><dt>Redacted evidence</dt><dd>{risk.evidence_redacted}</dd></div>
                  <div><dt>Created</dt><dd>{formattedCreatedAt(risk.created_at)}</dd></div>
                </dl>
                {canReview ? (
                  <p>
                    <button type="button" disabled={isBusy} onClick={() => void review(risk.id, "acknowledged")}>Acknowledge {risk.id}</button>
                    <button type="button" disabled={isBusy} onClick={() => void review(risk.id, "resolved")}>Resolve {risk.id}</button>
                  </p>
                ) : null}
              </article>
            );
          })}
        </div>
      ) : null}
    </aside>
  );
};
