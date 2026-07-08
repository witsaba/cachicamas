/**
 * HomeWorkspacesSection — /home workspaces display (R-WS-015).
 *
 * Reference: openspec/changes/2026-07-06-workspaces/specs
 *   R-WS-015 (S-WS-140..143) — `/home` workspaces integration.
 *
 * Rendering logic:
 *   - loading: shows "Loading…"
 *   - error: shows alert + Retry button
 *   - zero workspaces: shows empty CTA "Create your first workspace"
 *   - 1..N workspaces: shows up to 5 cards + "View all" link + Create CTA
 *   - truncated: shows "Showing 5 of N workspaces" warning when N > 5
 *
 * Aphantasic-friendly: text-first, no imagery, monochrome (bg-slate-900).
 */
import { component$, type QRL } from "@builder.io/qwik";
import { Button } from "~/components/ui/button/button";
import { WorkspaceCard } from "~/components/workspace-card/workspace-card";
import type { WorkspaceSummary } from "~/lib/api";

export interface HomeWorkspacesSectionProps {
  loading: boolean;
  error: string | null;
  workspaces: WorkspaceSummary[];
  truncated: boolean;
  onRetry: QRL<() => void>;
  onNavigate: QRL<(id: number) => void>;
  onCreateWorkspace: QRL<() => void>;
}

const MAX_DISPLAY = 5;

export const HomeWorkspacesSection = component$<HomeWorkspacesSectionProps>(
  ({ loading, error, workspaces, truncated, onRetry }) => {
    if (loading) {
      return (
        <section data-testid="home-workspaces-loading" class="mt-8">
          <p class="text-slate-700">Loading workspaces…</p>
        </section>
      );
    }
    if (error) {
      return (
        <section
          role="alert"
          data-testid="home-workspaces-error"
          class="mt-8 space-y-3"
        >
          <p class="text-red-700">{error}</p>
          <Button
            type="button"
            variant="primary"
            testId="home-workspaces-retry"
            onClick$={onRetry}
            class="px-3 py-1"
          >
            Retry
          </Button>
        </section>
      );
    }
    if (workspaces.length === 0) {
      return (
        <section
          data-testid="home-workspaces-empty"
          class="mx-auto mt-8 max-w-xl space-y-3 rounded border border-slate-200 bg-white p-8 text-center"
        >
          <h2 class="text-xl font-semibold text-slate-900">
            No workspaces yet
          </h2>
          <p class="text-slate-700">
            Create your first one to connect a GitHub repository.
          </p>
          <Button
            as="a"
            href="/workspaces/new"
            variant="primary"
            testId="home-workspaces-create-cta"
          >
            Create workspace
          </Button>
        </section>
      );
    }
    const visible = workspaces.slice(0, MAX_DISPLAY);
    const hiddenCount = workspaces.length - visible.length;
    return (
      <section data-testid="home-workspaces-list" class="mt-8 space-y-4">
        <div class="flex items-center justify-between">
          <h2 class="text-xl font-semibold text-slate-900">Your workspaces</h2>
          <Button
            as="a"
            href="/workspaces/new"
            variant="primary"
            testId="home-workspaces-create-cta"
            class="px-3 py-1"
          >
            Create workspace
          </Button>
        </div>
        {truncated ? (
          <p class="text-sm text-slate-600">
            Showing the first {MAX_DISPLAY} of {workspaces.length} workspaces.
          </p>
        ) : null}
        <ul class="space-y-2">
          {visible.map((w) => (
            <li key={w.id}>
              <WorkspaceCard workspace={w} />
            </li>
          ))}
        </ul>
        {hiddenCount > 0 || truncated ? (
          <a
            href="/workspaces"
            data-testid="home-workspaces-view-all"
            class="inline-block text-sm text-slate-700 underline"
          >
            View all {workspaces.length} workspaces
          </a>
        ) : null}
      </section>
    );
  },
);
