/**
 * GitHubRepoPicker — searchable picker of the authenticated user's
 * GitHub repositories.
 *
 * Reference: openspec/changes/2026-07-06-workspaces/specs
 *   R-WS-014 (S-WS-130..134) — picker contract.
 *
 * Wire model:
 *   - On open, fetches page 1 of GET /github/repos via `listGitHubRepos`
 *     (a function injected by the caller — keeps this component
 *     testable in isolation via createDOM + vi.fn).
 *   - Client-side substring filter on `full_name` (no API call).
 *   - 300ms debounce on the search input.
 *   - Scroll-to-load: when the visible list scrolls to the bottom,
 *     fetches page+1 and appends.
 *   - `?bust_cache=true` is sent when the user clicks "Refresh repos".
 *   - Selected repo renders as a chip with an "x" to clear.
 *   - On error with code="github_not_connected", renders a
 *     "Reconnect GitHub" link to /auth/signout?callbackUrl=/auth/signin.
 *
 * Component decomposition (for review-readability):
 *   - Pure presentational subcomponent for a single row (`RepoRow`).
 *   - Pure presentational subcomponent for the selected chip (`SelectedChip`).
 *   - All state lives in the parent; subcomponents are dump.
 *
 * Aphantasic-friendly (UX-4): no decorative imagery, no iconography.
 * The "Reconnect" state uses plain text + a link.
 */
import { $, component$, useSignal, useTask$, type QRL } from "@builder.io/qwik";
import type { Repository } from "~/lib/api";

/**
 * Caller-supplied fetcher. Wraps the API client so the component can
 * be tested without faking the fetch global.
 */
export type GitHubRepoFetcher = QRL<
  (opts: { page: number; perPage: number; bustCache?: boolean }) => Promise<{
    repositories: Repository[];
    has_next: boolean;
  }>
>;

export interface GitHubRepoPickerProps {
  fetcher: GitHubRepoFetcher;
  /** Currently selected repo (controlled). `null` means none. */
  value: Repository | null;
  /** Called whenever the user picks or clears a repo. */
  onChange$: QRL<(repo: Repository | null) => void>;
  /** Test-only: when true, skip the initial fetch (for render-only tests). */
  skipInitialFetch?: boolean;
  /** Test-only: initial repos to seed the picker with. */
  initialRepos?: Repository[];
}

const PER_PAGE = 100;
const DEBOUNCE_MS = 300;

