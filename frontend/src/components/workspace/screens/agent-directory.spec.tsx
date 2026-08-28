/**
 * agent-directory.spec.tsx — TDD contract for the polymorphic list
 * consumer on the agent directory screen
 * (feat/archetype-list-endpoint, slice 5 — RED).
 *
 * Three scenarios for the slice:
 *   - When the route loader resolves with a list of archetypes, the
 *     component renders one card per list entry (not the static
 *     AGENTS literal). The list from the loader is the directory.
 *   - When the loader fails (offline / 5xx / not authed) the
 *     component falls back to the static AGENTS literal. The
 *     assistant overlay (assistantView prop) still applies.
 *   - The assistant overlay (display_name + tagline from
 *     assistantView) takes precedence over both the loader's
 *     assistant row AND the static literal — the polymorphic view
 *     is authoritative for the assistant card.
 *
 * The `archetypes` prop is intentionally added by the slice 6
 * GREEN commit; the test passes the prop via a type cast so the
 * current (pre-GREEN) component still type-checks under
 * `strict: true`. The assertion is the RED: the current component
 * ignores the prop and falls back to the static AGENTS literal, so
 * the success-path assertion (3 mocked cards) fails because only
 * one card is rendered. The error-path + overlay-path tests are
 * regression guards for the GREEN behaviour.
 */
import { createDOM } from "@builder.io/qwik/testing";
import { describe, expect, it } from "vitest";

import { AgentDirectory } from "./agent-directory";
import type { ArchetypeView } from "~/lib/api/archetypes";

function viewFor(overrides: Partial<ArchetypeView>): ArchetypeView {
  return {
    slug: overrides.slug ?? "assistant",
    type: overrides.type ?? "system",
    display_name: overrides.display_name ?? "Assistant",
    tagline: overrides.tagline ?? "Default tagline",
    status: overrides.status ?? "active",
    archived_at: overrides.archived_at ?? null,
    created_at: overrides.created_at ?? "2026-08-27T10:00:00Z",
    created_by: overrides.created_by ?? "seed",
    bundle_version: overrides.bundle_version,
    is_critical: overrides.is_critical,
    override: overrides.override,
    is_override: overrides.is_override ?? false,
  };
}

