/**
 * format-relative-time.ts — small helper for "X minutes ago" / "Yesterday"-style
 * relative time formatting.
 *
 * Lives in chat/ because the only consumer is the conversation rail
 * (chat-app.tsx mount) — no other page needs the same shape. The
 * helper is browser-only (uses Date.now()); SSR returns the absolute
 * time so the wire doesn't ship a "just now" phrase that never ages.
 */
export function formatRelativeTime(iso: string, now: number = Date.now()): string {
  const then = Date.parse(iso);
  if (Number.isNaN(then)) return iso;
  const diffMs = now - then;
  if (diffMs < 0) return "Just now";
  const minutes = Math.floor(diffMs / 60_000);
  if (minutes < 1) return "Just now";
  if (minutes < 60) return `${minutes} minute${minutes === 1 ? "" : "s"} ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours} hour${hours === 1 ? "" : "s"} ago`;
  const days = Math.floor(hours / 24);
  if (days === 1) return "Yesterday";
  if (days < 7) return `${days} days ago`;
  // Beyond a week, fall back to a date. Keeps the rail scannable.
  const d = new Date(then);
  const monthNames = [
    "Jan",
    "Feb",
    "Mar",
    "Apr",
    "May",
    "Jun",
    "Jul",
    "Aug",
    "Sep",
    "Oct",
    "Nov",
    "Dec",
  ];
  const month = monthNames[d.getUTCMonth()];
  const day = d.getUTCDate();
  return `${month} ${day}`;
}
