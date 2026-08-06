# trace-tail-sampling Specification

> **Domain**: trace-tail-sampling
> **Change**: cachicamas-tail-sampling
> **Type**: New capability (full spec — no existing behavior)
> **Created**: 2026-06-20
> **Persistence**: hybrid (this file + engram `sdd/cachicamas-tail-sampling/spec`)

## Purpose

Defines the behavior of the OpenTelemetry Collector's tail-based sampling decision
pipeline for the cachicamas local development stack. The collector MUST evaluate
each trace after the root span completes (or after a wait timeout) and decide
whether to export the trace to Jaeger, based on errors, latency, status codes,
and probabilistic sampling of successful traffic.

## Requirements

### Requirement: Post-hoc trace evaluation

The otel-collector MUST evaluate each trace's complete set of spans before
deciding whether to export it to the `otlp/jaeger` exporter.

#### Scenario: Trace decision waits for root span completion

- GIVEN a trace is in flight with at least one root span not yet ended
- WHEN the root span ends
- THEN the sampler MUST evaluate the trace within `decision_wait` (default 10s)
- AND the trace MUST be either exported to Jaeger or discarded

#### Scenario: Trace decision after wait timeout

- GIVEN a trace is in flight and the root span has not ended
- WHEN `decision_wait` expires
- THEN the sampler MUST evaluate the trace with the spans received so far
- AND the trace MUST NOT block the pipeline indefinitely

### Requirement: Mandatory retention of error traces

The otel-collector MUST export 100% of traces that contain an error status code
(OTel `StatusCode = Error`) or at least one exception event.

#### Scenario: Span with error status forces export

- GIVEN a trace contains at least one span with `status_code = ERROR`
- WHEN the sampler evaluates the trace
- THEN the trace MUST be exported to `otlp/jaeger`
- AND the trace MUST appear in the Jaeger UI

#### Scenario: Span with exception event forces export

- GIVEN a trace contains at least one span with an `exception` event
- WHEN the sampler evaluates the trace
- THEN the trace MUST be exported to `otlp/jaeger`

### Requirement: Mandatory retention of slow traces

The otel-collector MUST export 100% of traces whose end-to-end duration exceeds
1000 ms.

#### Scenario: Latency above threshold forces export

- GIVEN a trace whose end-to-end duration is 1500 ms
- WHEN the sampler evaluates the trace
- THEN the trace MUST be exported to `otlp/jaeger`

#### Scenario: Latency exactly at threshold

- GIVEN a trace whose end-to-end duration is 1000 ms
- WHEN the sampler evaluates the trace
- THEN the trace MAY be evaluated by the probabilistic samplers (boundary
  semantics are implementation-defined and documented in design.md decision #6)

### Requirement: Mandatory retention of HTTP and gRPC failures

The otel-collector MUST export 100% of traces that contain a span with HTTP
status code in the 5xx class OR a gRPC span with `rpc.grpc.status_code` not
equal to `OK`.

#### Scenario: HTTP 500 forces export

- GIVEN a server span with `http.response.status_code = 500`
- WHEN the sampler evaluates the parent trace
- THEN the trace MUST be exported to `otlp/jaeger`

#### Scenario: gRPC non-OK forces export

- GIVEN a client or server span with `rpc.grpc.status_code = UNKNOWN`
- WHEN the sampler evaluates the parent trace
- THEN the trace MUST be exported to `otlp/jaeger`

### Requirement: Probabilistic sampling of successful traffic

The otel-collector MUST apply probabilistic sampling to traces that do NOT
match any of the mandatory-retention policies, at a configurable rate with
a default of 5%.

#### Scenario: Default happy-path rate is 5%

- GIVEN a successful trace with no errors and latency under 1s
- AND the policy is configured with `probabilistic_happy.rate = 0.05`
- WHEN 1000 such traces are sent through the pipeline
- THEN approximately 50 traces (5% ± statistical noise) MUST be exported
- AND the remaining traces MUST be discarded

#### Scenario: Rate is configurable

- GIVEN the policy is configured with `probabilistic_happy.rate = 0.01`
- WHEN 1000 successful traces are sent through the pipeline
- THEN approximately 10 traces MUST be exported
- AND a collector restart MUST be acceptable to apply the rate change
  (hot-reload is not wired in this PR)

### Requirement: Catch-all minimum retention

The otel-collector MUST apply a catch-all probabilistic policy that ensures at
least 1% of any unmatched traffic is exported, even when no other policy
matches.

#### Scenario: Unmatched trace is still exported at low rate

- GIVEN a trace that matches none of the mandatory retention policies
- AND the catch-all policy is configured with `catch_all.rate = 0.01`
- WHEN 10000 such traces are sent through the pipeline
- THEN approximately 100 traces (1% ± statistical noise) MUST be exported
- AND the "zero traces exported" silent failure mode MUST NOT occur

### Requirement: Memory headroom for the sampler

The otel-collector MUST operate the `memory_limiter` processor at 60% or less
of the container's memory limit, to leave headroom for the tail-sampling
processor's in-memory decision cache.

#### Scenario: memory_limiter configured at 60%

- GIVEN the collector is configured with `memory_limiter.limit_percentage: 60`
- WHEN the container is started with `docker compose up -d otel-collector`
- THEN the healthcheck (`/otelcol-contrib validate --config=...`) MUST pass
- AND the collector's RSS MUST stay below the configured limit under
  synthetic load of 1000 traces/min for 5 minutes

#### Scenario: Headroom is documented in the config

- GIVEN a developer reads `infra/otel/collector-config.yaml`
- WHEN they reach the `memory_limiter` block
- THEN an inline comment MUST explain WHY 60% (instead of the prior 80%)
  is required (tail-sampling cache + num_traces dimensioning)

## Cross-references

- Proposal: `openspec/changes/cachicamas-tail-sampling/proposal.md`
- Engram artifact: `sdd/cachicamas-tail-sampling/spec`
- Next phase: `sdd-design` (single config-file diff, expected < 100 lines)
