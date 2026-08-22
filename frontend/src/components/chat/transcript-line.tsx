import { component$, type QRL } from "@builder.io/qwik";
import { Button } from "~/components/ui/button/button";
import { StateLamp } from "~/components/os/lamp/lamp";
import { renderSanitizedMarkdown } from "~/lib/markdown";
import type { TranscriptEntry } from "~/lib/mock/chat";

/**
 * TranscriptLine — one line of a conversation, in the terminal's grammar.
 *
 * There are no chat bubbles here, and their absence is a decision. A bubble
 * column implies two peers exchanging messages; what is actually happening is
 * a run being narrated — a turn opens, language arrives, a tool is proposed, a
 * person decides, the turn closes with its cost. So the transcript is a
 * speaker-labelled log with a fixed left gutter: the gutter tells you who or
 * what is speaking, and everything lines up down the page whatever it is.
 *
 * Model prose is rendered through `renderSanitizedMarkdown`, never injected
 * raw — the allowlist in `lib/markdown.ts` is the only thing standing between
 * a model's output and the DOM.
 */
export interface TranscriptLineProps {
  readonly entry: TranscriptEntry;
  /**
   * Called with the person's answer when the run is suspended on permission.
   * A QRL, not a bare function: the handler crosses into a `$` closure, and
   * Qwik has to be able to serialise the reference across resumption.
   */
  readonly onDecide$?: QRL<(granted: boolean) => void>;
}

/** The fixed gutter every line shares, so the page has one column of truth. */
const GUTTER = "w-14 shrink-0 text-legend uppercase tracking-[0.14em]";

