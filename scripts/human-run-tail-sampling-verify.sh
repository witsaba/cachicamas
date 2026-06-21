#!/usr/bin/env bash
# ============================================================================
# human-run-tail-sampling-verify.sh
#
# Manual verification script for the tail-sampling feature in the otel-collector.
# Maps 1:1 to Phase 3 of `openspec/changes/cachicamas-tail-sampling/tasks.md`.
#
# Run from the repo root with the stack already up:
#
#     docker compose up -d --build
#     ./scripts/human-run-tail-sampling-verify.sh
#
# Exits 0 if every check passes, 1 on the first failure.
# Pure bash + curl + jq + docker — no extra deps.
#
# Why a shell script (and not uv/python like validate-infra.py):
#   - All checks are docker/curl/jq, no need for pyyaml
#   - It runs OUTSIDE the agent (you, the human) and needs to be copy-pasteable
#     one block at a time when debugging a single scenario
#   - Each scenario is a labeled function that you can `unset -f` and re-run
#     individually after fixing the config
# ============================================================================

set -euo pipefail

# ---- Pretty output -----------------------------------------------------
RED=$'\033[0;31m'
GRN=$'\033[0;32m'
YLW=$'\033[0;33m'
BLU=$'\033[0;34m'
RST=$'\033[0m'

pass() { echo "${GRN}✓${RST} $1"; }
fail() { echo "${RED}✗${RST} $1" >&2; exit 1; }
info() { echo "${BLU}ℹ${RST} $1"; }
warn() { echo "${YLW}⚠${RST} $1"; }
heading() { echo; echo "${BLU}━━━ $1 ━━━${RST}"; }

# ---- Pre-flight --------------------------------------------------------
SERVICE_URL="${SERVICE_URL:-http://localhost:8080}"
JAEGER_URL="${JAEGER_URL:-http://localhost:16686}"
OTEL_COLLECTOR_HEALTH="${OTEL_COLLECTOR_HEALTH:-http://localhost:13133}"
SERVICE_NAME="${SERVICE_NAME:-database_administrator}"

for cmd in curl docker jq; do
  command -v "$cmd" >/dev/null 2>&1 || fail "missing required command: $cmd"
done

heading "Pre-flight: collector health"
STATUS=$(docker compose ps otel-collector --format '{{.Status}}' 2>/dev/null || echo "not found")
info "otel-collector status: $STATUS"
case "$STATUS" in
  *healthy*)  pass "otel-collector is healthy" ;;
  *starting*) warn "otel-collector still in 'starting' state — wait 30s and re-run"; exit 1 ;;
  *)          fail "otel-collector is not running healthy: $STATUS" ;;
esac

# Wait an extra 15s after healthcheck so the first batch has time to flush.
# The first batch has timeout=1s in the collector, so 5s is technically enough;
# 15s is a defensive margin for the very first span to make the round trip.
info "waiting 15s for first batch to flush..."
sleep 15

# ---- Helpers -----------------------------------------------------------
# count_traces_in_jaeger: query Jaeger's API for traces in a service within
# the last N minutes. Uses the legacy /api/traces endpoint (still works on
# Jaeger v2) — the new search endpoint at /api/search requires a different
# shape and an indexer setup we don't have in dev.
#
# NOTE: only `lookback` is sent. We deliberately omit `start`/`end` because
# in Jaeger 2.19.0, when all three are present, `start`+`end` are honored
# over `lookback` and return 0 traces for a "now - 5min" window depending
# on millisecond drift between the script's clock and Jaeger's evaluation
# of the request. `lookback` alone is reliable.
count_traces_in_jaeger() {
  local service="$1"
  local lookback_min="${2:-5}"
  curl -sS -G "$JAEGER_URL/api/traces" \
    --data-urlencode "service=$service" \
    --data-urlencode "lookback=${lookback_min}m" \
    --data-urlencode "limit=10000" \
  | jq '.data | length'
}

# ---- Test 3.1: errors are 100% retained --------------------------------
# Strategy: hit the dev-only `?fail=true` flag on /health, which makes the
# handler return HTTP 500. The otel.Middleware then marks the span with
# status_code=500 AND status=Error — both sub-policies (`http-5xx` and
# `errors`) of the composite `keep-errors-and-slows` policy match, so the
# tail sampler retains the trace at 100%.
#
# Why not stop postgres? The current /health handler is a stub that does
# not touch the DB — stopping postgres produces 10 happy-path 200s and
# proves nothing about the sampler.
test_3_1_error_retention() {
  heading "Test 3.1: error spans are 100% retained"
  info "firing 10 GET /health?fail=true requests (dev-only, returns 500)"

  BASELINE=$(count_traces_in_jaeger "$SERVICE_NAME" 5)
  info "baseline trace count (last 5 min): $BASELINE"

  HTTP_CODES=""
  for i in $(seq 1 10); do
    CODE=$(curl -s -o /dev/null -w "%{http_code}" "$SERVICE_URL/health?fail=true" || echo "000")
    HTTP_CODES="$HTTP_CODES $CODE"
  done
  info "HTTP codes observed: $HTTP_CODES"

  info "waiting 20s for the sampler to evaluate and export..."
  sleep 20

  AFTER=$(count_traces_in_jaeger "$SERVICE_NAME" 5)
  info "trace count after 10 forced errors: $AFTER"

  NEW_TRACES=$((AFTER - BASELINE))
  info "new traces in Jaeger: $NEW_TRACES (expected ≥ 10, ideally exactly 10)"

  if [ "$NEW_TRACES" -ge 10 ]; then
    pass "all 10 error traces are in Jaeger (100% retention of errors)"
  else
    fail "only $NEW_TRACES error traces retained, expected ≥ 10"
  fi
}

