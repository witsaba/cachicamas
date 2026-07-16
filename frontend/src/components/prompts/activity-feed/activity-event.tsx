/**
 * ActivityEvent — a single event in the activity feed.
 *
 * Props:
 *   event — the event to render
 */

import { component$ } from "@builder.io/qwik";
import { eventClasses, type EventType } from "./classes";

export interface ActivityEventData {
  type: EventType;
  revisionNumber: number;
  timestamp: string;
}

export interface ActivityEventProps {
  event: ActivityEventData;
  testId?: string;
}

function formatTime(iso: string): string {
  const d = new Date(iso);
  if (isNaN(d.getTime())) return "—";
  const pad = (n: number) => n.toString().padStart(2, "0");
  return `${pad(d.getHours())}:${pad(d.getMinutes())}`;
}

/** Get the human-readable label for an event type. */
function eventLabel(type: EventType, revisionNumber: number): string {
  switch (type) {
    case "created":
      return `v${revisionNumber} created`;
    case "edited":
      return `v${revisionNumber} saved`;
    case "restored":
      return `v${revisionNumber} restored to current`;
    case "deleted":
      return "Prompt deleted";
  }
}

/** Get the SVG icon path for an event type. */
function eventIconPath(type: EventType): string {
  switch (type) {
    case "created":
      // Plus circle
      return "M12 9v6m3-3H9m12 0a9 9 0 11-18 0 9 9 0 0118 0z";
    case "edited":
      // Pencil
      return "M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.586-3.586a2 2 0 112.828 2.828l-3 3a2 2 0 01-2.828 0 1 1 0 00-1.414 1.414 4 4 0 005.656 0l3-3a4 4 0 00-5.656-5.656z";
    case "restored":
      // Arrow counterclockwise (restore)
      return "M3 10h10a5 5 0 015 5v0a5 5 0 01-5 5H3m0-10l4 4m-4-4l4-4";
    case "deleted":
      // Trash
      return "M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16";
  }
}

export const ActivityEvent = component$<ActivityEventProps>(
  ({ event, testId }) => {
    const classes = eventClasses(event.type);
    const label = eventLabel(event.type, event.revisionNumber);

    return (
      <li
        class={`flex items-start gap-2 py-1.5 ${classes.container}`}
        data-testid={testId ?? "activity-event"}
      >
        <svg
          class="mt-0.5 h-4 w-4 flex-shrink-0"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
          stroke-width={2}
          aria-hidden="true"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            d={eventIconPath(event.type)}
          />
        </svg>
        <span class={`flex-1 text-sm ${classes.text}`}>{label}</span>
        <span class="text-xs text-slate-400">
          {formatTime(event.timestamp)}
        </span>
      </li>
    );
  },
);
