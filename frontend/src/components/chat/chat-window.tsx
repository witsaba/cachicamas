/**
 * chat-window.tsx — the chat's visual surface.
 *
 * Reference: openspec/changes/cachicamas-frontend-chat-layer1/design.md
 *   §3 (composes MessageBubble + ChatInput; reads signals from
 *   useChatStream),
 *   §4 (REQ-1 happy path: user message + streaming assistant bubble;
 *        REQ-4 S-4.a inline error alert when the session's last
 *        message has status='error').
 *
 * Render shape:
 *   <section chat-window>
 *     <header>Chat — Cachicamas</header>
 *     <ol message-list>
 *       {messages.map(m => <MessageBubble message={m} key={m.id} />)}
 *       {streaming ? <StreamingPill /> : null}
 *     </ol>
 *     <ChatInput onSubmit$={submit} onCancel$={cancel} disabled={!isIdle} />
 *   </section>
 *
 * Aphantasic-friendly (UX-4): text-first, no decorative imagery.
 * Status pill uses a short uppercase label.
 */
import { component$ } from "@builder.io/qwik";

import { useChatStream } from "./use-chat-stream";
import { ChatInput } from "./chat-input";
import { MessageBubble } from "./message-bubble";

export const ChatWindow = component$(() => {
  const stream = useChatStream();
  const { session } = stream;

  const isIdle = session.status === "idle";
  const isStreaming = session.status === "streaming";
  const lastMessage = session.messages[session.messages.length - 1];

  return (
    <section
      data-testid="chat-window"
      aria-label="Chat"
      class="mx-auto flex max-w-3xl flex-col gap-4 px-4 py-8"
    >
      <header class="border-b border-slate-200 pb-3">
        <h1
          data-testid="chat-heading"
          class="text-2xl font-semibold text-slate-900"
        >
          Chat
        </h1>
        <p data-testid="chat-subheading" class="mt-1 text-sm text-slate-600">
          Type a prompt and press Enter. The reply streams in below.
        </p>
      </header>

      <ol
        data-testid="chat-message-list"
        class="flex flex-col gap-3"
        aria-live="polite"
      >
        {session.messages.length === 0 ? (
          <li
            data-testid="chat-empty"
            class="rounded-md border border-dashed border-slate-300 bg-slate-50 p-6 text-sm text-slate-600"
          >
            No messages yet. Send a prompt to start.
          </li>
        ) : (
          session.messages.map((m) => (
            <li
              key={m.id}
              data-testid={`message-bubble-${m.id}`}
              data-role={m.role}
            >
              <MessageBubble
                role={m.role}
                text={m.text}
                status={m.status}
                error={m.error ?? null}
              />
            </li>
          ))
        )}

        {isStreaming ? (
          <li
            data-testid="chat-streaming-pill"
            class="self-start rounded-md bg-slate-100 px-3 py-1 text-xs font-medium tracking-wide text-slate-700 uppercase"
            aria-live="polite"
          >
            Streaming…
          </li>
        ) : null}

        {lastMessage?.status === "error" && lastMessage.error ? (
          <li
            data-testid="chat-error-alert"
            role="alert"
            class="rounded-md border border-red-300 bg-red-50 p-3 text-sm text-red-800"
          >
            <strong class="font-semibold">Error:</strong>{" "}
            <span data-testid="chat-error-message">
              {lastMessage.error.message}
            </span>
          </li>
        ) : null}
      </ol>

      <ChatInput
        disabled={!isIdle}
        onSubmit$={stream.submit}
        onCancel$={stream.cancel}
      />
    </section>
  );
});
