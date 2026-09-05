import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";
import { SectorTopology } from "./sector-topology";
import type { SectorNode, SectorRoute } from "./control-tower-model";

const nodes: SectorNode[] = [
  { id: "ops", callsign: "AGT_01", label: "Orchestrator", tone: "healthy", x: -2, y: 1, z: 0 },
  { id: "guard", callsign: "AGT_02", label: "Policy Guard", tone: "critical", x: 2, y: -1, z: 0.2 },
];

const routes: SectorRoute[] = [
  { traceId: "trc-1", fromId: "ops", toId: "guard", tone: "critical" },
];

afterEach(cleanup);

describe("SectorTopology", () => {
  it("renders the semantic topology and agent buttons without WebGL", () => {
    render(<SectorTopology nodes={nodes} routes={routes} />);

    const topology = screen.getByRole("img", { name: "Live Agent sector topology" });
    const orchestrator = screen.getByRole("button", { name: "Inspect AGT_01 Orchestrator" });
    const policyGuard = screen.getByRole("button", { name: "Inspect AGT_02 Policy Guard" });

    expect(topology).toBeInTheDocument();
    expect(orchestrator).toBeInTheDocument();
    expect(policyGuard).toBeInTheDocument();
    expect(topology).not.toContainElement(orchestrator);
    expect(topology).not.toContainElement(policyGuard);
  });

  it("selects the clicked agent", async () => {
    const onSelect = vi.fn();
    render(<SectorTopology nodes={nodes} routes={routes} onSelect={onSelect} />);

    await userEvent.click(screen.getByRole("button", { name: "Inspect AGT_02 Policy Guard" }));

    expect(onSelect).toHaveBeenCalledWith("guard");
  });
});