# ---- Test 3.3: slow traces are 100% retained ---------------------------
# We can't easily force a >1s latency from outside without instrumenting the
# backend, so this test is OPT-IN. Set RUN_SLOW_TEST=1 to enable.
test_3_3_slow_retention() {
  heading "Test 3.3: traces with latency > 1s are 100% retained"
  if [ "${RUN_SLOW_TEST:-0}" != "1" ]; then
    warn "skipped (set RUN_SLOW_TEST=1 to enable; requires an endpoint that sleeps ≥1s)"
    return 0
  fi

  BASELINE=$(count_traces_in_jaeger "$SERVICE_NAME" 5)
  info "baseline trace count: $BASELINE"

  SLOW_URL="${SLOW_URL:-$SERVICE_URL/slow}"
  info "hitting slow endpoint: $SLOW_URL"
  curl -s -o /dev/null -w "  request took %{time_total}s\n" --max-time 10 "$SLOW_URL"

  info "waiting 20s for sampler decision..."
  sleep 20

  AFTER=$(count_traces_in_jaeger "$SERVICE_NAME" 5)
  NEW_TRACES=$((AFTER - BASELINE))
  info "new traces in Jaeger: $NEW_TRACES (expected ≥ 1)"

  if [ "$NEW_TRACES" -ge 1 ]; then
    pass "slow trace retained by tail sampler"
  else
    fail "slow trace was dropped — latency policy is not matching"
  fi
}

# ---- Test 3.6: happy-path volume reduction -----------------------------
test_3_6_volume_reduction() {
  heading "Test 3.6: happy-path volume reduction (5% probabilistic)"
  N_REQUESTS="${N_REQUESTS:-1000}"

  BASELINE=$(count_traces_in_jaeger "$SERVICE_NAME" 5)
  info "baseline trace count: $BASELINE"

  info "firing $N_REQUESTS successful requests against $SERVICE_URL/health..."
  for i in $(seq 1 "$N_REQUESTS"); do
    curl -s -o /dev/null "$SERVICE_URL/health" &
    # Batch 50 in flight to keep the loop fast without overwhelming the server
    if [ $((i % 50)) -eq 0 ]; then wait; fi
  done
  wait
  info "all $N_REQUESTS requests completed"

  info "waiting 25s for the last batch to flush through the sampler..."
  sleep 25

  AFTER=$(count_traces_in_jaeger "$SERVICE_NAME" 5)
  NEW_TRACES=$((AFTER - BASELINE))
  info "new traces in Jaeger: $NEW_TRACES (expected ~$((N_REQUESTS / 20)) at 5% sampling)"

  # Lower bound: 1% of N_REQUESTS (the catch-all floor).
  # Upper bound: 10% of N_REQUESTS (allowing statistical noise + a few errors).
  LOWER=$((N_REQUESTS / 100))
  UPPER=$((N_REQUESTS / 10))
  if [ "$NEW_TRACES" -ge "$LOWER" ] && [ "$NEW_TRACES" -le "$UPPER" ]; then
    pass "volume reduction within expected range [$LOWER, $UPPER]"
  elif [ "$NEW_TRACES" -gt "$UPPER" ]; then
    fail "retention too high ($NEW_TRACES > $UPPER) — probabilistic sampler may not be wired"
  else
    fail "retention too low ($NEW_TRACES < $LOWER) — catch-all may be off"
  fi
}

# ---- Test 3.8: memory headroom ------------------------------------------
test_3_8_memory_headroom() {
  heading "Test 3.8: memory headroom under load"
  RSS_KB=$(docker stats cachicamas-otel-collector --no-stream --format '{{.MemUsage}}' | awk '{print $1}')
  info "current RSS: $RSS_KB"

  # Extract the numeric part (e.g. "123.4MiB" -> 123.4)
  RSS_MIB=$(echo "$RSS_KB" | sed -E 's/^([0-9.]+).*/\1/')
  # Threshold: 400 MB. Use awk for float comparison.
  UNDER_LIMIT=$(awk -v rss="$RSS_MIB" 'BEGIN { print (rss < 400) ? 1 : 0 }')

  if [ "$UNDER_LIMIT" -eq 1 ]; then
    pass "RSS (${RSS_MIB} MiB) is under the 400 MiB budget"
  else
    fail "RSS (${RSS_MIB} MiB) exceeded the 400 MiB budget — memory_limiter may be misconfigured"
  fi
}

# ---- Test 3.9: healthcheck stability -----------------------------------
test_3_9_health_stability() {
  heading "Test 3.9: healthcheck stability (snapshot only — full window is 5 min)"
  STATUS=$(docker compose ps otel-collector --format '{{.Status}}' 2>/dev/null || echo "not found")
  if [[ "$STATUS" == *healthy* ]]; then
    pass "otel-collector is healthy right now"
    warn "for a 5-min soak test, run: watch -n 30 'docker compose ps otel-collector'"
  else
    fail "otel-collector is not healthy: $STATUS"
  fi
}

# ---- Run ---------------------------------------------------------------
test_3_1_error_retention
test_3_3_slow_retention
test_3_6_volume_reduction
test_3_8_memory_headroom
test_3_9_health_stability

heading "All tests passed"
info "if you want a full 5-min memory soak, run the loop in test 3.9 manually"
info "if any test failed, the first failing line tells you which policy to check in"
info "  infra/otel/collector-config.yaml under the tail_sampling block"
