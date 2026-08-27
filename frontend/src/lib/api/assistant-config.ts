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
 * getArchetypeConfig fetches the persisted config for the supplied
 * archetype `slug` via the polymorphic surface, then flattens to the
 * legacy `ArchetypeConfig` shape ConfigureSection and the agents/*
 * routes still consume.
 *
 * The slug is required (feat/archetype-list-endpoint: v1 only
 * surfaces the assistant, but the directory list may include general
 * / owned archetypes; the call-sites always know which slug they
 * belong to).
 */
export async function getArchetypeConfig(
  slug: string,
): Promise<ApiResult<ArchetypeConfig>> {
  const result = await getArchetype(slug);
  if (!result.ok) {
    return result;
  }
  return { ok: true, value: viewToConfig(result.value) };
}

/**
 * @deprecated Use `getArchetypeConfig(slug)` — the slug is required.
 * Kept as a thin alias that hard-codes "assistant" for the v1 default
 * so any out-of-tree consumer that still expects the no-arg shape
 * does not break. Will be removed when ConfigureSection is the sole
 * consumer.
 */
export async function getAssistantConfig(): Promise<ApiResult<ArchetypeConfig>> {
  return getArchetypeConfig("assistant");
}

/**
 * putArchetypeConfig validates + persists a new per-org config for
 * the supplied `slug` via the polymorphic surface.
 */
export async function putArchetypeConfigFlat(
  slug: string,
  update: AssistantConfigUpdate,
): Promise<ApiResult<ArchetypeConfig>> {
  const result = await putArchetypeConfig(slug, {
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

/**
 * @deprecated Use `putArchetypeConfigFlat(slug, update)`.
 * Kept as a thin alias that hard-codes "assistant" for the v1 default.
 * Will be removed when ConfigureSection is the sole consumer.
 */
export async function putAssistantConfig(
  update: AssistantConfigUpdate,
): Promise<ApiResult<ArchetypeConfig>> {
  return putArchetypeConfigFlat("assistant", update);
}

// -----------------------------------------------------------------------------
// URL helpers — exposed for the few call-sites that need to inspect
// the wire URL directly (e.g. test assertions, route debugging).
// -----------------------------------------------------------------------------

/** The polymorphic per-org config URL for the supplied slug. */
export function archetypeConfigURLFor(slug: string): string {
  return archetypeConfigURL(slug);
}
