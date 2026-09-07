import type { RiskEvent } from "../../lib/api/types";

const timestamp = (value: string) => {
  const parsed = Date.parse(value);
  return Number.isNaN(parsed) ? Number.NEGATIVE_INFINITY : parsed;
};

export const risksForTrace = (risks: RiskEvent[], traceId: string): RiskEvent[] =>
  risks
    .filter((risk) => risk.trace_id === traceId)
    .sort((left, right) => {
      const timestampDifference = timestamp(right.created_at) - timestamp(left.created_at);
      if (timestampDifference !== 0) {
        return timestampDifference;
      }

      return left.id < right.id ? -1 : left.id > right.id ? 1 : 0;
    });
