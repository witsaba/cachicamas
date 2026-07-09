/**
 * api.spec.ts — TDD coverage for the workspaces API client methods.
 *
 * Reference: openspec/changes/2026-07-06-workspaces/specs
 *   R-WS-002 (S-WS-010..013) — listWorkspaces contract.
 *   R-WS-003 (S-WS-020..022) — getWorkspace contract.
 *   R-WS-008 — REMOVED in 2026-07-08-workspaces-simplify (the workspace_repository table is gone).
 *
 * Strict TDD posture: tests cover the locked wire shapes only. Server
 * implementation lives in backend/database_administrator.
 */
import { describe, expect, it, beforeEach, vi, afterEach } from "vitest";import {
  listWorkspaces,
  getWorkspace,
  deleteWorkspace,
  createWorkspace,
  listGitHubRepos,
} from "~/lib/api";
describe("api.ts — workspaces client (PR2-i)", () => {
  const originalFetch = globalThis.fetch;

  beforeEach(() => {
    vi.restoreAllMocks();
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  // -------------------------------------------------------------------
  // listWorkspaces (R-WS-002, T-WS-2i-001..003)
  // -------------------------------------------------------------------

  describe("listWorkspaces", () => {
    it("RED-T-WS-2i-001: returns {ok: true, value: {workspaces: [...], truncated: false}} on a 200 with the expected shape", async () => {
      globalThis.fetch = vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            workspaces: [
              {
                id: 1,
                name: "alpha",
                repository: {
                  github_id: 100,
                  full_name: "octocat/hello-world",
                  owner: "octocat",
                  name: "hello-world",
                },

                created_at: "2026-07-01T00:00:00Z",
              },
            ],
            truncated: false,
          }),
          { status: 200, headers: { "Content-Type": "application/json" } },
        ),
      );

      const result = await listWorkspaces();

      expect(result.ok).toBe(true);
      if (!result.ok) return;
      expect(result.value.workspaces).toHaveLength(1);
      expect(result.value.workspaces[0]!.name).toBe("alpha");
expect(result.value.workspaces[0]!.repository.full_name).toBe(
        "octocat/hello-world",
      );
      expect(result.value.truncated).toBe(false);
    });

    it("TRIANGULATE-T-WS-2i-003(a): 401 with auth error envelope → kind=server", async () => {
      globalThis.fetch = vi
        .fn()
        .mockResolvedValue(
          new Response(
            JSON.stringify({ error: "unauthorized", message: "auth required" }),
            { status: 401 },
          ),
        );
      const result = await listWorkspaces();
      expect(result.ok).toBe(false);
      if (result.ok) return;
      expect(result.kind).toBe("server");
    });

    it("TRIANGULATE-T-WS-2i-003(b): 500 → kind=server with the message", async () => {
      globalThis.fetch = vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ error: "server", message: "boom" }), {
          status: 500,
        }),
      );
      const result = await listWorkspaces();
      expect(result.ok).toBe(false);
      if (result.ok) return;
      expect(result.kind).toBe("server");
      expect(result.message).toContain("boom");
    });

    it("TRIANGULATE-T-WS-2i-003(c): fetch throws (network) → kind=offline", async () => {
      globalThis.fetch = vi.fn().mockRejectedValue(new Error("ECONNREFUSED"));
      const result = await listWorkspaces();
      expect(result.ok).toBe(false);
      if (result.ok) return;
      expect(result.kind).toBe("offline");
      expect(result.message).toContain("ECONNREFUSED");
    });
  });

  // -------------------------------------------------------------------
  // getWorkspace (R-WS-003, T-WS-2i-004..005)
  // -------------------------------------------------------------------

  describe("getWorkspace", () => {
    it("RED-T-WS-2i-004: returns full workspace detail on 200", async () => {
      globalThis.fetch = vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            id: 7,
            name: "beta",
            repository: {
              github_id: 200,
              full_name: "octocat/widgets",
              owner: "octocat",
              name: "widgets",
            },
created_at: "2026-07-01T00:00:00Z",
            updated_at: "2026-07-02T00:00:00Z",
          }),
          { status: 200, headers: { "Content-Type": "application/json" } },
        ),
      );
      const result = await getWorkspace(7);
      expect(result.ok).toBe(true);
      if (!result.ok) return;
      expect(result.value.id).toBe(7);
      expect(result.value.name).toBe("beta");
      // linked_repositories field removed in 2026-07-08-workspaces-simplify.
    });

    it("TRIANGULATE-T-WS-2i-005(a): 404 not_found → kind=server (or not_found)", async () => {
      globalThis.fetch = vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            error: "not_found",
            message: "Workspace not found.",
          }),
          { status: 404 },
        ),
      );
      const result = await getWorkspace(999);
      expect(result.ok).toBe(false);
      if (result.ok) return;
      // We don't preserve not_found in the api result shape (the lock only
      // adds it for the workspace handler envelope). The apiResult
      // collapses 404→server kind per existing organization pattern. The
      // detail page checks 404 server-side via its own fetch + status.
      expect(["server", "not_found"]).toContain(result.kind);
    });
  });

  // -------------------------------------------------------------------
  // listLinkedRepos (R-WS-008, T-WS-2i-006..007)
  describe("deleteWorkspace", () => {
    it("RED-T-WS-2ii-006: 204 returns ok with null", async () => {
      globalThis.fetch = vi
        .fn()
        .mockResolvedValue(new Response(null, { status: 204 }));

      const result = await deleteWorkspace(7);

      expect(result.ok).toBe(true);
      if (!result.ok) return;
      expect(result.value).toBeNull();

      const fetchMock = globalThis.fetch as ReturnType<typeof vi.fn>;
      const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
      expect(url).toContain("/workspaces/7");
      expect(init.method).toBe("DELETE");
    });

    it("TRIANGULATE-T-WS-2ii-007: 404 (already deleted) → kind=server", async () => {
      globalThis.fetch = vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            error: "not_found",
            message: "Workspace not found.",
          }),
          { status: 404 },
        ),
      );
      const result = await deleteWorkspace(7);
      expect(result.ok).toBe(false);
      if (result.ok) return;
      expect(["server", "not_found"]).toContain(result.kind);
    });
  });
});

