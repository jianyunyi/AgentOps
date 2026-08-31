"use client";
import { FormEvent, useState } from "react";
import { useRouter } from "next/navigation";
import { login } from "../../lib/api/auth";

export default function LoginPage() {
  const router = useRouter(); const [email, setEmail] = useState(""); const [password, setPassword] = useState(""); const [error, setError] = useState<string | null>(null);
  async function submit(event: FormEvent) { event.preventDefault(); setError(null); try { await login(email, password); router.push("/dashboard/traces"); } catch { setError("Email or password is invalid"); } }
  return <main className="auth-card"><h1>Sign in to AgentScope</h1><form onSubmit={submit}><label>Email<input value={email} onChange={(e) => setEmail(e.target.value)} type="email" required /></label><label>Password<input value={password} onChange={(e) => setPassword(e.target.value)} type="password" required /></label>{error && <div role="alert" className="error-state">{error}</div>}<button type="submit">Sign in</button></form></main>;
}
