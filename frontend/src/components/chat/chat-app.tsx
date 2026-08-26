import {
  $,
  component$,
  useSignal,
  useStore,
  useVisibleTask$,
} from "@builder.io/qwik";
import { AgentAvatar } from "~/components/workspace/avatar/avatar";
import { Status } from "~/components/workspace/status/status";
import { initialsOf } from "~/lib/initials";
import { AGENTS, agentBySlug } from "~/lib/mock/staff";
import { loadMostRecentConversation } from "~/lib/chat-api";
import type {
  ConversationSummary,
  ToolCallDTO,
  ToolResultDTO,
  PermissionDecisionDTO,
} from "~/lib/chat-types";
import type { TranscriptEntry } from "~/lib/mock/chat";
import { Composer } from "./composer";
import { ConversationList } from "./conversation-list";
import { TranscriptLine } from "./transcript-line";
import { useChatStream } from "./use-chat-stream";
import { SCROLL_EVENT } from "./events";

/**
 * Chat — a conversation with one colleague.
 *
 * What is worth trying, because it is the product's actual argument: ask a
 * colleague to *send* something. The conversation stops, shows the exact thing
 * it is about to do, and waits. Nothing moves until a person answers. That
 * pause is the whole reason a company can let an agent near its systems.
 *
 * CH-05.1 (D-3): the conversations rail was dropped from the CH-05.1
 * wire-up. CH-08 undoes that — the rail mounts again, driven by the
 * real wire (R-CRI-002, R-CRI-005, REQ-8). The `?with=<slug>`
 * deep-link (D-6) still resolves to an `agentSlug` from staff.ts so
 * the workplace shell's `/chat?with=finance` link keeps working
 * (D-4 inert: it does not drive which conversation loads — D-1).
 *
 * CH-08 mount data flow (REQ-8):
 *   1. `useVisibleTask$` on first paint fires
 *      `GET /api/agent/conversations/:id`  and
 *      `GET /api/agent/conversations` in parallel.
 *   2. The reload endpoint's `ExchangeDTO[]` is projected to
 *      `TranscriptEntry[]` and `useChatStream.reset(entries)` seeds
 *      the buffer — no EventSource opened, no false streaming claim
 *      (REQ-9: a reload never claims the turn is still in flight).
 *   3. The list endpoint's `ConversationSummary[]` populates the
 *      rail via `<ConversationList />` (R-CRI-005).
 *   4. The cross-participant / 404 / empty-list boundaries are
 *      enforced by the backend — see
 *      `backend/agent/src/chat/http.go:HandleReloadConversation`
 *      and `HandleListConversations`.
 */
export interface ChatAppProps {
  readonly youName: string;
  readonly youEmail: string;
  /**
   * Authenticated participant id. Surfaced from `requireSession` in
   * `routes/chat/index.tsx` so the page knows which conversation to
   * load (D-1: `conversationID == participantID`). The page never
   * trusts a URL param or a header for this — the route hands the
   * session's resolved id in as a prop.
   */
  readonly participantID: string;
}

/**
 * exchangesToEntries rebuilds a transcript from a recorded exchange
 * slice (CH-08 R-CRI-001 reload surface). The output sequence for
 * each exchange is:
 *
 *   1. said:you {text: promptText, state: final}
 *   2. said:chat {text: assistantText, state: final|streaming}
 *   3. tool {tool, intent, args, state, result} — appended AFTER the
 *      assistant said entry, one per (toolCall, toolResult) pair.
 *      Success / result_failure outcomes → state "done";
 *      execution_failure → state "failed" with the typed failure
 *      category (R-CCP-008 / D6 mirror — no provider text on
 *      failure).
 *   4. hold {tool, intent, args, decision} — appended AFTER the
 *      tool entries, one per recorded permission decision
 *      (R-CPM-006, S-FCL-022). The decision is "granted" for
 *      "allow_once", "denied" for "deny". The decision is
 *      terminal — the wire does not carry "waiting" on the reload
 *      surface (the participant's click happened before the
 *      exchange was committed to the store).
 *
 * The DTO shape carries ToolCalls + ToolResults + PermissionDecisions
 * as optional readonly arrays; a turn without activity omits the
 * keys entirely (the JSON wire omits them per `omitempty`).
 *
 * Exported for direct testability of the S-CTS-019 covering test
 * (CH-09 WU-4a — verify #3974 found the function had no runtime
 * test). The export is a pure testability seam; the function
 * signature is unchanged from its prior module-private form.
 */
