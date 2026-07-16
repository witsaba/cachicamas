/**
 * Icon for the Prompts setting tile — pencil over a document.
 *
 * Reference: `sdd/settings-app-grid/{proposal,spec,design}.md` (engram).
 *   - REQ-3 (`stroke="currentColor"` SVG contract — monochrome rule)
 *   - SCN-3.2 (first consumer of SettingCard obeys the contract)
 *
 * Sizing:
 *   - 48×48 width/height (fills the 64×64 SettingCard icon container
 *     with 8px gutters — the container's text color drives the
 *     stroke via `currentColor`).
 *   - 24×24 viewBox (Lucide's native coordinate system).
 *
 * Stroke:
 *   - `stroke-width="1.75"` — between empty-state's 1.5 and
 *     sign-out's 2. Heavy enough to read at 48×48, lighter than
 *     default 2 to feel refined at this scale.
 *   - `stroke="currentColor"` so the icon container's text color
 *     (text-slate-700 → group-hover:text-slate-900) drives the
 *     stroke.
 *
 * A11y:
 *   - `aria-hidden="true"` because the visible "Prompts" label
 *     announces the affordance.
 *   - `focusable="false"` so IE/Edge legacy tab order skips it.
 *
 * Source: Lucide `file-pen` (ISC-licensed).
 *   https://lucide.dev/icons/file-pen
 *   Created v0.68.0; last revised v0.552.0.
 * Path data copied verbatim from
 *   https://raw.githubusercontent.com/lucide-icons/lucide/main/icons/file-pen.svg
 * Any re-syndication requires paired updates to this file AND the
 * spec at `./prompts-icon.spec.tsx` (which pins the d= attrs
 * byte-equal).
 */
import { component$ } from "@builder.io/qwik";

export const PromptsIcon = component$(() => (
  <svg
    xmlns="http://www.w3.org/2000/svg"
    width="48"
    height="48"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    stroke-width="1.75"
    stroke-linecap="round"
    stroke-linejoin="round"
    aria-hidden="true"
    focusable="false"
    data-testid="prompts-icon"
  >
    <path d="M12.659 22H18a2 2 0 0 0 2-2V8a2.4 2.4 0 0 0-.706-1.706l-3.588-3.588A2.4 2.4 0 0 0 14 2H6a2 2 0 0 0-2 2v9.34" />
    <path d="M14 2v5a1 1 0 0 0 1 1h5" />
    <path d="M10.378 12.622a1 1 0 0 1 3 3.003L8.36 20.637a2 2 0 0 1-.854.506l-2.867.837a.5.5 0 0 1-.62-.62l.836-2.869a2 2 0 0 1 .506-.853z" />
  </svg>
));
