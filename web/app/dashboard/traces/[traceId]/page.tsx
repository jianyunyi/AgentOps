"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { APIError, getTrace } from "../../../../lib/api/client";
import type { TraceDetail } from "../../../../lib/api/types";
import { SpanTree } from "../../../../components/traces/span-tree";

export default function TraceDetailPage() {
  const params = useParams<{ traceId: string }>();
  const [trace, setTrace] = useState<TraceDetail | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    getTrace(params.traceId)
      .then(setTrace)
      .catch((cause: unknown) => setError(cause instanceof APIError ? cause.message : "Unable to load trace"));
  }, [params.traceId]);

  if (error) return <div role="alert" className="error-state">{error}</div>;
  if (!trace) return <div className="panel">Loading trace…</div>;

  return (
    <section>
      <h1 className="page-title">Trace {trace.traceId}</h1>
      <p className="page-description">{trace.agentName} · {trace.status} · {trace.riskLevel} · {trace.durationMs} ms</p>
      <div className="panel" style={{ padding: 24 }}>
        <h2>Execution timeline</h2>
        <SpanTree spans={trace.spans} />
      </div>
    </section>
  );
}
