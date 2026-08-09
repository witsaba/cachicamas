# OpenRouter live smoke test

This package's `TestOpenRouterAdapter_LiveSmoke` is an **opt-in, human-run
only** test that makes one real, bounded request against the live
OpenRouter gateway. It is not part of any automatic run: this repository
has no CI workflow at all (ADR 0005 § Enforcement — no `.github/workflows/`
directory exists), and `make test` passes with neither environment
variable below set.

## Quick start

Export both environment variables in your own shell — never write either
one to a file inside this repository:

```sh
# Paste the key at the silent prompt: it is never echoed and never enters shell history.
read -rs OPENROUTER_API_KEY && export OPENROUTER_API_KEY
export RUN_LIVE_OPENROUTER_SMOKE=1
```

Then run the exact test:

```sh
go test -run TestOpenRouterAdapter_LiveSmoke ./src/ai/openaicompat/openrouter/internal/smoke/ -v
```

(from `backend/agent/`, matching this repository's usual test invocation
root.)

With either variable unset, empty, or `RUN_LIVE_OPENROUTER_SMOKE` set to
anything other than the exact literal `1`, the test reports `--- SKIP`
immediately and makes no outbound request.

## Bound and cost

| | |
|---|---|
| Timeout | 60 seconds total. The request context carries this deadline; the stream drain derives its own bound from whatever remains of that same deadline when the drain starts, so both stages are cut off by one shared hard deadline — never two independent 60-second timers. |
| Approximate cost | ~1¢ (US) best case per run — one short `openai/gpt-4o` completion via OpenRouter. Up to ~4x that if a retryable transport failure triggers the adapter's own retry policy (see "Requests per run" below). |
| Requests per run | Exactly one `provider.Stream` invocation from this test — no retry or loop in the smoke's own code. The OpenRouter adapter underneath carries its own ratified HTTP-layer retry policy (unmodified by this change), which may issue up to 4 billed HTTP attempts on a retryable transport failure. |

## What is asserted

- A response-start event is present.
- At least one content event (a text delta or a tool-call event) is
  present.
- Exactly one terminal event exists, in terminal position.

## What is NOT asserted

- The model's generated text, tool-call arguments, or token counts — no
  assertion reads, compares, or matches provider-chosen content.

## Credential safety

Every diagnostic the live run produces — request metadata, drain outcome,
error renderings, and terminal diagnostics — is captured into a single
buffer and swept for the credential, the credential's prefix, and the
test's own planted prompt marker before any of it reaches the test log.
The sweep's own positive control runs first, so a broken sweep fails the
run rather than silently reporting a false "clean" result. If the sweep
finds a match, the test fails naming only which vector matched — it never
reprints the matched bytes.

**The credential is never written to any file inside this repository.**
Provide it only through the shell `export` shown above, in the process
that invokes `go test`.
