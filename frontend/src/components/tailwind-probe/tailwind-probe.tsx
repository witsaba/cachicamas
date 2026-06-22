import { component$ } from "@builder.io/qwik";

/**
 * TailwindProbe — a small Qwik component that uses Tailwind utility classes.
 * It exists to (a) prove the Tailwind v4 + @tailwindcss/vite integration is
 * working end-to-end and (b) give the Vitest spec a stable target to assert
 * against. The classes used here are stable across Tailwind v4 minor versions.
 */
export const TailwindProbe = component$(() => {
  return (
    <div
      data-testid="tailwind-probe"
      class="rounded bg-blue-100 p-4 text-red-500"
    >
      Tailwind probe — if this is a blue box with red text, Tailwind is working.
    </div>
  );
});
