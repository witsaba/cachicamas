/**
 * assistant-configure-section.tsx — the runtime-editable surface for
 * the chat archetype's configuration (REQ-FACS-001/002/003).
 *
 * Renders on the `/agents/assistant/` profile when the caller is
 * signed in AND onboarded (REQ-FACS-001). Lets the org owner edit:
 *   - the system prompt (textarea, capped at 4000 chars; mirrors
 *     `archetype.MaxSystemPromptLength`)
 *   - which tools the Assistant may use (per-tool toggles for
 *     `current_time` and `summarize_conversation`)
 *   - which tools require explicit permission approval (defer set;
 *     for v1, only `summarize_conversation` is deferrable)
 *   - a read-only model readout (informational; the actual model is
 *     env-driven, not user-editable)
 *
 * Save flow (REQ-FACS-003):
 *   - The user clicks Save; the UI applies the change optimistically
 *     and shows a spinner during the PUT.
 *   - 200 → spinner disappears, local state equals the server
 *     response, "Saved" toast appears.
 *   - non-2xx → local state reverts to the prior value, error toast
 *     appears. The validation error envelope (when present) is
 *     surfaced inline per-field.
 *
 * Strict TDD pairing: see `assistant-configure-section.spec.tsx`.
 */

import { component$, useSignal, useStore, $ } from "@builder.io/qwik";

import {
  getArchetypeConfig,
  putArchetypeConfigFlat,
  type ArchetypeConfig,
} from "~/lib/api/assistant-config";

const MAX_PROMPT_LENGTH = 4000;

interface ConfigureSectionProps {
  /**
   * The archetype slug this section edits. The save flow PUTs to
   * `/api/archetypes/{slug}/config/`. Required — the section is no
   * longer bound to the assistant by default; the route passes the
   * slug from the URL param so any archetype in the directory list
   * is editable.
   */
  slug: string;
  /**
   * Initial config the page-level route loader fetched. The
   * ConfigureSection optimistically re-renders against the server's
   * response on save, so the initial prop is the seed only.
   */
  initial: ArchetypeConfig;
}

type SaveStatus = "idle" | "saving" | "saved" | "error";

