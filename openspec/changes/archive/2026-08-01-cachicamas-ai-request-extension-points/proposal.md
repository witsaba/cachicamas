# Proposal — request extension points

> **Change**: `cachicamas-ai-request-extension-points`
> **Milestone**: AI-12 — Add per-request options, the escape hatch, and rebuild
> **Nodes**: AI-12.1 `[leaf]` · AI-12.2 `[leaf]` · AI-12.3 `[leaf]` · AI-12.4 `[leaf]`
> **Status**: proposed · **Phase**: proposal
> **Project**: cachicamas (witsaba) · **Date**: 2026-08-01 · **Driver**: braejan
> **Branch**: `feat/ai-12-request-extension-points` (worktree `cachicamas-worktrees/ai-12`, planned against `07d2027`, **rebased onto finished Wave 1 head `1c4171e`**)
> **Scope**: `openspec/changes/cachicamas-ai-request-extension-points/`, **one new production file plus one new test file** under `backend/agent/src/ai/`, edits to `request.go` and `request_test.go`, and — **added by the Phase 0 re-verification gate** — edits to `backend/agent/src/agenttest/request_test.go`, whose round-trip walk must learn the new region or silently stop covering it (`design.md` § 7.2). No `go.mod` change, **no new dependency, no register amendment, no appended rule class**
> **Re-verified 2026-08-01**: all five commitments below survived verification against the landed AI-10 surface. `design.md` § 13 carries the resolved register.
> **Predecessor artifact**: `explore.md` (this change)
> **Depends on**: AI-10, AI-11 · **Blocks**: AI-24, AI-26.7, Layer 2's pre-request hook

---

## Intent

Make the request **derivable**. AI-10 landed a value you can build and read; a pre-request hook needs a value you can *modify*, and doc 0001 § 6 seam 1 says that moment — "the only point where the outgoing request still exists as data" — cannot be reached from anywhere else. Three things follow, and this milestone lands all three:

1. **Copy-on-write rebuild** (`V-REQ-29`) — derive a modified request; the original stays observably unmodified.
2. **Per-request option overrides** (`V-REQ-27`) — vary one call without rebuilding the caller's defaults.
3. **A typed-but-opaque namespaced pass-through** (`V-REQ-28`) — carry a provider-specific value the neutral vocabulary deliberately does not model.

Closing gap **G9**, and supplying the mechanism **G11**'s hook taxonomy stands on.

## The design principle this change encodes

Doc 0001 § 3.3's conclusion, and doc 0002's AI-12 charter note, state it identically:

> **The correct response to provider divergence is a typed pass-through, not a wider neutral vocabulary.** Every field added to the neutral model for one provider becomes a field every other adapter must ignore, and the model grows without bound.

It is the acceptance criterion in disguise: "a provider-specific value survives to its adapter **without any other adapter needing to know it exists**" is a clause no neutral field can satisfy. AI-10 `design.md` § 8.2 already assigned six rejected option candidates — `top_k`, frequency/presence penalties, `seed`, thinking budget, user/metadata identifiers — to this hatch. None may become a neutral option.

## Locked constraints (inherited, not proposed)

1. **The terms are the register's.** `V-REQ-27`, `V-REQ-28`, `V-REQ-29` already exist and already name AI-12. This change *implements* them and **never edits** `openspec/specs/ai-contract-vocabulary/spec.md`.
2. **The failure vocabulary is AI-04's.** Both new rules are `ErrEmpty`. No class is appended; `explore.md` § 8 walks the fit.
3. **One validation path.** Derive-time validation is the **same rule slice** as construction, not a second one (AI-12.2 item 2). AI-06's "one rule set, two callers".
4. **Zero requires.** The pass-through is designed so no encoder is ever needed to carry it. `go.mod` stays at zero until AI-24.
5. **Trap 2 binds.** The escape hatch is not a way to use a provider that has no adapter.
6. **Strict TDD**, `Test<Subject>_<Behavior>_<Expectation>`, banners citing the leaf ID, behavioral tests in `ai_test`.
7. **Evidence gate**: recorded green `make test` (`go test -race -v ./...`) in `backend/agent/`, plus clean `make lint`, before every commit.

## Capabilities

### New Capabilities

- `ai-request-extension-points`: rebuild, per-request option overrides, and the namespaced typed-opaque pass-through on the normalized request, plus the read-back determinism of those surfaces.

### Modified Capabilities

- None. `ai-model-request` is AI-10's canonical spec and is not yet archived; AI-12 adds a sibling capability rather than a delta against an unmerged one. Where AI-12 constrains a surface AI-10 owns (the option seal, the rule order, equality), it does so by **citing** `R-AMR-*` and adding its own requirement, never by restating AI-10's.

## The five choices this proposal commits to

AI-12 has no `[decision]` node, so there is no `decision.md`; every choice is made in `design.md` and answers to a scenario in `spec.md`.

