"use client";

import { useEffect, useState, type FormEvent } from "react";
import { APIError } from "../../../lib/api/client";
import { changeMemberRole, createInvitation, disableMember, listMembers, transferOwner } from "../../../lib/api/members";
import { getCurrentUser } from "../../../lib/api/auth";
import type { CurrentUser, Member } from "../../../lib/api/types";

const roles = ["admin", "developer", "auditor", "viewer"];

export default function MembersPage() {
  const [members, setMembers] = useState<Member[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState<string | null>(null);
  const [currentUser, setCurrentUser] = useState<CurrentUser | null>(null);
  const [inviteEmail, setInviteEmail] = useState(""); const [inviteRole, setInviteRole] = useState("developer"); const [inviteToken, setInviteToken] = useState<string | null>(null);
  const canWrite = currentUser?.permissions.includes("member:write") ?? false;
  const load = () => listMembers().then(setMembers).catch((err) => setError(err instanceof APIError ? err.message : "Unable to load members"));
  useEffect(() => { getCurrentUser().then(setCurrentUser).then(load).catch((err) => setError(err instanceof APIError ? err.message : "Unable to load member permissions")); }, []);
  async function update(id: string, role: string) {
    setBusy(id); setError(null);
    try { await changeMemberRole(id, role); await load(); } catch (err) { setError(err instanceof APIError ? err.message : "Unable to update member"); } finally { setBusy(null); }
  }
  async function disable(id: string) {
    if (!window.confirm("Disable this member's access?")) return;
    setBusy(id); setError(null);
    try { await disableMember(id); await load(); } catch (err) { setError(err instanceof APIError ? err.message : "Unable to disable member"); } finally { setBusy(null); }
  }
  async function invite(event: FormEvent) { event.preventDefault(); setError(null); try { const result = await createInvitation(inviteEmail, inviteRole); setInviteToken(result.invite_token); setInviteEmail(""); } catch (err) { setError(err instanceof APIError ? err.message : "Unable to create invitation"); } }
  async function makeOwner(id: string) { if (!window.confirm("Transfer tenant ownership to this member?")) return; setBusy(id); try { await transferOwner(id); await load(); } catch (err) { setError(err instanceof APIError ? err.message : "Unable to transfer ownership"); } finally { setBusy(null); } }
  if (currentUser && !currentUser.permissions.includes("member:read")) return <section><h1 className="page-title">Members</h1><div role="alert" className="error-state">You do not have permission to view members.</div></section>;
  return <section><h1 className="page-title">Members</h1><p className="page-description">Manage tenant access without exposing credentials.</p>{error && <div role="alert" className="error-state">{error}</div>}{canWrite && <div className="panel"><h2>Invite member</h2><form onSubmit={invite}><input aria-label="Invite email" value={inviteEmail} onChange={(event) => setInviteEmail(event.target.value)} type="email" placeholder="member@example.com" required /><select aria-label="Invite role" value={inviteRole} onChange={(event) => setInviteRole(event.target.value)}>{roles.map((role) => <option key={role}>{role}</option>)}</select><button type="submit">Create invitation</button></form>{inviteToken && <p role="status">Copy this one-time invitation token: <code>{inviteToken}</code></p>}</div>}<div className="panel"><table className="trace-table"><thead><tr><th>Email</th><th>Role</th><th>Status</th><th>Actions</th></tr></thead><tbody>{members.map((member) => <tr key={member.id}><td>{member.email}</td><td>{member.role === "owner" ? <span>owner</span> : <select aria-label={`Role for ${member.email}`} value={member.role} disabled={!canWrite || busy === member.id} onChange={(event) => update(member.id, event.target.value)}>{roles.map((role) => <option key={role}>{role}</option>)}</select>}</td><td>{member.status}</td><td>{canWrite && member.role !== "owner" && member.status === "active" && <><button disabled={busy === member.id} onClick={() => disable(member.id)}>Disable</button><button disabled={busy === member.id} onClick={() => makeOwner(member.id)}>Make owner</button></>}</td></tr>)}</tbody></table></div></section>;
}
