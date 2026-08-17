"use client";

import { useEffect, useState } from "react";
import { APIError, listTraces } from "../../../lib/api/client";
import type { TraceSummary } from "../../../lib/api/types";
import { TraceTable } from "../../../components/traces/trace-table";

export default function TracesPage() {
  const [traces, setTraces] = useState<TraceSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    listTraces()
      .then((result) => setTraces(result.data))
      .catch((cause: unknown) => setError(cause instanceof APIError ? cause.message : "Unable to load traces"))
      .finally(() => setLoading(false));
  }, []);

  return (
    <section>
      <h1 className="page-title">Agent traces</h1>
      <p className="page-description">Inspect execution chains, cost, latency, and risk signals from production Agents.</p>
      {loading && <div className="panel">Loading traces…</div>}
      {!loading && error && <div role="alert" className="error-state">{error}</div>}
      {!loading && !error && <div className="panel"><TraceTable traces={traces} /></div>}
    </section>
  );
}
