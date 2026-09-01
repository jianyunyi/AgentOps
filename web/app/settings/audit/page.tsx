"use client";

import { useEffect, useState } from "react";
import { APIError } from "../../../lib/api/client";
import { listAuditLogs } from "../../../lib/api/governance";
import type { AuditRecord } from "../../../lib/api/types";

export default function AuditPage() {
  const [records, setRecords] = useState<AuditRecord[]>([]); const [error, setError] = useState<string | null>(null);
  useEffect(() => { listAuditLogs().then((result) => setRecords(result.data)).catch((err) => setError(err instanceof APIError ? err.message : "Unable to load audit logs")); }, []);
  return <section><h1 className="page-title">Audit logs</h1><p className="page-description">Immutable tenant activity records.</p>{error && <div role="alert" className="error-state">{error}</div>}<div className="panel"><table className="trace-table"><thead><tr><th>Action</th><th>Resource</th><th>Actor</th><th>Time</th></tr></thead><tbody>{records.map((record) => <tr key={record.id}><td>{record.action}</td><td>{record.resource_type}/{record.resource_id}</td><td>{record.actor_id}</td><td>{record.created_at}</td></tr>)}</tbody></table></div></section>;
}
