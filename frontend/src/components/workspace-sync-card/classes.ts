// classes.ts — pure function table for the WorkspaceSyncCard.
// Kept separate from the .tsx (Qwik component) so the logic is
// testable without the expensive createDOM machinery. Mirrors
// the pattern in components/ui/button/classes.ts (PR-46).
//
// Status vocabulary (locked, matches domain.SyncJobStatus in
// backend/database_administrator/src/domain/sync.go):
//   - "pending"  — job enqueued, not yet started
//   - "running"  — workspace_syncer is cloning + probing
//   - "done"     — clone succeeded, workspace.last_synced_* updated
//   - "failed"   — clone failed; error_message + error_code populated
//
// Aphantasic-friendly (UX-4): text-first. The status pill uses
// a short uppercase label and a color (per Tailwind 4 palette).
// The button state is encoded in the disabled attribute; the
// label changes to "Syncing…" during running.

import type { SyncJob } from "~/lib/api";

export type SyncStatus = SyncJob["status"];

export interface StatusPillClasses {
  container: string;
  label: string;
}

export interface ButtonClasses {
  root: string;
  label: string;
}

/** Status pill classes. The dot is purely decorative; the label
 *  is the actual text the user reads. Aphantasic-friendly: the
 *  text alone is sufficient to convey state. */
export function statusPillClasses(status: SyncStatus): StatusPillClasses {
  const base =
    "inline-flex items-center gap-2 rounded-full px-3 py-1 text-xs font-medium";
  switch (status) {
    case "pending":
      return {
        container: `${base} bg-slate-100 text-slate-700`,
        label: "Pending",
      };
    case "running":
      return {
        container: `${base} bg-blue-100 text-blue-800`,
        label: "Syncing",
      };
    case "done":
      return {
        container: `${base} bg-emerald-100 text-emerald-800`,
        label: "Synced",
      };
    case "failed":
      return {
        container: `${base} bg-red-100 text-red-800`,
        label: "Failed",
      };
  }
}

/** Sync-button classes + label. The "Syncing…" label is locked;
 *  the disabled state mirrors the project design system rules. */
export function syncButtonClasses(
  job: SyncJob | null,
  isStarting: boolean,
): ButtonClasses {
  const disabled =
    isStarting ||
    (job !== null && (job.status === "pending" || job.status === "running"));
  if (disabled) {
    return {
      root: "inline-flex items-center gap-2 rounded bg-slate-200 px-4 py-2 text-sm font-medium text-slate-500 cursor-not-allowed",
      label: job?.status === "running" ? "Syncing\u2026" : "Pending\u2026",
    };
  }
  if (job?.status === "failed") {
    return {
      root: "inline-flex items-center gap-2 rounded bg-slate-900 px-4 py-2 text-sm font-medium text-white hover:bg-slate-800 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-slate-900",
      label: "Retry sync",
    };
  }
  return {
    root: "inline-flex items-center gap-2 rounded bg-slate-900 px-4 py-2 text-sm font-medium text-white hover:bg-slate-800 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-slate-900",
    label: "Sync now",
  };
}

/** Format a commit SHA for display. Full SHA is 40 chars; the
 *  card shows the first 7 (GitHub's UI convention). Returns
 *  "—" for null. */
export function formatCommitSha(sha: string | null): string {
  if (!sha) return "\u2014";
  return sha.length > 7 ? sha.slice(0, 7) : sha;
}

/** Format an ISO-8601 timestamp for display. Returns the local
 *  date-time in a compact form (YYYY-MM-DD HH:MM). Returns "—"
 *  for null. The frontend does not use a date library; a
 *  custom parser keeps the bundle small. */
export function formatTimestamp(iso: string | null): string {
  if (!iso) return "\u2014";
  const d = new Date(iso);
  if (isNaN(d.getTime())) return "\u2014";
  const pad = (n: number) => n.toString().padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`;
}
