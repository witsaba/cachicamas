# Proposal — AI-40: Publish the Layer 2 readiness contract

> **Change**: `cachicamas-ai-layer2-handoff`
> **Milestone**: AI-40 (doc 0002, lines 2369–2410) · **Wave**: 6 — Hand off, **last** milestone · Layer 1's exit
> **Artifact store**: hybrid (file + Engram) · **Explore**: Engram `#2799`
> **Depends on**: AI-38 (landed on `main`, `5bc2da4e`); AI-39 optional (merged `b062be74`, PR #142)
> **Blocks**: doc 0003 `AG-03` onward — doc 0003 line 4 names AI-40 the **normative entry gate**
> **Execution mode**: auto — every fork below is resolved with a position marked
> *recommended, pending maintainer ratification*, never deferred back as a blocking question.

## Intent

Layer 1 is code-complete (41 of 42 milestones SDD-closed) but **not handed off**. Doc 0003 lines 70–78
state that "the normative gate is AI-40. Layer 1's surface freezes there, and every code-bearing
milestone in this document (AG-03 onward) depends on it." Today a Layer 2 author has no answer to
three questions, and each gap is measurable:

- **What may I rely on?** No frozen-surface declaration exists anywhere. `src/ai/doc.go` documents
  contracts milestone by milestone but never says which of them are v1-stable versus experimental.
- **What does the adapter actually support?** The nine-entry capability record is real and
  mechanically asserted (`expectedOpenRouterRecord()`, `.../openrouter/conformance/capability_record_test.go`)
  but published nowhere a consumer reads. Its `CAP-O-01` *absent* row is a **struck verdict** with a
  reopen trigger, which a consumer must know before designing around reasoning.
- **How do I write a Layer 2 test?** `Glob **/example_test.go` over `backend/agent` returns **empty** —
  the module ships zero runnable examples. Documentation that is not compiled is documentation that rots.

Two publication duties were also *delegated* to this node and are currently unpublished: item 6's
wire clause (not exercisable in v1) and the Layer-2-strips-reasoning duty for OpenRouter's
`reasoning_details`. Both are recorded only inside AI-29's `decision.md` and doc 0002's own amendments.

Finally, doc 0002's completion checklist is **stale on disk**: items 11, 12, 14, 15, 16 and 17 still read
`[ ]` at lines 2428–2434 even though AI-33, AI-34, AI-23/AI-38.1, AI-38, AI-36/AI-23.7 and AI-39.1 all
closed and the Status line at line 3 already counts them. Layer 1 cannot declare itself complete against
a checklist that contradicts its own status line.

**Success**: a Layer 2 author runs `go doc` on `src/ai`, sees what is frozen, sees the nine-row matrix
with its reopen caveat, sees four compiled examples, and can copy a working consumer test — while a
single durable `decision.md` walks all eighteen checklist items to their closing nodes, and doc 0002's
boxes finally match the evidence.

## Scope

### In scope — the three charter nodes

| Node | Deliverable | Concrete artifacts |
|---|---|---|
| **AI-40.1** — consumer proof `[leaf]` | A tiny external-package test — the future Layer 2 in miniature — constructs a request, invokes the AI-21 fake, drains events, handles a scripted error **and** a cancellation, importing only stdlib + `src/ai` + `src/agenttest` | New sibling package `backend/agent/src/handoff/` (`doc.go` + `handoff_test.go`, `package handoff_test`) — see **D2** |
| **AI-40.2** — capability matrix and examples `[leaf]` | Four runnable examples (request construction, streaming, tool-call reconstruction, error inspection) that compile and run under the normal test run; the AI-38.2 nine-row matrix published in package documentation; **both inherited publication duties** stated | `backend/agent/src/ai/example_test.go` (`package ai_test`) — see **D3**; matrix + duties in `backend/agent/src/ai/doc.go` — see **D4** |
| **AI-40.3** — compatibility statement `[decision]` | v1 surface enumerated and declared frozen, experimental marked as such; the eighteen-item completion checklist walked with each row citing its closing node; the abandoned-consumer-who-never-cancels posture restated as documented contract | `openspec/changes/cachicamas-ai-layer2-handoff/decision.md` + a short pointer paragraph in `src/ai/doc.go` — see **D1** |

### In scope — inherited publication duties on AI-40.2 (both mandatory)

1. **Item 6's wire clause, published as not exercisable in v1.** Cause named: `AI-26.6` landed as a
   refusal and `AI-29.2` is struck by AI-29, so there is no reasoning on this wire to round-trip and no
   v1 node can close the wire half. The **stream** half — closed by **AI-17** (`R-ARE-009`/`R-ARE-010`)
   and recorded on the **G12(b)** spine row — is unaffected and **MUST NOT** be reopened, restated as
   open, or implied to be reopened by this publication. The distinction language MUST be lifted from
   doc 0002 line 2446 and the AI-40.2 amendment at line 2402, not paraphrased.
2. **Layer 2 strips reasoning on the wire.** Delegated from Wave 4 close (doc 0002 line 14): Layer 2
   MUST strip OpenRouter's `reasoning_details` — a recorded absence (AI-29), not an oversight. This duty
   sits **beside** duty 1 on this node, not in place of it.

### In scope — doc 0002 checkbox reconciliation

- Tick items **11, 12, 14, 15, 16, 17** *individually, each with its closing evidence cited* — never as a
  blanket sweep. Item 18 (`AI-40.1`, `AI-40.3`) ticks at this change's own close.
- **Item 6 stays `[ ]` by design.** Doc 0002 line 2446 says so explicitly. AI-40 publishes the
  not-exercisable-in-v1 restatement; it does not tick the box and does not strike AI-17's closure.
- The `> Amended 2026-08-10 — AI-40 close` blockquote follows the AI-33…AI-39 pattern exactly.

### Out of scope

- **Charter out-of-scope, verbatim**: implementing anything in Layer 2.
- Modifying the conformance suite (`src/agenttest/conformance_*.go`) or any adapter
  (`src/ai/openaicompat/**`) behavior. The matrix is **transcribed from** the committed expectation,
  never regenerated, re-decided, or amended here.
- Reopening `CAP-O-01`. A generated `Satisfied` for reasoning remains a hard stop needing an ADR
  (`R-OR-05`, `R-ACR-004`); this change publishes the outcome and its trigger, it does not relitigate it.
- Re-deciding items 11/12/14/15/16/17. AI-40.3 **reports** their status as of the milestones that closed
  them; it does not re-verify their evidence.
- Any file move or restructuring. `src/agenttest` MUST remain a direct sibling of `src/ai`
  (`provider_signature_guard_test.go` resolves `../ai/provider.go` via `runtime.Caller(0)`).
- The `R-CNF-020`…`R-CNF-026` identifier gap (AI-36's recorded, separately tracked repository defect).
- Opening a doc 0003 milestone. AI-40 unblocks `AG-03`; it does not start it.

## Capabilities

### New capabilities

- `ai-layer2-handoff`: the milestone contract itself — the external-package consumer proof, the four
  compiled examples, the published nine-row capability matrix with its reopen caveat, the two inherited
  publication duties, the frozen-v1-surface declaration, the eighteen-item checklist walk, and the
  never-cancelled abandoned-consumer posture as documented contract.

### Modified capabilities

- `ai-provider-conformance-suite`: acceptance item 10 (`spec.md:343`) still reads "exactly **eight**
  entries", stale since AI-35's nine-entry amendment and recorded-but-unfixed by AI-38. See **D6**.

### Conditional — `sdd-spec` MUST NOT assume this

- `ai-first-provider-decision` (`spec.md:125`, "eight entries") is left alone: it describes the AI-24-era
  decision artifact as of its own date, and correcting it would rewrite history rather than fix drift.

## Approach

Approach 3 from exploration — **`decision.md` as the authoritative cited artifact, plus a bounded
in-code echo in `src/ai/doc.go`**. It follows two conventions this repo already established
independently (`[decision]`-typed nodes produce a `decision.md`, cf. AI-29's
`cachicamas-ai-provider-reasoning-stream/decision.md` § 11 and `src/ai/openaicompat/decision.md`;
`src/ai/doc.go` grows one guarded paragraph per milestone, already true for AI-14 and AI-20) rather than
inventing a third pattern or dropping either one. No evidence against it was found during proposal.

| Step | Work | Node |
|---|---|---|
| A | New `src/handoff/` package: `doc.go` + the consumer-proof test (request → fake → drain → scripted error → cancellation) | 40.1 |
| B | Four `Example*` functions in `src/ai/example_test.go`; first RED step compiles the file to prove the external-test-package import of `agenttest` creates no cycle | 40.2 |
| C | `src/ai/doc.go` grows two sections: the nine-row capability matrix (citing its generating test) and the frozen-v1-surface pointer, carrying both inherited publication duties | 40.2 / 40.3 |
| D | Doc-drift guard so the published matrix cannot rot silently (**D4**) | 40.2 |
| E | `decision.md`: frozen-surface enumeration, eighteen-item checklist walk against the spine, never-cancelled posture | 40.3 |
| F | Doc 0002 checkbox reconciliation + AI-40 close amendment; `ai-provider-conformance-suite` item-10 fix (**D6**) | close |

## Decision forks — resolved

| # | Fork | Position (*recommended, pending maintainer ratification*) |
|---|---|---|
| D1 | Artifact shape for AI-40.3 | **Both** — `decision.md` for the cited, structured eighteen-item walk (the format the spine template needs and a GoDoc comment cannot carry), plus a short pointer paragraph in `src/ai/doc.go` so a consumer running `go doc` learns a freeze happened without digging into `openspec/changes/`. `doc.go` carries a summary and a path, never a duplicate of the walk. |
| D2 | Where the AI-40.1 consumer proof lives | **A new sibling test-only package `backend/agent/src/handoff/`** (one 5-line `doc.go` + one `handoff_test.go` in `package handoff_test`). The charter's words are "the future Layer 2 in miniature": a package that is neither `ai` nor `agenttest` proves a *third-party* consumer compiles, whereas `agenttest_test` would be the testing library testing itself. Costs exactly +1 production / +1 test file and moves nothing, so the `src/agenttest` sibling constraint is untouched. **Zero vendor imports needs no new guard**: `import_boundary_test.go`'s `layer1Pattern = modulePath + "/..."` with `go list -deps -test` already sweeps the whole module recursively, so a new package anywhere under `backend/agent/` is covered on the next run — which is exactly the charter's "proven by the AI-00.3 guard mechanism rather than by inspection". Rejected: `agenttest_test` (self-referential); a brand-new top-level directory outside `src/` (trips the sibling constraint's spirit and the module layout). |
| D3 | Where the four examples live | **All four in `backend/agent/src/ai/example_test.go` (`package ai_test`), importing `src/agenttest` for the streaming and tool-call examples.** Go's external-test-package rule permits `ai_test` to import a package that imports `ai`, so this creates no cycle — and it puts every example on the `go doc`/pkg.go.dev surface of `src/ai`, the package a Layer 2 author actually reads. `sdd-design`'s first obligation is to prove the no-cycle claim by compiling, before any example body is written. Fallback if it does not hold: split (b) and (c) into `src/agenttest/example_test.go`, accepting the split godoc surface. |
| D4 | Rot-protection for the published matrix | **A doc-drift guard test that parses the nine rows out of `src/ai/doc.go` and compares them entry-by-entry to the committed expectation**, placed in `.../openrouter/conformance/` and resolving `doc.go` by relative path — the established mechanism (`provider_signature_guard_test.go` reaches `../ai/provider.go` the same way), and the only place `expectedOpenRouterRecord()` is reachable, since it is unexported in a `_test` file. **Boundary note for the maintainer**: this adds a new *read-only* test file inside the conformance directory. It changes no suite behavior, no adapter, and no expectation — but it is adjacent to the "do not modify the conformance suite" line and is flagged here rather than assumed. If ratified against, fall back to a manual `doc.go` comment citing `TestOpenRouterAdapter_FullConformance` as generator and accept that the table can drift silently. |
| D5 | Checkbox update policy | **Per-item, evidence-cited, never blanket.** Each of items 11/12/14/15/16/17 is ticked with a one-line citation of the closing milestone's own doc 0002 amendment blockquote; item 18 ticks at this close; **item 6 stays open by design** with the not-exercisable-in-v1 restatement, and AI-17's stream-half closure is explicitly named as unaffected in the same sentence. |
| D6 | The stale "eight entries" line | **Fix it, append-only.** `ai-provider-conformance-suite/spec.md:343` contradicts the nine-row matrix AI-40 is about to publish as Layer 1's exit contract; shipping the freeze on top of a canonical spec that says "eight" would publish a contradiction. One-line correction under a dated amendment note, delta spec in this change folder, promoted at archive. `ai-first-provider-decision` is deliberately not touched. |
| D7 | PR boundary | **One PR**, per the session's locked delivery constraint (review budget 1000 lines, `size:exception` pre-accepted with automatic raising authorized for AI-40 only). The three nodes are not independently shippable: AI-40.3 `Depends on: AI-40.1, AI-40.2` in the charter, and its frozen-surface enumeration cites the examples and the matrix by name. If `sdd-tasks` forecasts an overrun, slice along the work units in *Approach*, not along a code/docs seam. |

## Affected areas

| Area | Impact | Description |
|---|---|---|
| `backend/agent/src/handoff/doc.go` | New | Package doc: the Layer-2-in-miniature consumer proof (D2) |
| `backend/agent/src/handoff/handoff_test.go` | New | AI-40.1: request → `agenttest.NewProvider` → drain → scripted error → cancellation |
| `backend/agent/src/ai/example_test.go` | New | AI-40.2: four `Example*` functions (D3) |
| `backend/agent/src/ai/doc.go` | Modified | Nine-row capability matrix + reopen caveat; both inherited publication duties; frozen-v1-surface paragraph pointing at `decision.md` |
| `backend/agent/src/agenttest/doc.go` | Modified (minor) | One pointer sentence once the examples and consumer proof land |
| `backend/agent/src/ai/openaicompat/openrouter/conformance/` (new file) | New | Doc-drift guard for the published matrix (D4) |
| `.../openrouter/conformance/capability_record_test.go`, `run_for_test.go` | **Read-only** | Source of the nine rows; not modified |
| `openspec/changes/cachicamas-ai-layer2-handoff/decision.md` | New | AI-40.3's compatibility statement and eighteen-item walk |
| `openspec/changes/cachicamas-ai-layer2-handoff/specs/` | New | `ai-layer2-handoff` full spec; `ai-provider-conformance-suite` delta (D6) |
| `openspec/specs/ai-provider-conformance-suite/spec.md` | Modified at archive | Acceptance item 10: eight → nine entries |
| `docs/architecture/milestones/0002-...md` | Modified | Checklist items 11/12/14/15/16/17/18; AI-40 close amendment; Status line 41 → **42 of 42** |

## Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| **Checkbox staleness** — the walk is written against a stale snapshot | **High (already observed)** | Items 11/12/14/15/16/17 are confirmed `[ ]` on disk at lines 2428–2434 while the Status line counts them closed. `sdd-design` and `sdd-apply` MUST each re-read lines 2414–2436 fresh, immediately before writing, and cite the on-disk state they saw |
| **`CAP-O-01` reopening hazard** — the "reasoning absent" row is a struck verdict that becomes wrong the day a reasoning-capable model becomes the OpenRouter default | Med | The published row MUST cite `TestOpenRouterAdapter_FullConformance` as its generator and name the reopen trigger (`R-OR-05`, `R-ACR-004`, ADR required) inline; never phrased as a permanent property. D4's drift guard makes the doc rot-detect itself |
| **`src/agenttest` sibling constraint** — any file-move bundled into this change breaks `provider_signature_guard_test.go`'s `runtime.Caller(0)` resolution of `../ai/provider.go` | Low | No move is in scope. D2 adds a new package instead of relocating one. `sdd-apply` MUST reject any refactor that relocates `src/agenttest` or `src/ai` |
| **Item 6 wording precision** — the wire-half restatement reads as reopening G12(b)'s stream half | **High** | Lift the distinction verbatim from doc 0002 lines 2402 and 2446; name AI-17 / `R-ARE-009` / `R-ARE-010` as unaffected in the same sentence; `sdd-verify` treats a missing "not reopened" clause as CRITICAL |
| Example import cycle — `ai_test` importing `agenttest` (D3) | Low | First RED step is a compile-only proof; documented fallback splits examples across two packages |
| D4's guard is judged to breach the "do not modify the conformance suite" boundary | Med | Flagged as a maintainer-visible fork, not assumed; documented fallback is a citing comment with no guard |
| 1000-line review budget exceeded | Med | `sdd-tasks` owns the binding forecast; `size:exception` pre-accepted with automatic raising for AI-40 only |

## Rollback

Confined to worktree `cachicamas-worktrees/feat-ai-40-layer2-handoff`, branch `feat/ai-40-layer2-handoff`
based at `b062be74`. Every deliverable is additive: three new files, two doc-comment additions, one
checklist reconciliation. Reverting the branch restores `main` exactly, and nothing on `main` depends on
anything this change creates — doc 0003's `AG-03` has not started. Delta specs never touch
`openspec/specs/**` before archive, so an abandoned change leaves the promoted spec tree exactly as AI-39
left it, including the stale item-10 line. Steps are independently revertible: D (drift guard), F
(checkbox reconciliation) and the D6 spec fix each revert alone; A, B and C revert as a unit only because
E's `decision.md` cites them by name.

## Dependencies

- **AI-38** — landed on `main` (`5bc2da4e`, PR #140). Source of the nine-entry capability record.
- **AI-39** — optional per charter; merged (`b062be74`, PR #142). Its `[[FILE-COUNTS]]` reconciliation was
  discharged at its own close (95 production / 168 test `.go` files under `backend/agent`).
- **AI-21** (`agenttest.Provider`, `NewProvider`, `Script`/`Step`/`Emit`/`Hold`, `ErrScriptsExhausted`,
  `Requests()`) and **AI-22.3** (`ai.CheckStream`) — reused, never modified.
- **AI-00.3** `import_boundary_test.go` — reused as-is; no guard change is needed or permitted.
- **AI-29's `decision.md` § 11** — the primary source AI-40.2's publication duty cites by identifier.
- **Test runner**: `cd backend/agent && make test` (`go test -race -v ./...`). Engram `#2055` names
  `backend/database_administrator/Makefile` as the backend runner — that observation is **subsystem-scoped
  to `database_administrator` and does not apply here**; every AI-33…AI-39 close cites `make test` from
  `backend/agent`, and there is no root Makefile. Strict TDD is active.

## Success criteria — traceability to the charter

**Charter acceptance, verbatim**: *"A tiny external-package test constructs a request, invokes a fake
provider, drains events, handles cancellation and a scripted error, and compiles with zero vendor
imports."*

| Node / duty | Criterion |
|---|---|
| AI-40.1 | [ ] An external-package test constructs a request, invokes the AI-21 fake, drains events, handles a scripted error **and** a cancellation, and compiles with zero vendor imports — proven by the AI-00.3 guard mechanism rather than by inspection |
| AI-40.2 (1) | [ ] Four runnable examples cover request construction, streaming, tool-call reconstruction and error inspection, and are compiled **and run** by `make test`, so documentation cannot rot silently |
| AI-40.2 (2) | [ ] The AI-38.2 nine-row matrix is published in package documentation, entry-for-entry identical to the committed expectation, citing its generating test and naming the `CAP-O-01` reopen trigger |
| AI-40.2 duty 1 | [ ] Item 6's wire clause is published as not exercisable in v1, naming AI-26.6's refusal and AI-29.2's striking — with AI-17's stream-half closure (`R-ARE-009`/`R-ARE-010`) explicitly stated as unaffected and not reopened |
| AI-40.2 duty 2 | [ ] Layer 2's obligation to strip OpenRouter `reasoning_details` on the wire is published beside duty 1, attributed to AI-29's recorded absence |
| AI-40.3 (1) | [ ] The v1 surface is enumerated and declared frozen; anything experimental is marked as such; the statement names exactly what Layer 2 may rely on |
| AI-40.3 (2) | [ ] All eighteen completion-checklist items are walked in order, each row citing its closing node(s) per the traceability spine (doc 0002 line 2514) — including item 6's struck `AI-29.2` and item 18's `AI-40.1`/`AI-40.3` |
| AI-40.3 (3) | [ ] The abandoned-consumer-who-never-cancels posture is restated as documented contract: the caller owns the context, the producer blocks until the context ends, abandoning without cancelling is a contract violation — with abandoned-then-cancelled coverage (AI-23.5, AI-33.3) named and the never-cancelled case marked untestable-to-termination |
| Reconciliation | [ ] Doc 0002 items 11/12/14/15/16/17 ticked with per-item evidence; item 18 ticked; **item 6 still `[ ]`**; Status line reads 42 of 42 |
| Reconciliation | [ ] `ai-provider-conformance-suite` acceptance item 10 reads nine entries |
| Gates | [ ] `make test` green under `-race`; `make lint` 0 issues; `make build` exit 0; `backend/agent/go.mod` and `go.sum` byte-identical to base |

## Proposal question round

Execution mode is `auto` and this executor cannot prompt directly, so the questions that would have been
asked are recorded here. Each fork above already carries a resolved position; these are the four where a
maintainer answer would change the artifact rather than confirm it.

1. **D4 boundary** — the matrix drift guard adds a read-only test file inside
   `.../openrouter/conformance/`. Does that count as "modifying the conformance suite" (out of scope), or
   is a non-behavioral guard acceptable there? Assumption taken: acceptable, with a documented fallback.
2. **D2 shape** — is a new `src/handoff/` package (+1 production file for a 5-line `doc.go`) the right
   home for the consumer proof, or should it live in `agenttest_test` at zero file cost? Assumption
   taken: the new package, because "the future Layer 2 in miniature" argues for a third-party consumer.
3. **Frozen-surface granularity** — should AI-40.3 enumerate the v1 surface *by exported identifier*, or
   *by capability/behavior* per the repo's own no-Go-identifier rule for `decision.md` artifacts?
   Assumption taken: **by capability**, matching AI-29's `decision.md` convention, with `doc.go` carrying
   the identifier-level detail where identifiers are native.
4. **D6 blast radius** — is correcting one stale canonical spec line acceptable inside AI-40, or should it
   be deferred to a separate housekeeping change? Assumption taken: fix it here, since publishing a
   nine-row matrix on top of a spec that says "eight" ships a contradiction at Layer 1's exit.
