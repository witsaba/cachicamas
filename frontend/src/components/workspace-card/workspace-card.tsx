/**
 * WorkspaceCard — list card for the /workspaces page.
 *
 * Reference: openspec/changes/2026-07-06-workspaces/specs
 *   R-WS-011 (S-WS-100..103) — list card contract.
 *
 * Design system rule (cachicamas UAT catch): the primary CTAs use
 * bg-slate-900 (project monochrome). Tinted "open" link.
 *
 * Aphantasic-friendly (UX-4): text-first. Renders name + primary repo +
 * linked count + an "Open" link to /workspaces/:id. No decorative iconography.
 */
import { component$ } from "@builder.io/qwik";
import type { WorkspaceSummary } from "~/lib/api";

export interface WorkspaceCardProps {
  workspace: WorkspaceSummary;
}

export const WorkspaceCard = component$<WorkspaceCardProps>(({ workspace }) => {
  const { id, name, primary_repository, linked_repos_count } = workspace;
  const fullName = primary_repository.full_name;
  const linkedLabel =
    linked_repos_count === 0
      ? "No linked repos"
      : linked_repos_count === 1
        ? "1 linked repo"
        : `${linked_repos_count} linked repos`;
  return (
    <article
      data-testid="workspace-card"
      data-workspace-id={id}
      class="flex items-center justify-between gap-4 rounded-lg border border-slate-200 bg-white px-5 py-4 shadow-sm transition-shadow hover:shadow"
    >
      <div class="min-w-0">
        <h2
          data-testid="workspace-card-name"
          class="truncate text-lg font-semibold text-slate-900"
        >
          {name}
        </h2>
        <p
          data-testid="workspace-card-repo"
          class="truncate text-sm text-slate-600"
        >
          <span class="font-mono">{fullName}</span>
          <span class="mx-2 text-slate-300">·</span>
          <span data-testid="workspace-card-linked-count">{linkedLabel}</span>
        </p>
      </div>
      <a
        href={`/workspaces/${id}`}
        data-testid="workspace-card-open"
        class="inline-flex shrink-0 items-center rounded bg-slate-900 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-slate-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-900"
      >
        Open
      </a>
    </article>
  );
});
