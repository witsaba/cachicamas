# Design: AG-17 — Add the context strategy seam and token accounting (`cachicamas-agent-context-strategy`)

> Change `cachicamas-agent-context-strategy` · Milestone AG-17 · Worktree `cachicamas-worktrees/ag17`, base `origin/main@b0de5bf6`
> Inputs: `proposal.md` (Decisions 1–5, binding), `explore.md` §1–§8. Every citation below re-read against the worktree during this phase.
> Settled upstream, cited not re-litigated: seam is a `Harness` field consulted once per LOGICAL turn between `harness.go:498` and `:530` (`agent-run-driver/spec.md:342`); no new counting interface (`R-AMP-017`, `ai/provider.go:123-125`); three type-level provenance states; presence-carrying budget; estimate over UTF-8 bytes; no `L2C` row; prefix `R-CTX-`/`S-CTX-`; five spec deltas.

## Technical Approach

Mint the seam in one new file (`context_strategy.go`) mirroring `failover_policy.go` symbol-for-symbol: one-method interface, exported-field prompt struct, empty verdict struct, shipped no-op, `var _` guard. Mint the measurement in a second new file (`token_accounting.go`): a three-state provenance type, a resolver that type-asserts the shipped `ai.TokenCounter` (`ai/provider.go:130-134`), and a pure byte-based estimate. Wire both into `Harness.Run`'s outer loop with one guarded block inserted immediately after `transcript := transcriptFromHistory(hist)` (`harness.go:498`), so the nil path is byte-for-byte the pre-AG-17 path. Extend `agenttest` with one exported counting-capable fixture. `doc.go`, `doc_contract_guard_test.go`, `stream_check.go`, `history.go`, `cost_events.go`, `cost_usage.go`, `compaction_events.go`, `event_descriptor.go`, `event_registry_test.go`, `go.mod`/`go.sum`, and all of `src/ai/**` stay byte-unchanged.

## Architecture Decisions

### DD1 — The seam types: exact declarations, and why a non-default verdict is unconstructible

New file `src/agent/context_strategy.go`:

```go
// ContextStrategy is the turn-boundary context seam (R-CTX-001).
// Method name mirrors PermissionPolicy.Resolve and FailoverPolicy.Resolve
// (failover_policy.go:25-26) — the house one-method seam convention.
type ContextStrategy interface {
	// Resolve is consulted exactly once per LOGICAL turn, at AG-13's
	// turn boundary in Harness.Run's outer loop — never per retry
	// attempt (see the R-RTY-002 argument at harness.go:518-529).
	Resolve(ctx context.Context, prompt ContextPrompt) ContextVerdict
}

// ContextPrompt is the typed report Resolve receives — the same
// exported-field posture as FailoverPrompt (failover_policy.go:44-53).
type ContextPrompt struct {
	// Transcript is a fresh clone of the slice the coming logical
	// turn's attempts will send (request.go:367-370's own argument:
	// a consumer that rewrites what it received must not be able to
	// rewrite the harness's slice).
	Transcript []ai.Message

	// Budget is Layer 3's stated budget, possibly absent.
	Budget ContextBudget

	// Accounting is the transcript's measured size with type-level
	// provenance (DD4). Even TokenSourceReported is exact only for
	// the PRE-hook request (DD6).
	Accounting TokenAccounting
}

// ContextVerdict is Resolve's typed return value. v1 ships NO field:
// the zero value is the only constructible value, so a verdict that
// requests compaction is unconstructible by ANY implementation — the
// FailoverVerdict posture (failover_policy.go:55-63). AG-18 adds
// compaction fields non-breakingly; every implementation returning
// the zero verdict keeps compiling.
type ContextVerdict struct{}

// NoOpContextStrategy is the one shipped, installable never-compact
// default. Installing it changes nothing observable versus leaving
// Harness.ContextStrategy nil (the inertness pin).
type NoOpContextStrategy struct{}

func (NoOpContextStrategy) Resolve(context.Context, ContextPrompt) ContextVerdict {
	return ContextVerdict{}
}

var _ ContextStrategy = NoOpContextStrategy{}
```