export function exchangesToEntries(
  exchanges: readonly {
    promptText: string;
    assistantText: string;
    partial: boolean;
    toolCalls?: readonly ToolCallDTO[];
    toolResults?: readonly ToolResultDTO[];
    permissionDecisions?: readonly PermissionDecisionDTO[];
  }[],
): TranscriptEntry[] {
  const out: TranscriptEntry[] = [];
  exchanges.forEach((ex, idx) => {
    out.push({
      kind: "said",
      id: `u${idx}`,
      who: "you",
      text: ex.promptText,
      state: "final",
    });
    out.push({
      kind: "said",
      id: `a${idx}`,
      who: "chat",
      text: ex.assistantText,
      state: ex.partial ? "streaming" : "final",
    });
    // CH-09 — tool entries from the reload surface. The reload DTO
    // carries both slices in issuance order (R-CCS-016); we walk
    // them in lockstep so each tool call gets a matching result.
    const calls = ex.toolCalls ?? [];
    const results = ex.toolResults ?? [];
    const max = Math.max(calls.length, results.length);
    for (let i = 0; i < max; i++) {
      const call = calls[i];
      const result = results[i];
      const toolName = call?.tool ?? result?.tool ?? "tool";
      const wireId = call?.wireCallId ?? result?.wireCallId ?? `r-${idx}-${i}`;
      let state: "done" | "failed" = "done";
      let resultText: string | undefined;
      if (result) {
        if (result.outcome === "execution_failure") {
          state = "failed";
          resultText = result.failureCategory;
        } else {
          state = "done";
          resultText = result.content;
        }
      }
      out.push({
        kind: "tool",
        id: `tool-${wireId}-${idx}`,
        tool: toolName,
        intent: toolName,
        args: parseToolArgs(call?.arguments ?? ""),
        result: resultText,
        state,
      });
    }
    // CH-10 (R-CPM-006, S-FCL-022) — permission-decision entries
    // from the reload surface. Issued in the same order as the
    // stored records (the store's position field, projected
    // by the postgres sibling-table ORDER BY position ASC or
    // the in-memory slice order). Decision is terminal — the
    // reload surface does not carry the "waiting" state because
    // the click happened before the exchange committed.
    const decisions = ex.permissionDecisions ?? [];
    for (let i = 0; i < decisions.length; i++) {
      const pd = decisions[i];
      const decision: "granted" | "denied" =
        pd.outcome === "allow_once" ? "granted" : "denied";
      out.push({
        kind: "hold",
        id: `hold-${pd.wireCallId}-${idx}`,
        tool: pd.tool,
        intent: pd.tool,
        args: [],
        risk: "",
        decision,
      });
    }
  });
  return out;
}

/**
 * parseToolArgs converts the tool-call DTO's arguments string
 * (JSON object) into the entry's args tuple array. Mirrors the
 * wire-side parseArgs in use-chat-stream.ts.
 */
function parseToolArgs(
  jsonText: string,
): readonly (readonly [string, string])[] {
  if (!jsonText) return [];
  try {
    const obj = JSON.parse(jsonText) as Record<string, unknown>;
    if (typeof obj !== "object" || obj === null) return [];
    return Object.entries(obj).map(
      ([k, v]) => [k, String(v)] as readonly [string, string],
    );
  } catch {
    return [];
  }
}

