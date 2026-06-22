import { component$ } from "@builder.io/qwik";
import type { DocumentHead } from "@builder.io/qwik-city";
import { TailwindProbe } from "~/components/tailwind-probe/tailwind-probe";

export default component$(() => {
  return (
    <>
      <h1 class="text-3xl font-bold text-slate-900">Hi 👋</h1>
      <div class="mt-2 text-slate-700">
        Can't wait to see what you build with qwik!
        <br />
        Happy coding.
      </div>
      <div class="mt-6">
        <TailwindProbe />
      </div>
    </>
  );
});

export const head: DocumentHead = {
  title: "Welcome to Qwik",
  meta: [
    {
      name: "description",
      content: "Qwik site description",
    },
  ],
};