**What AG-17 does with a non-default verdict: nothing, because none exists.** `ContextVerdict struct{}` has exactly one value; the harness receives it and discards it (the same bare-call posture as `h.Failover.Resolve(...)` at `harness.go:622`). This is deliberately stronger than "reject" or "ignore": an ignored accept-flag would make the never-compacts scenario a property of the shipped implementation only, while an empty struct makes it a property of the type — unfalsifiable by any caller-written strategy (`failover_policy.go:57-62` blesses exactly this extension path for AG-18). Rejected: a `Verdict` enum with one member (invites a second member landing without harness handling); an `error` return (a strategy is a consulted observer in v1, not a failure site).

### DD2 — Consultation site: exact placement, quoted, with the accounting inside the nil guard

The seam consults between `transcript := transcriptFromHistory(hist)` (`harness.go:498`) and the attempt loop `for attempt := 1; ; attempt++` (`harness.go:530`) — the binding placement. Within that window, it goes **immediately after `:498`**, before the retry-bound block at `:500-503`: the seam reads only transcript, budget, and accounting, and sitting adjacent to the transcript resolution makes its once-per-logical-turn pairing structurally identical to the timing resolution's own argument at `harness.go:505-511`.

Before (today, `harness.go:498-500`):

```go
		transcript := transcriptFromHistory(hist)

		bound := h.RetryAttempts
```

After:

```go
		transcript := transcriptFromHistory(hist)

		// AG-17 (R-CTX-001, R-CTX-002): the context seam, consulted
		// exactly once per LOGICAL turn — at AG-13's turn boundary,
		// outside the attempt loop below — never per attempt. A
		// per-attempt consultation would let a future compacting
		// verdict (AG-18) mutate the transcript between two attempts
		// of one logical turn, making R-RTY-002's "identical
		// transcript, reused BY REFERENCE" (harness.go:518-529)
		// unprovable by the exact argument its comment relies on.
		// A nil strategy is never consulted and no accounting is
		// resolved: the nil path is byte-for-byte the pre-AG-17 path.
		if h.ContextStrategy != nil {
			h.ContextStrategy.Resolve(runCtx, ContextPrompt{
				Transcript: slices.Clone(transcript),
				Budget:     h.ContextBudget,
				Accounting: resolveTokenAccounting(runCtx, h.Provider, h.Turn, h.System, transcript),
			})
		}

		bound := h.RetryAttempts
```

**Accounting runs only under the non-nil guard.** With a nil strategy, `CountTokens` is never invoked, so the nil path is identical even at the provider-interaction level, not just the event level. With `NoOpContextStrategy{}` installed and a counting provider, the one observable difference is the `CountTokens` call itself — provider-visible, event-invisible, and outside the inertness pin by design (the pin compares event streams and history, and its fixture uses the non-counting `agenttest.Provider` so even provider interactions are identical). Two new `Harness` fields land after `Failover` (`harness.go:91`), before the unexported block at `:93`: `ContextStrategy ContextStrategy` (nil-default) and `ContextBudget ContextBudget` (zero-default = absent).

### DD3 — The budget: constructor name, negative input, and reachability

New declarations in `context_strategy.go`, mirroring `ai.TokenCount` (`ai/usage.go:44-65`):

```go
// ContextBudget carries Layer 3's stated token budget. Its zero value
// is ABSENT — "Layer 3 stated no budget" — never "a budget of zero
// tokens"; ContextBudgetOf(0) is a stated zero and a different value
// (the ai.TokenCount discipline, usage.go:34-47).
type ContextBudget struct {
	limit   int64
	present bool
}

// ContextBudgetOf builds a stated budget. It is total: n < 0 yields
// the absent zero value — a negative limit is not a budget, and
// minting it present would hand AG-18 a bound under which every
// transcript overflows, the exact defect the presence bit prevents.
func ContextBudgetOf(n int64) ContextBudget {
	if n < 0 {
		return ContextBudget{}
	}
	return ContextBudget{limit: n, present: true}
}

// Limit returns the limit and whether Layer 3 stated one — the
// two-result idiom, so the limit is unreadable without its presence
// (usage.go:57-62).
func (b ContextBudget) Limit() (int64, bool) { return b.limit, b.present }
```

