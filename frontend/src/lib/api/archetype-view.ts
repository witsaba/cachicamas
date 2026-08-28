/**
 * archetype-view.ts — minimal display projection from a polymorphic
 * ArchetypeView into the Agent shape the workspace cards and profile
 * screens consume (D-ADR-02).
 *
 * S2-G part 1 (T5.1, CRL-S-016) of cachicamas-agent-catalog-config-reload:
 * the projection moves here from `agent-directory.tsx` so both the staff
 * directory and the `/agents/[slug]` profile route render server rows
 * through ONE pure function. It is deliberately minimal: the server is
 * the authority for every displayed field, so nothing is inherited from
 * the static AGENTS literal and no discovery/fallback enrichment happens.
 *
 * Field mapping:
 *   - slug      → view.slug
 *   - initials  → slug.slice(0, 2).toUpperCase()
 *   - name      → view.display_name
 *   - department / departmentName → "assistant" / "Front desk" (the
 *     directory shell's front-desk framing for every server row)
 *   - tagline   → view.tagline
 *   - status    → "working" when the view is active, "training" otherwise
 *   - statusWord→ "Configured" when view.is_override (the org has a
 *     per-org row), "Default" otherwise (C2/D-ADR-04: the row wins)
 *   - statusDetail / joined / summary — honest defaults; the view does
 *     not carry skills, tools, tenure, hands-off or workload figures, so
 *     those stay empty/null rather than being fabricated.
 */

import type { Agent } from "~/lib/mock/staff";
import type { ArchetypeView } from "./archetypes";

export function archetypeViewToAgent(view: ArchetypeView): Agent {
  return {
    slug: view.slug,
    initials: view.slug.slice(0, 2).toUpperCase(),
    name: view.display_name,
    department: "assistant",
    departmentName: "Front desk",
    tagline: view.tagline,
    summary: "",
    status: view.status === "active" ? "working" : "training",
    statusWord: view.is_override ? "Configured" : "Default",
    statusDetail:
      view.status === "active"
        ? "On staff and answering now."
        : "Hired, still being set up.",
    joined: view.created_at,
    tenure: null,
    skills: [],
    tools: [],
    handsOff: null,
    conversationsThisWeek: null,
  };
}
