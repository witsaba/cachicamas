/**
 * Behavioural spec for `routes/settings/skills/index.tsx`.
 *
 * Reference: `sdd/cachicamas-skills-foundational/{spec,design}` (engram).
 *   - S-FE-1..7: Settings tile → SkillStudio → CRUD flows
 *   - SCN-1: list empty → EmptyState
 *   - SCN-2: list populated → sidebar + editor
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
 * Module-scope loader signal:
 *   Each test sets `testLoaderValue` BEFORE calling `render()`. The
 *   TestWrapper then captures this value into the store's signal
 *   inside its component$ body, which runs BEFORE the route's
 *   useSignal() reads the loader value. This avoids the "first
 *   render reads empty, subsequent renders are ignored" race that
 *   plagues mutations inside the test render body.
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

// ---- Module-scope state (QRL serialization workaround) ---------------------

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
let validationFields: Record<string, string> = {};
let deletedSkillKind: "not_found" | null = null;
/** The value to inject into the route loader for this test. */
let testLoaderValue: unknown = { ok: true as const, skills: [] };

beforeEach(() => {
  createCalls = [];
  updateCalls = [];
  deleteCalls = [];
  restoreCalls = [];
  validationFields = {};
  deletedSkillKind = null;
  testLoaderValue = { ok: true as const, skills: [] };
});

// ---- Mock the api client so we can drive behaviour -------------------------

vi.mock("~/lib/skills-api", () => ({
  listSkills: vi.fn(async () => ({
    ok: true as const,
    value: ((testLoaderValue as { skills?: unknown[] })?.skills ?? []),
  })),
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
  createSkill: vi.fn(
    async (input: { name: string; description: string; body: string }) => {
      createCalls.push(input);
      return {
        ok: true as const,
        value: {
          id: 99,
          name: input.name,
          description: input.description,
          body: input.body,
          current_revision: 1,
          created_at: "2026-07-17T10:00:00Z",
          updated_at: "2026-07-17T10:00:00Z",
          deleted_at: null,
        },
      };
    },
  ),
  updateSkill: vi.fn(
    async (name: string, input: { description: string; body: string }) => {
      updateCalls.push({ name, ...input });
      return {
        ok: true as const,
        value: {
          id: 1,
          name,
          description: input.description,
          body: input.body,
          current_revision: 2,
          created_at: "2026-07-17T10:00:00Z",
          updated_at: "2026-07-17T10:01:00Z",
          deleted_at: null,
        },
      };
    },
  ),
  deleteSkill: vi.fn(async (name: string) => {
    deleteCalls.push(name);
    if (deletedSkillKind === "not_found") {
      return { ok: false, kind: "not_found", message: "Skill has been deleted." };
    }
    return { ok: true as const, value: undefined };
  }),
  listRevisions: vi.fn(async () => ({ ok: true as const, value: [] })),
  restoreRevision: vi.fn(async (name: string, revision: number) => {
    restoreCalls.push({ name, revision });
    return {
      ok: true as const,
      value: {
        id: 1,
        name,
        description: "(restored)",
        body: "---\nname: rest\n---\n",
        current_revision: 99,
        created_at: "2026-07-17T10:00:00Z",
        updated_at: "2026-07-17T10:02:00Z",
        deleted_at: null,
      },
    };
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

/**
 * Single TestWrapper. Reads `testLoaderValue` (a module-scope
 * variable) at the time the wrapper mounts. The test MUST set
 * `testLoaderValue` before calling `render()`.
 */
const TestWrapper = component$(() => {
  // Use the prompts test pattern: signal created empty, then value
  // set after useStore(). Setting inside useStore() (initial value)
  // is being lost in Qwik 1.20 render ordering.
  const loaderState = useStore<Record<string, unknown>>(
    { [loaderId]: useSignal() },
    { deep: false },
  );
  (loaderState[loaderId] as { value: unknown }).value = testLoaderValue;
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

// =========================================================================
// 7.6 — Empty state branch
// =========================================================================

describe("routes/settings/skills — empty state branch (7.6)", () => {
  it("TestSettingsSkillsRoute_RendersEmptyStateWhenListEmpty — shows the EmptyState CTA when the loader returns []", async () => {
    testLoaderValue = { ok: true as const, skills: [] };
    const { screen, render } = await createDOM();
    await render(<TestWrapper />);
    const cta = screen.querySelector(
      '[data-testid="empty-state-create"]',
    ) as HTMLElement | null;
    expect(cta).toBeTruthy();
  });

  it("the empty state CTA invokes handleNewSkill — clicking it switches mode to 'create' and renders SkillEditor", async () => {
    testLoaderValue = { ok: true as const, skills: [] };
    const { screen, render, userEvent } = await createDOM();
    await render(<TestWrapper />);
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

// =========================================================================
// 7.7 — Populated branch
// =========================================================================

const sampleSkill = {
  id: 1,
  name: "pdf-cleanup",
  description: "Cleans PDF files",
  body: "---\nname: pdf-cleanup\ndescription: Cleans PDF files\n---\n# Body",
  current_revision: 1,
  created_at: "2026-07-17T10:00:00Z",
  updated_at: "2026-07-17T10:00:00Z",
  deleted_at: null,
};

describe("routes/settings/skills — populated branch (7.7)", () => {
  it("TestSettingsSkillsRoute_RendersSidebarAndEditorWhenListPopulated — sidebar renders the populated skill list", async () => {
    testLoaderValue = { ok: true as const, skills: [sampleSkill] };
    const { screen, render } = await createDOM();
    await render(<TestWrapper />);
    // Sidebar filter input MUST render when populated.
    const filter = screen.querySelector(
      '[data-testid="skill-sidebar-filter"]',
    ) as HTMLElement | null;
    expect(filter).toBeTruthy();
    // The skill name MUST appear in the sidebar list.
    const list = screen.querySelector(
      '[data-testid="skill-sidebar-list"]',
    ) as HTMLElement | null;
    expect(list).toBeTruthy();
    expect(list?.textContent ?? "").toContain("pdf-cleanup");
  });

  it("clicking a skill in the sidebar selects it — sidebar shows the selected testid", async () => {
    testLoaderValue = { ok: true as const, skills: [sampleSkill] };
    const { screen, render, userEvent } = await createDOM();
    await render(<TestWrapper />);
    const item = screen.querySelector(
      '[data-testid="skill-list-item"]',
    ) as HTMLElement | null;
    expect(item).toBeTruthy();
    await userEvent(item!, "click");
    // After click, the list item becomes 'skill-list-item-selected'.
    const selected = screen.querySelector(
      '[data-testid="skill-list-item-selected"]',
    ) as HTMLElement | null;
    expect(selected).toBeTruthy();
    // The SkillEditor mounts in edit mode for the selected skill.
    const editor = screen.querySelector(
      '[data-testid="skill-editor-description"]',
    ) as HTMLElement | null;
    expect(editor).toBeTruthy();
  });
});