export const ChatApp = component$<ChatAppProps>(
  ({ youName, youEmail, participantID }) => {
    // Disable the browser's automatic scroll-position restoration
    // on refresh. With `history.scrollRestoration = "auto"` (the
    // default), the browser tries to put the page back where the
    // user was before F5 — which, for a chat that's mid-scroll
    // through history, lands the scroll AWAY from the bottom even
    // after our explicit scroll-to-bottom runs. "manual" keeps
    // the user's scroll exactly where the chat component puts it.
    if (typeof window !== "undefined") {
      if (window.history.scrollRestoration !== "manual") {
        window.history.scrollRestoration = "manual";
      }
    }

    const activeSlug = useSignal(AGENTS[0].slug);
    // CH-12 (S-SCROLL-001) — auto-scroll bus. The hook
    // (a) dispatches the `chat:scroll-to-bottom` CustomEvent on
    // `window` at every entry mutation site, AND
    // (b) bumps this counter as a public-API fallback.
    // The visible-task below listens for the event (decoupled from
    // Qwik's reactive task lifecycle — see the comment on the
    // listener below). The signal itself is kept on the hook's
    // public surface for backward compatibility.
    const scrollCounter = useSignal(0);
    const turn = useChatStream([], scrollCounter);
    const scroller = useSignal<HTMLElement>();

    // CH-08 (REQ-8 / REQ-9): mount-time fetch + seed. On first paint
    // the page fires the two resume GETs in parallel, seeds the
    // transcript with `reset(entries)`, populates the rail. The
    // helper returns silently when offline / cross-participant /
    // unknown — the mount does not throw, so a transient wire
    // failure leaves the page in its CH-05.1 empty-seed shape
    // rather than a blank-by-error state.
    //
    // Fix B (server-issued conversationID): `loadMostRecentConversation`
    // composes the two resume GETs — list first, then load — so the
    // reload URL carries the server-issued conversationID, NOT the
    // route-level `participantID` prop (which falls back to the user's
    // email when Auth.js issued no `user.id`; the backend refuses
    // that with 403 not_found, R-CHS-004.b).
    const railState = useStore<{
      summaries: ConversationSummary[];
      loaded: boolean;
      loadError: boolean;
      selectedId: string;
    }>({
      summaries: [],
      loaded: false,
      loadError: false,
      // Rail selection id starts as the route prop and is replaced
      // with the server-issued conversationID once the list resolves.
      // An empty string keeps the rail unhighlighted until the helper
      // resolves (no false-positive "current row" on a stale prop).
      selectedId: "",
    });

    // eslint-disable-next-line qwik/no-use-visible-task
    useVisibleTask$(({ track }) => {
      track(() => participantID);
      if (typeof window === "undefined") return;

      void (async () => {
        const resumed = await loadMostRecentConversation();

        if (resumed.ok) {
          // Reload — seed the buffer if the participant has recorded
          // exchanges; an empty list (S-CRI-004: 200 []) or no
          // exchanges leaves the seed empty.
          if (resumed.value.exchanges.length > 0) {
            await turn.reset(exchangesToEntries(resumed.value.exchanges));
          }
          // Pin the transcript to the bottom EXPLICITLY here, after
          // the awaited bulk reset. `reset()` itself dispatches
          // SCROLL_EVENT for the listener to catch — but on a chat
          // refresh with many entries the listener can fire BEFORE
          // Qwik has committed the bulk render of the replayed
          // exchanges, reading a stale scrollHeight and leaving
          // the scroll mid-conversation. Run the scroll across six
          // consecutive frames after the awaited reset: that gives
          // Qwik enough time to commit the bulk, and one of the
          // six attempts will land against the post-commit layout.
          // `nextFrame` is hoisted out of the closure so the
          // TS-inferred type of `resolve` is unambiguously
          // `() => void` (requestAnimationFrame's callback shape),
          // not the `PromiseLike` overload that breaks compilation
          // when the resolve callback is passed directly.
          const nextFrame = (): Promise<void> =>
            new Promise<void>((resolve) =>
              requestAnimationFrame(() => resolve()),
            );
          for (let i = 0; i < 6; i++) {
            await nextFrame();
            await nextFrame();
            const scrollerEl = scroller.value;
            if (scrollerEl) {
              void scrollerEl.offsetHeight;
              scrollerEl.scrollTo({
                top: scrollerEl.scrollHeight,
                behavior: "instant",
              });
            }
          }
          // List — populate the rail with the server-issued
          // conversationID; the rail marks only that row current.
          railState.summaries = [...resumed.value.summaries];
          railState.selectedId = resumed.value.conversationID;
          railState.loaded = true;
          railState.loadError = false;
        } else {
          railState.summaries = [];
          railState.loaded = false;
          railState.loadError = true;
        }
      })();
    });

    // `?with=<slug>` opens the conversation with that colleague. It is
    // read from the browser rather than the router so this component stays
    // renderable without a request context — the front-desk panel's link
    // (`front-desk.tsx:119`) routes here and the screen answers, neither
    // needs to know about Qwik City's location service.
    // eslint-disable-next-line qwik/no-use-visible-task
    useVisibleTask$(() => {
      if (typeof window === "undefined") return;
      const slug = new URL(window.location.href).searchParams.get("with");
      if (!slug) return;
      const match = agentBySlug(slug);
      if (match) activeSlug.value = match.slug;
    });

    // Following an arriving answer to the bottom of a scroller is a browser-only
    // concern by definition; there is no server-side equivalent to fall back to.
    //
    // CH-12 (S-SCROLL-001) — drive auto-scroll from an explicit DOM event
    // dispatched at the mutation site (use-chat-stream.ts:173-181,
    // `requestScroll`) instead of a Qwik signal counter / `useVisibleTask$`
    // tracker.
    //
    // Why the previous signal-counter approach (CH-11 / commit 81a24641)
    // did not work: Qwik re-runs a `useVisibleTask$` that `track`s a
    // signal in the SAME microtask as the signal change — BEFORE Qwik's
    // render commit. The rAF inside fires before the new <li> is in the
    // DOM, so `scrollTop = scrollHeight` reads the OLD height and the
    // new content lands below the fold (the scrollbar grows but the
    // text is hidden).
    //
    // The event-based approach decouples the scroll from Qwik's
    // reactive task lifecycle. The hook dispatches the event from
    // plain browser time, AFTER its own mutation. The listener runs
    // in plain browser time too — there is no `track` to race with
    // the commit. The double rAF inside the listener is critical:
    // the first rAF lets Qwik flush its render commit, the second
    // rAF lets the browser paint, and only then do we read
    // scrollHeight. By that point the new <li> is in the DOM and
    // laid out, so the scroll lands.
    //
    // The signal counter is preserved on the hook's public surface
    // (CH-11) so external observers (tests, analytics) can still
    // react to entry mutations. The actual scroll is event-driven.
    //
    // A second auto-scroll mechanism, a `MutationObserver` attached
    // to the transcript `<ol>`, runs as a SECONDARY path. It catches:
    //   1. New entries appended through any code path that did NOT
    //      call `requestScroll` (defence in depth for future hook
    //      variants).
    //   2. Streaming `message.delta` text appended to an EXISTING
    //      `<li>` — Qwik batches the `state.entries.map` write, so
    //      the SCROLL_EVENT may fire BEFORE the streaming text node
    //      is in the DOM, missing the height growth.
    //   3. The browser's default `overflow-anchor: auto` behaviour,
    //      which silently re-asserts a stale visual position on any
    //      content append (mitigated by `[overflow-anchor:none]` on
    //      the `<ol>` above AND backed up here for safety).
    // The observer fires `scrollToBottom` — the SAME rAF chain the
    // event listener uses — so a per-character stream coalesces into
    // one scroll per paint frame.
    //
    // Earlier experiments tried a 6-frame brute-force spread inside
    // this listener (commit 3bfe7fb1), which fixed the chat-on-
    // refresh case but produced "scrollbar goes crazy" symptoms on
    // send / stream because six scroll attempts fired inside a
    // 2-frame window against a still-mutating scrollHeight. The
    // bulk-replay case now lives in the reload visible-task's own
    // explicit brute-force spread AFTER `await turn.reset()` —
    // that's the path that needs the extra cadence. The streaming
    // / send path here is happy with a single double-rAF + raw
    // `scrollTop = scrollHeight` (guaranteed-instant across
    // browsers, no animation races, no scrollIntoView jitter).
    // eslint-disable-next-line qwik/no-use-visible-task
    useVisibleTask$(({ cleanup }) => {
      if (typeof window === "undefined") return;
      const scrollToBottom = () => {
        // Double rAF: first lets Qwik's render commit flush,
        // second lets the browser paint. After both, scrollHeight
        // reflects the post-commit content reliably.
        window.requestAnimationFrame(() => {
          window.requestAnimationFrame(() => {
            const scrollerEl = scroller.value;
            if (!scrollerEl) return;
            void scrollerEl.offsetHeight;
            scrollerEl.scrollTop = scrollerEl.scrollHeight;
          });
        });
      };
      window.addEventListener(SCROLL_EVENT, scrollToBottom);

      // Catches new <li>s appended to the transcript. `childList`
      // only — `subtree` + `characterData` would re-fire on every
      // streamed text node mutation and on every status-dot
      // re-render, multiplying the scroll attempts inside a
      // streaming window and producing visible jitter. A new
      // <li> is the meaningful signal; intermediate text updates
      // inside an existing <li> are caught by the EVENT listener
      // (the hook dispatches SCROLL_EVENT on every message.delta).
      const olElement = scroller.value;
      let observer: MutationObserver | null = null;
      if (olElement) {
        observer = new MutationObserver(() => {
          scrollToBottom();
        });
        observer.observe(olElement, {
          childList: true,
        });
      }

      cleanup(() => {
        window.removeEventListener(SCROLL_EVENT, scrollToBottom);
        observer?.disconnect();
      });
    });

    const agent = agentBySlug(activeSlug.value) ?? AGENTS[0];
    const youInitials = initialsOf(youName, youEmail);

    return (
      <div class="flex min-h-0 flex-1">
        {/* CH-08 (R-CRI-005): the rail is re-mounted against the wire.
          It was dropped in CH-05.1 (D-3); CH-08 undoes that drop
          because the page now needs to render the participant's
          own conversations. `selectedId` is the participant id
          (D-1) — the rail marks only the active row current. The
          onSelect handler is a no-op stub for v1 (the page has one
          active conversation at v1; future CH-08 leaves will
          expand the rail's interaction surface). */}
        <aside
          class="bg-surface border-line hidden w-64 shrink-0 border-r md:block"
          aria-label="Conversation list"
        >
          <ConversationList
            conversations={railState.summaries}
            selectedId={railState.selectedId}
            onSelect$={$(async (id: string) => {
              // v1: the rail is informational. The page holds one
              // active conversation; future CH work wires the rail
              // to a conversation switcher (deferred — CH-08.2
              // own-list surfaces only, the interaction model is a
              // follow-up). The id parameter is reserved for that
              // future and intentionally unused here.
              void id;
            })}
          />
        </aside>
        <div class="bg-canvas flex min-w-0 flex-1 flex-col">
          {/* who you are talking to */}
          <header class="border-line bg-surface flex items-center gap-3 border-b px-4 py-2.5">
            <AgentAvatar agent={agent} size="md" />
            <div class="min-w-0 flex-1">
              <p class="flex items-center gap-2">
                <a
                  href={`/agents/${agent.slug}/`}
                  class="text-ink truncate rounded-sm text-base font-semibold hover:underline"
                >
                  {agent.name}
                </a>
                <span class="text-2xs text-ink-soft shrink-0 font-medium">
                  Agent
                </span>
              </p>
              <div class="mt-0.5 flex items-center gap-2">
                <span class="text-ink-soft truncate text-xs">
                  {agent.departmentName}
                </span>
                <span class="ml-auto">
                  <Status
                    status={
                      turn.status === "streaming" ||
                      turn.status === "submitting" ||
                      turn.status === "cancelling"
                        ? "working"
                        : agent.status
                    }
                    word={
                      turn.status === "streaming" ||
                      turn.status === "submitting" ||
                      turn.status === "cancelling"
                        ? "Working now"
                        : agent.statusWord
                    }
                  />
                </span>
              </div>
            </div>
          </header>

          <ol
            ref={scroller}
            data-testid="transcript"
            aria-live="polite"
            aria-label={`Conversation with ${agent.name}`}
            // Every line keeps a reading measure. Without it an answer runs
            // the full width of a 1440px screen, which is about twice the length
            // anyone reads comfortably; the composer below is capped to the same
            // column so the two never disagree about where the conversation is.
            //
            // `[overflow-anchor:none]` opts out of the browser's
            // scroll-anchoring feature. The default `auto` lets the
            // browser AUTO-ADJUST `scrollTop` to keep whatever was
            // visible at the same visual position whenever new
            // content is appended to the scroller — which silently
            // undoes `scrollTop = scrollHeight` whenever the user
            // is scrolled up while a new message or a streaming
            // delta arrives. Disabling the anchor lets the explicit
            // scroll-to-bottom in this file actually stick.
            class="min-h-0 flex-1 overflow-y-auto px-4 pb-2 [overflow-anchor:none] sm:px-6 [&>li]:mx-auto [&>li]:w-full [&>li]:max-w-2xl"
          >
            {turn.entries.map((entry) => (
              <TranscriptLine
                key={entry.id}
                entry={entry}
                agent={agent}
                youName={youName}
                youInitials={youInitials}
              />
            ))}
          </ol>

          <Composer
            status={
              turn.status === "submitting" ||
              turn.status === "streaming" ||
              turn.status === "cancelling"
                ? ("running" as const)
                : ("idle" as const)
            }
            agentName={agent.name}
            onSubmit$={$(async (prompt: string) => {
              await turn.submit(prompt);
            })}
            onCancel$={$(async () => {
              await turn.cancel();
            })}
          />
        </div>
      </div>
    );
  },
);
