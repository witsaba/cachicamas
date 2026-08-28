/**
 * agent-directory.spec.tsx — TDD contract for the server-authoritative
 * directory component (cachicamas-agent-catalog-config-reload, S2-G T5.5).
 *
 * The component renders exactly the rows the route loader returned:
 *   - one card per list entry — the list from the server IS the directory;
 *   - an empty array renders the explicit "No colleagues yet" empty state
 *     (data-testid="agent-directory-empty"), never the static AGENTS mock;
 *   - the deleted assistantView/assistantConfigured override-hook props are
 *     ignored — a card's statusWord derives from its row's `is_override`
 *     (C2/D-ADR-04), so the override hook can no longer fabricate or
 *     restyle an assistant card (CRL-S-010, CRL-S-011).
 *
 * List-fetch failure is owned by the route (null → explicit error card),
 * so the component only ever receives real arrays.
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

describe("AgentDirectory (loader-provided list renders one card per entry)", () => {
  it("renders one card per entry in the loader-provided list", async () => {
    const { screen, render } = await createDOM();
    const archetypes = [
      viewFor({ slug: "assistant", type: "system", display_name: "Assistant" }),
      viewFor({ slug: "general-one", type: "general", display_name: "General One" }),
      viewFor({ slug: "owned-one", type: "owned", display_name: "Owned One" }),
    ];
    await render(<AgentDirectory archetypes={archetypes} />);
    for (const v of archetypes) {
      const card = screen.querySelector(`[data-testid="agent-card-${v.slug}"]`);
      expect(card, v.slug).toBeTruthy();
    }
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
          // Qwik's createDOM screen returns `undefined` (not `null`)
          // when a selector misses — assert falsiness, not null.
          screen.querySelector('[data-testid="agent-card-assistant"]'),
        ).toBeFalsy();
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
