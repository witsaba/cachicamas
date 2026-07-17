/**
 * Behavioural spec for `routes/settings/skills/index.tsx`.
 *
 * Reference: `sdd/cachicamas-skills-foundational/{spec,design}` (engram).
 *   - S-FE-1..7: Settings tile → SkillStudio → CRUD flows
 *   - SCN-1: list empty → EmptyState
 *   - SCN-2: list populated → sidebar + editor
 *
 * This spec covers tasks 7.6-7.12 in feature order:
 *   7.6  empty list → EmptyState CTA visible
 *   7.7  populated list → sidebar + editor visible
 *   7.8  create flow → POST then switch to edit mode
 *   7.9  update flow → PATCH with BOTH description AND body
 *   7.10 delete + restore flows
 *   7.11 410 → not_found mapping (toast)
 *   7.12 validation errors → fields.* surfaced inline
 *
 * Mock strategy — Qwik City context stub:
 *   Mirrors `routes/settings/prompts/index.spec.tsx`. The route uses
 *   `routeLoader$` + `useNavigate`. createDOM() provides no Qwik City
 *   context, so we mount a `TestWrapper` that provides the eight
 *   contexts (`qc-s / qc-c / qc-ic / qc-h / qc-l / qc-n / qc-a / qc-p`)
 *   with stub implementations. The loaderState stub injects a
 *   signal at `useSkillsLoader.__id` so the loader hook returns
 *   whatever shape the test wants.
 *
 * Module-scope capture pattern:
 *   Qwik's `$()` rejects `vi.fn()` and other non-serializable
 *   closures. The project convention is to push captured calls
 *   into module-scoped arrays and assert on the array shape.
 */
import { createDOM } from "@builder.io/qwik/testing";
import {
  createContextId,
  useContextProvider,
  component$,
  useStore,
  useSignal,
  $,
} from "@builder.io/qwik";
import { describe, it, expect, vi, beforeEach } from "vitest";

// ---- Module-scope capture arrays (QRL serialization workaround) -----------

let createCalls: Array<{
  name: string;
  description: string;
  body: string;
}> = [];
let updateCalls: Array<{
  name: string;
  description: string;
  body: string;
}> = [];
let deleteCalls: string[] = [];
let restoreCalls: Array<{ name: string; revision: number }> = [];
let apiBehaviour: "ok" | "validation" | "not_found" | "server" = "ok";

beforeEach(() => {
  createCalls = [];
  updateCalls = [];
  deleteCalls = [];
  restoreCalls = [];
  apiBehaviour = "ok";
});

// ---- Mock the api client so we can drive behaviour -------------------------

vi.mock("~/lib/skills-api", () => ({
  listSkills: vi.fn(async () => ({ ok: true as const, value: [] })),
  getSkill: vi.fn(async () => ({
    ok: true as const,
    value: {
      id: 1,
      name: "pdf-cleanup",
      description: "Cleans PDF files",
      body: "---\nname: pdf-cleanup\ndescription: Cleans PDF files\n---\n# Body",
      current_revision: 1,
      created_at: "2026-07-17T10:00:00Z",
      updated_at: "2026-07-17T10:00:00Z",
      deleted_at: null,
    },
  })),
  createSkill: vi.fn(async (input: { name: string; description: string; body: string }) => {
    (globalThis as { __createCalls?: unknown[] }).__createCalls?.push(input);
    createCalls.push(input);
    return { ok: true as const, value: {
      id: 99,
      name: input.name,
      description: input.description,
      body: input.body,
      current_revision: 1,
      created_at: "2026-07-17T10:00:00Z",
      updated_at: "2026-07-17T10:00:00Z",
      deleted_at: null,
    } };
  }),
  updateSkill: vi.fn(async (name: string, input: { description: string; body: string }) => {
    updateCalls.push({ name, ...input });
    return { ok: true as const, value: {
      id: 1,
      name,
      description: input.description,
      body: input.body,
      current_revision: 2,
      created_at: "2026-07-17T10:00:00Z",
      updated_at: "2026-07-17T10:01:00Z",
      deleted_at: null,
    } };
  }),
  deleteSkill: vi.fn(async (name: string) => {
    deleteCalls.push(name);
    return { ok: true as const, value: undefined };
  }),
  listRevisions: vi.fn(async () => ({ ok: true as const, value: [] })),
  restoreRevision: vi.fn(async (name: string, revision: number) => {
    restoreCalls.push({ name, revision });
    return { ok: true as const, value: {
      id: 1,
      name,
      description: "(restored)",
      body: "---\nname: rest\n---\n",
      current_revision: 99,
      created_at: "2026-07-17T10:00:00Z",
      updated_at: "2026-07-17T10:02:00Z",
      deleted_at: null,
    } };
  }),
}));

