"use client";

import type { ReactNode } from "react";
import { useState } from "react";
import { usePathname, useRouter } from "next/navigation";

import { request } from "../../lib/api/client";

const navigation = [
  ["Overview", "/"],
  ["Traces", "/dashboard/traces"],
  ["Risk", "/dashboard/risk"],
  ["Agents", "/settings/agents"],
  ["Members", "/settings/members"],
  ["Policies", "/settings/policies"],
  ["Audit", "/settings/audit"],
] as const;

function NavigationIcon({ name }: { name: (typeof navigation)[number][0] }) {
  const paths = {
    Overview: <><rect x="3" y="3" width="7" height="7" rx="1" /><rect x="14" y="3" width="7" height="7" rx="1" /><rect x="3" y="14" width="7" height="7" rx="1" /><rect x="14" y="14" width="7" height="7" rx="1" /></>,
    Traces: <><path d="M4 17V7a2 2 0 0 1 2-2h12" /><path d="m14 3 4 2-4 2" /><path d="m10 21-4-2 4-2" /><path d="M20 7v10a2 2 0 0 1-2 2H6" /></>,
    Risk: <><path d="M12 3 3 20h18L12 3Z" /><path d="M12 9v4" /><path d="M12 17h.01" /></>,
    Agents: <><circle cx="12" cy="8" r="3" /><path d="M5 21v-2a7 7 0 0 1 14 0v2" /><path d="M19 8h2" /><path d="M20 7v2" /></>,
    Members: <><circle cx="9" cy="8" r="3" /><circle cx="17" cy="10" r="2" /><path d="M3 21v-2a6 6 0 0 1 12 0v2" /><path d="M15 16a5 5 0 0 1 6 3v2" /></>,
    Policies: <><path d="M12 3 5 6v5c0 5 3 8 7 10 4-2 7-5 7-10V6l-7-3Z" /><path d="m9 12 2 2 4-4" /></>,
    Audit: <><path d="M6 3h9l3 3v15H6V3Z" /><path d="M15 3v4h4" /><path d="M9 12h6" /><path d="M9 16h6" /></>,
  } satisfies Record<(typeof navigation)[number][0], ReactNode>;

  return <svg aria-hidden="true" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.75">{paths[name]}</svg>;
}

export function ConsoleShell({ children }: { children: ReactNode }) {
  const router = useRouter();
  const pathname = usePathname();
  const [signOutError, setSignOutError] = useState(false);
  const [signingOut, setSigningOut] = useState(false);
  const isLogin = pathname === "/login";

  async function signOut() {
    if (signingOut) return;

    setSigningOut(true);
    setSignOutError(false);
    try {
      await request<void>("/api/v1/auth/logout", { method: "POST" });
      router.push("/login");
    } catch {
      setSignOutError(true);
      setSigningOut(false);
    }
  }

  if (isLogin) {
    return (
      <div className="app-shell">
        <header className="app-header">
          <a className="brand" href="/login">AgentOps</a>
          <span className="subtitle">Agent governance console</span>
        </header>
        <main className="page-frame">{children}</main>
      </div>
    );
  }

  return (
    <div className="app-shell">
      <aside className="command-rail">
        <a className="command-brand" href="/">AgentOps</a>
        <nav className="command-nav" aria-label="Primary navigation">
          {navigation.map(([label, href]) => {
            const isCurrent = href === "/" ? pathname === "/" : pathname === href || pathname.startsWith(`${href}/`);

            return (
              <a key={href} href={href} aria-current={isCurrent ? "page" : undefined}>
                <NavigationIcon name={label} />
                <span>{label}</span>
              </a>
            );
          })}
        </nav>
        <div className="command-operator">
          <span>Tenant workspace</span>
          <button type="button" className="button button-quiet" onClick={() => void signOut()} disabled={signingOut}>
            {signingOut ? "Signing out" : "Sign out"}
          </button>
        </div>
      </aside>
      {signOutError && <div className="shell-alert" role="alert">Unable to sign out. Please try again.</div>}
      <main className="console-workspace page-frame">{children}</main>
    </div>
  );
}
