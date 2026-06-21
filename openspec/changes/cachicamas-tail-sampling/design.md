# Design: cachicamas-tail-sampling

> **Change**: cachicamas-tail-sampling
> **Status**: designed
> **Created**: 2026-06-20
> **Persistence**: hybrid (this file + engram `sdd/cachicamas-tail-sampling/design`)

## Technical Approach

Single-file config change to `infra/otel/collector-config.yaml`. Insert the
`tail_sampling` processor into the `traces` pipeline between
`resourcedetection` and `batch`, and lower `memory_limiter.limit_percentage`
from 80 to 60. No Go code changes, no compose changes, no image bump. The
OTel Collector Contrib image at 0.137.0 ships the `tail_sampling` processor
out of the box. Maps to the 7 requirements in
`openspec/changes/cachicamas-tail-sampling/specs/trace-tail-sampling/spec.md`.

## Architecture Decisions

### Decision: Position of `tail_sampling` in the processor chain

**Choice**: `memory_limiter → resourcedetection → resource → tail_sampling → batch`
**Alternatives considered**:
- `tail_sampling` first (before `memory_limiter`) — rejected: an in-flight trace burst would OOM before the limiter could react.
- `tail_sampling` after `batch` (terminal processor) — rejected: the OTel docs explicitly place samplers *before* batchers; sampling after batch means the batcher already paid the per-span CPU/memory cost for traces that get discarded.
**Rationale**: The canonical order is `tail_sampling` immediately before `batch`. `memory_limiter` must stay first to protect the whole pipeline including the sampler cache.

### Decision: `decision_wait` = 10s, explicit

**Choice**: `decision_wait: 10s` in the sampler config, matching the OTel default but written explicitly with a comment.
**Alternatives**: 5s (tighter SLO but risks dropping in-flight spans for slow services); 30s (more headroom for long traces but inflates Jaeger search latency for a 100%-retention error trace by 30s).
**Rationale**: 10s is the documented default. Explicit (not implicit) so a future reader sees the value without consulting external docs.

### Decision: Composite policy structure (OR, no strategy key)