export const TranscriptLine = component$<TranscriptLineProps>((props) => {
  const e = props.entry;
  const decide = props.onDecide$;

  if (e.kind === "note") {
    const tone =
      e.tone === "fail"
        ? "text-fail"
        : e.tone === "live"
          ? "text-live"
          : "text-fg-dim";
    return (
      <li
        data-testid={`line-note-${e.id}`}
        class="flex items-center gap-3 py-2"
      >
        <span
          class={`${tone} text-legend tracking-[0.18em] whitespace-nowrap uppercase`}
        >
          {e.label}
        </span>
        <span aria-hidden="true" class="bg-rule h-px flex-1" />
        {e.detail ? (
          <span class="text-legend text-fg-dim whitespace-nowrap">
            {e.detail}
          </span>
        ) : null}
      </li>
    );
  }

  if (e.kind === "said") {
    const you = e.who === "you";
    return (
      <li
        data-testid={`line-said-${e.id}`}
        data-who={e.who}
        class="flex gap-3 py-2"
      >
        <span class={`${GUTTER} ${you ? "text-amber" : "text-cyan"}`}>
          {you ? "You" : "Chat"}
        </span>
        <div class="min-w-0 flex-1">
          {you ? (
            <p class="font-human text-body text-fg leading-relaxed break-words">
              {e.text}
            </p>
          ) : (
            <div
              class="prose-terminal font-human text-body text-fg-mid leading-relaxed break-words"
              // Sanitised by `renderSanitizedMarkdown` — a tight tag, attribute
              // and URI-scheme allowlist. Raw model HTML never reaches here.
              dangerouslySetInnerHTML={renderSanitizedMarkdown(e.text)}
            />
          )}
          {e.state === "streaming" ? (
            <span
              data-testid="stream-caret"
              aria-hidden="true"
              class="term-caret text-amber h-4 align-text-bottom"
            />
          ) : null}
        </div>
      </li>
    );
  }

  if (e.kind === "tool") {
    const tone =
      e.state === "running"
        ? { lamp: "live" as const, word: "Running", border: "border-rule" }
        : e.state === "denied"
          ? { lamp: "fail" as const, word: "Refused", border: "border-rule" }
          : e.state === "failed"
            ? { lamp: "fail" as const, word: "Failed", border: "border-fail" }
            : {
                lamp: "ready" as const,
                word: "Returned",
                border: "border-rule",
              };
    return (
      <li data-testid={`line-tool-${e.id}`} class="flex gap-3 py-2">
        <span class={`${GUTTER} text-fg-dim`}>Tool</span>
        <div class={`min-w-0 flex-1 border ${tone.border} bg-raise`}>
          <div class="border-rule flex items-baseline gap-2 border-b px-2.5 py-1.5">
            <span class="text-label text-fg tracking-[0.12em]">{e.tool}</span>
            <span class="flex-1" />
            <StateLamp
              tone={tone.lamp}
              word={tone.word}
              pulse={e.state === "running"}
            />
          </div>
          <div class="px-2.5 py-2">
            <p class="font-human text-data text-fg-mid leading-snug">
              {e.intent}
            </p>
            <dl class="mt-1.5 grid grid-cols-[auto_1fr] gap-x-3 gap-y-0.5">
              {e.args.map(([k, v]) => (
                <div key={k} class="contents">
                  <dt class="text-legend text-fg-dim tracking-[0.12em] uppercase">
                    {k}
                  </dt>
                  <dd class="text-data text-fg-mid break-all">{v}</dd>
                </div>
              ))}
            </dl>
            {e.result ? (
              <p
                data-testid={`tool-result-${e.id}`}
                class="border-rule text-data text-fg mt-2 border-t pt-2"
              >
                {e.result}
              </p>
            ) : null}
            {e.state === "denied" ? (
              <p class="border-rule text-data text-fail mt-2 border-t pt-2">
                Nothing ran. The call was refused before it left the archetype.
              </p>
            ) : null}
          </div>
        </div>
      </li>
    );
  }

  if (e.kind === "hold") {
    const pending = e.decision === "pending";
    return (
      <li data-testid={`line-hold-${e.id}`} class="flex gap-3 py-2">
        <span class={`${GUTTER} text-hold`}>Hold</span>
        <div class="border-hold bg-raise min-w-0 flex-1 border">
          <div class="border-hold/40 flex flex-wrap items-baseline gap-2 border-b px-2.5 py-1.5">
            <span class="text-label text-hold tracking-[0.14em] uppercase">
              Permission required
            </span>
            <span class="flex-1" />
            <span class="text-legend text-fg-dim tracking-[0.12em] uppercase">
              {pending
                ? "The run is suspended here"
                : `Decided · ${e.decision}`}
            </span>
          </div>
          <div class="px-2.5 py-2">
            <p class="font-human text-body text-fg leading-snug">{e.intent}</p>
            <dl class="mt-2 grid grid-cols-[auto_1fr] gap-x-3 gap-y-0.5">
              <div class="contents">
                <dt class="text-legend text-fg-dim tracking-[0.12em] uppercase">
                  tool
                </dt>
                <dd class="text-data text-fg-mid">{e.tool}</dd>
              </div>
              {e.args.map(([k, v]) => (
                <div key={k} class="contents">
                  <dt class="text-legend text-fg-dim tracking-[0.12em] uppercase">
                    {k}
                  </dt>
                  <dd class="text-data text-fg-mid break-all">{v}</dd>
                </div>
              ))}
            </dl>
            <p class="border-hold/40 font-human text-data text-hold mt-2 border-t pt-2 leading-snug">
              {e.risk}
            </p>
            {pending ? (
              <div class="mt-2.5 flex flex-wrap gap-2">
                <Button
                  variant="primary"
                  testId="permission-allow"
                  onClick$={() => decide?.(true)}
                >
                  Allow once
                </Button>
                <Button
                  variant="destructive"
                  testId="permission-refuse"
                  onClick$={() => decide?.(false)}
                >
                  Refuse
                </Button>
              </div>
            ) : null}
          </div>
        </div>
      </li>
    );
  }

  return (
    <li data-testid={`line-fault-${e.id}`} class="flex gap-3 py-2">
      <span class={`${GUTTER} text-fail`}>Fault</span>
      <div class="border-fail bg-raise min-w-0 flex-1 border">
        <div class="border-fail/40 flex items-baseline gap-2 border-b px-2.5 py-1.5">
          <span class="text-label text-fail tracking-[0.14em] uppercase">
            {e.code}
          </span>
          <span class="flex-1" />
          <span class="text-legend text-fg-dim tracking-[0.12em] uppercase">
            No automatic retry
          </span>
        </div>
        <div class="px-2.5 py-2">
          <p class="font-human text-body text-fg leading-snug">{e.message}</p>
          <p class="font-human text-data text-fg-mid mt-1.5 leading-snug">
            {e.recovery}
          </p>
        </div>
      </div>
    </li>
  );
});
