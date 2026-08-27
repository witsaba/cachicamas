/**
 * assistant-config.ts — backwards-compatible wrapper around the new
 * polymorphic /api/archetypes/{slug}/config/ client
 * (frontend/src/lib/api/archetypes.ts, T-21 PR-2 of
 * cachicamas-archetype-system-foundation).
 *
 * The new polymorphic client returns the nested `ArchetypeView`
 * shape from the Go backend (parent + optional `override` block).
 * ConfigureSection and the agents/* routes historically worked on a
 * flat `ArchetypeConfig` shape; this file is a thin adapter that
 * flattens the nested view back to the legacy shape so the call-sites
 * can migrate incrementally.
 *
 * Long-term plan: ConfigureSection + agents/* routes will move to the
 * new client directly (T-23 of cachicamas-archetype-system-foundation
 * migrates the agents/* routes; ConfigureSection migration is on the
 * roadmap). For now, the adapter keeps the existing call-sites working
 * without forcing a one-shot refactor of every component.
 *
 * Endpoint: GET /api/archetypes/assistant/  (per-slug read of the
 * polymorphic view, including the per-org override block when one
 * exists; mapped here to the flat ArchetypeConfig shape).
 *
 * Endpoint: PUT /api/archetypes/assistant/config/  (per-slug write of
 * the per-org override; the body is the same ArchetypeUpdate shape
 * the polymorphic client accepts).
 *
 * The slug is hard-coded to "assistant" because v1 has exactly one
 * user-customisable archetype. Future work that surfaces more slugs
 * (general, owned) will add a parameterised helper and remove this
 * unslug variant.
 */

import {
  archetypeConfigURL,
  archetypeURL,
  getArchetype,
  putArchetypeConfig,
  type ArchetypeView,
} from "~/lib/api/archetypes";
import type { ApiResult } from "~/lib/api";

/**
 * Legacy flat-shape view of an archetype's per-org config. Kept for
 * ConfigureSection + the agents/* routes' existing call-sites. The
 * fields mirror the chat-package's `ArchetypeConfig` (PR-1 of
 * cachicamas-assistant-configuration-ui) — see
 * `backend/agent/src/archetype/config.go:ArchetypeConfig`.
 *
 * Once ConfigureSection migrates to the nested `ArchetypeView` shape
 * directly, this type can be deleted.
 */
export interface ArchetypeConfig {
  kind: "chat";
  org_id: string;
  system_prompt: string;
  tool_allowlist: string[];
  defer_tool_names: string[];
  /** Informational; the actual model is selected by the env-driven
   * runtime, not the persisted config. May be absent. */
  model?: string | null;
  version: number;
  updated_by?: string;
  updated_at?: string;
  /** `true` when the config came from a per-org row that shadows the
   * system default. `false` when the org is on the system default. */
  is_override: boolean;
}

/**
 * Legacy PUT body shape. The polymorphic client's `ArchetypeUpdate` is
 * structurally identical, so this is just a type alias.
 */
export interface AssistantConfigUpdate {
  system_prompt: string;
  tool_allowlist: string[];
  defer_tool_names: string[];
  model?: string | null;
}

// -----------------------------------------------------------------------------
// Adapter: nested ArchetypeView -> flat ArchetypeConfig
// -----------------------------------------------------------------------------

function viewToConfig(view: ArchetypeView): ArchetypeConfig {
  // The polymorphic view carries the per-org override in a nested
  // `override` block (catalog.go:ArchetypeOverride). The flat legacy
  // shape inlines those fields at the top level + flips is_override
  // to the derived `true` (the view itself also carries the derived
  // `is_override` boolean).
  const override = view.override;
  if (override === undefined) {
    // No per-org row: synthesise a flat default view from the
    // system defaults. is_override is false; the tool allowlist /
    // prompt come from the system row in the Go side, but for
    // ConfigureSection we only need a stable shape — fields are
    // taken from the parent's defaults (display_name, tagline) and
    // the version is 1 (the DefaultConfig fallback).
    return {
      kind: "chat",
      org_id: "",
      system_prompt: "",
      tool_allowlist: [],
      defer_tool_names: [],
      model: null,
      version: 1,
      is_override: false,
    };
  }
  return {
    kind: "chat",
    org_id: "",
    system_prompt: override.system_prompt,
    tool_allowlist: [...override.tool_allowlist],
    defer_tool_names: [...override.defer_tool_names],
    model: override.model,
    version: override.version,
    updated_by: override.updated_by,
    updated_at: override.updated_at,
    is_override: true,
  };
}

// -----------------------------------------------------------------------------
// Public client (legacy API surface)
// -----------------------------------------------------------------------------

/**
 * getAssistantConfig fetches the persisted config for the caller's
 * org via the new polymorphic surface, then flattens to the legacy
 * `ArchetypeConfig` shape ConfigureSection and the agents/* routes
 * still consume.
 *
 * The function takes no slug because v1 has only one user-customisable
 * archetype (assistant). When multi-archetype editing lands, the
 * call-sites move to the polymorphic `getArchetype(slug)` directly.
 */
export async function getAssistantConfig(): Promise<ApiResult<ArchetypeConfig>> {
  const result = await getArchetype("assistant");
  if (!result.ok) {
    return result;
  }
  return { ok: true, value: viewToConfig(result.value) };
}

/**
 * putAssistantConfig validates + persists a new per-org config via
 * the new polymorphic surface. The body is the same shape ConfigureSection
 * already constructs.
 *
 * The server's response is the new persisted view; we re-fetch via
 * getAssistantConfig to keep the flat-shape contract (the new view's
 * nested shape needs flattening to match the legacy return).
 */
export async function putAssistantConfig(
  update: AssistantConfigUpdate,
): Promise<ApiResult<ArchetypeConfig>> {
  const result = await putArchetypeConfig("assistant", {
    system_prompt: update.system_prompt,
    tool_allowlist: update.tool_allowlist,
    defer_tool_names: update.defer_tool_names,
    model: update.model ?? null,
  });
  if (!result.ok) {
    return result;
  }
  return { ok: true, value: viewToConfig(result.value) };
}

// -----------------------------------------------------------------------------
// URL constants — exposed for the few call-sites that need to inspect
// the wire URL directly (e.g. test assertions, route debugging).
// -----------------------------------------------------------------------------

/** The polymorphic per-org config URL. */
export const ASSISTANT_CONFIG_ENDPOINT = archetypeConfigURL("assistant");

/** The polymorphic per-slug read URL. */
export const ASSISTANT_GET_ENDPOINT = archetypeURL("assistant");
