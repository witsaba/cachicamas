/**
 * api.sse.spec.ts — TDD coverage for the SSE parser.
 *
 * UAT bug discovered 2026-07-08: the polling-based card
 * (getWorkspaceSyncStatus every 3s) failed to update the UI on
 * click. The fix is push-based: the server emits a SSE event on
 * every sync_job state change, the client subscribes once and
 * updates job.value on every event.
 */
import { describe, expect, it } from "vitest";
import { parseSSEResponse } from "./api";

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
