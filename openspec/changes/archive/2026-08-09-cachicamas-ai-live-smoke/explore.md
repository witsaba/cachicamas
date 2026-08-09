# Explore — AI-39: Add the opt-in live smoke test (Wave 6 — Hand off)

> **Change**: `cachicamas-ai-live-smoke`
> **Milestone**: AI-39 (doc 0002, lines 2345–2365) · **Wave**: 6 — Hand off
> **Artifact store**: hybrid (file + Engram) · **Engram**: `#2785` (`sdd/cachicamas-ai-live-smoke/explore`)
> **Worktree**: `/Users/braejan/workspace/witsaba/repositories/cachicamas-worktrees/ai-39-live-smoke`
> **Depends on**: AI-38 (merged, `5bc2da4e`) · **Blocks**: nothing — AI-40 may treat AI-39 as optional

This file materializes Engram observation `#2785`. The exploration session had no Write tool, so
the body below is that observation's content; the closing section **Additions after
re-verification** carries findings the proposal phase established afterwards with Grep/Read over
the worktree. The original body is left as written so the two can be compared.

---

## Current state

A live-smoke package already exists on `main` at
`backend/agent/src/ai/openaicompat/openrouter/smoke/` (arrived pre-AI-39 as commit `f446526b`,
PR #3 of `add-openrouter-first-provider`; touched by AI-36 `3311aadb` and a comment refactor
`5f9f6ea9`). Three files:

- `smoke_test.go` (package `smoke_test`) — `decideLiveSmoke`, the two-stage env-var gate
  (`liveSmokeEnvVarName` = `OPENROUTER_API_KEY`, `liveSmokeRunFlagName` =
  `RUN_LIVE_OPENROUTER_SMOKE`, opens only on the literal `"1"`), `buildLiveSmokeRequest`,
  `TestOpenRouterAdapter_LiveSmoke`, and 4 `TestLiveSmokeGate_*` unit tests. Env-var literals are
  byte-assembled at runtime so the sentinel sweep does not flag its own source.
- `sentinel_sweep.go` (package `smoke`) — pure `Scan(captured []byte, denyList []DenyEntry) error`
  plus `BuildDenyList(envVarName, secretPrefix, plantedPrompt string) []DenyEntry`, delegating the
  match to the shared `agenttest/sweep` core (AI-36 AD-1). `DenyEntry` is a type alias of
  `sweep.Entry`.
- `sentinel_sweep_test.go` — 8 tests proving `Scan` against the 3 deny-list vectors (env-var name,
  secret prefix, planted prompt), a positive control (`<redacted>` never triggers), and 3
  deliberate-mutation bite-proofs.

No 4th file (setup docs) exists in the package directory.

## Gap matrix — AI-39.1 test list (5 items) vs the current package

| # | Requirement | Status | Evidence |
|---|---|---|---|
| 1 | Skips cleanly without credentials; `make test` never depends on it | **SATISFIED** | `TestOpenRouterAdapter_LiveSmoke` calls `decideLiveSmoke(liveSmokeEnvAdapter)` and `t.Skip(d.Reason)` when the gate is closed. AI-38's archive `verify-report.md` (Round 3) recorded this exact test as 1 of 26 benign `--- SKIP` under `make test`, exit 0. |
| 2 | With credentials: one bounded request under a hard timeout, asserting ONLY stream-shape invariants — start, ≥1 content event, exactly ONE terminal — never model output | **PARTIAL / weaker than spec** | Boundedness is solid (`context.WithTimeout(60s)` + `agenttest.DrainAndRecord` with the same deadline). The assertion is only `len(events) == 0` (fail) plus `hasAnyChunk(events)` — a disjunction over ResponseStart/TextDelta/ToolCallStart/ToolCallDelta/Completion, not three separate invariants. It does not assert a `ResponseStart` is present, does not assert a distinct content event, and explicitly declines exactly-one-terminal (`t.Logf("... final-kind-diagnostic only, not an assertion")`). |
| 3 | Output never contains the credential or the full prompt, even on failure, asserted sentinel-style over captured output | **GAP — the sweep exists but is not wired to the live test** | `sentinel_sweep.go` / `sentinel_sweep_test.go` prove `Scan`/`BuildDenyList` in isolation (8 green tests). `smoke_test.go` never imports the `smoke` package and never calls `Scan`/`BuildDenyList` — its full import list is `context, encoding/json, fmt, os, strings, testing, time, agenttest, ai, openaicompat, openrouter`. `TestOpenRouterAdapter_LiveSmoke` captures no output buffer and never scans one, so the "even on failure" clause is unmet. |
| 4 | Unreachable from any application entry point, proven mechanically: package is internal + a dependency walk from each composition root shows no path to it | **GAP — both halves missing, and the module has zero composition roots** | The package path contains no `internal/` segment, so Go's import-visibility rule does not restrict it: it is importable from anywhere in the module and, being a public module path, from outside it. `backend/agent/README.md` states "Only `src/ai` exists today"; the Makefile states the composition root arrives with doc 0004 (`src/cmd/cachicamas`). The only `package main` files under `backend/agent/` are two testdata fixtures for an unrelated constructor guard. So a literal "walk from each composition root" is today either unwritable or vacuous. |
| 5 | Credential-safe setup instructions ship with the package; following them never requires writing a credential to a file in the repo | **GAP — no doc exists** | `Glob` of the package directory returns only the 3 `.go` files. Needs a doc naming both env vars, instructing shell `export` (never a file), the exact invocation, and the cost bound (~60s timeout on `openai/gpt-4o` via OpenRouter, roughly one cent per run per the source comment). |