**Choice**: Single composite policy, OR-implicit (the contrib schema has no `composite_strategy` key — composite sub-policies are always OR-ed). The sampler matches if the composite OR either probabilistic matches. Net: `keep = (errors OR exceptions OR slow OR http5xx OR grpc_fail) OR (probabilistic_happy) OR (catch_all)`.
**Alternatives considered**:
- Flat policy list (5 separate top-level policies, each one independently OR-ed by the sampler) — equivalent semantics but bloats the policy list and loses the "interesting traces" grouping.
- `type: and` with `and_sub_policy:` — would mean "keep only if ALL 5 conditions match simultaneously". This is **wrong** for the stated intent: a slow-but-successful request with no error/exception/5xx/gRPC code is exactly the kind of trace we want to keep, not discard. AND-of-everything would discard the most common "interesting" case (a single slow query with no error code).
**Rationale**: The original design (Decision #3, draft) used the term "AND-of-ORs" which was both contradictory and a misrepresentation of the schema. The actual intent — keep any "interesting" trace — is OR over the 5 sub-policies, and that's what composite gives us by default. Fixed during apply after a runtime unmarshal error caught the mistake.

### Decision: `num_traces` = 50000, `expected_new_traces_per_sec` = 500

**Choice**: `num_traces: 50000`, `expected_new_traces_per_sec: 500`.
**Alternatives**: 10000 (too small for 1k req/min dev traffic bursts — would force the sampler to evict before the trace finishes); 100000 (2x memory cost for negligible benefit at dev volumes).
**Rationale**: 50000 traces at ~2KB/trace decision-cache entry ≈ 100MB resident. Combined with the rest of the collector at 80–100MB, the container lands at 200–300MB RSS, well below the 512MB default Docker limit. The 500/sec expected rate covers burst traffic up to ~1500 req/sec (the 3x headroom is conservative).

### Decision: Boundary semantics for "slow" (1000 ms)

**Choice**: `latency > 1000 ms` (strict greater-than). The spec's "exactly at threshold" scenario is documented as a no-op (the trace is evaluated by `probabilistic_happy` instead, not discarded).
**Alternatives**: `>= 1000 ms` — keeps one extra trace per ~million with no operational value.
**Rationale**: OTel latency attributes are typically reported at millisecond resolution; the inclusive boundary is dominated by clock noise. The exclusive form is the de-facto industry default (Honeycomb, Datadog, Grafana Cloud all use `>` for similar thresholds).

### Decision: `memory_limiter.limit_percentage` 80 → 60

**Choice**: Lower from 80 to 60, keeping `spike_limit_percentage: 20`.
**Alternatives**: 70 (compromise); 50 (more headroom but increases OOM risk on legitimate spikes).
**Rationale**: Sampler cache adds ~100MB working set on top of the existing pipeline. Cutting 20 percentage points keeps the collector's effective ceiling at the same absolute number while giving the sampler room to grow during a 5-minute synthetic-load test.

## Data Flow

```
[Go service: database_administrator]
        │  OTLP/gRPC (no change)
        ▼
[otel-collector]
        │
        ▼  pipeline "traces"
   ┌──────────────────────────────────────────────────┐
   │ 1. otlp receiver (4317)                          │
   │ 2. memory_limiter (60% / 20% spike)              │
   │ 3. resourcedetection (env detector only)         │
   │ 4. resource (upsert deployment.environment,      │
   │              service.namespace)                  │
   │ 5. ★ tail_sampling (NEW)                         │
   │      - num_traces: 50000                         │
   │      - decision_wait: 10s                        │
   │      - policies:                                 │
   │          keep-errors-and-slows  (composite OR, composite_implicit_OR)  │
   │          probabilistic-happy   (5%)              │
   │          catch-all             (1%)              │
   │ 6. batch (timeout 1s, size 1024)                 │
   └──────────────────────────────────────────────────┘
        │                │                │
        ▼                ▼                ▼
   otlp/jaeger         debug           (no other exporter)
   (traces stored)  (stdout)
```

The `metrics` and `logs` pipelines are unchanged.

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `infra/otel/collector-config.yaml` | Modify | Add `tail_sampling` block under `processors:`; reorder the `traces` pipeline's `processors` array; lower `memory_limiter.limit_percentage: 80` → `60`. Update the file's top comment to mention tail sampling in the topology diagram. |
| `README.md` | Modify | 3-5 line note in the observability section: "Jaeger retains ~5% of successful traffic and 100% of errors and slow requests." |
| `docker-compose.yaml` | No change | Image `otel/opentelemetry-collector-contrib:0.137.0` already includes `tail_sampling`. |
| `backend/**` | No change | The Go SDK is not touched. |
| `openspec/changes/cachicamas-tail-sampling/{proposal,specs/*}.md` | Already exists | Carried over from previous phases. |

## Interfaces / Contracts

The actual config is in `infra/otel/collector-config.yaml` — it uses the contrib 0.137.0 schema and does not include a `composite_strategy` key (composite sub-policies are OR-implicit; see Decision #3 for the rationale). This design doc does not attempt to replicate the full config inline.

**Boundary check** (Requirement 3, scenario 2): the spec demands a documented
inclusive/exclusive choice. The implementation uses `>` (strict greater-than):
spans with `duration = 1000ms` exactly do **not** match the latency policy and
fall through to the probabilistic samplers. Comment in the config will call
this out so reviewers see it without consulting this design doc.

## Testing Strategy

| Layer | What to Test | Approach |
|-------|-------------|----------|
| Static | Config validity | `docker compose up -d otel-collector` (compose healthcheck runs `/otelcol-contrib validate --config=...`). Must exit 0 and container must be healthy. |
| Integration | Mandatory retention of errors | Script: 10 requests, force the backend to return HTTP 500 (e.g., a `?fail=true` query string handled in dev only). Inspect Jaeger UI at `localhost:16686`. Assert all 10 traces are present. |
| Integration | Mandatory retention of slow traces | Script: 1 request to an endpoint instrumented with `time.Sleep(2500*time.Millisecond)`. Inspect Jaeger. Assert the trace is present. |
| Integration | Volume reduction on happy path | Script: 1000 successful requests. Compare `total traces in Jaeger` before vs. after. Reduction MUST be ≥ 80% (expected 95% ± 4.5% at 5% sampling). |
| Integration | Catch-all floor | Script: 10 traces with neither errors nor slow nor HTTP/gRPC attributes (use a span type that emits no status attributes). Assert ≥ 1 trace is in Jaeger. |
| Resource | Memory headroom | `docker stats cachicamas-otel-collector --no-stream` after the 5-minute synthetic load. Assert RSS < 400MB. |
| Negative | Healthcheck stability | Compose healthcheck (running every 15s) MUST stay green throughout the test window. |

## Migration / Rollout

No data migration, no schema change in Jaeger, no feature flag. The change
ships as a single PR (one config file + a README line). Rollback is `git
revert` of the PR, followed by `docker compose up -d --build otel-collector`.

If the change were to fail in production (e.g., the sampler OOMs), the
operational escape hatch is to comment out `tail_sampling` in the `processors`
array of the `traces` pipeline and restart the container — that drops the
sampler and returns to head-based 100% export within ~30 seconds. No code
change needed; just a config edit.

## Open Questions

- **Resolved (boundary semantics)**: the spec flagged `> 1000` vs `>= 1000`
  for the slow-trace threshold. Decision: strict `>` (see Architecture
  Decision #6). Documented in config comments.
- **Resolved (probabilistic rate re-config)**: spec Requirement 5 Scenario 2
  claims the rate can change without restarting the app. Confirmed: the
  collector's `file` config provider can reload on SIGHUP, but that is NOT
  wired in this PR. The scenario is true for the application but requires a
  collector restart for THIS change. We will document this caveat in the
  `README.md` note.
- **Open (deferred to follow-up PR)**: exposing the collector's own
  `otelcol_*` metrics on a `prometheus` exporter endpoint so a dashboard
  can show sampler hit rates per policy. Not in scope here; tracked as
  "self-metrics of the collector" in the proposal's Out-of-Scope section.

## Notes for the next phase

`sdd-tasks` should produce 6-8 tasks (1 per file change + 1 per integration
test scenario). Forecast against the 400-line PR review budget: **Low** (diff
expected < 100 lines, all in one YAML file plus a 3-5 line README edit). No
need to chain PRs.