Closing the proposal's three open sub-questions: constructor is **`ContextBudgetOf`**; a negative `n` **maps to absent** (rejected: an `error` return — a second validation surface for a value Layer 2 never interprets, heavier than `ai.Tokens`' total constructor at `usage.go:53-55`; rejected: clamp to present 0 — mints the worst possible value; rejected: panic — breaks Layer 2's totality posture); the budget is **not** reachable per-turn — one harness-scoped field only, since the model is fixed per harness in v1 and `TurnOptions` is the wrong home by Decision 1's own argument.

### DD4 — Accounting: three type-level states, the resolver, and where it lives

New file `src/agent/token_accounting.go` — **not** `cost_usage.go`. That file's package comment scopes it to AG-16 end-to-end (`cost_usage.go:1-12`), and both milestones' single-revert rollback plans depend on whole-file deletion; sharing a file would entangle them. Composition, not duplication: the resolver reads the reported figure through the same public `TokenCount.Count()` idiom `cost_usage.go:12` names as its own route (`costFromUsage`, `cost_usage.go:38-60`), constructs no `costPresence`, and touches no `cost_*` file.

```go
// TokenSource is the provenance of a token figure (R-CTX-00N).
// The zero value is Unavailable, so a zero TokenAccounting{} reads as
// "no figure", never as "0 tokens".
type TokenSource int

const (
	TokenSourceUnavailable TokenSource = iota // no figure exists
	TokenSourceReported                       // the provider's ai.TokenCounter reported it
	TokenSourceEstimated                      // Layer 2's documented heuristic produced it
)

func (s TokenSource) String() string // renders the three states distinctly (usage.go:72-77's log-line argument)

// TokenAccounting is a token figure that cannot be read without its
// provenance. Unexported fields; the only accessor is two-result.
type TokenAccounting struct {
	tokens int64
	source TokenSource
}

// Tokens returns the figure and its provenance. A consumer physically
// cannot obtain the number without also obtaining the source — the
// mechanical enforcement of "an estimate never masquerades as exact".
func (a TokenAccounting) Tokens() (int64, TokenSource) { return a.tokens, a.source }
```

The resolver (package-private — no Layer 3 exists, `0003:110`; external tests read `ContextPrompt.Accounting` through a recording strategy, the AG-16 `cost_usage` surface argument):

```go
func resolveTokenAccounting(ctx context.Context, provider ai.ModelProvider, opts TurnOptions, system string, transcript []ai.Message) TokenAccounting {
	req, err := buildLoopRequest(opts, system, transcript) // loop.go:713-726 — the SAME builder the turn uses
	if err != nil {
		return TokenAccounting{} // Unavailable: the turn's own build at loop.go:304 stays the single abort authority
	}
	counter, ok := provider.(ai.TokenCounter) // the ONLY discovery mechanism (ai/provider.go:106-113)
	if !ok {
		return TokenAccounting{tokens: estimateTokens(req), source: TokenSourceEstimated} // clean absence, R-AMP-018
	}
	tc, cerr := counter.CountTokens(ctx, req)
	n, present := tc.Count()
	if cerr != nil || !present {
		return TokenAccounting{} // advertised-but-not-answering is NON-CONFORMANT, not absent (R-AMP-019) — NEVER the estimate
	}
	return TokenAccounting{tokens: n, source: TokenSourceReported}
}
```

