import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const { push, mockedRequest, pathname } = vi.hoisted(() => ({
  push: vi.fn(),
  mockedRequest: vi.fn(),
  pathname: vi.fn(() => "/dashboard/traces"),
}));

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push }),
  usePathname: () => pathname(),
}));

vi.mock("../../lib/api/client", () => ({
  request: mockedRequest,
}));

import { ConsoleShell } from "./console-shell";
import { StatusBadge } from "../ui/status-badge";
import { StateView } from "../ui/state-view";

afterEach(() => cleanup());

describe("ConsoleShell", () => {
  beforeEach(() => {
    push.mockReset();
    mockedRequest.mockReset();
    pathname.mockReturnValue("/dashboard/traces");
  });

  it("renders the primary routes and tenant context", () => {
    render(
      <ConsoleShell>
        <div>content</div>
      </ConsoleShell>,
    );

    expect(screen.getByRole("link", { name: "Overview" })).toHaveAttribute("href", "/");
    expect(screen.getByRole("link", { name: "Traces" })).toHaveAttribute("href", "/dashboard/traces");
    expect(screen.getByText("Tenant workspace")).toBeInTheDocument();
    expect(screen.getByText("content")).toBeInTheDocument();
  });

  it("marks the matching nested route as current", () => {
    pathname.mockReturnValue("/dashboard/traces/run-123");

    render(<ConsoleShell>content</ConsoleShell>);

    expect(screen.getByRole("link", { name: "Traces" })).toHaveAttribute("aria-current", "page");
    expect(screen.getByRole("link", { name: "Overview" })).not.toHaveAttribute("aria-current");
  });

  it("logs out through the API helper and routes to login", async () => {
    mockedRequest.mockResolvedValue(undefined);

    render(<ConsoleShell>content</ConsoleShell>);
    await userEvent.click(screen.getByRole("button", { name: "Sign out" }));

    expect(mockedRequest).toHaveBeenCalledWith("/api/v1/auth/logout", { method: "POST" });
    expect(push).toHaveBeenCalledWith("/login");
  });

  it("shows a bounded error when logout fails", async () => {
    mockedRequest.mockRejectedValue(new Error("server response must stay private"));

    render(<ConsoleShell>content</ConsoleShell>);
    await userEvent.click(screen.getByRole("button", { name: "Sign out" }));

    const alert = await screen.findByRole("alert");
    expect(alert).toHaveTextContent("Unable to sign out. Please try again.");
    expect(alert).not.toHaveTextContent("server response must stay private");
    expect(push).not.toHaveBeenCalled();
  });

  it("hides console navigation on the login route", () => {
    pathname.mockReturnValue("/login");

    render(<ConsoleShell>login</ConsoleShell>);

    expect(screen.queryByRole("navigation")).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Sign out" })).not.toBeInTheDocument();
    expect(screen.getByText("login")).toBeInTheDocument();
  });
});

describe("StateView", () => {
  it("renders an accessible error state with an optional action", () => {
    const onRetry = vi.fn();

    render(
      <StateView
        kind="error"
        title="Could not load"
        description="Retry the request."
        action={{ label: "Retry", onClick: onRetry }}
      />,
    );

    expect(screen.getByRole("alert")).toHaveTextContent("Could not load");
    expect(screen.getByRole("alert")).toHaveTextContent("Retry the request.");
    expect(screen.getByRole("button", { name: "Retry" })).toBeInTheDocument();
  });

  it.each([
    ["loading", "status"],
    ["empty", "status"],
    ["forbidden", "alert"],
  ] as const)("uses the %s state role", (kind, role) => {
    render(<StateView kind={kind} title={`${kind} state`} />);

    expect(screen.getByRole(role)).toHaveTextContent(`${kind} state`);
  });
});

describe("StatusBadge", () => {
  it("maps domain statuses to semantic tones", () => {
    render(<StatusBadge domain="risk" status="critical" />);

    expect(screen.getByText("critical")).toHaveAttribute("data-tone", "danger");
    expect(screen.getByText("critical")).toHaveClass("status-badge--danger");
  });
});