describe("api.ts — workspaces client (PR2-iii)", () => {
  const originalFetch = globalThis.fetch;

  beforeEach(() => {
    vi.restoreAllMocks();
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  describe("createWorkspace", () => {
    it("RED-T-WS-2iii-001: 201 + body → ok with the new workspace detail", async () => {
const body = {
        id: 99,
        name: "ws-one",
        repository: {
          github_id: 12345,
          full_name: "octocat/hello",
          owner: "octocat",
          name: "hello",
        },
        created_at: "2026-07-06T00:00:00Z",
        updated_at: "2026-07-06T00:00:00Z",
      };
      globalThis.fetch = vi
        .fn()
        .mockResolvedValue(new Response(JSON.stringify(body), { status: 201 }));
      const result = await createWorkspace({
        name: "ws-one",
        repository: {
          github_id: 12345,
          full_name: "octocat/hello",
          owner: "octocat",
          name: "hello",
        },
      });
      expect(result.ok).toBe(true);
      if (!result.ok) return;
      expect(result.value.id).toBe(99);
      expect(result.value.name).toBe("ws-one");
      const fetchMock = globalThis.fetch as ReturnType<typeof vi.fn>;
      const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
      expect(url).toMatch(/\/workspaces$/);
      expect(init.method).toBe("POST");
      expect(JSON.parse(init.body as string)).toEqual({
        name: "ws-one",
        repository: {
          github_id: 12345,
          full_name: "octocat/hello",
          owner: "octocat",
          name: "hello",
        },
      });
    });

    it("TRIANGULATE-T-WS-2iii-003a: 400 with fields.name → kind=validation", async () => {
      globalThis.fetch = vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            error: "validation",
            fields: { name: "Name must be 3–60 characters." },
          }),
          { status: 400 },
        ),
      );
      const result = await createWorkspace({
        name: "ab",
        repository: {
          github_id: 1,
          full_name: "o/r",
          owner: "o",
          name: "r",
        },
      });
      expect(result.ok).toBe(false);
      if (result.ok) return;
      expect(result.kind).toBe("validation");
      if (result.kind === "validation") {
        expect(result.fields.name).toBe("Name must be 3–60 characters.");
      }
    });

    it("TRIANGULATE-T-WS-2iii-003b: 409 duplicate name → kind=conflict with message", async () => {
      globalThis.fetch = vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            error: "conflict",
            message: "A workspace with this name already exists.",
          }),
          { status: 409 },
        ),
      );
      const result = await createWorkspace({
        name: "ws-one",
        repository: {
          github_id: 1,
          full_name: "o/r",
          owner: "o",
          name: "r",
        },
      });
      expect(result.ok).toBe(false);
      if (result.ok) return;
      expect(result.kind).toBe("conflict");
      expect(result.message).toBe("A workspace with this name already exists.");
    });

    it("TRIANGULATE-T-WS-2iii-003c: 422 with fields.repository → kind=validation", async () => {
      globalThis.fetch = vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            error: "validation",
            fields: {
              repository: "Selected repository is not accessible.",
            },
          }),
          { status: 422 },
        ),
      );
      const result = await createWorkspace({
        name: "ws-one",
        repository: {
          github_id: 1,
          full_name: "o/r",
          owner: "o",
          name: "r",
        },
      });
      expect(result.ok).toBe(false);
      if (result.ok) return;
      expect(result.kind).toBe("validation");
      if (result.kind === "validation") {
        expect(result.fields.repository).toBe(
          "Selected repository is not accessible.",
        );
      }
    });

    it("TRIANGULATE-T-WS-2iii-003d: network error → kind=offline", async () => {
      globalThis.fetch = vi.fn().mockRejectedValue(new Error("ECONNREFUSED"));
      const result = await createWorkspace({
        name: "ws-one",
        repository: {
          github_id: 1,
          full_name: "o/r",
          owner: "o",
          name: "r",
        },
      });
      expect(result.ok).toBe(false);
      if (result.ok) return;
      expect(result.kind).toBe("offline");
    });
  });

  describe("listGitHubRepos", () => {
    it("RED-T-WS-2iii-004: 200 with repos + has_next=true → ok", async () => {
      const body = {
        repositories: [
          {
            id: 1,
            full_name: "octocat/hello",
            owner_login: "octocat",
            name: "hello",
            private: false,
            description: "demo",
            html_url: "https://github.com/octocat/hello",
            updated_at: "2026-07-06T00:00:00Z",
            stargazers_count: 5,
          },
        ],
        page: 1,
        per_page: 30,
        has_next: true,
      };
      globalThis.fetch = vi
        .fn()
        .mockResolvedValue(new Response(JSON.stringify(body), { status: 200 }));
      const result = await listGitHubRepos();
      expect(result.ok).toBe(true);
      if (!result.ok) return;
      expect(result.value.repositories).toHaveLength(1);
      expect(result.value.hasNext).toBe(true);
    });

    it("TRIANGULATE-T-WS-2iii-005a: bust_cache=true + page=2 → query string", async () => {
      globalThis.fetch = vi
        .fn()
        .mockResolvedValue(
          new Response(JSON.stringify({ repositories: [] }), { status: 200 }),
        );
      await listGitHubRepos({ bustCache: true, page: 2, perPage: 50 });
      const fetchMock = globalThis.fetch as ReturnType<typeof vi.fn>;
      const [url] = fetchMock.mock.calls[0] as [string];
      expect(url).toContain("bust_cache=true");
      expect(url).toContain("page=2");
      expect(url).toContain("per_page=50");
    });

    it("TRIANGULATE-T-WS-2iii-006a: 401 github_not_connected → kind=server with reconnect message", async () => {
      globalThis.fetch = vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            error: "github_not_connected",
            message: "Reconnect GitHub to list repositories.",
          }),
          { status: 401 },
        ),
      );
      const result = await listGitHubRepos();
      expect(result.ok).toBe(false);
      if (result.ok) return;
      expect(result.kind).toBe("server");
      expect(result.message).toContain("Reconnect GitHub");
    });

    it("TRIANGULATE-T-WS-2iii-006b: 502 github_rate_limited → kind=server", async () => {
      globalThis.fetch = vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            error: "github_rate_limited",
            message: "GitHub rate limit reached.",
          }),
          { status: 502 },
        ),
      );
      const result = await listGitHubRepos();
      expect(result.ok).toBe(false);
      if (result.ok) return;
      expect(result.kind).toBe("server");
      expect(result.message).toContain("rate limit");
    });
  });
});

