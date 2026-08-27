/**
 * use-system-archetype.ts — Qwik hook that resolves a single system
 * archetype by slug into a polymorphic ArchetypeView, with a static
 * AGENTS fallback when the live fetch fails.
 *
 * T-23 of cachicamas-archetype-system-foundation (PR-2) replaces the
 * per-archetype ad-hoc fetches on `/agents/*` with this hook so the
 * directory's Assistant card can render the live `display_name` +
 * `tagline` + `is_override` signal in one place. When the fetch fails
 * (offline, anonymous, server error, unknown slug) the hook falls back
 * to the AGENTS literal — the same fallback the chat-package used
 * pre-PR-1 — so a stale client still renders something readable.
 *
 * The fallback shape is constructed carefully: the polymorphic view's
 * required fields (`slug`, `type`, `display_name`, `tagline`, `status`,
 * `is_override`, `created_at`, `created_by`) are populated; the
 * `override` block is absent (the AGENTS literal carries no per-org
 * state); `bundle_version` and `is_critical` are populated for the
 * `assistant` slug only.
 *
 * Returning `null` (rather than the fallback) is reserved for the
 * "slug is not in AGENTS AND the fetch failed" case — i.e. an
 * unregistered slug. The route loaders treat null as a real 404.
 *
 * The fetch + fallback logic is split into `resolveSystemArchetype(slug)`
 * so the spec can drive it with a mocked `globalThis.fetch` without
 * going through the Qwik resource plumbing (which is hard to await in
 * `createDOM`). The hook itself is a thin wrapper.
 */

import { useResource$ } from "@builder.io/qwik";

import { getArchetype, type ArchetypeView } from "~/lib/api/archetypes";
import { AGENTS } from "~/lib/mock/staff";

/**
 * useSystemArchetype(slug) returns a Qwik Resource that resolves to
 * the polymorphic view for the slug, or to the AGENTS-derived fallback
 * when the fetch fails AND the slug is registered in AGENTS, or to
 * `null` when the slug is unknown AND the fetch failed.
 *
 * Call from inside a `routeLoader$` (or any component / hook context)
 * — `useResource$` requires the Qwik component context.
 */
export function useSystemArchetype(slug: string) {
  return useResource$<ArchetypeView | null>(() => resolveSystemArchetype(slug));
}

/**
 * resolveSystemArchetype(slug) is the pure async resolver that
 * `useSystemArchetype` wraps. Exposed for the spec — the hook is a
 * one-liner; the interesting behaviour (fetch, fallback, null-on-
 * unknown) lives here and can be driven by a `fetch` mock without
 * going through Qwik's resource scheduler.
 */
export async function resolveSystemArchetype(
  slug: string,
): Promise<ArchetypeView | null> {
  const result = await getArchetype(slug);
  if (result.ok) {
    return result.value;
  }
  const fallback = AGENTS.find((a) => a.slug === slug);
  if (!fallback) {
    return null;
  }
  return syntheticFallbackView(fallback);
}

/**
 * syntheticFallbackView produces an ArchetypeView from a static AGENTS
 * literal. Exported for the spec — the hook delegates to it so the
 * shape can be tested independently of the Qwik resource plumbing.
 *
 * No `override` block: the AGENTS literal carries no per-org state.
 * Callers that need a per-org config from a fallback must derive it
 * locally (the assistant-config.ts adapter already handles this case
 * for the ConfigureSection; the directory treats `is_override=false`
 * as "Default" without inspecting the override block).
 */
export function syntheticFallbackView(fallback: {
  slug: string;
  name: string;
  tagline: string;
}): ArchetypeView {
  return {
    slug: fallback.slug,
    type: "system",
    display_name: fallback.name,
    tagline: fallback.tagline,
    status: "active",
    archived_at: null,
    created_at: new Date(0).toISOString(),
    created_by: "system",
    bundle_version: "v1",
    is_critical: fallback.slug === "assistant",
    is_override: false,
  };
}
