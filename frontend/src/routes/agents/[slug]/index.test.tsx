/**
 * `/agents/[slug]` route spec — TDD contract for the polymorphic
 * per-slug loader (feat/archetype-list-endpoint, slice 5 — RED).
 *
 * The slice's contract:
 *   - `slug="assistant"` — the loader resolves the per-org
 *     config and the page renders the ConfigureSection.
 *   - `slug="general"` (or any non-assistant slug from the new
 *     list) — the loader ALSO resolves a config (the
 *     `params.slug === "assistant"` gate is dropped). The
 *     ConfigureSection renders.
 *   - unknown slug — `agentBySlug` returns undefined → the page
 *     renders the 404 state ("No such colleague").
 *   - fetch error — the profile renders, but the
 *     ConfigureSection is suppressed.
 *
 * The current (pre-GREEN) route has two relevant constraints:
 *   1. `useAssistantConfig` gates on `params.slug === "assistant"`,
 *      so non-assistant slugs get a null config even when the
 *      API would resolve them.
 *   2. The route is monolithic: the `routeLoader$` calls are
 *      inline, so there is no extracted loader function to
 *      test in isolation.
 *
 * The slice 6 GREEN commit extracts the loader logic into a
 * pure `loadArchetypeConfigForSlug(slug, agent)` function
 * exported from `./index` so the test can drive it directly
 * (no Qwik City request context required). The RED state: the
 * test file fails to compile because that exported function
 * does not exist yet.
 */
import { beforeEach, describe, expect, it, vi } from "vitest";

import type { Agent } from "~/lib/mock/staff";

// The slice 6 GREEN commit exports `loadArchetypeConfigForSlug`
// as a pure function. The RED state: this import fails to
// resolve, which is the canonical RED for a new exported
// function under strict TDD.
import { loadArchetypeConfigForSlug } from "./index";

// Mock the `assistant-config` module so the loader calls
// `getArchetypeConfig` (the polymorphic wrapper). The mock is set up
// in each test via
// `vi.mocked(getArchetypeConfig).mockResolvedValueOnce(...)`. The
// legacy aliases `getAssistantConfig` / `putAssistantConfig` are also
// mocked so the deprecated call-sites stay testable in isolation.
vi.mock("~/lib/api/assistant-config", () => ({
  getArchetypeConfig: vi.fn(),
  putArchetypeConfigFlat: vi.fn(),
  getAssistantConfig: vi.fn(),
  putAssistantConfig: vi.fn(),
}));

import { getArchetypeConfig } from "~/lib/api/assistant-config";

const assistantAgent: Agent = {
  slug: "assistant",
  initials: "AS",
  name: "Assistant",
  department: "assistant",
  departmentName: "Front desk",
  tagline: "The colleague everyone talks to first.",
  summary: "Handles the work that has no other home.",
  status: "working",
  statusDetail: "On staff and answering now.",
  joined: "2026-01-12",
  tenure: "7 months",
  skills: [],
  tools: [],
  handsOff: null,
  conversationsThisWeek: 64,
};

const generalAgent: Agent = {
  slug: "general",
  initials: "GN",
  name: "General Specialist",
  department: "finance",
  departmentName: "Finance",
  tagline: "A general-purpose specialist",
  summary: "Handles general work.",
  status: "working",
  statusDetail: "On staff and answering now.",
  joined: "2026-02-01",
  tenure: "6 months",
  skills: [],
  tools: [],
  handsOff: null,
  conversationsThisWeek: 12,
};

describe("/agents/[slug] loader (feat/archetype-list-endpoint slice 5 — RED)", () => {
  beforeEach(() => {
    vi.mocked(getArchetypeConfig).mockReset();
  });
  it("Test_AgentSlug_AssistantSlug_LoadsConfig: loader resolves the per-org config for slug='assistant'", async () => {
    vi.mocked(getArchetypeConfig).mockResolvedValueOnce({
      ok: true,
      value: {
        kind: "chat",
        org_id: "user_alice",
        system_prompt: "loaded prompt",
        tool_allowlist: ["current_time"],
        defer_tool_names: [],
        version: 4,
        is_override: true,
      },
    });
    const got = await loadArchetypeConfigForSlug("assistant", assistantAgent);
    expect(got).not.toBeNull();
    expect(got?.system_prompt).toBe("loaded prompt");
    expect(got?.version).toBe(4);
    // Lock the polymorphic contract: the loader must call the
    // API with the supplied slug, not a hard-coded "assistant".
    expect(vi.mocked(getArchetypeConfig)).toHaveBeenCalledWith("assistant");
  });

  it("Test_AgentSlug_GeneralSlug_AlsoLoadsConfig: loader ALSO resolves for slug='general' (the assistant-only gate is dropped) and the API is called with 'general'", async () => {
    vi.mocked(getArchetypeConfig).mockResolvedValueOnce({
      ok: true,
      value: {
        kind: "chat",
        org_id: "user_alice",
        system_prompt: "general prompt",
        tool_allowlist: ["current_time"],
        defer_tool_names: [],
        version: 2,
        is_override: true,
      },
    });
    const got = await loadArchetypeConfigForSlug("general", generalAgent);
    // The current code returns null for any slug !== "assistant".
    // The GREEN commit drops the gate; this assertion documents
    // the new contract.
    expect(got).not.toBeNull();
    expect(got?.system_prompt).toBe("general prompt");
    // CRITICAL: the API must be called with the actual slug, not
    // a hard-coded "assistant" — this is the bug the fix closes.
    expect(vi.mocked(getArchetypeConfig)).toHaveBeenCalledWith("general");
  });

  it("Test_AgentSlug_UnknownSlug_404: agent is null → loader returns null (the page renders the 404 state)", async () => {
    // When the agent is not in the static mock, the loader
    // short-circuits and returns null. The page component
    // renders the 404 state because `useAgent()` returns null.
    const got = await loadArchetypeConfigForSlug("no-such-slug", null);
    expect(got).toBeNull();
    // The API must NOT be called for an unknown slug — the
    // agent lookup short-circuits before any fetch.
    expect(vi.mocked(getArchetypeConfig)).not.toHaveBeenCalled();
  });

  it("Test_AgentSlug_FetchError_RendersProfileWithoutConfigureSection: getArchetypeConfig fails → loader returns null", async () => {
    // The profile is rendered (the agent is known), but the
    // ConfigureSection is suppressed (the loader returned null
    // because the API call failed). The user sees the profile
    // and the section is simply absent.
    vi.mocked(getArchetypeConfig).mockResolvedValueOnce({
      ok: false,
      kind: "server",
      message: "upstream 500",
    });
    const got = await loadArchetypeConfigForSlug("assistant", assistantAgent);
    expect(got).toBeNull();
  });
});