// -------------------------------------------------------------------
// getWorkspaceSyncStatus — R-WS-019 S-WS-196 (sync card polling)
//
// UAT bug discovered 2026-07-08: the prior implementation used
// plain `fetch` (not `serverAwareFetch`), so the SSR path sent
// an unauthenticated request to db_admin and got 401. The route's
// `useTask$` silently ignored the 401 (`if (syncResult.ok) ...`),
// leaving `initialSyncJob.value = null` and the card stuck on
// "Pending...". The fix: route through `serverAwareFetch` so the
// SSR path forwards the user's session cookie.
// -------------------------------------------------------------------
describe("getWorkspaceSyncStatus", () => {
  it("RED-R-WS-019-2026-07-08-001: SSR fetch forwards the session cookie", async () => {
    // Run in the Node SSR runtime (process.versions is defined).
    // We assert that the dispatched fetch carries the captured
    // cookie header (IdentityFromCookie on the db_admin side
    // needs it to authenticate the request).
    const capturedHeaders: Record<string, string> = {};
    globalThis.fetch = vi.fn().mockImplementation(async (url, init) => {
      const headers = (init?.headers ?? {}) as Record<string, string>;
      Object.assign(capturedHeaders, headers);
      return new Response(
        JSON.stringify({
          job_id: 42,
          workspace_id: 7,
          status: "done",
          commit_sha: "ec8fbc8a",
          default_branch: "main",
        }),
        { status: 200 },
      );
    });

    // Inject a captured session cookie (the route's onRequest
    // writes the inbound Cookie into the module-level
    // ssrCookieContext before the page renders).
    const { setSsrCookieHeader } = await import("~/lib/ssr-cookie-context");
    setSsrCookieHeader("authjs.session-token=PAYLOAD; Path=/; HttpOnly");

    const { getWorkspaceSyncStatus } = await import("~/lib/api");
    const result = await getWorkspaceSyncStatus(7);

    expect(result.ok).toBe(true);
    const cookie = capturedHeaders["cookie"] ?? capturedHeaders["Cookie"];
    if (!cookie || !cookie.includes("authjs.session-token=PAYLOAD")) {
      throw new Error(
        `SSR fetch did NOT forward the session cookie. ` +
        `Headers sent: ${JSON.stringify(capturedHeaders)}`,
      );
    }
  });
});
