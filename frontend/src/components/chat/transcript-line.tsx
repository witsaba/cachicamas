/**
 * One line of a conversation.
 *
 * Five kinds, and the differences between them are structural rather than
 * decorative, because the whole point of showing a colleague's work is that a
 * person can tell at a glance which part of it was *speech* and which part was
 * an *action*:
 *
 *   said   — a name, a face, and words. Nothing else.
 *   tool   — an inset panel: which tool, why, with what, and what came back.
 *   hold   — the same panel, waiting, with the two answers a person can give.
 *   fault  — said once, with the recovery, and never retried.
 *   note   — the conversation narrating itself, set quietly between the rest.
 *
 * The person's own words are rendered as text. The colleague's are rendered
 * through the sanitiser, which emits a small allowlisted set of tags — model
 * output never reaches the DOM as raw HTML.
 */
import { component$, type QRL } from "@builder.io/qwik";
import { Icon } from "~/components/icon/icon";
import { Button } from "~/components/ui/button/button";
import {
  AgentAvatar,
  PersonAvatar,
} from "~/components/workspace/avatar/avatar";
import { renderSanitizedMarkdown } from "~/lib/markdown";
import type { TranscriptEntry } from "~/lib/mock/chat";
import type { Agent } from "~/lib/mock/staff";

export interface TranscriptLineProps {
  readonly entry: TranscriptEntry;
  /** Whose conversation this is — supplies the name and the avatar. */
  readonly agent: Agent;
  readonly youName: string;
  readonly youInitials: string;
  /** Only a pending permission needs this; every other kind ignores it. */
  readonly onDecide$?: QRL<(granted: boolean) => void>;
}

const PANEL =
  "rounded-md border border-line bg-surface p-3 shadow-[var(--shadow-raised)]";

