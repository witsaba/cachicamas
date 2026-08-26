/**
 * WaspSpinner — the cachicama wasp replaces the green "Working now"
 * dot in the chat header while the assistant is streaming, submitting,
 * or cancelling.
 *
 * The body is static; only the wings flap. Pure CSS keyframes — no
 * JavaScript loop, no requestAnimationFrame, no hydration cost. The
 * animation stops the instant the indicator leaves the DOM.
 *
 * Mount this ONLY while the assistant is actively working. The parent
 * decides that; the spinner has no internal state.
 *
 * Reduced-motion users get a static wasp (handled globally in
 * global.css via `prefers-reduced-motion`).
 */
import { component$ } from "@builder.io/qwik";

export interface WaspSpinnerProps {
  /** Extra class applied to the root svg. Use it to size and position. */
  readonly class?: string;
}

export const WaspSpinner = component$<WaspSpinnerProps>(
  ({ class: className }) => (
    <svg
      role="img"
      aria-label="Assistant is working"
      data-testid="work-wasp"
      viewBox="0 0 32 32"
      class={["wasp-spinner", className]}
      xmlns="http://www.w3.org/2000/svg"
    >
      {/* Legs (drawn first so they sit beneath the body) */}
      <g stroke="#1e3a8a" stroke-width="0.7" stroke-linecap="round" fill="none">
        {/* Front pair — from the shoulders */}
        <path d="M13.6,11 L9.5,7.5" />
        <path d="M18.4,11 L22.5,7.5" />
        {/* Middle pair */}
        <path d="M13,12.5 L7,11.5" />
        <path d="M19,12.5 L25,11.5" />
        {/* Rear pair — from the abdomen */}
        <path d="M13.5,22 L8,24" />
        <path d="M18.5,22 L24,24" />
      </g>

      {/* Wings — the only animated parts. Pivots from the
          shoulders, flapped by CSS keyframes in global.css. */}
      <path
        class="wasp-wing-left"
        d="M12,11 Q5,11 3,17 Q4.5,22 12,19 Z"
        fill="rgba(70,130,220,0.5)"
        stroke="rgba(140,190,255,0.85)"
        stroke-width="0.5"
        stroke-linejoin="round"
      />
      <path
        class="wasp-wing-right"
        d="M20,11 Q27,11 29,17 Q27.5,22 20,19 Z"
        fill="rgba(70,130,220,0.5)"
        stroke="rgba(140,190,255,0.85)"
        stroke-width="0.5"
        stroke-linejoin="round"
      />

      {/* Petiole (narrow waist between thorax and abdomen) */}
      <rect x="15" y="15" width="2" height="2.5" fill="#0f172a" />

      {/* Abdomen (gaster) */}
      <path
        d="M16,17.5 C 11.5,20 11.5,25 16,28.5 C 20.5,25 20.5,20 16,17.5 Z"
        fill="#1e3a8a"
        stroke="#3b82f6"
        stroke-width="0.4"
      />

      {/* Stinger */}
      <path d="M15.4,28.5 L16.6,28.5 L16,30 Z" fill="#991b1b" />

      {/* Thorax */}
      <ellipse
        cx="16"
        cy="12"
        rx="3.4"
        ry="2.6"
        fill="#1e3a8a"
        stroke="#3b82f6"
        stroke-width="0.4"
      />

      {/* Head */}
      <ellipse
        cx="16"
        cy="7.5"
        rx="3"
        ry="2.2"
        fill="#0f2b5c"
        stroke="#1e3a8a"
        stroke-width="0.4"
      />

      {/* Compound eyes */}
      <ellipse cx="14.2" cy="7.5" rx="1.1" ry="1.5" fill="#0a0f1d" />
      <ellipse cx="17.8" cy="7.5" rx="1.1" ry="1.5" fill="#0a0f1d" />

      {/* Antennae */}
      <g stroke="#93c5fd" stroke-width="0.6" fill="none" stroke-linecap="round">
        <path d="M14.5,5.8 Q12.5,4 10.5,4.5" />
        <path d="M17.5,5.8 Q19.5,4 21.5,4.5" />
      </g>
    </svg>
  ),
);