Closing the proposal's two open sub-questions: the counter's error is **collapsed into `TokenSourceUnavailable`, not carried on the prompt** — the strategy is a measurement consumer, not an error handler; `R-AMP-019` non-conformance stays observable through the `Unavailable` label the scenario asserts, and AG-18 can widen `ContextPrompt` non-breakingly if diagnosis is ever needed (rejected: an `Err` field — invites strategies to make routing decisions that belong to the driver). A `buildLoopRequest` failure **yields `Unavailable`** — duplicating the abort decision would create a second failure site for one condition (proposal Decision 3's own recommendation). An advertised counter answering an **absent** `TokenCount` with a nil error is also `Unavailable`: "answered no figure" is not a report, and estimating there would launder it identically.

### DD5 — The estimate: constants, exact accessor walk, and asserted determinism

```go
// estimateBytesPerToken (D) = 4: byte-level BPE averages ~4 UTF-8
// bytes per token on mixed prose; bytes, not runes, because a 3-byte
// CJK ideograph is roughly one token (runes under-count it ~4x, bytes
// under 2x). Errs toward over-counting dense ASCII — the safe
// direction for a budget check (compaction asked earlier, never later).
const estimateBytesPerToken = 4

// estimateTokensPerMessage (M) = 4: chat encodings add role and
// delimiter tokens per message that no byte count sees (~3-4 per
// message); 4 keeps the same conservative direction. Dominant on
// short-message transcripts, where byte counting alone is worst.
const estimateTokensPerMessage = 4

// estimateTokens = ceil(B/D) + M*K over req. PURE: no clock, no env,
// no randomness, no I/O. The ambient-authority guard forbids only os,
// os/exec, syscall, io/ioutil (ambient_authority_test.go:73-94) — NOT
// time — so purity is asserted directly by test, never assumed from
// the guard. Approximate BY CONSTRUCTION, unbounded in both
// directions: over-counts dense ASCII, under-counts non-Latin
// scripts, base64-like content, unusual tool schemas. Its result is
// only ever labelled TokenSourceEstimated.
func estimateTokens(req ai.Request) int64
```

**B is the exact accessor walk** (pinned so a later Layer 1 field cannot be missed silently; the spec states this walk normatively):

| Region | Accessors (all opened this phase) |
|---|---|
| System instruction | `req.SystemInstruction()` (`system_instruction.go:154`) → `Segments()` (`:131`) → `Segment.Text()` (`:48`) |
| Messages (K = `len(req.Messages())`) | `req.Messages()` (`request.go:375`) → `Message.Content()` (`message.go:175`) → per part: `Part.Text()` (`text_content.go:115`); `Part.ToolCall()` (`tool_call.go:138`) → `Name()` (`:164`) + `Arguments()` (`:173`); `Part.ToolResult()` (`tool_result.go:123`) → `Content()` (`:140`); `Part.Reasoning()` (`reasoning_content.go:371`) → `Text()` (`:313`) |
| Tool schemas | `req.Tools()` (`request.go:486`) → `ToolSet.Tools()` (`tool_set.go:93`) → per tool: `Name()` (`tool.go:156`) + `Description()` (`:159`) + `Schema()` (`:172`) |

Ceiling as integer arithmetic: `(B + D - 1) / D`. Determinism is asserted directly: the table test computes the estimate twice per row and once against an independently re-constructed equal request, requiring identical figures.

### DD6 — The pre-hook inexactness is a stated semantic, not a footnote

`applyPreRequestHook` (`loop.go:326`) derives a new `ai.Request` **after** `buildLoopRequest` (`loop.go:304`), inside `Turn`, downstream of the turn boundary. Accounting therefore measures the **pre-hook** request on every path. Represented three ways, not merely noted:

1. **Type semantics.** `TokenSourceReported`'s doc comment defines the state as "the provider reported this figure **for the pre-hook request**" — never "for the bytes sent". `ContextPrompt.Accounting`'s comment carries the same sentence (DD1). The inexactness lives where a consumer must read it.
2. **A divergence-recording test.** Install a request-mutating `PreRequestHook`; the counting fixture captures the `CountTokens` request, `Provider.Requests()` (`fake_provider.go:157-161`) captures the sent request; the test asserts the accounting matches the pre-hook request and that the two captured requests **differ** — recording the divergence rather than asserting a false equality (proposal Risk 5's mandate).
3. **A spec obligation forwarded to `sdd-spec`.** The requirement MUST state accounting is over the pre-hook request and MUST NOT claim exactness even on the `Reported` path.

### DD7 — The exported `agenttest` fixture

New file `src/agenttest/counting_provider.go`, the `stubProviderWithTokenCounter` embedding shape (`agenttest/provider_test.go:133-142`) made exported and scriptable. Embedding is `*Provider` — `Stream` is on the pointer receiver (`fake_provider.go:74`), so value embedding would not satisfy `ai.ModelProvider` (proposal Risk 10).

```go
// CountingProvider embeds *Provider and advertises ai.TokenCounter by
// satisfying it, and by no other means (ai-minimum-capabilities §9).
type CountingProvider struct {
	*Provider
	mu        sync.Mutex
	result    ai.TokenCount
	err       error
	countReqs []ai.Request
}

// NewCountingProvider: a counter that reports result.
func NewCountingProvider(result ai.TokenCount, scripts ...Script) *CountingProvider
// NewFailingCountingProvider: an ADVERTISED counter that errs — the
// R-AMP-019 non-conformance fixture. Two constructors, so a
// (result, err) pair can never be half-stated.
func NewFailingCountingProvider(err error, scripts ...Script) *CountingProvider

func (p *CountingProvider) CountTokens(ctx context.Context, req ai.Request) (ai.TokenCount, error) // captures req under mu
func (p *CountingProvider) CountRequests() []ai.Request // fresh slice, the Requests() posture (fake_provider.go:157-161)

var _ ai.ModelProvider = (*CountingProvider)(nil)
var _ ai.TokenCounter = (*CountingProvider)(nil)
```

## Data Flow (one logical turn at the boundary)

```
Run outer loop (harness.go:479)
  ├─ transcript := transcriptFromHistory(hist)          harness.go:498
  ├─ h.ContextStrategy != nil ?                         ← inserted block
  │    ├─ buildLoopRequest(h.Turn, h.System, transcript)   loop.go:713   (pre-hook request)
  │    ├─ provider.(ai.TokenCounter) ── ok ── CountTokens ─ err/absent → Unavailable
  │    │                                └──── figure ────→ Reported
  │    │                     └── !ok ── estimateTokens ──→ Estimated
  │    └─ Strategy.Resolve(ContextPrompt{clone, budget, accounting}) → ContextVerdict{} (discarded; nothing else exists)
  ├─ retry bound + timing resolution                    harness.go:500-511
  └─ attempt loop (per attempt: buildLoopRequest → applyPreRequestHook → provider.Stream)   harness.go:530, loop.go:304/326/351
```

## File Changes

| File | Action | Description |
|---|---|---|
| `src/agent/context_strategy.go` | Create | `ContextStrategy`, `ContextPrompt`, `ContextVerdict`, `NoOpContextStrategy`, `ContextBudget`, guards (DD1, DD3) |
| `src/agent/token_accounting.go` | Create | `TokenSource`, `TokenAccounting`, `resolveTokenAccounting`, `estimateTokens`, both constants (DD4, DD5) |
| `src/agent/harness.go` | Modify | Two fields after `Failover` (`:91`); consultation block after `:498` (DD2) |
| `src/agenttest/counting_provider.go` | Create | `CountingProvider`, two constructors, capture, guards (DD7) |
| `src/agenttest/counting_provider_test.go` | Create | Fixture physics: report, error, capture order |
| `src/agent/context_strategy_test.go` | Create | External `package agent_test`: AG-17.1 scenarios, inertness pins, pre-hook divergence, discovery/non-conformance |
| `src/agent/token_accounting_test.go` | Create | Internal: estimate table + determinism (the AG-16 pure-converter allowance, archive `design.md:176`), resolver three-state table |
| `loop_test.go`, `loop_hook_test.go` | Modify | Substrate filters widened by exact filename, byte-in-sync, one entry per new `src/agent` file the filter's population rule covers (verified at apply against the filter's own predicate) |
| Spec deltas — FIVE | Delta | `agent-context-strategy` (new), `agent-run-driver` (`:342` check-half closed), `agent-history` (`:260` closed, byte-unchanged), `agent-v1-scope` (seams 5/6 back-annotation per `R-AGS-014`, `S-AGS-023` at `:145`), `agent-retry-failover` (`R-RTY-002` held), `agent-loop-skeleton` (`R-LSK-004`: no substrate release, filters widened) |
| `docs/architecture/milestones/0003-…` | Modify | AG-17 tick, R-18 seams 5/6 back-annotation, counter 17/24 |
| `doc.go`, `doc_contract_guard_test.go` (eight rows), `stream_check.go`, `event.go`, `event_descriptor.go`, `event_registry_test.go`, `compaction_events.go`, `cost_events.go`, `cost_usage.go`, `history.go`, `go.mod`/`go.sum`, `src/ai/**` | NOT TOUCHED | Proposal Decision 5; no new `EventKind`; no Layer 1 edit; no new dependency |

## Strict-TDD plan (RED-first; `cd backend/agent && make test`, evidence with `-count=1`)

| Test | Covers | RED-first bite |
|---|---|---|
| `TestHarness_ContextStrategy_ConsultedOncePerLogicalTurn` — recording strategy, N=2 clean turns | AG-17.1 cardinality | RED: no seam exists |
| `TestHarness_ContextStrategy_RetriedTurnConsultsOnce` — **the mandated anti-regression bite (proposal Risk 1)**: turn 1 scripted to fail retryably once then succeed (the `errorProvider` wrapper precedent cited by AG-16's design, archive `design.md:181`), turn 2 clean → 3 attempts, 2 logical turns; assert exactly **2** consultations | Decision 1b against the per-attempt reading | **Bite (a)**: move the consultation inside the attempt loop (`harness.go:530`) → test sees 3, FAILS |
| `TestHarness_ContextStrategy_PromptCarriesTranscriptAndBudget` — each consultation's `Transcript` element-equal to the messages of the sent request (`Provider.Requests()`), `Budget` echoes `ContextBudgetOf` | AG-17.1 inputs | RED: absent |
| `TestHarness_ContextStrategy_NilVsNoOpByteIdentical` — identical scripts on plain `agenttest.Provider`; run A nil field/absent budget, run B `NoOpContextStrategy{}`/present budget; **byte-identical** event streams and `hist.Entries()` read-backs; zero compaction kinds on either (`S-PRH-002`/`S-LSK-015`/`R-RUN-012` precedent) | AG-17.1 inertness, "no history mutated", "no compaction events" | **Bite (d)**: make the consultation block emit one `CompactionStarted` → byte-identity FAILS |
| `TestContextVerdict_HasNoFields` — `reflect.TypeOf(ContextVerdict{}).NumField() == 0` | unconstructibility, AG-18-inertness | RED: absent; guards AG-18's extension point |
| `TestResolveTokenAccounting_ThreeStates` (internal table) — `CountingProvider` → `Reported`+figure; `agenttest.Provider` → `Estimated`; `NewFailingCountingProvider` → `Unavailable`; advertised absent-count → `Unavailable` | AG-17.2 discovery + non-conformance (`R-AMP-018/019`) | **Bite (b)**: collapse to two states (error → estimate) → non-conformance row FAILS |
| `TestTokenAccounting_UnreadableWithoutSource` — figure obtainable only via `Tokens() (int64, TokenSource)` | distinguishability | **Bite (c)**: expose a bare `Tokens() int64` → compile failure IS the bite evidence |
| `TestEstimateTokens_TableDriven` (internal) — ASCII, CJK, empty transcript, tool-schema-bearing, system-bearing; plus determinism: computed twice + against a re-constructed equal request | DD5, proposal Risk 9 (clock unguarded) | RED: absent |
| `TestHarness_Accounting_PreHookDivergenceRecorded` — mutating `PreRequestHook`; accounting equals pre-hook request's count; captured count-request ≠ sent request | DD6, proposal Risk 5 | RED: absent |
| No-masquerade check | `cost_events.go`/`cost_usage.go` byte-unchanged; no path constructs `ai.TokenCount` from an estimate | Verified by `sdd-verify` reading the diff, not by assertion (proposal Risk 3) |

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary.

## Migration / Rollout

No migration. Single revert of the merge commit per the proposal's Rollback Plan: both new `src/agent` files, the fixture, and all new tests delete; `harness.go` returns to `:498` → `:530` directly; filters revert; the five deltas drop; doc 0003 un-ticks. No live consumer exists (`0003:110`), and the shipped default is inert by type, so no caller observation changes in either direction.

## Open Questions

None blocking. Two apply-time verifications flagged: (1) whether the substrate filters' population rule covers `_test.go` files (decides the exact entry set; both filters stay byte-in-sync either way); (2) exact retry-scripting shape — reuse the `errorProvider` wrapper precedent (archive `design.md:181`) or a `Script` composition, whichever the existing AG-15 tests already use at apply time.
