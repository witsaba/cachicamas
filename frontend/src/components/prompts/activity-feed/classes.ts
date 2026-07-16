/**
 * activity-feed classes — pure function table for activity event styling.
 *
 * Aphantasic-friendly (UX-4): text-first. Each event type has both an icon
 * (colored SVG) and text, so the message is conveyed even without color.
 *
 * Event types and their colors:
 *   created  — emerald (new, positive)
 *   edited   — blue (change)
 *   restored — amber (revert, attention)
 *   deleted  — red (destructive)
 */

export type EventType = "created" | "edited" | "restored" | "deleted";

export interface EventClasses {
  container: string;
  iconColor: string;
  text: string;
}

export function eventClasses(type: EventType): EventClasses {
  const baseContainer = "flex items-start gap-2 text-sm";
  const baseText = "text-slate-700";

  switch (type) {
    case "created":
      return {
        container: `${baseContainer} text-emerald-700`,
        iconColor: "text-emerald-500",
        text: baseText,
      };
    case "edited":
      return {
        container: `${baseContainer} text-blue-700`,
        iconColor: "text-blue-500",
        text: baseText,
      };
    case "restored":
      return {
        container: `${baseContainer} text-amber-700`,
        iconColor: "text-amber-500",
        text: baseText,
      };
    case "deleted":
      return {
        container: `${baseContainer} text-red-700`,
        iconColor: "text-red-500",
        text: baseText,
      };
  }
}
