"use client";

import { useEffect, useState } from "react";
import { APIError } from "../../../lib/api/client";
import { changeMemberRole, disableMember, listMembers } from "../../../lib/api/members";
import type { Member } from "../../../lib/api/types";

const roles = ["admin", "developer", "auditor", "viewer"];

export default function MembersPage() {
  const [members, setMembers] = useState<Member[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState<string | null>(null);
  const load = () => listMembers().then(setMembers).catch((err) => setError(err instanceof APIError ? err.message : "Unable to load members"));
  useEffect(() => { load(); }, []);
  async function update(id: string, role: string) {
    setBusy(id); setError(null);
    try { await changeMemberRole(id, role); await load(); } catch (err) { setError(err instanceof APIError ? err.message : "Unable to update member"); } finally { setBusy(null); }
  }
  async function disable(id: string) {
    if (!window.confirm("Disable this member's access?")) return;
    setBusy(id); setError(null);
    try { await disableMember(id); await load(); } catch (err) { setError(err instanceof APIError ? err.message : "Unable to disable member"); } finally { setBusy(null); }
  }
  return <section><h1 className="page-title">Members</h1><p className="page-description">Manage tenant access without exposing credentials.</p>{error && <div role="alert" className="error-state">{error}</div>}<div className="panel"><table className="trace-table"><thead><tr><th>Email</th><th>Role</th><th>Status</th><th>Actions</th></tr></thead><tbody>{members.map((member) => <tr key={member.id}><td>{member.email}</td><td>{member.role === "owner" ? <span>owner</span> : <select aria-label={`Role for ${member.email}`} value={member.role} disabled={busy === member.id} onChange={(event) => update(member.id, event.target.value)}>{roles.map((role) => <option key={role}>{role}</option>)}</select>}</td><td>{member.status}</td><td>{member.role !== "owner" && member.status === "active" && <button disabled={busy === member.id} onClick={() => disable(member.id)}>Disable</button>}</td></tr>)}</tbody></table></div></section>;
}
