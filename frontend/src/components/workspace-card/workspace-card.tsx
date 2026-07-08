/**
 * WorkspaceCard — list card for the /workspaces page.
 *
 * Reference: openspec/changes/2026-07-08-workspaces-simplify/specs/workspaces/spec.md
 *   R-WS-002 — list card contract (post-1:1)
 *
 * 2026-07-08-workspaces-simplify changelog:
 *   - Dropped the linked_repos_count display + the "No linked repos"
 *     / "N linked repos" label. In the 1:1 model the workspace IS
 *     the repo; there is nothing else to count.
 *   - Renamed `primary_repository` -> `repository` to match the new
 *     WorkspaceSummary wire shape.
 *
 * Design system rule (cachicamas UAT catch): the primary CTAs use
 * bg-slate-900 (project monochrome). Tinted "open" link.
 *
 * Aphantasic-friendly (UX-4): text-first. Renders name + repository +
 * an "Open" link to /workspaces/:id. No decorative iconography.
 */
import { component$ } from "@builder.io/qwik";
import { Button } from "~/components/ui/button/button";
import type { WorkspaceSummary } from "~/lib/api";

export interface WorkspaceCardProps {
  workspace: WorkspaceSummary;
}

export const WorkspaceCard = component$<WorkspaceCardProps>(({ workspace }) => {
  const { id, name, repository } = workspace;
  const fullName = repository.full_name;
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
        </p>
      </div>
      <Button
        as="a"
        href={`/workspaces/${id}`}
        variant="primary"
        testId="workspace-card-open"
        class="shrink-0"
      >
        Open
      </Button>
    </article>
  );
});
