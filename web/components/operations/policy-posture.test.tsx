import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("next/link", () => ({
  default: ({ href, children }: { href: string; children: React.ReactNode }) => (
    <a href={href}>{children}</a>
  ),
}));

import { PolicyPosture } from "./policy-posture";

const activePolicy = {
  id: "policy-1",
  tenant_id: "tenant-1",
  name: "Production guardrails",
  version: 7,
  enabled: true,
  rules_enabled: true,
  llm_enabled: false,
  max_input_bytes: 2048,
  created_by: "user-1",
  created_at: "2026-09-06T10:00:00Z",
};

describe("PolicyPosture", () => {
  afterEach(() => {
    cleanup();
  });

  it("renders a real active policy posture", () => {
    render(<PolicyPosture state={{ kind: "ready", posture: { enabledCount: 3, active: activePolicy } }} />);

    expect(screen.getByRole("region", { name: "Policy posture" })).toBeInTheDocument();
    expect(screen.getByText("Governance configuration")).toBeInTheDocument();
    expect(screen.getByText("3 enabled in loaded results")).toBeInTheDocument();
    expect(screen.getByText("Production guardrails · v7")).toBeInTheDocument();
    expect(screen.getByText("Rules enabled")).toBeInTheDocument();
    expect(screen.getByText("LLM analysis disabled")).toBeInTheDocument();
    expect(screen.getByRole("link", { name: "Manage policies" })).toHaveAttribute("href", "/settings/policies");
  });

  it("allows a long external active policy name to break inside its value", () => {
    render(
      <PolicyPosture
        state={{
          kind: "ready",
          posture: { enabledCount: 1, active: { ...activePolicy, name: "policy-with-an-unbroken-external-name" } },
        }}
      />,
    );

    expect(screen.getByText(/policy-with-an-unbroken-external-name/)).toHaveClass("policy-posture-value");
    expect(screen.getByText(/policy-with-an-unbroken-external-name/)).toHaveStyle({ overflowWrap: "anywhere" });
  });

  it("renders the explicit empty state", () => {
    render(<PolicyPosture state={{ kind: "ready", posture: { enabledCount: 0 } }} />);

    expect(screen.getByText("No policies configured.")).toBeInTheDocument();
  });

  it("renders an error and retries", async () => {
    const onRetry = vi.fn();
    render(<PolicyPosture state={{ kind: "error", onRetry }} />);

    expect(screen.getByRole("alert")).toHaveTextContent("Policy posture is temporarily unavailable.");
    await userEvent.click(screen.getByRole("button", { name: "Retry policy posture" }));
    expect(onRetry).toHaveBeenCalledOnce();
  });

  it("announces loading status", () => {
    render(<PolicyPosture state={{ kind: "loading" }} />);

    expect(screen.getByRole("status")).toHaveTextContent("Reading policy posture");
  });
});
