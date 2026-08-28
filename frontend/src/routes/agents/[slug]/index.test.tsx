/**
 * `/agents/[slug]` route spec — TDD contract for the server-authoritative
 * profile load (cachicamas-agent-catalog-config-reload, S2-R — RED).
 *
 * S2-G target: the route no longer gates rendering on the static AGENTS
 * literal (`agentBySlug`). A pure `resolveAgentProfile(slug)` performs the
 * per-slug server loads and classifies the result into the three-state
 * `AgentProfileResolution`:
 *
 *   - profile GET /api/archetypes/{slug} → 200
 *       → { kind: "ok", view, config }   (config is null when the config GET
 *         failed — profile renders WITHOUT the ConfigureSection, no
 *         synthesized config — CRL-S-014)
 *   - profile GET → 404 (slug not registered on the server)
 *       → { kind: "unknown" } — the page maps this to status(404) + the
 *         "No such colleague" state (CRL-S-013)
 *   - profile GET → transient failure (5xx / offline)
 *       → { kind: "unavailable", message } — an explicit error card, NOT a
 *         404, and never a profile fabricated from AGENTS (CRL-S-015)
 *
 * A slug absent from AGENTS but known to the server MUST open the profile
 * (CRL-S-012). The fixtures below use "general-specialist", which is not in
 * AGENTS, so the pre-GREEN AGENTS-gated loader would render a 404.
 *
 * The route also projects the ArchetypeView minimally into the AgentProfile
 * props via a pure `agentProfileProps(view)` helper (CRL-S-016):
 * name=display_name, tagline, slug — no extra transformation.
 *
 * RED seam: both functions are added by the S2-G GREEN commit. The tests
 * access them through a namespace cast so the RED is a behavioral assertion
 * failure ("expected a function, got undefined"), not a module-link error
 * that would mask which scenario failed.
 *
 * fetch is mocked at the same globalThis.fetch seam the API client spec
 * (lib/api/archetypes.spec.ts) uses, so the per-slug URL contract is
 * asserted against the real client code path.
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import * as routeIndex from "./index";
import type { ArchetypeView } from "~/lib/api/archetypes";

type AgentProfileResolution =
  | { kind: "ok"; view: ArchetypeView; config: ArchetypeView | null }
  | { kind: "unknown" }
  | { kind: "unavailable"; message: string };

// RED seam (S2-G): these exports do not exist yet. The cast turns the
// missing exports into per-test behavioral failures.
const routeModule = routeIndex as unknown as {
  resolveAgentProfile?: (slug: string) => Promise<AgentProfileResolution>;
  agentProfileProps?: (
    view: ArchetypeView,
  ) => { name: string; tagline: string; slug: string };
};

const resolveAgentProfile = routeModule.resolveAgentProfile;
const agentProfileProps = routeModule.agentProfileProps;

const originalFetch = globalThis.fetch;

/** Not in AGENTS — the pre-GREEN AGENTS-gated loader would 404 this slug. */
function generalSpecialistView(overrides: Partial<ArchetypeView> = {}): ArchetypeView {
  return {
    slug: overrides.slug ?? "general-specialist",
    type: overrides.type ?? "general",
    display_name: overrides.display_name ?? "General Specialist",
    tagline: overrides.tagline ?? "A general-purpose specialist",
    status: overrides.status ?? "active",
    archived_at: overrides.archived_at ?? null,
    created_at: overrides.created_at ?? "2026-08-27T10:00:00Z",
    created_by: overrides.created_by ?? "seed",
    is_override: overrides.is_override ?? false,
    ...overrides,
  };
}

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "content-type": "application/json" },
  });
}

