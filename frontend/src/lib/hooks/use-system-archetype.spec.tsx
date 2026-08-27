/**
 * use-system-archetype.spec.ts — TDD contract for the Qwik hook
 * `useSystemArchetype(slug)` (T-23 of cachicamas-archetype-system-foundation).
 *
 * Two units under test:
 *   - `syntheticFallbackView(agent)` — pure helper that builds an
 *     ArchetypeView from a static AGENTS literal. Tested directly so
 *     the spec does not need a Qwik DOM to verify the wire shape.
 *   - `resolveSystemArchetype(slug)` — the async resolver the hook
 *     wraps. Driven by a `globalThis.fetch` mock; covers the success
 *     path, the fetch-failure → AGENTS fallback path, and the
 *     unknown-slug → null path.
 *
 * The hook itself is a one-liner over `useResource$(resolveSystemArchetype)`;
 * a smoke render via `createDOM` proves the wrapper does not throw and
 * the Qwik resource plumbing accepts it.
 */

import { component$ } from "@builder.io/qwik";
import { createDOM } from "@builder.io/qwik/testing";
import { afterEach, describe, expect, it, vi } from "vitest";

import { archetypeURL } from "~/lib/api/archetypes";
import type { ArchetypeView } from "~/lib/api/archetypes";
import { AGENTS } from "~/lib/mock/staff";

import {
  resolveSystemArchetype,
  syntheticFallbackView,
  useSystemArchetype,
} from "./use-system-archetype";

const originalFetch = globalThis.fetch;

// -----------------------------------------------------------------------------
// Polymorphic ArchetypeView fixtures (the API's wire shape)
// -----------------------------------------------------------------------------

function assistantView(overrides: Partial<ArchetypeView> = {}): ArchetypeView {
  return {
    slug: "assistant",
    type: "system",
    display_name: "Assistant",
    tagline: "The colleague everyone talks to first.",
    status: "active",
    archived_at: null,
    created_at: "2026-08-27T10:00:00Z",
    created_by: "seed",
    bundle_version: "v1",
    is_critical: true,
    is_override: true,
    override: {
      system_prompt: "you are a helpful assistant",
      tool_allowlist: ["current_time", "summarize_conversation"],
      defer_tool_names: ["summarize_conversation"],
      model: null,
      version: 5,
      updated_at: "2026-08-27T10:00:00Z",
      updated_by: "user_alice",
    },
    ...overrides,
  };
}

function jsonResponse(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "content-type": "application/json" },
  });
}

// -----------------------------------------------------------------------------
// syntheticFallbackView: pure shape coverage (no Qwik DOM needed)
// -----------------------------------------------------------------------------

describe("syntheticFallbackView", () => {
  it("produces a valid ArchetypeView from the assistant AGENTS literal", () => {
    const fallback = AGENTS.find((a) => a.slug === "assistant");
    expect(fallback).toBeDefined();
    if (!fallback) return;
    const view = syntheticFallbackView(fallback);
    expect(view.slug).toBe("assistant");
    expect(view.type).toBe("system");
    expect(view.display_name).toBe(fallback.name);
    expect(view.tagline).toBe(fallback.tagline);
    expect(view.status).toBe("active");
    expect(view.bundle_version).toBe("v1");
    expect(view.is_critical).toBe(true);
    expect(view.is_override).toBe(false);
    expect(view.override).toBeUndefined();
  });

  it("sets is_critical only for the assistant slug", () => {
    const fallback = AGENTS.find((a) => a.slug === "assistant")!;
    const view = syntheticFallbackView(fallback);
    expect(view.is_critical).toBe(true);
    // A future non-critical slug stays non-critical.
    const view2 = syntheticFallbackView({
      slug: "future",
      name: "Future",
      tagline: "Not here yet",
    });
    expect(view2.is_critical).toBe(false);
  });

  it("does NOT carry per-org state (no override block)", () => {
    // The fallback is the "no signal" case: callers must not assume a
    // per-org override exists when the API was unreachable.
    const fallback = AGENTS.find((a) => a.slug === "assistant")!;
    const view = syntheticFallbackView(fallback);
    expect(view.override).toBeUndefined();
    expect(view.is_override).toBe(false);
  });
});

