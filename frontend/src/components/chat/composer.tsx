import { component$, useSignal, type QRL } from "@builder.io/qwik";
import { Icon } from "~/components/icon/icon";
import { Button } from "~/components/ui/button/button";

/**
 * Where you say something.
 *
 * A bordered field that grows to the message, Enter to send, Shift+Enter for a
 * new line — the shape every messaging surface has, because a person should
 * not have to learn how to type here.
 *
 * While a colleague is working, the field is disabled and the action becomes
 * Stop. A colleague you cannot interrupt is the failure this product exists to
 * avoid, so Stop is never more than one key away.
 */
export interface ComposerProps {
  readonly status: "idle" | "running" | "held";
  readonly agentName: string;
  readonly onSubmit$: QRL<(prompt: string) => void>;
  readonly onCancel$: QRL<() => void>;
}

export const Composer = component$<ComposerProps>((props) => {
  const draft = useSignal("");
  const busy = props.status !== "idle";

  return (
    <form
      data-testid="composer"
      preventdefault:submit
      onSubmit$={async () => {
        if (busy) return;
        const text = draft.value;
        draft.value = "";
        await props.onSubmit$(text);
      }}
      class="border-line bg-surface border-t px-4 py-3"
    >
      <div
        class={[
          "bg-surface mx-auto flex w-full max-w-2xl items-end gap-2 rounded-lg border p-2 transition-colors duration-150",
          busy ? "border-line" : "border-line-control",
        ].join(" ")}
      >
        <textarea
          id="composer-input"
          data-testid="composer-input"
          rows={1}
          disabled={busy}
          value={draft.value}
          aria-label={`Message ${props.agentName}`}
          placeholder={
            props.status === "held"
              ? "Answer the request above before carrying on."
              : props.status === "running"
                ? `${props.agentName} is working. Stop to type again.`
                : `Message ${props.agentName}…`
          }
          class="text-md text-ink max-h-40 min-h-9 flex-1 resize-none border-none bg-transparent px-1.5 py-1.5 leading-relaxed outline-none disabled:opacity-45"
          onInput$={(_, el) => {
            draft.value = el.value;
            el.style.height = "auto";
            el.style.height = `${Math.min(el.scrollHeight, 160)}px`;
          }}
          onKeyDown$={async (event, el) => {
            if (event.key === "Enter" && !event.shiftKey) {
              event.preventDefault();
              if (busy) return;
              const text = draft.value;
              draft.value = "";
              el.value = "";
              el.style.height = "auto";
              await props.onSubmit$(text);
            }
          }}
        />
        {props.status === "running" ? (
          <Button
            variant="destructive"
            size="sm"
            testId="composer-stop"
            onClick$={() => props.onCancel$()}
          >
            <Icon name="stop" size={14} />
            Stop
          </Button>
        ) : (
          <Button
            type="submit"
            variant="primary"
            size="sm"
            testId="composer-send"
            disabled={busy}
          >
            <Icon name="send" size={14} />
            Send
          </Button>
        )}
      </div>
      <p class="text-ink-soft mx-auto w-full max-w-2xl pt-2 text-xs">
        Enter to send, Shift + Enter for a new line. Nothing is sent outside
        your company without you approving it first.
      </p>
    </form>
  );
});
