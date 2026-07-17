/**
 * /settings/skills — Skill Studio.
 *
 * Reference: `sdd/cachicamas-skills-foundational/{spec,design}` (engram).
 *   - S-FE-1..7: Settings tile → SkillStudio → CRUD flows
 *   - design §4.2: route-level signals (mode, selectedName, currentSkill,
 *     currentRevisions, editorSaving, editorError)
 *   - design §4.3: handleSave sends BOTH description AND body in PATCH
 *     (anti-drift from obs #1959 item 4)
 *
 * Renders: SkillStudio — split-panel layout with sidebar + editor
 * (mirrors the prompts route at /settings/prompts/index.tsx).
 *
 * States (PR2c covers empty + populated; full CRUD flows land in 7.8-7.10):
 *   loading   → <p>Loading…</p>
 *   error     → error alert with retry
 *   empty     → EmptyState (no skills)
 *   loaded    → SkillSidebar + SkillEditor
 *
 * This skeleton (task 7.5) ships the routeLoader$ + onRequest chain.
 * Subsequent tasks add: empty/populated branches (7.6/7.7), handleCreate
 * (7.8), handleUpdate (7.9), handleDelete + handleRestore (7.10),
 * 410 → not_found mapping (7.11), validation errors inline (7.12).
 *
 * Navigation:
 *   - The `← Back` affordance uses `window.history.back()` with a
 *     `/settings` fallback for deep-link entries. NOT a hardcoded
 *     `<Link href="...">` — semantic actions survive navigation
 *     flow changes (deep links, bookmarks, cross-navigation).
 */

import { $, component$, useSignal } from "@builder.io/qwik";
import {
  routeLoader$,
  useNavigate,
  type DocumentHead,
  type RequestHandler,
} from "@builder.io/qwik-city";
import {
  listSkills,
  type Skill,
  type SkillRevision,
} from "~/lib/skills-api";
import { setSsrCookieHeader } from "~/lib/ssr-cookie-context";
import { requireAuthRedirect } from "~/lib/require-auth-redirect";
import { requireOwnboarding } from "~/lib/require-ownboarding";

// ---------------------------------------------------------------------------
// Request guard — cookie capture + auth + ownboarding
// ---------------------------------------------------------------------------

export const onRequest: RequestHandler = async (event) => {
  setSsrCookieHeader(event.request.headers.get("cookie") ?? "");
  requireAuthRedirect(event);
  await requireOwnboarding(event);
};

// ---------------------------------------------------------------------------
// Route loader — SSR initial skill list
// ---------------------------------------------------------------------------

export const useSkillsLoader = routeLoader$(async (event) => {
  await requireOwnboarding(event);
  const result = await listSkills();
  if (result.ok) return { ok: true as const, skills: result.value };
  return { ok: false as const, message: result.message };
});

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

export default component$(() => {
  const skillsLoader = useSkillsLoader();

  // Skills list (from SSR loader, updated client-side)
  const skills = useSignal<Skill[]>(
    skillsLoader.value.ok ? skillsLoader.value.skills : [],
  );
  const loaderError = useSignal<string | null>(
    skillsLoader.value.ok ? null : skillsLoader.value.message,
  );
  const loading = useSignal(false);

  // Editor state (signals declared; full wires land in 7.6-7.12)
  const mode = useSignal<"list" | "edit" | "create">("list");
  const selectedName = useSignal<string | null>(null);
  const currentSkill = useSignal<Skill | null>(null);
  const currentRevisions = useSignal<SkillRevision[]>([]);
  const editorSaving = useSignal(false);
  const editorError = useSignal<string | null>(null);

  // Real-history back: navigate to wherever the user came from.
  // Falls back to /settings (the URL hierarchy parent) for
  // deep-link / new-tab entries that have no history.
  const nav = useNavigate();
  const handleBack = $(() => {
    if (typeof window === "undefined") return; // SSR guard
    if (window.history.length > 1) {
      window.history.back();
    } else {
      nav("/settings");
    }
  });

  // -----------------------------------------------------------------------
  // Render — skeleton; full UI branches land in tasks 7.6-7.10.
  // -----------------------------------------------------------------------

  if (loading.value) {
    return (
      <main class="mx-auto max-w-3xl px-4 py-16">
        <p data-testid="skill-studio-loading">Loading…</p>
      </main>
    );
  }

  if (loaderError.value && skills.value.length === 0) {
    return (
      <main class="mx-auto max-w-3xl px-4 py-16">
        <div
          role="alert"
          class="rounded border border-red-300 bg-red-50 px-4 py-3 text-red-800"
          data-testid="skill-studio-error"
        >
          {loaderError.value}
        </div>
      </main>
    );
  }

  void mode.value;
  void selectedName.value;
  void currentSkill.value;
  void currentRevisions.value;
  void editorSaving.value;
  void editorError.value;
  void handleBack;
  void skillsLoader;

  return (
    <main class="mx-auto flex max-w-5xl flex-col px-4 py-8" data-testid="skill-studio-shell">
      <p data-testid="skill-studio-skeleton">Skill Studio (skeleton)</p>
    </main>
  );
});

export const head: DocumentHead = {
  title: "Skills — Settings — Cachicamas",
};