1. **One option vocabulary, widened to every region.** `RequestOption` gains `WithModel` and `WithMessages`, so the draft is the single seat of all regions. `NewRequest` and `With` become the same operation over two seeds — parameters, or a frozen request. This is the only shape under which "no second, weaker validation path" is structural rather than a discipline. `explore.md` § 4 records the two rejected shapes.
2. **Rebuild is `func (r Request) With(opts ...RequestOption) (Request, error)`**, the signature AI-10 `design.md` § 12.2 recorded. On failure it returns the zero request; the receiver is never touched, because the receiver is a value and the draft is a fresh copy.
3. **The pass-through carries `[]byte` behind a sealed, constructor-free value type** — `ProviderExtension`, produced only by the request on read. `any` is rejected because it lets a vendor wire type cross the Layer 1 method boundary, which doc 0001 § 8's checklist forbids; a map is rejected because map iteration is exactly the nondeterminism AI-12.4 exists to exclude; an encoding-named alias is rejected because Layer 1 has zero requires and because naming an encoding invites parsing.
4. **The extension region is an ordered slice, last-wins per namespace, replacing in place.** Last-wins because a namespace is what "the same option" means here, and a hook that cannot revise an extension contradicts AI-12.1's totality. In place, because moving a replaced namespace to the end would make read-back order depend on revision history.
5. **Extensions participate in equality, structurally.** "Inert in equality" (AI-12.3 item 3) means *never interpreted and never special-cased per namespace* — not *excluded*. Excluding them would let a rebuild silently drop every extension and still pass AI-10.5's round-trip pin. `design.md` § 8 argues it; `spec.md` requires it.

## Scope

### In scope

| Artifact | Content |
| --- | --- |
| `explore.md` | Landed — the principle, the totality tension, the payload comparison, the two concurrency assumption tables |
| `proposal.md` | This file |
| `specs/ai-request-extension-points/spec.md` | `R-REX-001` … `R-REX-010`, RFC 2119, with Given/When/Then scenarios |
| `design.md` | Every Go shape, the widened rule order, the five dispositions, the AI-10/AI-11 re-verification anchors |
| `tasks.md` | Four phases, one per leaf, one task per doc 0002 test-list item, with the resuming agent's entry point and the workload forecast |
| `src/ai/request_extension.go` | **New.** `ProviderExtension`, `WithProviderExtension`, `Request.ProviderExtension`, `Request.ProviderExtensions`, the region's rules, redaction |
| `src/ai/request_extension_test.go` | **New.** AI-12.3 and AI-12.4's items, in `ai_test`, including the two fake translators |
| `src/ai/request.go` | **Modified.** `WithModel`, `WithMessages`, `Request.With`, the rule slice extracted to a draft-scoped method, `requestDraft` widened, `appliedNames` extended |
| `src/ai/request_test.go` | **Modified.** AI-12.1 and AI-12.2's items |

### Out of scope (explicit)

| Excluded | Owner | Why not here |
| --- | --- | --- |
| The pre-request hook itself | Layer 2 (`V-OUT-13`) | Layer 1 supplies rebuildability; the hook is not Layer 1's |
| Any concrete adapter or real provider namespace | AI-24 … AI-31 | Trap 2. AI-12.3's acceptance is proven with fake translators in `ai_test` |
| A registry of known namespaces, or namespace rules beyond emptiness | nobody | Would make Layer 1 know every provider — the coupling the hatch exists to avoid |
| **Unsetting** an option or removing an extension | reopened by AI-24 / AI-26.7 on demand | No consumer, and it would give the package a second spelling of absence |
| Wire-byte determinism | **AI-26.1 / AI-26.4** | Wire bytes do not exist yet. AI-12.4 guarantees only that the neutral surface is not the source |
| Serialization of the extension region | **AI-26.7** | Zero requires until AI-24; the hatch is designed to need no encoder |
| Promoting a namespace to a neutral generation option | a future milestone | Requires `V-REQ-26`'s admission test to pass; not a refactor |
| Cache-marker semantics, the breakpoint cap, the cascade | **AI-11** | AI-12 requires markers to be *reachable* by the rebuild; AI-11 owns what a marker is |

## Approach

1. **Widen, do not fork.** Every region reaches the draft through the one option type; every rule runs from the one slice; `With` and `NewRequest` differ only in how the draft is seeded.
2. **Take the rebuild first.** AI-12.1 is the walking skeleton: without it neither the override nor the hatch has a derive path to be proven on. AI-12.2 and AI-12.3 widen it in parallel; AI-12.4 closes over both.
3. **Prove totality by enumeration, not by assertion.** AI-12.1 item 2's test walks the region list and shows each is reachable, so a region added later without an option fails the item rather than passing silently.
4. **Keep opacity mechanical.** The package never parses, re-encodes, trims or normalizes an extension value; the only thing it decides about one is whether it is empty.
5. **Keep the redaction posture.** A namespace is caller data. `String`/`GoString` render the extension **count**, never a namespace or a value; positions use `extensions[i]`, an ordinal, exactly as `AtIndex`'s GoDoc requires.
6. **Land nothing without a citable case.** No namespace registry, no `Validate()` beyond construction, no builder, no second equality.

