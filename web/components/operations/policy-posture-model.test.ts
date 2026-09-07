import { describe, expect, it } from "vitest";
import { buildPolicyPosture } from "./policy-posture-model";

const policy = (
  id: string,
  version: number,
  enabled = true,
  created_at = "2026-09-06T10:00:00Z",
) => ({
  id,
  tenant_id: "tenant-1",
  name: `Policy ${id}`,
  version,
  enabled,
  rules_enabled: true,
  llm_enabled: false,
  max_input_bytes: 2048,
  created_by: "user-1",
  created_at,
});

describe("buildPolicyPosture", () => {
  it("counts enabled policies and selects the greatest enabled version", () => {
    expect(
      buildPolicyPosture([
        policy("old", 2),
        policy("current", 3),
        policy("disabled", 9, false),
      ]),
    ).toMatchObject({ enabledCount: 2, active: { id: "current" } });
  });

  it("uses newest timestamp then lexical id for equal versions", () => {
    expect(
      buildPolicyPosture([
        policy("zeta", 3, true, "2026-09-06T00:00:00Z"),
        policy("alpha", 3, true, "2026-09-06T00:00:00Z"),
        policy("older", 3, true, "2026-09-05T00:00:00Z"),
      ]).active?.id,
    ).toBe("alpha");
  });

  it("treats invalid timestamps as older than valid timestamps", () => {
    expect(
      buildPolicyPosture([
        policy("invalid", 3, true, "not-a-timestamp"),
        policy("valid", 3, true, "2026-09-06T00:00:00Z"),
      ]).active?.id,
    ).toBe("valid");
  });

  it("does not invent an active policy", () => {
    expect(buildPolicyPosture([])).toEqual({
      enabledCount: 0,
      active: undefined,
    });
    expect(buildPolicyPosture([policy("disabled", 1, false)])).toEqual({
      enabledCount: 0,
      active: undefined,
    });
  });
});
