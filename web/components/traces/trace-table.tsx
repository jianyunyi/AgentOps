import Link from "next/link";
import type { TraceSummary } from "../../lib/api/types";

function formatNumber(value: number): string {
  return new Intl.NumberFormat("en-US").format(value);
}

export function TraceTable({ traces }: { traces: TraceSummary[] }) {
  if (traces.length === 0) {
    return (
      <div role="status" className="empty-state">
        <strong>No traces found</strong>
        <p>Try changing the filters or wait for a new Agent event.</p>
      </div>
    );
  }

  return (
    <table className="trace-table">
      <thead>
        <tr>
          <th>Agent</th>
          <th>Status</th>
          <th>Risk</th>
          <th>Duration</th>
          <th>Tokens</th>
          <th>Cost</th>
          <th>Started</th>
        </tr>
      </thead>
      <tbody>
        {traces.map((trace) => (
          <tr key={trace.traceId}>
            <td><Link href={`/dashboard/traces/${trace.traceId}`}>{trace.agentName}</Link></td>
            <td>{trace.status}</td>
            <td>{trace.riskLevel}</td>
            <td>{trace.durationMs} ms</td>
            <td>{formatNumber(trace.totalTokens)}</td>
            <td>${trace.estimatedCost.toFixed(4)}</td>
            <td>{new Date(trace.startedAt).toLocaleString()}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
