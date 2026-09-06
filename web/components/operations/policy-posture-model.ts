import type { Policy } from "../../lib/api/types";

export type PolicyPosture = {
  enabledCount: number;
  active?: Policy;
};

const timestamp = (value: string) => {
  const parsed = Date.parse(value);
  return Number.isNaN(parsed) ? Number.NEGATIVE_INFINITY : parsed;
};

const isNewerPolicy = (candidate: Policy, current: Policy) => {
  if (candidate.version !== current.version) {
    return candidate.version > current.version;
  }

  const candidateTimestamp = timestamp(candidate.created_at);
  const currentTimestamp = timestamp(current.created_at);
  if (candidateTimestamp !== currentTimestamp) {
    return candidateTimestamp > currentTimestamp;
  }

  return candidate.id < current.id;
};

export const buildPolicyPosture = (policies: Policy[]): PolicyPosture => {
  const enabledPolicies = policies.filter((policy) => policy.enabled);
  const active = enabledPolicies.reduce<Policy | undefined>(
    (current, candidate) =>
      current === undefined || isNewerPolicy(candidate, current) ? candidate : current,
    undefined,
  );

  return { enabledCount: enabledPolicies.length, active };
};
