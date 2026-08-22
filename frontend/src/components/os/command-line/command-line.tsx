import { $, component$, useOnDocument, useSignal } from "@builder.io/qwik";
import { resolveCommand, suggestCommands } from "~/lib/mock/registry";

/**
 * CommandLine — the launcher, in the world's own grammar.
 *
 * This is the signature interaction of the whole interface. A terminal's front
 * door is a line you type a code into, so that is what opens an archetype
 * here: `CHAT` and Enter. There is no icon grid to hunt through and no menu to
 * unfold; the dock below is the same set of destinations, keyed, for when you
 * would rather point.
 *
 * Three things it must do, and does:
 *   - Focus from anywhere on ⌘K / Ctrl-K, and let Escape hand focus back.
 *   - Suggest as you type, so the code set is discoverable rather than lore.
 *   - Refuse out loud. An unknown code prints what it does accept — never a
 *     silent no-op, which is the failure mode every command bar has.
 *
 * Navigation is a full document load rather than a client-side route change.
 * That is deliberate: it matches an OS launching an application, and it keeps
 * this component free of the router context, which the layout's test harness
 * does not provide.
 */
export const CommandLine = component$(() => {
  const input = useSignal("");
  const open = useSignal(false);
  const refused = useSignal<string | null>(null);
  const inputId = "command-line-input";

  const go = $((href: string) => {
    if (typeof window !== "undefined") window.location.assign(href);
  });

  const submit = $(() => {
    const result = resolveCommand(input.value);
    if (result.ok) {
      refused.value = null;
      open.value = false;
      go(result.href);
      return;
    }
    refused.value = result.message;
  });

  useOnDocument(
    "keydown",
    $((event: Event) => {
      const e = event as KeyboardEvent;
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "k") {
        // A document-level key handler has no `preventdefault:` attribute to
        // declare; this is the only place the browser's own ⌘K can be
        // suppressed.
        // eslint-disable-next-line qwik/no-async-prevent-default
        e.preventDefault();
        const el = document.getElementById(inputId) as HTMLInputElement | null;
        el?.focus();
        el?.select();
      }
    }),
  );

  const suggestions = open.value
    ? suggestCommands(input.value).slice(0, 7)
    : [];

  return (
    <div
      data-testid="command-line"
      class="border-rule bg-void relative border-b px-3 py-1.5"
    >
      <div class="flex items-center gap-2">
        <label
          for={inputId}
          class="text-label text-amber select-none"
          data-testid="command-line-prompt"
        >
          &gt;
        </label>
        <input
          id={inputId}
          data-testid="command-line-input"
          type="text"
          autocomplete="off"
          spellcheck={false}
          aria-label="Command line — type an archetype code and press Enter"
          placeholder="Type a code — CHAT, DBA, SYSTEM — and press Enter"
          value={input.value}
          class="text-body text-fg min-w-0 flex-1 border-none bg-transparent outline-none"
          onFocus$={() => {
            open.value = true;
          }}
          onInput$={(_, el) => {
            input.value = el.value;
            open.value = true;
            refused.value = null;
          }}
          onKeyDown$={(event) => {
            if (event.key === "Enter") {
              void submit();
            } else if (event.key === "Escape") {
              open.value = false;
              (event.target as HTMLInputElement).blur();
            }
          }}
        />
        <kbd
          aria-hidden="true"
          class="border-rule text-legend text-fg-dim hidden border px-1.5 py-px tracking-[0.12em] uppercase sm:inline"
        >
          ⌘K
        </kbd>
      </div>

      {refused.value ? (
        <p
          data-testid="command-line-refusal"
          class="text-legend text-fail pt-1 pl-4"
        >
          {refused.value}
        </p>
      ) : null}

      {suggestions.length > 0 ? (
        <ul
          data-testid="command-line-suggestions"
          class="border-rule-strong bg-panel absolute top-full left-0 z-30 max-h-72 w-full overflow-y-auto border-x border-b"
        >
          {suggestions.map((s) => (
            <li key={s.code}>
              <a
                href={s.href}
                data-testid={`command-suggestion-${s.code}`}
                class="hover:bg-raise flex items-baseline gap-3 px-3 py-1.5 no-underline"
              >
                <span class="text-label text-amber w-24 shrink-0 tracking-[0.12em] uppercase">
                  {s.code}
                </span>
                <span class="text-data text-fg flex-1 truncate">{s.label}</span>
                <span class="text-legend text-fg-dim tracking-[0.12em] uppercase">
                  {s.hint}
                </span>
              </a>
            </li>
          ))}
        </ul>
      ) : null}
    </div>
  );
});
