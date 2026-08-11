/**
 * SkillListItem — a single skill in the sidebar list.
 *
 * Props:
 *   skill     — the Skill object
 *   selected  — whether this item is currently selected
 *   onClick$  — handler called when the item is clicked
 *
 * Anti-drift gate (obs #1959 item 2):
 *   MUST render `v{skill.current_revision}` (never `v{undefined}`).
 *   Backend emits `current_revision` via SQL JOIN (ADR-SK-008) but
 *   we defend against runtime undefined so a future fixture change
 *   cannot reintroduce the prompts `vundefined` regression.
 */
import { component$, type QRL } from "@builder.io/qwik";
import type { Skill } from "~/lib/skills-api";
import { listItemClasses } from "./classes";

export interface SkillListItemProps {
  skill: Skill;
  selected: boolean;
  onClick$: QRL<() => void>;
}

/** Format an ISO timestamp as YYYY-MM-DD for compact display. */
function formatDate(iso: string | null): string {
  if (!iso) return "—";
  const d = new Date(iso);
  if (isNaN(d.getTime())) return "—";
  const pad = (n: number) => n.toString().padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

/** Safe accessor for current_revision. Falls back to 0 if missing. */
function safeRevision(rev: number | undefined | null): number {
  return typeof rev === "number" && !isNaN(rev) ? rev : 0;
}

export const SkillListItem = component$<SkillListItemProps>(
  ({ skill, selected, onClick$ }) => {
    const classes = listItemClasses(selected);
    const rev = safeRevision(skill.current_revision);

    return (
      <li>
        <button
          type="button"
          onClick$={onClick$}
          class={classes.container}
          aria-current={selected ? "true" : undefined}
          data-testid={
            selected ? "skill-list-item-selected" : "skill-list-item"
          }
        >
          <span class={classes.name} data-testid="skill-list-item-name">
            {skill.name}
          </span>
          <span class={classes.meta} data-testid="skill-list-item-meta">
            v{rev} &middot; {formatDate(skill.updated_at)}
          </span>
        </button>
      </li>
    );
  },
);