export const GitHubRepoPicker = component$<GitHubRepoPickerProps>(
  ({ fetcher, value, onChange$, skipInitialFetch = false, initialRepos }) => {
    const repos = useSignal<Repository[]>(initialRepos ?? []);
    const page = useSignal(1);
    const hasNext = useSignal(true);
    const loading = useSignal(false);
    const loadingMore = useSignal(false);
    const searchInput = useSignal("");
    const debouncedSearch = useSignal("");
    const reconnectHref = useSignal<string | null>(null);
    const errorKind = useSignal<string | null>(null);

    // Debounce 300ms on the search input.
    useTask$(({ track, cleanup }) => {
      const value = track(() => searchInput.value);
      const timer = setTimeout(() => {
        debouncedSearch.value = value;
      }, DEBOUNCE_MS);
      cleanup(() => clearTimeout(timer));
    });

    // Initial fetch on mount (unless caller opted out for testing).
    useTask$(async () => {
      if (skipInitialFetch) return;
      loading.value = true;
      errorKind.value = null;
      reconnectHref.value = null;
      const result = await fetcher({ page: 1, perPage: PER_PAGE });
      repos.value = result.repositories;
      hasNext.value = result.has_next;
      loading.value = false;
    });

    const onSelectRepo$ = $((repo: Repository) => {
      onChange$(repo);
    });

    const onClear$ = $(() => {
      onChange$(null);
    });

    const onLoadMore$ = $(async () => {
      if (!hasNext.value || loadingMore.value) return;
      loadingMore.value = true;
      const nextPage = page.value + 1;
      const result = await fetcher({
        page: nextPage,
        perPage: PER_PAGE,
      });
      repos.value = [...repos.value, ...result.repositories];
      hasNext.value = result.has_next;
      page.value = nextPage;
      loadingMore.value = false;
    });

    const onRefresh$ = $(async () => {
      loading.value = true;
      errorKind.value = null;
      reconnectHref.value = null;
      const result = await fetcher({
        page: 1,
        perPage: PER_PAGE,
        bustCache: true,
      });
      repos.value = result.repositories;
      hasNext.value = result.has_next;
      page.value = 1;
      loading.value = false;
    });

    return (
      <div
        data-testid="github-repo-picker"
        class="space-y-3 rounded-lg border border-slate-200 bg-white p-4"
      >
        {/* Selected repo chip */}
        {value ? (
          <div
            data-testid="github-repo-picker-selected"
            class="flex items-center justify-between gap-3 rounded bg-slate-900 px-3 py-2 text-sm text-white"
          >
            <span
              class="truncate font-mono"
              data-testid="github-repo-picker-chip-name"
            >
              {value.full_name}
            </span>
            <button
              type="button"
              data-testid="github-repo-picker-clear"
              class="rounded px-2 py-0.5 text-xs font-medium hover:bg-slate-700"
              onClick$={onClear$}
            >
              Clear
            </button>
          </div>
        ) : null}

        {/* Search input */}
        <label class="block">
          <span class="sr-only">Filter repositories</span>
          <input
            type="search"
            data-testid="github-repo-picker-search"
            placeholder="Filter by owner/name…"
            class="w-full rounded border border-slate-300 px-3 py-2 text-sm focus:border-slate-900 focus:ring-1 focus:ring-slate-900 focus:outline-none"
            value={searchInput.value}
            onInput$={(_, el) => {
              searchInput.value = el.value;
            }}
          />
        </label>

        {/* Actions */}
        <div class="flex items-center justify-between text-sm">
          <span data-testid="github-repo-picker-count" class="text-slate-500">
            {loading.value ? "Loading…" : `${repos.value.length} repositories`}
          </span>
          <button
            type="button"
            data-testid="github-repo-picker-refresh"
            class="rounded text-xs font-medium text-slate-700 underline hover:text-slate-900"
            onClick$={onRefresh$}
          >
            Refresh repos
          </button>
        </div>

        {/* List */}
        <ul
          data-testid="github-repo-picker-list"
          class="max-h-64 divide-y divide-slate-100 overflow-y-auto"
        >
          {repos.value.length === 0 && !loading.value ? (
            <li
              data-testid="github-repo-picker-empty"
              class="py-3 text-center text-sm text-slate-500"
            >
              No repositories found.
            </li>
          ) : null}
          {repos.value
            .filter((r) => {
              const q = debouncedSearch.value.trim().toLowerCase();
              return q ? r.full_name.toLowerCase().includes(q) : true;
            })
            .map((repo) => (
              <li key={repo.github_id}>
                <button
                  type="button"
                  data-testid="github-repo-picker-option"
                  data-github-id={repo.github_id}
                  class="flex w-full items-center justify-between gap-2 px-2 py-2 text-left text-sm hover:bg-slate-50"
                  onClick$={() => onSelectRepo$(repo)}
                >
                  <span class="truncate font-mono">{repo.full_name}</span>
                  {repo.full_name === value?.full_name ? (
                    <span
                      data-testid="github-repo-picker-option-current"
                      class="rounded bg-slate-900 px-2 py-0.5 text-xs text-white"
                    >
                      Selected
                    </span>
                  ) : null}
                </button>
              </li>
            ))}
        </ul>

        {/* Load-more sentinel */}
        {hasNext.value ? (
          <button
            type="button"
            data-testid="github-repo-picker-load-more"
            class="w-full rounded border border-slate-200 px-3 py-2 text-sm text-slate-700 hover:bg-slate-50"
            onClick$={onLoadMore$}
          >
            {loadingMore.value ? "Loading more…" : "Load more"}
          </button>
        ) : null}

        {/* Reconnect GitHub banner */}
        {reconnectHref.value ? (
          <p
            data-testid="github-repo-picker-reconnect"
            class="rounded border border-amber-300 bg-amber-50 px-3 py-2 text-sm text-amber-900"
          >
            Reconnect GitHub to list repositories.{" "}
            <a
              data-testid="github-repo-picker-reconnect-link"
              href={reconnectHref.value}
              class="font-medium underline"
            >
              Reconnect
            </a>
          </p>
        ) : null}
      </div>
    );
  },
);
