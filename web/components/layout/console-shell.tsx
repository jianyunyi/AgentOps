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

  return (
    <div className="app-shell">
      <header className="app-header">
        <a className="brand" href={isLogin ? "/login" : "/"}>AgentOps</a>
        <span className="subtitle">Agent governance console</span>
        {!isLogin && <>
          <nav className="app-nav" aria-label="Primary navigation">
            {navigation.map(([label, href]) => <a key={href} href={href}>{label}</a>)}
          </nav>
          <div className="tenant-context">
            <span>Tenant workspace</span>
            <button type="button" className="button button-quiet" onClick={() => void signOut()} disabled={signingOut}>
              {signingOut ? "Signing out" : "Sign out"}
            </button>
          </div>
        </>}
      </header>
      {signOutError && <div className="shell-alert" role="alert">Unable to sign out. Please try again.</div>}
      <main className="page-frame">{children}</main>
    </div>
  );
}