describe("AgentDirectory (feat/archetype-list-endpoint slice 5 — RED)", () => {
  it("renders one card per entry in the loader-provided list", async () => {
    const { screen, render } = await createDOM();
    const archetypes = [
      viewFor({ slug: "assistant", type: "system", display_name: "Assistant" }),
      viewFor({ slug: "general-one", type: "general", display_name: "General One" }),
      viewFor({ slug: "owned-one", type: "owned", display_name: "Owned One" }),
    ];
    // Type cast: the `archetypes` prop is added by the slice 6 GREEN
    // commit. The cast keeps the test type-checking under strict TS
    // without forcing a forward-reference to the new prop name.
    await render(
      <AgentDirectory
        {...({ archetypes } as unknown as Parameters<typeof AgentDirectory>[0])}
      />,
    );
    for (const v of archetypes) {
      const card = screen.querySelector(`[data-testid="agent-card-${v.slug}"]`);
      expect(card, v.slug).toBeTruthy();
    }
  });

  it("falls back to the static AGENTS literal when the loader list is absent", async () => {
    // The loader is absent (the route fell back to null after a
    // 5xx / offline / not-authed response). The directory renders
    // the static AGENTS literal so the user still sees the
    // assistant card + overlay path.
    const { screen, render } = await createDOM();
    await render(
      <AgentDirectory
        assistantConfigured={false}
        assistantView={viewFor({ slug: "assistant", display_name: "From Overlay" })}
      />,
    );
    // The assistant card is rendered (from AGENTS) with the
    // overlay's display_name taking precedence.
    const card = screen.querySelector('[data-testid="agent-card-assistant"]');
    expect(card).toBeTruthy();
    expect(card?.textContent).toContain("From Overlay");
  });

  it("keeps the assistant overlay authoritative when the loader list is present", async () => {
    // When the loader resolves AND the caller passes an explicit
    // assistantView prop, the prop's display_name + tagline win
    // for the assistant card. The loader's entry for the
    // assistant is treated as base data; the overlay is the
    // per-org customisation surface.
    const { screen, render } = await createDOM();
    const archetypes = [
      viewFor({ slug: "assistant", type: "system", display_name: "From Loader" }),
      viewFor({ slug: "general-one", type: "general", display_name: "General One" }),
    ];
    await render(
      <AgentDirectory
        {...({
          archetypes,
          assistantConfigured: true,
          assistantView: viewFor({
            slug: "assistant",
            display_name: "From Overlay",
            tagline: "Overlay tagline",
          }),
        } as unknown as Parameters<typeof AgentDirectory>[0])}
      />,
    );
    const card = screen.querySelector('[data-testid="agent-card-assistant"]');
    expect(card).toBeTruthy();
    // Overlay wins.
    expect(card?.textContent).toContain("From Overlay");
    expect(card?.textContent).toContain("Overlay tagline");
    // The non-assistant card still uses the loader's data.
        const other = screen.querySelector('[data-testid="agent-card-general-one"]');
        expect(other).toBeTruthy();
        expect(other?.textContent).toContain("General One");
      });
    });

    // ---------------------------------------------------------------------------
    // cachicamas-agent-catalog-config-reload (S2-R — RED) — the directory becomes
    // server-authoritative: the route owns list-fetch failure and passes REAL
    // arrays only, the AGENTS fallback is gone, and the assistant's statusWord
    // derives from its row's is_override — not the deleted override hook
    // (CRL-S-010/011, C2/D-ADR-04).
    //
    // GREEN contract notes:
    //   - `archetypes` becomes a REQUIRED prop; `assistantConfigured` and
    //     `assistantView` are deleted. The legacy props below are passed through
    //     a cast so they type-check today AND stay meaningful after GREEN (the
    //     component must simply IGNORE them — the row's is_override wins).
    //   - the empty list renders an explicit empty state carrying
    //     data-testid="agent-directory-empty".
    // ---------------------------------------------------------------------------
    describe("AgentDirectory (cachicamas-agent-catalog-config-reload S2-R — RED)", () => {
      it("Test_AgentDirectory_EmptyList_RendersEmptyState: an empty list renders the empty state, NOT AGENTS cards (CRL-S-010)", async () => {
        // The route handles list-fetch failure and hands the component REAL
        // arrays only. An empty array therefore means "no archetypes": the
        // component must render an explicit empty state and never fall back
        // to the static AGENTS literal.
        const { screen, render } = await createDOM();
        await render(<AgentDirectory archetypes={[]} />);
        expect(
          screen.querySelector('[data-testid="agent-directory-empty"]'),
        ).toBeTruthy();
        expect(
          screen.querySelectorAll('[data-testid^="agent-card-"]').length,
        ).toBe(0);
      });

      it("Test_AgentDirectory_NoAssistantCard_DerivedFromAgents: even with the legacy override-hook props present, no assistant card is derived from AGENTS (CRL-S-011)", async () => {
        // The deleted override hook (assistantConfigured/assistantView) must
        // not be able to fabricate an assistant card when the list has none.
        const { screen, render } = await createDOM();
        await render(
          <AgentDirectory
            {...({
              archetypes: [],
              assistantConfigured: true,
              assistantView: viewFor({ slug: "assistant", display_name: "From Overlay" }),
            } as unknown as Parameters<typeof AgentDirectory>[0])}
          />,
        );
        expect(
          screen.querySelector('[data-testid="agent-card-assistant"]'),
        ).toBeNull();
        expect(
          screen.querySelectorAll('[data-testid^="agent-card-"]').length,
        ).toBe(0);
      });

      it("Test_AgentDirectory_AssistantStatusWord_FromRowIsOverride: the assistant statusWord derives from its row's is_override, not the deleted override hook (CRL-S-011, C2/D-ADR-04)", async () => {
            // The row says Default (is_override=false) but the deleted override
            // hook would claim Configured. The row must win. The tagline is set
            // explicitly so the "Default" assertion can only match the status word.
            const { screen, render } = await createDOM();
            await render(
              <AgentDirectory
                {...({
                  archetypes: [
                    viewFor({ slug: "assistant", is_override: false, tagline: "Server row tagline" }),
                  ],
                  assistantConfigured: true,
                } as unknown as Parameters<typeof AgentDirectory>[0])}
              />,
            );
        const card = screen.querySelector('[data-testid="agent-card-assistant"]');
        expect(card).toBeTruthy();
        expect(card?.textContent).toContain("Default");
        expect(card?.textContent).not.toContain("Configured");
      });
    });
