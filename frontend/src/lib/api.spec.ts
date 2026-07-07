/**
 * api.spec.ts — TDD coverage for the workspaces API client methods.
 *
 * Reference: openspec/changes/2026-07-06-workspaces/specs
 *   R-WS-002 (S-WS-010..013) — listWorkspaces contract.
 *   R-WS-003 (S-WS-020..022) — getWorkspace contract.
 *   R-WS-008 (S-WS-070..071) — listLinkedRepos contract.
 *
 * Strict TDD posture: tests cover the locked wire shapes only. Server
 * implementation lives in backend/database_administrator.
 */
import { describe, expect, it, beforeEach, vi, afterEach } from "vitest";
import {
  listWorkspaces,
  getWorkspace,
  listLinkedRepos,
  addRepoToWorkspace,
  removeRepoFromWorkspace,
  deleteWorkspace,
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
                primary_repository: {
                  github_id: 100,
                  full_name: "octocat/hello-world",
                  owner: "octocat",
                  name: "hello-world",
                },
                linked_repos_count: 2,
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
      expect(result.value.workspaces[0]!.primary_repository.full_name).toBe(
        "octocat/hello-world",
      );
      expect(result.value.workspaces[0]!.linked_repos_count).toBe(2);
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
      globalThis.fetch = vi
        .fn()
        .mockResolvedValue(
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
            primary_repository: {
              github_id: 200,
              full_name: "octocat/widgets",
              owner: "octocat",
              name: "widgets",
            },
            linked_repositories: [
              {
                id: 11,
                github_id: 201,
                github_full_name: "octocat/gizmos",
                github_owner: "octocat",
                github_name: "gizmos",
                added_at: "2026-07-02T00:00:00Z",
              },
            ],
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
      expect(result.value.linked_repositories).toHaveLength(1);
    });

    it("TRIANGULATE-T-WS-2i-005(a): 404 not_found → kind=server (or not_found)", async () => {
      globalThis.fetch = vi
        .fn()
        .mockResolvedValue(
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
  // -------------------------------------------------------------------

  describe("listLinkedRepos", () => {
    it("RED-T-WS-2i-006: returns {repositories: [...]} on 200", async () => {
      globalThis.fetch = vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            repositories: [
              {
                id: 11,
                github_id: 201,
                github_full_name: "octocat/gizmos",
                github_owner: "octocat",
                github_name: "gizmos",
                added_at: "2026-07-02T00:00:00Z",
              },
            ],
          }),
          { status: 200, headers: { "Content-Type": "application/json" } },
        ),
      );
      const result = await listLinkedRepos(7);
      expect(result.ok).toBe(true);
      if (!result.ok) return;
      expect(result.value.repositories).toHaveLength(1);
      expect(result.value.repositories[0]!.github_full_name).toBe(
        "octocat/gizmos",
      );
    });

        it("TRIANGULATE-T-WS-2i-007: empty repositories array is ok", async () => {
          globalThis.fetch = vi.fn().mockResolvedValue(
            new Response(JSON.stringify({ repositories: [] }), {
              status: 200,
              headers: { "Content-Type": "application/json" },
            }),
          );
          const result = await listLinkedRepos(7);
          expect(result.ok).toBe(true);
          if (!result.ok) return;
          expect(result.value.repositories).toEqual([]);
        });
      });

      // -------------------------------------------------------------------
      // addRepoToWorkspace (R-WS-006, T-WS-2ii-001..003)
      // -------------------------------------------------------------------

      describe("addRepoToWorkspace", () => {
        const repoPayload = {
          github_id: 555,
          github_full_name: "octocat/widgets",
          github_owner: "octocat",
          github_name: "widgets",
        };

        it("RED-T-WS-2ii-001: returns the linked repo on 201", async () => {
          globalThis.fetch = vi
            .fn()
            .mockResolvedValue(
              new Response(
                JSON.stringify({
                  id: 99,
                  github_id: 555,
                  github_full_name: "octocat/widgets",
                  github_owner: "octocat",
                  github_name: "widgets",
                  added_at: "2026-07-03T00:00:00Z",
                }),
                { status: 201, headers: { "Content-Type": "application/json" } },
              ),
            );

          const result = await addRepoToWorkspace(7, repoPayload);

          expect(result.ok).toBe(true);
          if (!result.ok) return;
          expect(result.value.id).toBe(99);
          expect(result.value.github_id).toBe(555);
          expect(result.value.github_full_name).toBe("octocat/widgets");

          // Verify the request shape
          const fetchMock = globalThis.fetch as ReturnType<typeof vi.fn>;
          expect(fetchMock).toHaveBeenCalledTimes(1);
          const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
          expect(url).toContain("/workspaces/7/repositories");
          expect(init.method).toBe("POST");
          const body = JSON.parse(init.body as string) as Record<string, unknown>;
          expect(body).toEqual(repoPayload);
        });

        it("TRIANGULATE-T-WS-2ii-003(a): 409 conflict → kind=conflict", async () => {
          globalThis.fetch = vi
            .fn()
            .mockResolvedValue(
              new Response(
                JSON.stringify({
                  error: "conflict",
                  message: "This repository is already linked.",
                }),
                { status: 409 },
              ),
            );
          const result = await addRepoToWorkspace(7, repoPayload);
          expect(result.ok).toBe(false);
          if (result.ok) return;
          expect(result.kind).toBe("conflict");
          expect(result.message).toContain("already linked");
        });

        it("TRIANGULATE-T-WS-2ii-003(b): 422 with fields.primary_repository → kind=validation", async () => {
          globalThis.fetch = vi
            .fn()
            .mockResolvedValue(
              new Response(
                JSON.stringify({
                  error: "validation",
                  fields: {
                    primary_repository:
                      "Selected repository is not accessible.",
                  },
                }),
                { status: 422 },
              ),
            );
          const result = await addRepoToWorkspace(7, repoPayload);
          expect(result.ok).toBe(false);
          if (result.ok) return;
          expect(result.kind).toBe("validation");
          if (result.kind === "validation") {
            expect(result.fields.primary_repository).toContain("not accessible");
          }
        });
      });

      // -------------------------------------------------------------------
      // removeRepoFromWorkspace (R-WS-007, T-WS-2ii-004..005)
      // -------------------------------------------------------------------

      describe("removeRepoFromWorkspace", () => {
        it("RED-T-WS-2ii-004: 204 returns ok with null", async () => {
          globalThis.fetch = vi
            .fn()
            .mockResolvedValue(new Response(null, { status: 204 }));

          const result = await removeRepoFromWorkspace(7, 99);

          expect(result.ok).toBe(true);
          if (!result.ok) return;
          expect(result.value).toBeNull();

          const fetchMock = globalThis.fetch as ReturnType<typeof vi.fn>;
          const [url, init] = fetchMock.mock.calls[0] as [string, RequestInit];
          expect(url).toContain("/workspaces/7/repositories/99");
          expect(init.method).toBe("DELETE");
        });

        it("TRIANGULATE-T-WS-2ii-005: 404 → kind=server", async () => {
          globalThis.fetch = vi
            .fn()
            .mockResolvedValue(
              new Response(
                JSON.stringify({ error: "not_found", message: "Workspace not found." }),
                { status: 404 },
              ),
            );
          const result = await removeRepoFromWorkspace(7, 99);
          expect(result.ok).toBe(false);
          if (result.ok) return;
          expect(["server", "not_found"]).toContain(result.kind);
        });
      });

      // -------------------------------------------------------------------
      // deleteWorkspace (R-WS-005, T-WS-2ii-006..007)
      // -------------------------------------------------------------------

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
          globalThis.fetch = vi
            .fn()
            .mockResolvedValue(
              new Response(
                JSON.stringify({ error: "not_found", message: "Workspace not found." }),
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
