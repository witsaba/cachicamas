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
export OPENROUTER_API_KEY=your-own-openrouter-api-key
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
| Timeout | 60 seconds, shared by the request context and the stream drain |
| Approximate cost | ~1¢ (US) per run — one short `openai/gpt-4o` completion via OpenRouter |
| Requests per run | Exactly one — no retry, no loop |

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