export const TranscriptLine = component$<TranscriptLineProps>((props) => {
  const { entry, agent, youName, youInitials } = props;
  // Read the QRL into a local before the closure: a prop captured directly
  // inside an event handler trips Qwik's lexical-scope rule.
  const decide = props.onDecide$;

  if (entry.kind === "note") {
    return (
      <li
        class="flex items-center gap-3 py-3"
        data-testid={`line-note-${entry.id}`}
      >
        <span class="bg-line h-px flex-1" aria-hidden="true" />
        <span class="text-2xs text-ink-soft shrink-0">
          <span
            class={
              entry.tone === "fail"
                ? "text-stop font-medium"
                : entry.tone === "live"
                  ? "text-ok font-medium"
                  : "text-ink-mid font-medium"
            }
          >
            {entry.label}
          </span>
          {entry.detail ? ` · ${entry.detail}` : null}
        </span>
        <span class="bg-line h-px flex-1" aria-hidden="true" />
      </li>
    );
  }

  if (entry.kind === "said") {
    const you = entry.who === "you";
    return (
      <li
        class="flex gap-3 py-3"
        data-testid={`line-said-${entry.id}`}
        data-who={you ? "you" : "agent"}
      >
        <span class="pt-0.5">
          {you ? (
            <PersonAvatar name={youName} initials={youInitials} size="md" />
          ) : (
            <AgentAvatar agent={agent} size="md" />
          )}
        </span>
        <div class="min-w-0 flex-1">
          <p class="flex items-baseline gap-2 pb-0.5">
            <span class="text-ink text-base font-semibold">
              {you ? youName || "You" : agent.name}
            </span>
            {you ? null : (
              <span class="text-2xs text-ink-soft font-medium">Agent</span>
            )}
          </p>
          {you ? (
            <p class="text-md text-ink-mid whitespace-pre-wrap">{entry.text}</p>
          ) : (
            <div
              class="prose-work"
              // Sanitised at the boundary; the allowlist is in lib/markdown.ts.
              dangerouslySetInnerHTML={renderSanitizedMarkdown(entry.text)}
            />
          )}
          {entry.state === "streaming" ? (
            <span
              class="work-caret text-ink-mid ml-0.5 align-baseline"
              data-testid="stream-caret"
              aria-hidden="true"
            />
          ) : null}
        </div>
      </li>
    );
  }

  if (entry.kind === "tool") {
    return (
      <li class="py-3 pl-11" data-testid={`line-tool-${entry.id}`}>
        <div class={PANEL}>
          <p class="flex flex-wrap items-center gap-x-2 gap-y-1">
            <Icon name="tool" size={16} class="text-ink-soft shrink-0" />
            <span class="text-ink text-base font-semibold">
              {entry.state === "running" ? "Running" : "Used"} {entry.tool}
            </span>
            <span class="text-ink-soft text-xs">{entry.intent}</span>
          </p>
          <dl class="border-line mt-2 grid grid-cols-[minmax(5rem,auto)_minmax(0,1fr)] gap-x-4 gap-y-1 border-t pt-2 text-xs">
            {entry.args.map(([k, v]) => (
              <div key={k} class="contents">
                <dt class="text-ink-soft">{k}</dt>
                <dd class="text-ink-mid overflow-x-auto">{v}</dd>
              </div>
            ))}
          </dl>
          {/* Every state gets a word, because a colour alone never tells
              anyone whether something happened. */}
          {entry.state === "denied" || entry.state === "failed" ? (
            <p class="border-line text-stop mt-2 border-t pt-2 text-xs font-medium">
              {entry.state === "denied"
                ? "Nothing ran — you refused it."
                : "Failed."}
              {entry.result ? (
                <span class="text-ink-mid font-normal"> {entry.result}</span>
              ) : null}
            </p>
          ) : entry.result ? (
            <p
              class="border-line text-ink-mid mt-2 border-t pt-2 text-xs"
              data-numeric
            >
              {entry.result}
            </p>
          ) : null}
        </div>
      </li>
    );
  }

  if (entry.kind === "hold") {
    // Only a pending decision that something is listening for can be answered.
    const pending = entry.decision === "pending" && decide !== undefined;
    return (
      <li class="py-3 pl-11" data-testid={`line-hold-${entry.id}`}>
        <div
          class={[
            "rounded-md border p-3",
            pending
              ? "border-waiting/35 bg-waiting/[0.06]"
              : "border-line bg-surface",
          ].join(" ")}
        >
          <p class="flex flex-wrap items-center gap-x-2 gap-y-1">
            <Icon
              name="shield"
              size={16}
              class={
                pending ? "text-waiting shrink-0" : "text-ink-soft shrink-0"
              }
            />
            <span class="text-ink text-base font-semibold">
              {pending ? "Waiting for you" : "You were asked"}
            </span>
            <span class="text-ink-soft text-xs">{entry.intent}</span>
          </p>
          <dl class="border-line/70 mt-2 grid grid-cols-[minmax(5rem,auto)_minmax(0,1fr)] gap-x-4 gap-y-1 border-t pt-2 text-xs">
            {entry.args.map(([k, v]) => (
              <div key={k} class="contents">
                <dt class="text-ink-soft">{k}</dt>
                <dd class="text-ink-mid overflow-x-auto">{v}</dd>
              </div>
            ))}
          </dl>
          <p class="text-ink-mid mt-2 text-xs">{entry.risk}</p>

          {pending ? (
            <div class="mt-3 flex flex-wrap gap-2">
              <Button
                size="sm"
                variant="primary"
                testId="permission-allow"
                onClick$={() => decide?.(true)}
              >
                Allow it
              </Button>
              <Button
                size="sm"
                variant="secondary"
                testId="permission-refuse"
                onClick$={() => decide?.(false)}
              >
                Don&rsquo;t
              </Button>
            </div>
          ) : (
            <p class="border-line mt-2 border-t pt-2 text-xs font-medium">
              {entry.decision === "granted" ? (
                <span class="text-ok">You allowed it.</span>
              ) : (
                <span class="text-stop">You refused. Nothing happened.</span>
              )}
            </p>
          )}
        </div>
      </li>
    );
  }

  return (
    <li class="py-3 pl-11" data-testid={`line-fault-${entry.id}`}>
      <div class="border-stop/30 bg-stop/[0.05] rounded-md border p-3">
        <p class="text-stop text-base font-semibold">{entry.code}</p>
        <p class="text-ink-mid mt-1 text-base">{entry.message}</p>
        <p class="border-line text-ink-soft mt-2 border-t pt-2 text-xs">
          {entry.recovery}
        </p>
        {/* Said once, and said out loud: nothing here quietly retries in the
            background. A colleague that keeps trying without telling you is
            how a small failure becomes an expensive one. */}
        <p class="text-ink-soft pt-1 text-xs font-medium">
          No automatic retry.
        </p>
      </div>
    </li>
  );
});
