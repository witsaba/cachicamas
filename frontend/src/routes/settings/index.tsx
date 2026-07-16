/**
 * /settings — Settings app launcher (Launchpad-style grid).
 *
 * Reference: `sdd/settings-app-grid/{proposal,spec,design}.md` (engram).
 *   - REQ-10 (canonical guard chain + tile grid)
 *   - REQ-11 (headerless — tile grid IS the visual identifier;
 *     brand chrome at the root layout provides context)
 *   - SCN-10.1 / SCN-10.2 / SCN-11.1
 *
 * Renders: a 2-col (mobile) / 3-col (`sm`) / 4-col (`md`) grid of
 * `<SettingCard>` tiles. One tile today (Prompts); future tiles
 * (Profile, Billing, Notifications, ...) slot in without any layout
 * change.
 *
 * Guard chain (canonical — mirrors `routes/settings/prompts/index.tsx`):
 *   1. `setSsrCookieHeader` — capture the inbound request cookie so
 *      the api.ts fetch helpers can forward it during SSR.
 *   2. `requireAuthRedirect` — 302 to `/auth/signin` if anonymous.
 *   3. `requireOwnboarding` — 302 to `/ownboarding` if no org.
 * The order matters: the SSR cookie capture MUST happen before any
 * guard throws, otherwise downstream api.ts fetches miss the cookie.
 *
 * Headerless (design §7): no `<h1>`. The grid is the visual
 * identifier. If a future change wants `<h1>Settings</h1>`, the
 * addition is ~5 lines.
 */
import { component$ } from "@builder.io/qwik";
import { type DocumentHead, type RequestHandler } from "@builder.io/qwik-city";
import { setSsrCookieHeader } from "~/lib/ssr-cookie-context";
import { requireAuthRedirect } from "~/lib/require-auth-redirect";
import { requireOwnboarding } from "~/lib/require-ownboarding";
import { SettingCard } from "~/components/ui/setting-card/setting-card";
import { PromptsIcon } from "./icons/prompts-icon";

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
  <main class="mx-auto max-w-3xl px-4 py-16">
    <div
      class="grid grid-cols-2 gap-6 sm:grid-cols-3 md:grid-cols-4"
      data-testid="settings-grid"
    >
      <SettingCard
        href="/settings/prompts"
        label="Prompts"
        icon={<PromptsIcon />}
        testId="settings-card-prompts"
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
