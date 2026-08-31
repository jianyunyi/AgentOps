import { describe, expect, it } from "vitest";
import { hasPermission } from "./permissions";

describe("permissions", () => {
  it("uses the server-provided permission set", () => {
    expect(hasPermission({ user_id: "u", tenant_id: "t", role: "viewer", permissions: ["agent:read"] }, "agent:write")).toBe(false);
    expect(hasPermission({ user_id: "u", tenant_id: "t", role: "admin", permissions: ["agent:write"] }, "agent:write")).toBe(true);
  });
});