## Existing mechanical-guard precedent relevant to item 4

`backend/agent/src/ai/import_boundary_test.go` (AI-00.3 forward guard) is the established pattern
for this class of proof: deny-by-default
`go list -deps -test -f '{{if not .Standard}}{{.ImportPath}}{{end}}'` over a module pattern,
checked against a forbidden-prefix table and an allowlist, plus an exact-closure pin and an
anti-vacuity `Fatal`. `backend/database_administrator/src/domain/imports_test.go` does the
mirror-image negative-import guard for cross-module isolation. Both are directly reusable models
in place of a literal walk from the (nonexistent) composition root.

## Env vars and gating mechanism (confirmed, byte-exact)

- `OPENROUTER_API_KEY` — bearer credential. Absence → Skip (stage 1).
- `RUN_LIVE_OPENROUTER_SMOKE` — must equal the literal `"1"` exactly (boundary-tested: `"true"`,
  `"yes"`, `"0"`, `" 1 "`, `"1\n"` all still skip). Two-stage gate: the secret alone is never
  sufficient consent.
- Both names are byte-assembled at runtime in `smoke_test.go`, never a contiguous literal, so the
  sentinel sweep does not flag its own source file.
- `liveSmokePromptMarker = "live-smoke-prompt-marker-9b3a8f2c"` is the planted, non-secret marker
  embedded in the live request's prompt, reused as the sweep's third deny-list entry.
- `liveSmokeRequestTimeout = 60 * time.Second` bounds both the outbound `context.WithTimeout` and
  the `agenttest.DrainAndRecord` deadline.

## Sentinel-sweep mechanism (confirmed)

`smoke.Scan(captured []byte, denyList []DenyEntry) error` walks 3 deny-list entries in declaration
order (env-var name, 4-character secret prefix, planted prompt) via the shared
`agenttest/sweep.Scan` core, returning an error naming only the vector
(`"env-var name"` / `"secret prefix"` / `"planted prompt"`), never the matched bytes. The
machinery is solid and reusable; the gap is purely that nothing in `smoke_test.go` calls it over
the live test's own captured output.

## Composition-root inventory

- `backend/agent`: **zero composition roots today.** `src/cmd/cachicamas` is the single planned
  root, deferred to doc 0004 (Layer 3). README layering: `src/cmd/cachicamas → src/coding →
  src/agent → src/ai`; only `src/ai` exists. The two `package main` files
  (`src/ai/testdata/{handrolled,constructed}/main.go`) are compile fixtures, not entry points.
- `backend/database_administrator` and `backend/workspace_syncer`: each has its own
  `src/cmd/server/main.go` — real composition roots, but in separate Go modules (root `go.work`
  lists all three) with a mechanically guarded zero-import boundary against `backend/agent`.

## Go package-visibility analysis for the "internal" requirement

