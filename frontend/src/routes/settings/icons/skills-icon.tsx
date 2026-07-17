/**
 * Icon for the Skills setting tile — book with a bookmark (Lucide `book-marked`).
 *
 * Reference: `sdd/cachicamas-skills-foundational/{proposal,spec,design}` (engram).
 *   - §4.8 design: Lucide `book-marked` for the Skills tile
 *   - REQ-3 (`stroke="currentColor"` SVG contract — monochrome rule)
 *
 * Sizing:
 *   - 48×48 width/height (fills the 64×64 SettingCard icon container
 *     with 8px gutters — the container's text color drives the
 *     stroke via `currentColor`).
 *   - 24×24 viewBox (Lucide's native coordinate system).
 *
 * Stroke:
 *   - `stroke-width="1.75"` — matches `prompts-icon.tsx` for visual
 *     consistency across the settings grid.
 *   - `stroke="currentColor"` so the icon container's text color
 *     drives the stroke.
 *
 * A11y:
 *   - `aria-hidden="true"` because the visible "Skills" label
 *     announces the affordance.
 *   - `focusable="false"` so IE/Edge legacy tab order skips it.
 *
 * Source: Lucide `book-marked` (ISC-licensed).
 *   https://lucide.dev/icons/book-marked
 * Path data copied verbatim from
 *   https://raw.githubusercontent.com/lucide-icons/lucide/main/icons/book-marked.svg
 * Any re-syndication requires paired updates to this file AND the
 * spec at `./skills-icon.spec.tsx` (which pins the d= attrs
 * byte-equal).
 *
 * Deviation note (PR2a): the orchestrator task spec mentioned "3
 * path elements", but the upstream Lucide book-marked SVG ships
 * with **2** paths. The load-bearing constraint is "byte-equal to
 * Lucide", so this file uses the real upstream count. The spec
 * asserts path.length === 2.
 */
import { component$ } from "@builder.io/qwik";

/**
 * Verbatim Lucide `book-marked` path data (locked).
 * Two paths: 1) bookmark/star at top of book, 2) book outline with
 * folded binding on the bottom-right.
 */
const LUCIDE_PATH_1 = "M10 2v8l3-3 3 3V2";
const LUCIDE_PATH_2 =
  "M4 19.5v-15A2.5 2.5 0 0 1 6.5 2H19a1 1 0 0 1 1 1v18a1 1 0 0 1-1 1H6.5a1 1 0 0 1 0-5H20";

export const SkillsIcon = component$(() => (
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
    data-testid="skills-icon"
  >
    <path d={LUCIDE_PATH_1} />
    <path d={LUCIDE_PATH_2} />
  </svg>
));
