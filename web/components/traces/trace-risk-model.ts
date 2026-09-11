import type { RiskEvent } from "../../lib/api/types";

const timestamp = (value: string): number | null => {
  const parsed = Date.parse(value);
  return Number.isNaN(parsed) ? null : parsed;
};

export const risksForTrace = (risks: RiskEvent[], traceId: string): RiskEvent[] =>
  risks
    .filter((risk) => risk.trace_id === traceId)
    .sort((left, right) => {
      const leftTimestamp = timestamp(left.created_at);
      const rightTimestamp = timestamp(right.created_at);

      if (leftTimestamp !== null && rightTimestamp === null) {
        return -1;
      }

      if (leftTimestamp === null && rightTimestamp !== null) {
        return 1;
      }

      if (leftTimestamp !== null && rightTimestamp !== null && leftTimestamp !== rightTimestamp) {
        return rightTimestamp - leftTimestamp;
      }

      return left.id < right.id ? -1 : left.id > right.id ? 1 : 0;
    });
