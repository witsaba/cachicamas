/**
 * ActivityFeed — shows per-prompt activity events.
 *
 * Props:
 *   events  — list of activity events (newest first)
 *   prompt  — the current prompt (to derive the "created" event)
 */

import { component$ } from "@builder.io/qwik";
import type { Prompt, PromptRevision } from "~/lib/prompts-api";
import { ActivityEvent, type ActivityEventData } from "./activity-event";

export interface ActivityFeedProps {
  prompt: Prompt;
  revisions: PromptRevision[];
  testId?: string;
}

/**
 * Build activity events from the prompt and its revisions.
 * - "created" event from the oldest revision
 * - "edited" events for each revision that is not the first
 * - We derive event types from revision timestamps (not explicit in the API)
 *
 * The API doesn't provide explicit event types — we infer them from the
 * revision list. The first revision = created. Subsequent revisions = edited.
 */
function buildActivityEvents(
  prompt: Prompt,
  revisions: PromptRevision[],
): ActivityEventData[] {
  if (revisions.length === 0) {
    return [
      {
        type: "created",
        revisionNumber: prompt.current_revision,
        timestamp: prompt.created_at,
      },
    ];
  }

  const events: ActivityEventData[] = revisions
    .map((rev) => ({
      type: "edited" as const,
      revisionNumber: rev.revision_number,
      timestamp: rev.created_at,
    }))
    .sort((a, b) => b.revisionNumber - a.revisionNumber); // newest first

  // Add a "created" event for the oldest revision
  const oldest = revisions[revisions.length - 1];
  if (oldest) {
    events.push({
      type: "created",
      revisionNumber: oldest.revision_number,
      timestamp: oldest.created_at,
    });
  }

  return events.sort(
    (a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime(),
  );
}

export const ActivityFeed = component$<ActivityFeedProps>(
  ({ prompt, revisions, testId }) => {
    const events = buildActivityEvents(prompt, revisions);

    return (
      <div
        class="mt-4 rounded border border-slate-200 bg-white"
        data-testid={testId ?? "activity-feed"}
        role="feed"
        aria-label="Prompt activity"
      >
        <div class="border-b border-slate-100 px-3 py-2">
          <h3 class="text-xs font-semibold tracking-wide text-slate-500 uppercase">
            Activity
          </h3>
        </div>
        <ul class="px-3 py-2">
          {events.length === 0 ? (
            <li class="py-2 text-sm text-slate-400">No activity yet.</li>
          ) : (
            events.map((event, idx) => (
              <ActivityEvent
                key={idx}
                event={event}
                testId={`activity-event-${idx}`}
              />
            ))
          )}
        </ul>
      </div>
    );
  },
);
