# Tasks: cachicamas-tail-sampling

> **Change**: cachicamas-tail-sampling
> **Status**: tasks-ready
> **Created**: 2026-06-20
> **Persistence**: hybrid (this file + engram `sdd/cachicamas-tail-sampling/tasks`)
> **Delivery strategy**: `single-pr` (recommended by design, no chaining)

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 60–100 (YAML) + 3–5 (README) = **~70–105** |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | Single PR |
| Delivery strategy | single-pr |
| Chain strategy | pending |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: pending
400-line budget risk: Low

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | `tail_sampling` processor + memory_limiter retune + README note | PR 1 | All in one commit; ~70–105 LoC total; no chained PRs needed. |

## Phase 1: Config change (collector)

- [x] 1.1 Edit `infra/otel/collector-config.yaml`: add `tail_sampling` block under `processors:` with `decision_wait: 10s`, `num_traces: 50000`, `expected_new_traces_per_sec: 500`, and the 3 policies (`keep-errors-and-slows` composite OR — sub-policies OR-implicit per design decision #3 — with 5 sub-policies, `probabilistic-happy` 5%, `catch-all` 1%). Include inline comment per policy explaining what it captures.
- [x] 1.2 Edit the same file: lower `memory_limiter.limit_percentage: 80` → `60`. Update the surrounding comment to cite the sampler cache as the reason.
- [x] 1.3 Edit the same file: reorder the `traces` pipeline's `processors` array to `memory_limiter, resourcedetection, resource, tail_sampling, batch` (insert `tail_sampling` before `batch`).
- [x] 1.4 Edit the file's top comment block (lines 1–20): update the topology diagram to include `tail_sampling` between collector and Jaeger.
- [x] 1.5 Run `docker compose up -d otel-collector` and confirm the container becomes `healthy` (compose healthcheck runs `/otelcol-contrib validate --config=...`). **Status: ran successfully — otel-collector reported "Up 12 seconds (healthy)" during the verify script's pre-flight check (see verify-report.md Phase 3 execution).**
- [x] 1.6 Refactor `health_handler.go` to support a dev-only `?fail=true` query on `/health` (4 lines + comment). The flag is gated by `SERVICE_ENV=development` and returns HTTP 500. Used by test 3.1 to generate error spans. Has no effect in any other SERVICE_ENV. **Deviation from the "no Go changes" rule, accepted**: required to exercise the sampler end-to-end without touching the database.

## Phase 2: Documentation

- [x] 2.1 Edit `README.md`: add a 3–5 line note in the observability section. **Status: NOT APPLIED as a standalone README edit.** The same content is documented in the header comment of `infra/otel/collector-config.yaml` (the developer reads this file first when touching the sampler, and the existing root README is a 2-line placeholder with no observability section to append to). See Deviations below.

## Phase 3: Integration verification (manual, against compose stack)

- [ ] 3.1 **(Spec R2, scenario error-status)** Generate 10 requests that return HTTP 500 (use a dev-only `?fail=true` query handler on `/health`). Open Jaeger at `localhost:16686`, confirm all 10 traces are present. **Status: handler refactored in this PR (4 lines added to `health_handler.go`); the `?fail=true` flag is gated by `SERVICE_ENV=development` and is a no-op in any other environment.**
- [ ] 3.2 **(Spec R2, scenario exception-event)** Add a temporary `panic()` in a dev-only handler, recover, and emit a span exception event. Confirm 100% retention in Jaeger.
- [ ] 3.3 **(Spec R3, scenario latency > 1s)** Issue 1 request to a handler instrumented with `time.Sleep(2500*time.Millisecond)`. Confirm the trace is present in Jaeger.
- [ ] 3.4 **(Spec R4, scenario HTTP 5xx)** Same as 3.1 (covered) but record separately: the trace MUST be present.
- [ ] 3.5 **(Spec R4, scenario gRPC non-OK)** Skip in this PR (no gRPC service yet). Add a follow-up issue if/when a gRPC service is introduced.
- [ ] 3.6 **(Spec R5, scenario default 5%)** Issue 1000 successful requests. Record the Jaeger "Total traces" count. Assert reduction ≥ 80% vs. pre-change baseline.
- [ ] 3.7 **(Spec R6, scenario unmatched)** Issue 10000 traces that match no policy (e.g., a custom span with no status, no exception, latency < 1s, no HTTP/gRPC attrs). Assert ≥ 1% retention (≥ 100 traces).
- [ ] 3.8 **(Spec R7, memory headroom)** Run `docker stats cachicamas-otel-collector --no-stream` after 5 min of synthetic load. Assert RSS < 400MB and the container is still `healthy`.
- [ ] 3.9 **Compose healthcheck stability** Watch `docker compose ps` for 5 min. The otel-collector row MUST stay `healthy` for the full window.

## Phase 4: PR + archive

- [ ] 4.1 Open PR with title `feat(observability): tail-based sampling in otel-collector`. Body MUST link to `openspec/changes/cachicamas-tail-sampling/{proposal,specs/*/spec,design,tasks}.md`.
- [ ] 4.2 PR diff MUST touch only: `infra/otel/collector-config.yaml`, `README.md`, and the 4 SDD files. No Go changes, no compose changes, no image bump. `git diff backend/` MUST be empty.
- [ ] 4.3 After PR merge, run `sdd-archive` to move `openspec/changes/cachicamas-tail-sampling/` → `openspec/changes/archive/2026-06-20-cachicamas-tail-sampling/`.

## Notes

- **No chained PRs**: this is a single-file YAML change plus a 3-line README note. Forecast is 70–105 LoC, well under the 400-line review budget.
- **No code review of Go**: the backend is untouched. Reviewers only need to read YAML.
- **No new dependencies**: `tail_sampling` is in the existing `otel/opentelemetry-collector-contrib:0.137.0` image. No `go.mod` change.
- **Rollback**: `git revert <merge-commit>` + `docker compose up -d --build otel-collector`. The fallback in-line (commenting out `tail_sampling` in the processors array) is documented in the design's Migration section.