// -----------------------------------------------------------------------------
// resolveSystemArchetype: fetch-driven behaviour (mocked globalThis.fetch)
// -----------------------------------------------------------------------------

describe("resolveSystemArchetype", () => {
  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  it("Test_useSystemArchetype_ReturnsArchetypeView_OnSuccess", async () => {
    const view = assistantView({
      display_name: "Live Assistant",
      tagline: "Polymorphic tagline from API",
    });
    const fetchMock = vi.fn(async () => jsonResponse(200, view));
    globalThis.fetch = fetchMock as unknown as typeof fetch;

    const resolved = await resolveSystemArchetype("assistant");

    expect(fetchMock).toHaveBeenCalledWith(
      archetypeURL("assistant"),
      expect.objectContaining({ method: "GET", credentials: "include" }),
    );
    expect(resolved).not.toBeNull();
    expect(resolved?.slug).toBe("assistant");
    expect(resolved?.display_name).toBe("Live Assistant");
    expect(resolved?.tagline).toBe("Polymorphic tagline from API");
    expect(resolved?.is_override).toBe(true);
  });

  it("Test_useSystemArchetype_FallsBackToAGENTS_OnFetchError", async () => {
    const fetchMock = vi.fn(async () => {
      throw new Error("connection refused");
    });
    globalThis.fetch = fetchMock as unknown as typeof fetch;

    const resolved = await resolveSystemArchetype("assistant");

    expect(resolved).not.toBeNull();
    const fallback = AGENTS.find((a) => a.slug === "assistant")!;
    expect(resolved?.slug).toBe("assistant");
    expect(resolved?.display_name).toBe(fallback.name);
    expect(resolved?.tagline).toBe(fallback.tagline);
    expect(resolved?.is_override).toBe(false);
    // The fallback does not carry per-org state.
    expect(resolved?.override).toBeUndefined();
  });

  it("returns null when the slug is unknown AND the fetch failed", async () => {
    const fetchMock = vi.fn(async () => {
      throw new Error("network unreachable");
    });
    globalThis.fetch = fetchMock as unknown as typeof fetch;

    const resolved = await resolveSystemArchetype("not-a-colleague");
    expect(resolved).toBeNull();
  });

  it("propagates a successful API response verbatim (no AGENTS overlay)", async () => {
    const view = assistantView({
      display_name: "Server-authoritative name",
      tagline: "Server-authoritative tagline",
      is_override: false,
    });
    const fetchMock = vi.fn(async () => jsonResponse(200, view));
    globalThis.fetch = fetchMock as unknown as typeof fetch;

    const resolved = await resolveSystemArchetype("assistant");

    expect(resolved?.display_name).toBe("Server-authoritative name");
    expect(resolved?.tagline).toBe("Server-authoritative tagline");
    // The AGENTS tagline must NOT bleed through.
    expect(resolved?.tagline).not.toBe(
      AGENTS.find((a) => a.slug === "assistant")!.tagline,
    );
    expect(resolved?.is_override).toBe(false);
  });
});

// -----------------------------------------------------------------------------
// Smoke coverage of the hook wrapper (proves useResource$ accepts it)
// -----------------------------------------------------------------------------

const Smoke = component$<{ slug: string }>(({ slug }) => {
  // The hook wrapper is a one-liner over `resolveSystemArchetype`; the
  // deep behaviour is covered above. This smoke test only proves the
  // resource can be wired into a component without Qwik throwing.
  const arch = useSystemArchetype(slug);
  void arch;
  return <div data-testid="smoke-ok">ok</div>;
});

describe("useSystemArchetype (hook wrapper smoke)", () => {
  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  it("accepts the hook as a useResource$ body (no throw on mount)", async () => {
    globalThis.fetch = vi.fn(async () =>
      jsonResponse(200, assistantView()),
    ) as unknown as typeof fetch;

    const { screen, render } = await createDOM();
    await render(<Smoke slug="assistant" />);
    expect(screen.querySelector('[data-testid="smoke-ok"]')).toBeTruthy();
  });
});