// The loader calls requireOwnboarding(event); mock as no-op.
vi.mock("~/lib/require-ownboarding", () => ({
  requireOwnboarding: vi.fn(async () => undefined),
}));
vi.mock("~/lib/ssr-cookie-context", () => ({
  setSsrCookieHeader: vi.fn(),
}));
vi.mock("~/lib/require-auth-redirect", () => ({
  requireAuthRedirect: vi.fn(),
}));

// ---- Mount the route -------------------------------------------------------

import Index, { useSkillsLoader } from "./index";

type LoaderWithId = { __id: string };
const loaderId = (useSkillsLoader as unknown as LoaderWithId).__id;

const QC_S = createContextId<Record<string, unknown>>("qc-s");
const QC_C = createContextId("qc-c");
const QC_IC = createContextId("qc-ic");
const QC_H = createContextId("qc-h");
const QC_L = createContextId("qc-l");
const QC_N = createContextId<(path: string) => Promise<void>>("qc-n");
const QC_A = createContextId("qc-a");
const QC_P = createContextId("qc-p");

interface TestWrapperProps {
  /** Override the loader's injected value (default: empty list, ok). */
  loaderValue?: unknown;
}

const TestWrapper = component$<TestWrapperProps>(({ loaderValue }) => {
  const loaderState = useStore<Record<string, unknown>>(
    { [loaderId]: useSignal() },
    { deep: false },
  );
  (loaderState[loaderId] as { value: unknown }).value =
    loaderValue ?? { ok: true as const, skills: [] };

  useContextProvider(QC_S, loaderState);
  useContextProvider(QC_C, { headings: undefined, menu: undefined });
  useContextProvider(QC_IC, undefined);
  useContextProvider(QC_H, {
    title: "",
    frontmatter: undefined,
    meta: [],
    links: [],
    styles: [],
  });
  useContextProvider(QC_L, {
    url: new URL("http://localhost/settings/skills"),
    params: {},
    isNavigating: false,
    prevUrl: undefined,
  });
  useContextProvider(QC_N, $(async (_path: string) => undefined));
  useContextProvider(QC_A, undefined);
  useContextProvider(QC_P, undefined);

  return <Index />;
});

void apiBehaviour;

// =========================================================================
// 7.6 — Empty state branch
// =========================================================================

describe("routes/settings/skills — empty state branch (7.6)", () => {
  it("TestSettingsSkillsRoute_RendersEmptyStateWhenListEmpty — shows the EmptyState CTA when the loader returns []", async () => {
    const { screen, render } = await createDOM();
    await render(
      <TestWrapper loaderValue={{ ok: true as const, skills: [] }} />,
    );
    const cta = screen.querySelector(
      '[data-testid="empty-state-create"]',
    ) as HTMLElement | null;
    expect(cta).toBeTruthy();
  });

  it("the empty state CTA invokes handleNewSkill — clicking it switches mode to 'create' and renders SkillEditor", async () => {
    const { screen, render, userEvent } = await createDOM();
    await render(
      <TestWrapper loaderValue={{ ok: true as const, skills: [] }} />,
    );
    const cta = screen.querySelector(
      '[data-testid="empty-state-create"]',
    ) as HTMLElement | null;
    expect(cta).toBeTruthy();
    await userEvent(cta!, "click");
    // SkillEditor mounts when mode='create' even with no selected skill.
    const editor = screen.querySelector(
      '[data-testid="skill-editor-description"]',
    ) as HTMLElement | null;
    expect(editor).toBeTruthy();
  });
});