A package under a path containing a literal `internal` directory segment is importable only by
packages rooted at that segment's parent. The smoke package's current path has no such segment.
Inserting one — e.g. `.../openrouter/internal/smoke`, mirroring `src/ai/internal/retry` — makes
the compiler itself enforce "unreachable except from within `.../openrouter/`" forever, for any
future composition root. That is stronger and more durable than a `go list` test alone, which
remains useful as a regression detector that reports *why*.

## Reconciliation target (a) — canonical spec never promoted

`openspec/specs/ai-openrouter-first-provider/spec.md` header (lines 1–11) is unchanged since
2026-08-06 and still reads `> **Status**: DRAFT`, `> **Change**: add-openrouter-first-provider`,
and `> **Links**: [proposal](../../proposal.md) · [explore](../../explore.md)`. It never went
through the four-part promotion transform every other canonical spec in this repo gets (per
`WAVE-1-ARCHIVE.md` § 2, and per AI-38's archive report, which explicitly declined the fix and
fenced it to AI-39): header rewrite `Status: DRAFT` → `Status: live` plus `Introduced by:`,
relative links re-resolved from active-change depth to canonical depth (`../../proposal.md` 404s
today), and an added `## Status` canonical-home section. AI-38 did modify `R-OR-05`/`R-OR-06`
in place in this file, so reconciliation is header/links-only — except for R-OR-07/08 below.

## Reconciliation target (b) — orphan change folder + superseded R-OR-07

- **Archived** (`openspec/changes/archive/2026-08-06-add-openrouter-first-provider/`) — closed;
  its R-OR-07 reads "Live smoke is opt-in via workflow dispatch and repo secret" and requires a
  `.github/workflows/agent-openrouter-smoke.yml` `workflow_dispatch`-only file. **Superseded**:
  the repository has no `.github/` directory at all (ADR 0005's no-CI posture) and the shipped
  implementation gates on env vars only.
- **Orphan, still live** (`openspec/changes/add-openrouter-first-provider/`, containing only
  `specs/ai-openrouter-first-provider/spec.md`, no proposal/design/tasks) — R-OR-07 here reads
  "Live smoke is opt-in via env vars only (no CI workflow)" and correctly matches the shipped
  gate, stating "No CI workflow file is required by this spec".
- The **canonical** spec still carries the old workflow-dispatch R-OR-07 text (AI-38 fenced
  R-OR-07/08 out of its edits), so canonical text today describes CI that does not exist and
  never will.
- AI-38's delta spec header states: "**Scope fence**: R-OR-07 / R-OR-08 (live-smoke gating and
  sentinel sweep) and the orphan `openspec/changes/add-openrouter-first-provider/` directory are
  AI-39's, not touched here."

## Reconciliation target (c) — carried forward, not closed by AI-39 alone

AI-38's archive report records: "One deliberate reporting gap: production/test file-count
reconciliation was not performed at this close, for lack of shell tooling in that archive
session." It defers the `find`-measured delta to the next milestone's close with shell access —
this change.

## Backend test runner

`backend/agent/Makefile` — `make test` runs `go test -race -v ./...` (pinned body per its own
comment); `make test/cover`, `make lint`, `make vet`, `make build` round out the local pipeline
(no `.github/workflows/`, per ADR 0005). Root `go.work` unifies `./backend/agent`,
`./backend/database_administrator`, `./backend/workspace_syncer`.

## Candidate approaches

| Approach | Pros | Cons | Effort |
|---|---|---|---|
| **A. Minimal in-place fix** — move `smoke/` under `.../openrouter/internal/smoke/`; add an AI-00.3-style `go list` guard; wire `Scan`+`BuildDenyList` into the live test over captured output; strengthen the stream-shape assertion; add a setup doc | Smallest diff; reuses three proven repo patterns; no new package-boundary design | The capture-on-failure mechanism needs a concrete Go decision (no built-in `testing.T` output hook) | Medium |
| **B. New internal harness package** with an exported `Run(t, opts)` owning credential capture, timeout, drain, sweep and assertions | Cleaner separation; positions for reuse | More surface to design and review than the milestone's acceptance asks for | Medium-High |
| **C. Defer item 4's dependency walk** to a `t.Skip`'d placeholder until doc 0004 creates a real root, relying only on the compiler `internal/` boundary | Avoids a test that cannot observe what it claims | Weakens "proven mechanically" to a naming convention | Low |

