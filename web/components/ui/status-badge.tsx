import type { HTMLAttributes } from "react";

export type StatusDomain = "trace" | "risk" | "agent" | "member" | "policy";
type StatusTone = "healthy" | "warning" | "danger" | "info" | "muted";

const toneByDomain: Record<StatusDomain, Record<string, StatusTone>> = {
  trace: { completed: "healthy", success: "healthy", ok: "healthy", running: "info", pending: "warning", queued: "warning", failed: "danger", error: "danger" },
  risk: { critical: "danger", high: "danger", open: "danger", escalated: "danger", blocked: "danger", medium: "warning", warning: "warning", at_risk: "warning", low: "healthy", clear: "healthy", resolved: "healthy" },
  agent: { active: "healthy", healthy: "healthy", online: "healthy", ready: "healthy", pending: "warning", credential_pending: "warning", disabled: "danger", revoked: "danger", credential_revoked: "danger", failed: "danger", error: "danger" },
  member: { active: "healthy", invited: "info", pending: "warning", suspended: "danger", disabled: "danger" },
  policy: { active: "healthy", enforced: "healthy", enabled: "healthy", draft: "warning", pending: "warning", disabled: "muted", archived: "muted" },
};

export function StatusBadge({ domain, status, ...props }: { domain: StatusDomain; status: string } & Omit<HTMLAttributes<HTMLSpanElement>, "children">) {
  const tone = toneByDomain[domain][status.trim().toLowerCase()] ?? "muted";

  return (
    <span {...props} className={`status-badge status-badge--${tone}${props.className ? ` ${props.className}` : ""}`} data-tone={tone}>
      {status}
    </span>
  );
}
