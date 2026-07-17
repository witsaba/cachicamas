/**
 * SkillSidebar — left panel with a filterable list of skills.
 *
 * Props:
 *   skills        — all skills
 *   selectedName  — currently selected skill name (or null)
 *   onSelect$     — handler when a skill is clicked (receives the name)
 *   onNewSkill$   — handler when "+ New Skill" is clicked
 */

import { component$, type QRL, useSignal, $ } from "@builder.io/qwik";
import type { Skill } from "~/lib/skills-api";
import { SkillListItem } from "~/components/skills/skill-list-item/skill-list-item";
import { Button } from "~/components/ui/button/button";

export interface SkillSidebarProps {
  skills: Skill[];
  selectedName: string | null;
  onSelect$: QRL<(name: string) => void>;
  onNewSkill$: QRL<() => void>;
}

export const SkillSidebar = component$<SkillSidebarProps>(
  ({ skills, selectedName, onSelect$, onNewSkill$ }) => {
    const filter = useSignal("");

    // Filter via case-insensitive substring on the skill name. Mirrors
    // the prompts sidebar behavior; agentskills.io names are lowercase
    // so this is mostly defensive.
    const filtered = skills.filter((s) =>
      s.name.toLowerCase().includes(filter.value.trim().toLowerCase()),
    );

    // Wrapper QRL: SkillListItem expects QRL<() => void>, but our
    // onSelect$ carries the name. Adapt in place.
    const selectFor = (name: string) =>
      $(() => {
        onSelect$(name);
      });

    return (
      <div class="flex h-full w-64 flex-col border-r border-slate-200 bg-white">
        {/* Filter input */}
        <div class="border-b border-slate-100 p-3">
          <input
            type="text"
            placeholder="Filter skills..."
            value={filter.value}
            onInput$={(e) => {
              filter.value = (e.target as HTMLInputElement).value;
            }}
            class="w-full rounded border border-slate-200 px-2 py-1 text-sm text-slate-900 placeholder-slate-400 focus:border-slate-400 focus:ring-1 focus:ring-slate-400 focus:outline-none"
            aria-label="Filter skills"
            data-testid="skill-sidebar-filter"
          />
        </div>

        {/* Skill list */}
        <ul
          class="flex-1 overflow-y-auto py-1"
          role="listbox"
          aria-label="Skills"
          data-testid="skill-sidebar-list"
        >
          {filtered.length === 0 ? (
            <li class="px-3 py-4 text-center text-sm text-slate-400">
              {skills.length === 0
                ? "No skills yet"
                : "No skills match your filter"}
            </li>
          ) : (
            filtered.map((skill) => (
              <SkillListItem
                key={skill.name}
                skill={skill}
                selected={skill.name === selectedName}
                onClick$={selectFor(skill.name)}
              />
            ))
          )}
        </ul>

        {/* New skill button */}
        <div class="border-t border-slate-100 p-3">
          <Button
            type="button"
            variant="primary"
            class="w-full justify-center"
            onClick$={onNewSkill$}
            testId="skill-sidebar-new"
          >
            <svg
              class="mr-1.5 h-4 w-4"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              stroke-width={2}
              aria-hidden="true"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                d="M12 4v16m8-8H4"
              />
            </svg>
            New Skill
          </Button>
        </div>
      </div>
    );
  },
);