describe("/agents/[slug] — server-authoritative profile load (S2-R — RED)", () => {
  beforeEach(() => {
    globalThis.fetch = vi.fn();
  });
  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  it("Test_AgentSlug_ServerKnownSlug_OpensViaPerSlugGet: slug absent from AGENTS opens via the per-slug GET, no 404 (CRL-S-012)", async () => {
    expect(
      resolveAgentProfile,
      "resolveAgentProfile is added by the S2-G GREEN commit",
    ).toBeTypeOf("function");
    (globalThis.fetch as ReturnType<typeof vi.fn>)
      .mockResolvedValueOnce(jsonResponse(200, generalSpecialistView()))
      .mockResolvedValueOnce(
        jsonResponse(200, generalSpecialistView({ is_override: true })),
      );

    const res = await resolveAgentProfile!("general-specialist");
    // "ok" — NOT the 404 the AGENTS-gated loader produces for a slug that
    // is not in the static literal.
    expect(res.kind).toBe("ok");
    if (res.kind !== "ok") return;
    expect(res.view.display_name).toBe("General Specialist");
    // The per-slug GET must be made with the actual slug.
    expect(globalThis.fetch).toHaveBeenCalledWith(
      "/api/archetypes/general-specialist",
      expect.objectContaining({ method: "GET", credentials: "include" }),
    );
  });

  it("Test_AgentSlug_UnknownSlug_KindUnknown: server 404 → kind 'unknown' (page maps to status(404) + 'No such colleague', CRL-S-013)", async () => {
    expect(resolveAgentProfile).toBeTypeOf("function");
    (globalThis.fetch as ReturnType<typeof vi.fn>).mockResolvedValueOnce(
      jsonResponse(404, {
        kind: "not_found",
        message: "archetype slug is not registered",
      }),
    );

    const res = await resolveAgentProfile!("no-such-slug");
    expect(res.kind).toBe("unknown");
  });

  it("Test_AgentSlug_TransientProfileFailure_KindUnavailable: network failure → kind 'unavailable', not 404, no AGENTS fabrication (CRL-S-015)", async () => {
    expect(resolveAgentProfile).toBeTypeOf("function");
    (globalThis.fetch as ReturnType<typeof vi.fn>)
      .mockRejectedValueOnce(new Error("network unreachable"))
      .mockRejectedValueOnce(new Error("network unreachable"));

    const res = await resolveAgentProfile!("general-specialist");
    // Transient failure is a distinct state from "unknown": an explicit
    // error card, never the 404 page, never a profile fabricated from AGENTS.
    expect(res.kind).toBe("unavailable");
    expect((res as { message?: string }).message).toBeTypeOf("string");
  });

  it("Test_AgentSlug_ConfigGetFailure_ProfileWithoutConfigureSection: config GET failure → kind 'ok' with config null, no synthesized config (CRL-S-014)", async () => {
    expect(resolveAgentProfile).toBeTypeOf("function");
    (globalThis.fetch as ReturnType<typeof vi.fn>)
      .mockResolvedValueOnce(jsonResponse(200, generalSpecialistView()))
      .mockRejectedValueOnce(new Error("config endpoint down"));

    const res = await resolveAgentProfile!("general-specialist");
    expect(res.kind).toBe("ok");
    // The profile still resolves; the ConfigureSection is suppressed and
    // no config object is synthesized from the failure.
    expect((res as { config?: unknown }).config).toBeNull();
  });

  it("Test_AgentSlug_ViewProjectsMinimallyIntoProfileProps: ArchetypeView projects name=display_name, tagline, slug — no extra transformation (CRL-S-016)", () => {
    expect(
      agentProfileProps,
      "agentProfileProps is added by the S2-G GREEN commit",
    ).toBeTypeOf("function");
    const view = generalSpecialistView();
    const props = agentProfileProps!(view);
    expect(props).toEqual({
      name: "General Specialist",
      tagline: view.tagline,
      slug: "general-specialist",
    });
    // Minimal projection: exactly these three keys, nothing else.
    expect(Object.keys(props).sort()).toEqual(["name", "slug", "tagline"]);
  });
});
