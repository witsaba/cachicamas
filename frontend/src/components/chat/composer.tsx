import { component$, useSignal, type QRL } from "@builder.io/qwik";
import { Button } from "~/components/ui/button/button";

/**
 * Composer — where a turn is opened.
 *
 * It is deliberately the same object as the command line at the top of the
 * screen: an amber prompt character, a bare field, a key hint. One is how you
 * address the system, the other is how you address a specialist, and the
 * repetition is the point — the interface has exactly one input shape.
 *
 * While a turn is running the field is disabled and the primary action becomes
 * STOP, because a run you cannot interrupt is the failure mode this product
 * exists to avoid (PRODUCT.md § Product Principles 3).
 */
export interface ComposerProps {
  readonly status: "idle" | "running" | "held";
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
      class="border-rule bg-panel border-t"
    >
      <div class="flex items-start gap-2 px-3 py-2">
        <label
          for="composer-input"
          class="text-label text-amber pt-1 select-none"
          aria-hidden="true"
        >
          &gt;
        </label>
        <textarea
          id="composer-input"
          data-testid="composer-input"
          rows={1}
          disabled={busy}
          value={draft.value}
          aria-label="Message the chat archetype"
          placeholder={
            props.status === "held"
              ? "The run is suspended — answer the permission request above."
              : props.status === "running"
                ? "The turn is running. Stop it to type again."
                : "Ask the chat archetype something. Enter to send, Shift+Enter for a new line."
          }
          class="font-human text-body text-fg max-h-40 min-h-8 flex-1 resize-none border-none bg-transparent leading-relaxed outline-none disabled:opacity-40"
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
            testId="composer-stop"
            onClick$={() => props.onCancel$()}
          >
            Stop
          </Button>
        ) : (
          <Button
            type="submit"
            variant="primary"
            testId="composer-send"
            disabled={busy}
          >
            Send
          </Button>
        )}
      </div>
      <p class="text-legend text-fg-dim px-3 pb-2 pl-7 tracking-[0.1em] uppercase">
        Demonstration only · no turn reaches a model · doc 0005 is 0 of 12
      </p>
    </form>
  );
});