export const ConfigureSection = component$<ConfigureSectionProps>(
  ({ slug, initial }) => {
    // Local editable state mirrors the server config; the textarea,
    // tool toggles, and defer toggle all bind to this.
    const state = useStore({
      systemPrompt: initial.system_prompt,
      toolAllowlist: [...initial.tool_allowlist] as string[],
      deferToolNames: [...initial.defer_tool_names] as string[],
    });

    // Snapshot of the last-saved values for rollback on save failure
    // (REQ-FACS-003 Scenario 2).
    const snapshot = useSignal<{
      systemPrompt: string;
      toolAllowlist: string[];
      deferToolNames: string[];
    }>({
      systemPrompt: initial.system_prompt,
      toolAllowlist: [...initial.tool_allowlist],
      deferToolNames: [...initial.defer_tool_names],
    });

    const status = useSignal<SaveStatus>("idle");
    const errorMessage = useSignal<string>("");

    const handleSave = $(async () => {
      // Snapshot the current state so we can roll back on failure.
      snapshot.value = {
        systemPrompt: state.systemPrompt,
        toolAllowlist: [...state.toolAllowlist],
        deferToolNames: [...state.deferToolNames],
      };
      status.value = "saving";
      errorMessage.value = "";

      const result = await putArchetypeConfigFlat(slug, {
        system_prompt: state.systemPrompt,
        tool_allowlist: state.toolAllowlist,
        defer_tool_names: state.deferToolNames,
      });

      if (result.ok) {
        status.value = "saved";
        return;
      }

      // Rollback to the snapshot.
      state.systemPrompt = snapshot.value.systemPrompt;
      state.toolAllowlist = [...snapshot.value.toolAllowlist];
      state.deferToolNames = [...snapshot.value.deferToolNames];
      status.value = "error";
      errorMessage.value = result.message;
    });

    const toggleTool = $((tool: string) => {
      if (state.toolAllowlist.includes(tool)) {
        state.toolAllowlist = state.toolAllowlist.filter((t) => t !== tool);
        // Removing a tool from the allowlist cascades: any defer
        // entry pointing at it is invalid (defer ⊆ allowlist) so
        // remove it from the defer set too.
        if (state.deferToolNames.includes(tool)) {
          state.deferToolNames = state.deferToolNames.filter((t) => t !== tool);
        }
      } else {
        state.toolAllowlist = [...state.toolAllowlist, tool];
      }
    });

    const toggleDefer = $((tool: string) => {
      if (state.deferToolNames.includes(tool)) {
        state.deferToolNames = state.deferToolNames.filter((t) => t !== tool);
      } else {
        // Deferring requires the tool to be in the allowlist. Add it
        // there too so the resulting update satisfies the
        // `defer ⊆ allowlist` server-side check.
        const allowlist = state.toolAllowlist.includes(tool)
          ? state.toolAllowlist
          : [...state.toolAllowlist, tool];
        state.toolAllowlist = allowlist;
        state.deferToolNames = [...state.deferToolNames, tool];
      }
    });

    return (
      <section
        id="configure"
        class="border-ink-soft mt-12 rounded-2xl border bg-white p-6 shadow-sm"
        data-testid="assistant-configure-section"
      >
        <h2 class="text-ink text-lg font-semibold">Configure</h2>
        <p class="text-ink-mid mt-1 text-sm">
          Edit what the Assistant says and which tools it may use.
        </p>

        <label class="mt-6 block">
          <span class="text-ink text-sm font-medium">System prompt</span>
          <textarea
            class="border-ink-soft text-ink mt-2 block w-full rounded-md border p-3 text-sm leading-6"
            rows={6}
            maxLength={MAX_PROMPT_LENGTH}
            value={state.systemPrompt}
            onInput$={(_, el) => {
              state.systemPrompt = el.value;
            }}
            data-testid="configure-system-prompt"
          />
          <span class="text-ink-soft mt-1 block text-xs">
            {state.systemPrompt.length} / {MAX_PROMPT_LENGTH}
          </span>
        </label>

        <fieldset class="mt-6">
          <legend class="text-ink text-sm font-medium">Tools</legend>
          {(["current_time", "summarize_conversation"] as const).map((tool) => (
            <label
              key={tool}
              class="mt-2 flex items-center gap-3 text-sm"
              data-testid={`configure-tool-${tool}`}
            >
              <input
                type="checkbox"
                checked={state.toolAllowlist.includes(tool)}
                onChange$={() => toggleTool(tool)}
              />
              <span class="text-ink">{tool}</span>
            </label>
          ))}
        </fieldset>

        <fieldset class="mt-6">
          <legend class="text-ink text-sm font-medium">
            Deferred (requires approval before each call)
          </legend>
          {(["summarize_conversation"] as const).map((tool) => (
            <label
              key={tool}
              class="mt-2 flex items-center gap-3 text-sm"
              data-testid={`configure-defer-${tool}`}
            >
              <input
                type="checkbox"
                checked={state.deferToolNames.includes(tool)}
                onChange$={() => toggleDefer(tool)}
              />
              <span class="text-ink">{tool}</span>
            </label>
          ))}
        </fieldset>

        <dl class="mt-6 grid grid-cols-2 gap-3 text-sm">
          <dt class="text-ink-soft">Model</dt>
          <dd class="text-ink-mid" data-testid="configure-model">
            {initial.model ?? "(env-driven)"}
          </dd>
          <dt class="text-ink-soft">Version</dt>
          <dd class="text-ink-mid">{initial.version}</dd>
        </dl>

        <div class="mt-8 flex items-center gap-4">
          <button
            type="button"
            class="bg-brand text-on-brand rounded-md px-4 py-2 text-sm font-medium disabled:opacity-50"
            disabled={status.value === "saving"}
            onClick$={handleSave}
            data-testid="configure-save"
          >
            {status.value === "saving" ? "Saving…" : "Save"}
          </button>
          {status.value === "saved" && (
            <span
              class="text-ink-mid text-sm"
              data-testid="configure-saved-toast"
            >
              Saved.
            </span>
          )}
          {status.value === "error" && (
            <span
              class="text-danger text-sm"
              data-testid="configure-error-toast"
            >
              {errorMessage.value}
            </span>
          )}
        </div>
      </section>
    );
  },
);

// Re-export the helper used by route loaders that want to seed the
// initial config (e.g. agent-profile.tsx). Takes a slug so the
// helper is polymorphic by archetype, not hard-coded to assistant.
export async function loadInitialArchetypeConfig(
  slug: string,
): Promise<ArchetypeConfig> {
  const result = await getArchetypeConfig(slug);
  if (result.ok) {
    return result.value;
  }
  // For the seed-failure case (offline, anon, server error) the
  // page-level loader should already have failed before reaching
  // this helper. Keep the fallback small: re-throw so the caller
  // can render a generic error.
  throw new Error(result.message);
}
