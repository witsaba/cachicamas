/**
 * SkillEditor — the main editing panel for skills.
 *
 * Modes:
 *   edit    — editing an existing skill
 *   create  — creating a new skill
 *
 * Features (v1):
 *   - Description field
 *   - Body textarea (NO markdown preview per decision #6)
 *   - Save / Cancel buttons
 *   - Delete button (edit mode only)
 *   - History panel via DiffViewer (PR2c wires dialogs)
 *
 * Anti-drift gates (obs #1959):
 *   - onSave$ payload MUST include BOTH description AND body
 *   - canSave starts FALSE; only TRUE after hasChanges + non-empty
 *   - Signals reset when currentSkill changes (including null)
 *
 * The component is intentionally lightweight in v1: dialogs are in
 * PR2c. We expose `onCancel$`/`onDelete$`/`onRestore$` as QRLs and
 * keep dialog state in the parent route (per the design §4.7).
 */

import {
  component$,
  useSignal,
  useTask$,
  useComputed$,
  $,
  type QRL,
} from "@builder.io/qwik";
import type { Skill, SkillRevision } from "~/lib/skills-api";
import { Button } from "~/components/ui/button/button";
import { skillEditorClasses } from "./classes";

export interface SkillEditorProps {
  skill: Skill | null; // null in create mode
  revisions: SkillRevision[];
  mode: "edit" | "create";
  saving: boolean;
  error: string | null;
  onSave$: QRL<(input: { description: string; body: string }) => void>;
  onCancel$: QRL<() => void>;
  onDelete$: QRL<() => void>;
  onRestore$: QRL<(revisionNumber: number) => void>;
}

export const SkillEditor = component$<SkillEditorProps>(
  ({
    skill,
    saving,
    error,
    onSave$,
    onCancel$,
    onDelete$,
  }) => {
    // Class table for the form controls. Pure function — see ./classes.
    const classes = skillEditorClasses();

    // Local editing state — reset by useTask$ below when skill changes.
    const description = useSignal(skill?.description ?? "");
    const body = useSignal(skill?.body ?? "");
    const hasChanges = useSignal(false);

    // Reset signals when the bound skill changes (including null).
    // This locks the prompts-style "form keeps old values after switching
    // to create mode" bug — see obs #1959 item 4 for the parallel prompt bug.
    useTask$(({ track }) => {
      track(() => skill?.name ?? null);
      if (skill) {
        description.value = skill.description;
        body.value = skill.body;
      } else {
        // No skill — create mode. Clear both signals.
        description.value = "";
        body.value = "";
      }
      // Any load (including the initial render) is NOT a change.
      hasChanges.value = false;
    });

    // canSave is FALSE initially (no changes), FALSE after a load
    // (no changes), TRUE only after the user edits AND both fields
    // are non-empty. The saving flag overrides (already-saved gate).
    const canSave = useComputed$(
      () =>
        !saving &&
        hasChanges.value &&
        description.value.trim().length > 0 &&
        body.value.trim().length > 0,
    );

    const handleSave = $(() => {
      if (!canSave.value) return;
      onSave$({
        description: description.value,
        body: body.value,
      });
    });

    return (
      <div class="flex flex-1 flex-col">
        {/* Header: description */}
        <div class="space-y-2 border-b border-slate-200 px-4 py-3">
          <div class="flex items-center gap-2">
            <label
              class="text-xs font-medium text-slate-500"
              for="skill-description"
            >
              Description
            </label>
            <input
              id="skill-description"
              type="text"
              value={description.value}
              onInput$={(e) => {
                description.value = (e.target as HTMLInputElement).value;
                hasChanges.value = true;
              }}
              placeholder="Short description..."
              class={classes.descriptionInput}
              data-testid="skill-editor-description"
            />
          </div>
        </div>

        {/* Body textarea */}
        <div class="flex flex-1 p-4">
          <textarea
            value={body.value}
            onInput$={(e) => {
              body.value = (e.target as HTMLTextAreaElement).value;
              hasChanges.value = true;
            }}
            placeholder="Write your SKILL.md (YAML frontmatter + markdown body)..."
            class={classes.bodyTextarea}
            data-testid="skill-editor-body"
          />
        </div>

        {/* Error alert */}
        {error && (
          <div
            role="alert"
            class="mx-4 mt-2 rounded border border-red-300 bg-red-50 px-3 py-2 text-sm text-red-800"
            data-testid="skill-editor-error"
          >
            {error}
          </div>
        )}

        {/* Footer: actions */}
        <div class={classes.footerRow}>
          <div>
            {skill && (
              <Button
                type="button"
                variant="destructive"
                class="border border-red-300 !bg-transparent !text-red-700 hover:!bg-red-50"
                onClick$={onDelete$}
                testId="skill-editor-delete"
              >
                Delete skill
              </Button>
            )}
          </div>
          <div class="flex items-center gap-2">
            {!hasChanges.value && skill && (
              <span class={classes.noChangesHint}>No changes to save</span>
            )}
            <Button
              type="button"
              variant="secondary"
              onClick$={onCancel$}
              testId="skill-editor-cancel"
            >
              Cancel
            </Button>
            <Button
              type="button"
              variant="primary"
              disabled={!canSave.value}
              loading={saving}
              onClick$={handleSave}
              testId="skill-editor-save"
            >
              {saving ? "Saving…" : "Save"}
            </Button>
          </div>
        </div>
      </div>
    );
  },
);
