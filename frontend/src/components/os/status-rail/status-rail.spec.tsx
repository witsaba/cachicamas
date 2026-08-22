/**
 * StatusRail — the one persistent band across the top of the system.
 *
 * The rail is where two honesty affordances live: the demonstration marker,
 * and the refusal to invent an organization name when none can be read. Both
 * are asserted here, along with the anonymous case, where the rail must not
 * surface a context the visitor does not have.
 */
import { createDOM } from "@builder.io/qwik/testing";
import { describe, it, expect, vi } from "vitest";

vi.mock("~/lib/api", () => ({
  getCurrentOrganization: () => Promise.resolve(null),
}));

import { StatusRail } from "./status-rail";

describe("components/os/status-rail", () => {
  it("always carries the wordmark, pointed where the caller says", async () => {
    const { screen, render } = await createDOM();
    await render(<StatusRail authenticated={true} brandHref="/home/" />);
    const brand = screen.querySelector('[data-testid="status-rail-brand"]');
    expect(brand?.textContent).toBe("cachicamas");
    expect(brand?.getAttribute("href")).toBe("/home/");
  });

  it("keeps the wordmark lowercase, always", async () => {
    const { screen, render } = await createDOM();
    await render(<StatusRail authenticated={true} />);
    expect(
      screen.querySelector('[data-testid="status-rail-brand"]')?.textContent,
    ).toBe("cachicamas");
  });

  it("says 'No organization' rather than inventing a company", async () => {
    const { screen, render } = await createDOM();
    await render(<StatusRail authenticated={true} org={null} />);
    expect(
      screen.querySelector('[data-testid="status-rail-org"]')?.textContent,
    ).toBe("No organization");
  });

  it("shows the organization it was given", async () => {
    const { screen, render } = await createDOM();
    await render(<StatusRail authenticated={true} org="Acme Industrial" />);
    expect(
      screen.querySelector('[data-testid="status-rail-org"]')?.textContent,
    ).toBe("Acme Industrial");
  });

  it("drops the org reading entirely for an anonymous visitor", async () => {
    const { screen, render } = await createDOM();
    await render(<StatusRail authenticated={false} />);
    expect(screen.querySelector('[data-testid="status-rail-org"]')).toBeFalsy();
    expect(
      screen.querySelector('[data-testid="status-rail-brand"]'),
    ).toBeTruthy();
  });

  it("marks demonstration data, and explains the marker on hover", async () => {
    const { screen, render } = await createDOM();
    await render(<StatusRail authenticated={true} demo />);
    const marker = screen.querySelector('[data-testid="status-rail-demo"]');
    expect(marker?.textContent).toBe("Demo data");
    expect(marker?.getAttribute("title") ?? "").toContain("demonstration data");
  });

  it("shows no marker when the caller has not claimed one", async () => {
    const { screen, render } = await createDOM();
    await render(<StatusRail authenticated={true} />);
    expect(
      screen.querySelector('[data-testid="status-rail-demo"]'),
    ).toBeFalsy();
  });

  it("reserves the clock slot server-side instead of shipping a stale time", async () => {
    const { screen, render } = await createDOM();
    await render(<StatusRail authenticated={true} />);
    const clock = screen.querySelector('[data-testid="status-rail-clock"]');
    expect(clock).toBeTruthy();
    expect(clock?.textContent).toBe("");
  });

  it("slots the identity affordance on the right", async () => {
    const { screen, render } = await createDOM();
    await render(
      <StatusRail authenticated={true}>
        <span data-testid="identity">me</span>
      </StatusRail>,
    );
    expect(screen.querySelector('[data-testid="identity"]')).toBeTruthy();
  });
});
