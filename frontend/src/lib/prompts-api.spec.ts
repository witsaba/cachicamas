import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Prompt, PromptRevision } from "./prompts-api";

// ---------------------------------------------------------------------------
// Helpers — mock fetch factory
// ---------------------------------------------------------------------------

/**
 * Build a mock Response with the given status and optional JSON body.
 */
function mockResponse(
  body: unknown,
  init: ResponseInit & { status?: number } = {},
): Response {
  const { status = 200, ...rest } = init;
  return new Response(body === undefined ? undefined : JSON.stringify(body), {
    ...rest,
    status,
    headers: {
      "Content-Type": "application/json",
      ...(rest.headers as Record<string, string> | undefined),
    },
  });
}

// ---------------------------------------------------------------------------
// Wire shape tests — verify the API functions produce correct requests
// ---------------------------------------------------------------------------

describe("prompts-api wire shapes", () => {
  // We use a simplified approach: verify the URL + method by capturing
  // the fetch call via globalThis.fetch patching.

  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  const PROMPT: Prompt = {
    id: 1,
    slug: "test-prompt",
    description: "A test prompt",
    body: "# Hello\n\nWorld",
    current_revision: 2,
    created_at: "2026-07-16T10:00:00Z",
    updated_at: "2026-07-16T12:00:00Z",
    deleted_at: null,
  };

  const REVISION: PromptRevision = {
    id: 10,
    prompt_id: 1,
    revision_number: 1,
    body: "# Version 1",
    created_at: "2026-07-16T10:00:00Z",
  };

  // -------------------------------------------------------------------------
  // listPrompts
  // -------------------------------------------------------------------------

  it("listPrompts calls GET /prompts?deleted=false", async () => {
    vi.mocked(fetch).mockResolvedValue(mockResponse([PROMPT], { status: 200 }));
    // We need to re-import to pick up the mocked fetch
    const { listPrompts } = await import("./prompts-api");
    const result = await listPrompts();
    expect(result.ok).toBe(true);
    if (result.ok) {
      expect(result.value).toHaveLength(1);
      expect(result.value[0].slug).toBe("test-prompt");
    }
    expect(vi.mocked(fetch)).toHaveBeenCalled();
    const call = vi.mocked(fetch).mock.calls[0];
    expect(call[0]).toContain("/prompts?deleted=false");
  });

  it("listPrompts returns offline on network error", async () => {
    vi.mocked(fetch).mockRejectedValue(new Error("ECONNREFUSED"));
    const { listPrompts } = await import("./prompts-api");
    const result = await listPrompts();
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.kind).toBe("offline");
    }
  });

  // -------------------------------------------------------------------------
  // createPrompt
  // -------------------------------------------------------------------------

  it("createPrompt calls POST /prompts with JSON body", async () => {
    vi.mocked(fetch).mockResolvedValue(mockResponse(PROMPT, { status: 201 }));
    const { createPrompt } = await import("./prompts-api");
    const result = await createPrompt({
      slug: "test-prompt",
      description: "A test",
      body: "# Hello",
    });
    expect(result.ok).toBe(true);
    const call = vi.mocked(fetch).mock.calls[0];
    expect(call[0]).toContain("/prompts");
    expect(call[1]?.method).toBe("POST");
    // JSON body — backend uses json.NewDecoder
    const parsed = JSON.parse(call[1]?.body as string) as Record<
      string,
      string
    >;
    expect(parsed.slug).toBe("test-prompt");
    expect(parsed.description).toBe("A test");
    expect(parsed.body).toBe("# Hello");
    expect((call[1]?.headers as Record<string, string>)?.["Content-Type"]).toBe(
      "application/json",
    );
  });

  it("createPrompt returns conflict on 409", async () => {
    vi.mocked(fetch).mockResolvedValue(
      mockResponse({ message: "Slug already taken" }, { status: 409 }),
    );
    const { createPrompt } = await import("./prompts-api");
    const result = await createPrompt({
      slug: "taken",
      body: "# x",
    });
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.kind).toBe("conflict");
      expect(result.message).toContain("Slug already taken");
    }
  });

  // -------------------------------------------------------------------------
  // updatePrompt
  // -------------------------------------------------------------------------

  it("updatePrompt calls PATCH /prompts/:slug with JSON body", async () => {
    vi.mocked(fetch).mockResolvedValue(mockResponse(PROMPT, { status: 200 }));
    const { updatePrompt } = await import("./prompts-api");
    const result = await updatePrompt("test-prompt", "# Updated body");
    expect(result.ok).toBe(true);
    const call = vi.mocked(fetch).mock.calls[0];
    expect(call[0]).toContain("/prompts/test-prompt");
    expect(call[1]?.method).toBe("PATCH");
    // JSON body — backend uses json.NewDecoder
    const parsed = JSON.parse(call[1]?.body as string) as Record<
      string,
      string
    >;
    expect(parsed.body).toBe("# Updated body");
  });

  it("updatePrompt returns validation error on 400", async () => {
    vi.mocked(fetch).mockResolvedValue(
      mockResponse(
        { message: "Body is required", fields: { body: "required" } },
        { status: 400 },
      ),
    );
    const { updatePrompt } = await import("./prompts-api");
    const result = await updatePrompt("test-prompt", "");
    expect(result.ok).toBe(false);
    if (!result.ok && result.kind === "validation") {
      expect(result.fields).toHaveProperty("body");
    }
  });

  // -------------------------------------------------------------------------
  // deletePrompt
  // -------------------------------------------------------------------------

  it("deletePrompt calls DELETE /prompts/:slug", async () => {
    vi.mocked(fetch).mockResolvedValue(
      new Response(undefined, { status: 204 }),
    );
    const { deletePrompt } = await import("./prompts-api");
    const result = await deletePrompt("test-prompt");
    expect(result.ok).toBe(true);
    const call = vi.mocked(fetch).mock.calls[0];
    expect(call[0]).toContain("/prompts/test-prompt");
    expect(call[1]?.method).toBe("DELETE");
  });

  // -------------------------------------------------------------------------
  // getPrompt
  // -------------------------------------------------------------------------

  it("getPrompt returns not_found on 404", async () => {
    vi.mocked(fetch).mockResolvedValue(
      mockResponse({ message: "Not found" }, { status: 404 }),
    );
    const { getPrompt } = await import("./prompts-api");
    const result = await getPrompt("nonexistent");
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.kind).toBe("not_found");
    }
  });

  it("getPrompt returns not_found on 410", async () => {
    vi.mocked(fetch).mockResolvedValue(
      new Response(undefined, { status: 410 }),
    );
    const { getPrompt } = await import("./prompts-api");
    const result = await getPrompt("deleted-prompt");
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.kind).toBe("not_found");
      expect(result.message).toBe("Prompt not found.");
    }
  });

  // -------------------------------------------------------------------------
  // listRevisions
  // -------------------------------------------------------------------------

  it("listRevisions calls GET /prompts/:slug/revisions", async () => {
    vi.mocked(fetch).mockResolvedValue(
      mockResponse([REVISION], { status: 200 }),
    );
    const { listRevisions } = await import("./prompts-api");
    const result = await listRevisions("test-prompt");
    expect(result.ok).toBe(true);
    const call = vi.mocked(fetch).mock.calls[0];
    expect(call[0]).toContain("/prompts/test-prompt/revisions");
  });

  // -------------------------------------------------------------------------
  // restoreRevision
  // -------------------------------------------------------------------------

  it("restoreRevision calls POST /prompts/:slug/revisions/:n/restore", async () => {
    vi.mocked(fetch).mockResolvedValue(mockResponse(PROMPT, { status: 200 }));
    const { restoreRevision } = await import("./prompts-api");
    const result = await restoreRevision("test-prompt", 1);
    expect(result.ok).toBe(true);
    const call = vi.mocked(fetch).mock.calls[0];
    expect(call[0]).toContain("/prompts/test-prompt/revisions/1/restore");
    expect(call[1]?.method).toBe("POST");
  });

  // -------------------------------------------------------------------------
  // getRevision
  // -------------------------------------------------------------------------

  it("getRevision calls GET /prompts/:slug/revisions/:n", async () => {
    vi.mocked(fetch).mockResolvedValue(mockResponse(REVISION, { status: 200 }));
    const { getRevision } = await import("./prompts-api");
    const result = await getRevision("test-prompt", 2);
    expect(result.ok).toBe(true);
    const call = vi.mocked(fetch).mock.calls[0];
    expect(call[0]).toContain("/prompts/test-prompt/revisions/2");
  });
});