## Affected areas

| Area | Change | Risk |
| --- | --- | --- |
| `src/ai/request_extension.go`, its test | New files | Medium — the shape AI-24's first adapter reads |
| `src/ai/request.go` | `WithModel`, `WithMessages`, `With`, rule-slice extraction, widened draft | **High** — jointly owned with AI-10.3 this once; rebase conflicts expected and cheap |
| `src/ai/request_test.go` | AI-12.1 and AI-12.2 items appended | Low |
| `src/ai/validation.go` and both registry mirrors | **None** — no class appended | — |
| `src/ai/system_instruction.go`, `message.go`, `tool*.go`, `reasoning_content.go` | **None** — read-only | — |
| `openspec/specs/ai-contract-vocabulary/spec.md` | **None**. Nothing to report; all three terms exist | — |
| `backend/agent/go.mod`, `go.work` | **None** — zero requires preserved | — |
| doc 0002 | **None expected.** If AI-11's marker surface forces a request-level region, the discovered case is *appended* to AI-12.1's test list under the revert-and-record clause | — |

## Risks

| Risk | Likelihood | Impact | Mitigation |
| --- | --- | --- | --- |
| AI-10.3/AI-10.6 land shapes this plan assumed differently | **High** | Medium | `explore.md` § 6.1 states every assumption with file and symbol; apply re-verifies before its first test and rebases |
| AI-11 models markers as a request-level region | Medium | Low | Both branches recorded; the extra option is one constructor plus one assertion, appended never substituted |
| The hatch becomes a dumping ground for genuinely neutral things | Medium | **High** | `V-REQ-26`'s admission test is the gate; AI-10 `design.md` § 8.2 fixes the initial customer list |
| A second validation path appears on the derive side | Low | **High** | Structurally impossible under choice 1; asserted anyway — same class **and** same position from both doors, per rule |
| `WithModel` inside `NewRequest` surprises a reader | Medium | Low | Pinned by test and documented on the constructor |
| Extensions silently dropped by a rebuild | Medium | **High** | Choice 5 — equality covers the region; totality item enumerates it |
| The milestone busts the review budget | **Certain** | Medium | Forecast in `tasks.md`; `exception-ok` accepted up front; leaf boundary = commit boundary |

## Rollback plan

Purely additive: one new file, plus added symbols in `request.go`. No landed signature changes, no rule class appended, no register edit. Rollback is `git revert` of the commit range; the only consumer of anything introduced here is this change's own tests.

The asymmetry worth stating: **widening `RequestOption` is cheap to keep and expensive to withdraw** once a caller ships against `WithMessages`. That is the argument for deciding the totality shape now, while no adapter exists, rather than after AI-24.

## Dependencies

- **AI-10** — consumed and extended; `request.go` is jointly owned with AI-10.3 during this wave.
- **AI-11** — consumed for the marker region only, and only as a reachability requirement.
- **AI-04 … AI-09** — consumed, none modified.
- **No new Go dependency, no ADR required.** `slices`, `bytes`, `strconv`, `strings` are standard library.

## Success criteria

- [ ] Every test-list item of AI-12.1 … AI-12.4 taken red → green → refactored, **in order**, both outputs recorded in `tasks.md`.
- [ ] `make test` green with `-race`, `make lint` clean, both import guards passing, `go.mod` at zero requires.
- [ ] A derived request differs where asked and the original is unchanged under deep comparison, before and after.
- [ ] The rebuild is total: every region — model, system segments, messages, tools, tool choice, generation options, markers — is reachable, proven by enumeration.
- [ ] A request carrying reasoning round-trip tokens rebuilds with every token byte-identical.
- [ ] A per-request override wins over the construction-time value; an absent override falls through.
- [ ] Derive-time validation reports the **same class and the same position** as construction for every rule.
- [ ] A namespaced opaque value survives byte-exact to the translator claiming its namespace, and a foreign translator's output is byte-identical with and without it.
- [ ] Two requests differing only in a third provider's namespace validate identically.
- [ ] Reading the option set and the extension set twice yields identical order and content.
- [ ] `openspec/specs/ai-contract-vocabulary/spec.md` is untouched, and `validation.go` carries no new class.

## Notes for the following phases

- **`spec.md`** — requirement IDs `R-REX-0NN`, scenario IDs `S-REX-0NN`; the system under test is runtime behavior; every scenario verifiable by one test.
- **`design.md`** — owns every Go spelling, the widened rule order, the five dispositions with their reasoning, and the AI-10/AI-11 re-verification anchors restated as a checklist for the apply agent.
- **`tasks.md`** — four phases, one per leaf, one task per doc 0002 test-list item **reproduced faithfully and never pruned**; an "Entry point for the resuming agent" section at the top; the Review Workload Forecast.