**Recommendation**: Approach A. For item 4's zero-root problem, combine the compiler-enforced
`internal/` boundary with a `go list`-based regression guard scoped honestly to "no non-internal
package in this module imports the smoke path today", with the same anti-vacuity `Fatal` that
`import_boundary_test.go` already uses.

## Open product questions for the propose phase

1. Exact capture mechanism for item 3's "captured output, even on failure".
2. Whether item 4's guard should hard-fail today or document its own scope, with anti-vacuity.
3. Whether reconciliation targets (a)/(b) ship in the same PR as the smoke fixes.
4. Whether the standalone `go test -run` invocation must stay valid after the path move
   (setup instructions must follow the new path).

---

## Additions after re-verification (proposal phase, same worktree)

Three blast-radius facts the original exploration did not surface. Each was confirmed by reading
the cited file in this worktree, and each changes the shape of the recommended fix.

### A-1. `src/agenttest` imports the smoke package today — the move breaks a compile

`backend/agent/src/agenttest/sweep_convergence_test.go:16` imports
`github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter/smoke` and calls
`smoke.Scan` / `smoke.BuildDenyList` at lines 31 and 49. `src/agenttest` is **not** under
`.../openrouter/`, so after the move to `.../openrouter/internal/smoke` this file can no longer
import it and the package will not compile.

That test is the AI-36.1 behavioural pin for the "both reach that one implementation" clause of
**S-CNF-080**, a live requirement in `openspec/specs/ai-provider-conformance-suite/spec.md:431`.
Its agenttest-side counterpart, `scanTextForSentinel`, is unexported
(`agenttest/conformance_redaction.go:108`), so the pin cannot simply move into the smoke package.

Resolution space: (i) export a narrow agenttest helper — widens the surface AI-40 is about to
freeze; (ii) re-anchor convergence to the shared core from both sides — the smoke package's own
test asserts `smoke.Scan` ≡ `sweep.Scan` over one corpus, agenttest keeps asserting
`scanTextForSentinel` ≡ `sweep.Scan`, making convergence transitive through the single
implementation S-CNF-080 already names; (iii) do not move the package. The proposal selects (ii).

### A-2. The credential scan's allowlist is keyed by exact relative path

`backend/agent/src/ai/openaicompat/credential_scan_test.go:84-85` enumerates
`filepath.Join("openrouter", "smoke", "smoke_test.go")` and
`filepath.Join("openrouter", "smoke", "sentinel_sweep_test.go")`. The scan's own doc (lines
15–19) states that a file is exempt only by carrying an in-file declaration **and** appearing on
this allowlist, and that updating one without the other still fails. The recursive `WalkDir`
keeps covering the files after a move, but both allowlist entries must be updated in the same
commit as the move, or the scan fails.

### A-3. Two exported reuse targets already exist that the exploration did not find

- `agenttest.RequireValidStream(tb, rec)` (`agenttest/stream_kit_ordering.go:21`) is **exported**
  and delegates to `ai.CheckStream`, whose documented rules include "terminal exclusivity and
  placement" (R-STK-005), then `CheckContiguity` (R-STK-006). The item-2 helpers named in the
  original exploration (`terminalExactlyOneCase`, `requireDrainedKinds`) are unexported and
  unreachable from `package smoke_test`; `RequireValidStream` is the reachable equivalent for the
  exactly-one-terminal half. Because `ai.CheckStream` reports no violation for an empty slice,
  non-emptiness, start-present and content-present remain separate local assertions.
- `sweep.SelfTest(deny)` (`agenttest/sweep/sweep.go:85`) is the module's mandatory positive
  control: the package doc states "Every sweep call site runs SelfTest(deny) before trusting a
  clean Scan result over the real corpus", and `conformance_redaction.go:182`,
  `a_i-36_1_test.go:157`, `ai37_denylist_test.go:181` all comply. The smoke package calls it
  **nowhere**. Item 3's wiring must run it, or a clean sweep over live output is exactly the
  vacuous absence assertion AI-36 exists to prevent.
