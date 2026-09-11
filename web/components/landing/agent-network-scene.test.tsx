import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

const { push } = vi.hoisted(() => ({ push: vi.fn() }));

vi.mock("next/navigation", () => ({ useRouter: () => ({ push }) }));

import { AgentNetworkScene } from "./agent-network-scene";

afterEach(() => {
  cleanup();
  push.mockReset();
});

describe("AgentNetworkScene", () => {
  it("renders an accessible Agent network visualization", () => {
    render(<AgentNetworkScene />);

    expect(screen.getByRole("img", { name: "Agent network visualization" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Trace Explorer" })).toBeInTheDocument();
    expect(screen.getByRole("button", { name: "Risk Review" })).toBeInTheDocument();
  });

  it("navigates to the selected domain when activated", async () => {
    render(<AgentNetworkScene />);

    await userEvent.click(screen.getByRole("button", { name: "Trace Explorer" }));

    expect(push).toHaveBeenCalledWith("/dashboard/traces");
  });
});
