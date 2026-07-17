/**
 * /settings — Settings app launcher (Launchpad-style grid).
 *
 * Reference: `sdd/settings-app-grid/{proposal,spec,design}.md` (engram #1922).
 *   - REQ-10 (canonical guard chain + tile grid)
 *   - REQ-11 (page identity section — REVISED 2026-07-16, was: headerless)
 *   - REQ-17 (layout centering — NEW 2026-07-16)
 *   - SCN-10.1 / SCN-10.2 / SCN-11.1 (revised) / SCN-17.1
 *
 * Renders: a 2-col (mobile) / 3-col (`sm`) / 4-col (`md`) grid of
 * `<SettingCard>` tiles, preceded by a small page identity section
 * (`<h1>` + `<p data-testid="settings-subtitle">`) and centered
 * vertically + horizontally inside the root-layout viewport. One
 * tile today (Prompts); future tiles (Profile, Billing,
 * Notifications, ...) slot in without any layout change.
 *
 * Guard chain (canonical — mirrors `routes/settings/prompts/index.tsx`):
 *   1. `setSsrCookieHeader` — capture the inbound request cookie so
 *      the api.ts fetch helpers can forward it during SSR.
 *   2. `requireAuthRedirect` — 302 to `/auth/signin` if anonymous.
 *   3. `requireOwnboarding` — 302 to `/ownboarding` if no org.
 * The order matters: the SSR cookie capture MUST happen before any
 * guard throws, otherwise downstream api.ts fetches miss the cookie.
 *
 * Page identity (REQ-11, REVISED 2026-07-16): a `<h1>Settings</h1>`
 * + subtitle band sits above the tile grid so the page reads as
 * "settings home" rather than as an orphan Prompts tile in a desert.
 * The original C1 (headerless) decision was falsified by UAT on
 * 2026-07-16 — the URL alone does not orient the user.
 *
 * Positioning (REQ-17, REVISED 2026-07-16 rev 2, was: centering):
 * `<main>` is `w-full max-w-3xl px-4 py-8` so the content block
 * sits near the top of the viewport, just below the root-layout
 * navbar. Vertical centering was dropped because it pushed content
 * ~150px below the navbar (UAT immediately after rev 1 ship). The
 * grid STILL carries `justify-items-center` so a single tile centers
 * in its cell.
 */
import { component$ } from "@builder.io/qwik";
import { type DocumentHead, type RequestHandler } from "@builder.io/qwik-city";
import { setSsrCookieHeader } from "~/lib/ssr-cookie-context";
import { requireAuthRedirect } from "~/lib/require-auth-redirect";
import { requireOwnboarding } from "~/lib/require-ownboarding";
import { SettingCard } from "~/components/ui/setting-card/setting-card";
import { PromptsIcon } from "./icons/prompts-icon";
import { SkillsIcon } from "./icons/skills-icon";

// ---------------------------------------------------------------------------
// Request guard — cookie capture + auth + ownboarding (canonical chain)
// ---------------------------------------------------------------------------

export const onRequest: RequestHandler = async (event) => {
  setSsrCookieHeader(event.request.headers.get("cookie") ?? "");
  requireAuthRedirect(event);
  await requireOwnboarding(event);
};

// ---------------------------------------------------------------------------
// Component — Launchpad-style grid
// ---------------------------------------------------------------------------

export default component$(() => (
  <main class="mx-auto w-full max-w-3xl px-4 py-8">
    {/*
      REQ-11 (revised 2026-07-16, was: headerless). UAT showed the
      page reads as a "desert" without visual identity — the URL is
      not enough. The h1 + subtitle orient the user. The tile grid
      remains the primary interaction surface.
    */}
    <header class="mb-10 text-center">
      <h1 class="text-2xl font-semibold tracking-tight text-slate-900">
        Settings
      </h1>
      <p
        class="mt-2 text-sm text-slate-500"
        data-testid="settings-subtitle"
      >
        Customize your workspace.
      </p>
    </header>
    {/*
      REQ-17 (revised 2026-07-16 rev 2). <main> drops vertical
      centering so content sits near the top of the viewport;
      justify-items-center on the grid still centers a single tile
      in its cell rather than hugging the left edge.
    */}
    <div
      class="grid w-full grid-cols-2 justify-items-center gap-6 sm:grid-cols-3 md:grid-cols-4"
      data-testid="settings-grid"
    >
      <SettingCard
        href="/settings/prompts"
        label="Prompts"
        icon={<PromptsIcon />}
        testId="settings-card-prompts"
      />
      <SettingCard
        href="/settings/skills"
        label="Skills"
        icon={<SkillsIcon />}
        testId="settings-card-skills"
      />
      {/*
        Future tiles (Profile, Billing, Notifications, ...) slot in
        here as additional <SettingCard .../> children. Grid layout,
        key, and testId naming convention stay the same.
      */}
    </div>
  </main>
));

// ---------------------------------------------------------------------------
// Document head
// ---------------------------------------------------------------------------

export const head: DocumentHead = {
  title: "Settings — Cachicamas",
};
