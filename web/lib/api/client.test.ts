import { afterEach, describe, expect, it, vi } from "vitest";
import { request } from "./client";

afterEach(() => {
  vi.restoreAllMocks();
  delete window.__AGENTOPS_RUNTIME_CONFIG__;
});

describe("request API base", () => {
  it("uses same-origin paths by default", async () => {
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({ data: { ok: true } }), { status: 200 })));
    await request("/api/v1/auth/me");
    expect(fetch).toHaveBeenCalledWith("/api/v1/auth/me", expect.objectContaining({ credentials: "include" }));
  });

  it("uses a runtime-configured absolute base when provided", async () => {
    window.__AGENTOPS_RUNTIME_CONFIG__ = { apiBaseUrl: "https://api.example.test/" };
    vi.stubGlobal("fetch", vi.fn().mockResolvedValue(new Response(JSON.stringify({ data: { ok: true } }), { status: 200 })));
    await request("/api/v1/auth/me");
    expect(fetch).toHaveBeenCalledWith("https://api.example.test/api/v1/auth/me", expect.objectContaining({ credentials: "include" }));
  });
});
