/**
 * api.sse.spec.ts — TDD coverage for the SSE parser.
 *
 * UAT bug discovered 2026-07-08: the polling-based card
 * (getWorkspaceSyncStatus every 3s) failed to update the UI on
 * click. The fix is push-based: the server emits a SSE event on
 * every sync_job state change, the client subscribes once and
 * updates job.value on every event.
 */
import { describe, expect, it, beforeEach, afterEach } from "vitest";
import { parseSSEResponse, subscribeWorkspaceSyncStream } from "./api";

describe("parseSSEResponse", () => {
  it("RED-R-WS-019-2026-07-08-005: parses a data frame with a SyncJob JSON", () => {
    const frame =
      'data: {"job_id":42,"workspace_id":7,"status":"done","commit_sha_after":"ec8fbc8a"}\n\n';
    const out = parseSSEResponse(frame);
    expect(out).toEqual([
      {
        data: {
          job_id: 42,
          workspace_id: 7,
          status: "done",
          commit_sha_after: "ec8fbc8a",
        },
      },
    ]);
  });

  it("RED-R-WS-019-2026-07-08-006: parses a comment (keepalive) frame as {data: null}", () => {
    const frame = ": keepalive\n\n";
    const out = parseSSEResponse(frame);
    expect(out).toEqual([{ data: null }]);
  });

  it("RED-R-WS-019-2026-07-08-007: parses a malformed JSON data frame as {data: null}", () => {
    const frame = "data: not json\n\n";
    const out = parseSSEResponse(frame);
    expect(out).toEqual([{ data: null }]);
  });

  it("RED-R-WS-019-2026-07-08-008: parses multiple concatenated frames in one chunk", () => {
    const frames =
      'data: {"status":"pending"}\n\n' +
      ": keepalive\n\n" +
      'data: {"status":"done"}\n\n';
    const out = parseSSEResponse(frames);
    expect(out).toEqual([
      { data: { status: "pending" } },
      { data: null },
      { data: { status: "done" } },
    ]);
  });

it("RED-R-WS-019-2026-07-08-009: a partial frame (no trailing \\n\\n) is parsed but yields {data: null}", () => {
        // The SSE parser is stateless: it splits on \n\n and emits
        // any complete frames. A trailing partial frame (no \n\n yet)
        // is treated as a complete frame, JSON.parse fails (truncated
        // JSON), and we surface as { data: null } so the caller's
        // event loop keeps running. In practice the caller always
        // passes full chunks (the EventSource mock feeds complete
        // events), so this edge case is mostly defensive.
        const partial = 'data: {"status":"pendi';
        const out = parseSSEResponse(partial);
        expect(out).toEqual([{ data: null }]);
      });
    });

    // RED-R-WS-019-2026-07-08-010: the SSE subscriber MUST close
    // the EventSource when it receives the null marker (job_id=0).
    // The backend closes the stream after sending the marker (no
    // more events expected). If the client doesn't close its own
    // EventSource, the browser auto-reconnects on EOF — the user
    // observed an endless SSE loop in the 4th UAT pass.
    describe("subscribeWorkspaceSyncStream auto-close on null marker", () => {
      let originalEventSource: typeof EventSource | undefined;
      let lastCreated: FakeES | null = null;

      class FakeES {
        url: string;
        withCredentials = false;
        onmessage: ((ev: MessageEvent<string>) => void) | null = null;
        onerror: ((ev: Event) => void) | null = null;
        closed = false;
        constructor(url: string, opts?: { withCredentials?: boolean }) {
          this.url = url;
          this.withCredentials = opts?.withCredentials ?? false;
          lastCreated = this;
        }
        close() {
          this.closed = true;
        }
      }

      beforeEach(() => {
        originalEventSource = (globalThis as any).EventSource;
        (globalThis as any).EventSource = FakeES;
      });

      afterEach(() => {
        if (originalEventSource === undefined) {
          delete (globalThis as any).EventSource;
        } else {
          (globalThis as any).EventSource = originalEventSource;
        }
        lastCreated = null;
      });

      it("closes the EventSource on the null marker (job_id=0)", async () => {
        const updates: Array<unknown> = [];
        const unsubscribe = subscribeWorkspaceSyncStream(
          7,
          (job) => updates.push(job),
          () => {},
        );
        expect(lastCreated).not.toBeNull();
        lastCreated!.onmessage!({
          data: JSON.stringify({
            job_id: 0,
            workspace_id: 7,
            status: "",
            triggered_by: "",
            started_at: null,
            finished_at: null,
            commit_sha_after: null,
            error_message: null,
            error_code: null,
            attempts: 0,
            created_at: "0001-01-01T00:00:00Z",
          }),
        } as MessageEvent<string>);
        expect(lastCreated!.closed).toBe(true);
        expect(updates).toEqual([null]);
        unsubscribe();
      });

      it("does NOT close on a real event (job_id > 0)", async () => {
        const updates: Array<unknown> = [];
        const unsubscribe = subscribeWorkspaceSyncStream(
          7,
          (job) => updates.push(job),
          () => {},
        );
        lastCreated!.onmessage!({
          data: JSON.stringify({
            job_id: 42,
            workspace_id: 7,
            status: "done",
            triggered_by: "manual",
            started_at: null,
            finished_at: null,
            commit_sha_after: null,
            error_message: null,
            error_code: null,
            attempts: 1,
            created_at: "2026-07-08T00:00:00Z",
          }),
        } as MessageEvent<string>);
        expect(lastCreated!.closed).toBe(false);
        expect(updates).toHaveLength(1);
        unsubscribe();
      });
    });
