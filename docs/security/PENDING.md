# Pending security topics

This document tracks security follow-ups identified during the
2026-07-08-workspace-sync-clone change (PR #47) that were
**deliberately not addressed in that PR**. Each row references
the locked artifact (ADR, design, call site) so a future PR
can pick it up without rediscovering the context.

When a row reaches done: strike the title and move it to the
`Resolved` subsection at the bottom. Do **not** delete history
— keeping the row documents what was deferred and why.

## Tracked items

| # | Severity | Title | Why deferred | Where (file:line) | Next action |
| --- | ---------- | ------- | -------------- | -------------------- | ------------- |
| 1 | **Medium** | OAuth token persists in bare mirror `.git/config` | The runner intentionally embeds the user's GitHub access token in the clone URL (`https://x-access-token:gho_…@github.com/…`). `git clone --bare` writes that URL into `remote.origin.url` of the bare mirror's `.git/config`. Safe while the syncer-data volume is internal-only. Becomes a leak if the volume is bind-mounted on the host or copied out (backup, debug share, etc.). | `backend/workspace_syncer/src/infrastructure/git/runner.go:Clone` | After a successful clone, rewrite the remote URL: `git --git-dir=<path> remote set-url origin https://github.com/{owner}/{repo}`. Add a TDD test that pins the post-condition. |
| 2 | **Low** | Internal auth v1 → v2 migration | v1 uses static bearer (`INTERNAL_SERVICE_TOKEN`) + HMAC for the callback + docker network isolation. The ADR reserves v2 = HMAC-signed short-lived JWT exchanged at a `/token` endpoint on `database_administrator`. No external pressure to migrate; current posture passes v1's stated threat model. | `docs/adr/workspace-syncer-internal-auth.md` (forthcoming) | Land the ADR with the v2 shape; implement JWT issuance + validation on both sides; rotate `INTERNAL_SERVICE_TOKEN` out of `.env`. |
| 3 | **Medium** | Service-to-service plaintext HTTP | `database_administrator ↔ workspace_syncer`, `database_administrator ↔ postgres`, and the `frontend → database_administrator` link are all plaintext HTTP inside the docker network. Acceptable for local + CI; not acceptable for any non-localhost deployment (VPS compose file). | `docker-compose.yaml`, `docker-compose.vps.yaml` | For prod: terminate TLS at a reverse proxy (caddy / traefik) or use an overlay network with mTLS (SPIFFE / cert-manager). Document the chosen posture in an ADR. |
| 4 | **Low** | OTLP exporter is a no-op | `otel/otel.go` initializes the tracer as `noop (real OTLP deferred to follow-up)` to avoid dep bloat in v1. Jaeger is running but receives no spans from `workspace_syncer`. Not directly security, but the absence of a production-grade audit trail makes incident review harder. | `backend/workspace_syncer/src/otel/otel.go` | Wire the real OTLP exporter; document the `OTEL_EXPORTER_OTLP_ENDPOINT` env contract; add spans around `clone.execute`, `callback.post`, `sweep.run`. |
| 5 | **Low** | Token redaction coverage in error paths | The slog handler claims `[REDACTED]` for the OAuth token. The `clone: github accessor error` log line concatenates `err.Error()` and could surface the token if a downstream SDK message includes it. Not confirmed a leak; needs a sweep. | `backend/workspace_syncer/src/application/clone_service.go:CloneAndValidate` and the wider syncer codebase | Add a sanitizer wrapper around `err.Error()` for known sensitive substrings (`gho_*`, `xox*-*`, GitHub emails). Add tests that pin the redaction across the warn/error paths. |
| 6 | **Low** | Workspace_syncer orphan-sweep is destructive on every boot | The sweep removes ANY directory under `/data/workspaces/` whose numeric name is not in the `liveIDs` set passed by `main.go`. Today `liveIDs` is empty (the syncer has no Postgres access), so **every boot wipes all clones**. Safe for v1 (the next sync recreates them) but it means an offline clone survives only one syncer restart. Becomes a problem if we ever cache a clone across restarts for latency. | `backend/workspace_syncer/src/infrastructure/git/sweep.go` | v1.1: pass a real `liveIDs` set via a future `GET /internal/live-workspaces` endpoint on `database_administrator`. Or: tag clones with a metadata file and let the sweep skip mismatches. |

## How to use this doc

- **Before opening a security-related PR**: scan this list to fold related items.
- **After a security audit or pentest**: add a new row with severity and source.
- **On resolving a row**: keep the row but strike the title (`~~…~~`) and move it to the `Resolved` subsection. Do not delete history.
- **Severity scale**: Low = theoretical / requires another condition. Medium = realistic exploitation if a stated assumption breaks. High = would block merge if discovered now (none in this list).

## Resolved

_(empty — first resolved row will land here.)_
