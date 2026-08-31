import { APIError } from "./client";

export async function login(email: string, password: string): Promise<void> {
  const response = await fetch("/api/v1/auth/login", { method: "POST", headers: { "content-type": "application/json" }, credentials: "include", body: JSON.stringify({ email, password }) });
  if (!response.ok) {
    const body = (await response.json()) as { error?: { code: string; message: string } };
    throw new APIError(body.error?.code ?? "LOGIN_FAILED", body.error?.message ?? "Login failed", response.status);
  }
}
