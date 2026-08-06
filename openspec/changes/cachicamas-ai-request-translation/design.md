# Design: AI-26 — Translate normalized requests to wire requests

## Technical Approach

One pure exported function in the package AI-25 creates: `Translate(req ai.Request) ([]byte, error)` in `backend/agent/src/ai/openaicompat/` — no client, no I/O, no mutation (NFR-ART-C). The body is **hand-assembled in fixed source order** by a small byte appender, never struct-marshalled, because a verified stdlib fact disqualifies `encoding/json` for this milestone: `json.Marshal` pipes every `Marshaler` result — including `json.RawMessage` — through `appendCompact` (`encoding/json/encode.go:483–488`, `indent.go:51`), eliding whitespace and HTML-escaping `<>&`, so schema bytes with distinctive whitespace cannot survive verbatim (R-ART-010, S-ART-035). Refusals reuse AI-19 via one shared helper (R-ART-015, S-ART-079); the feature policy is a production table walked by a hybrid inventory (R-ART-020/021); expectations are inline literals in a self-registering harness (NFR-ART-D). Stdlib only; `go.mod` untouched (NFR-ART-A); `package ai` read-only through public accessors — all verified present (NFR-ART-B).

## Architecture Decisions

| Decision | Choice | Rejected | Rationale |
|---|---|---|---|
| Wire-body representation | Unexported appender over `[]byte`: fields appended in one fixed source-code order (`model`, `messages`, `tools?`, `tool_choice?`, options, `stream`, `stream_options`, own-namespace extension members last); every string leaf encoded via `json.Marshal(string)` (deterministic escaping); schema and extension bytes spliced **raw** | Struct marshal + `json.RawMessage` (compaction/escaping breaks S-ART-035 verbatim — verified against Go source); post-marshal placeholder splicing (fragile, two mechanisms) | Field order = code order, so cross-run determinism is structural, not probabilistic (R-ART-003); verbatim splice is natural (R-ART-010); extension member-merge is natural (R-ART-019) |
| Map discipline | **Ranging over a map is forbidden anywhere on the Translate path**; key-lookup on a fixed table is permitted (deterministic). Every wire sequence is built by appending in accessor order — all neutral collections are already caller-ordered slices (verified: `Messages()`, `Content()`, `Segments()`, `ToolSet.Tools()`, `ProviderExtensions()`) | A static no-`map` guard over translation sources | Scoping a static guard to "translation files" is a fragile file list; the leak class is instead caught by checked-in expectations plus the staged map-flatten mutations of S-ART-011/S-ART-040, recorded across repeated independent invocations |
| Determinism proof | Expectation literals are fixed in source; every `go test` invocation (RED runs, slice-close runs, CI, review re-runs) is an independent process with an independently seeded map hasher — the accumulated invocations are the cross-run proof (S-ART-009). The same-process double-translate lives in `translation_test.go`, comment-labelled **pin, not proof** (S-ART-010/012) | Presenting double-translate as proof | Go re-randomizes map-range start per range-statement execution; a same-process check passes by luck (R-ART-003) |
| Relation to AI-25's `request.go` | `Translate` produces body bytes only. AI-25's unexported `newRequest` (join + Bearer) is untouched; wiring bytes into it is AI-28's work. New file `translation.go`; zero edits to AI-25 files | Growing `request.go` | AI-25 slice A is landing concurrently; file-level separation removes the collision. Plain-noun convention preserved |
| Refusal machinery | One unexported helper in `policy.go`: `refuse(feature string) error` → `ai.PreStreamFailure(ai.FailureReport{Category: ai.FailureCategoryUnsupportedCapability, Cause: <error naming feature>})`. `errors.Is(err, ai.ErrUnsupportedCapability)` holds via `Failure.Is` (provider_failure.go:366); the feature name rides `Cause`, reachable via `Unwrap` (S-ART-053). AI-26.6 and the 26.8.2 walk both call it — one door | Per-site construction; any new sentinel | Uniform category across 26.6/26.8.2, zero new sentinels (S-ART-054/079). AI-25's `Violation` is a different failure class — construction fault vs capability failure, AI-03 § 10.4 — recorded in `doc.go` (NFR-ART-E, S-ART-087) |
| Inventory (26.8.1 `[decision]`) | **Hybrid.** (a) Runtime enumeration of the five exported closed vocabularies — `PartKinds()` (content_part.go:128), `ReasoningStates()` (reasoning_content.go:132), `ToolChoiceModes()` (tool_choice.go:116), `CacheRegions()` (cache_boundary.go:67), `Roles()` (role.go:96) — all verified. (b) The repo's AST-scan idiom (`content_part_registry_test.go`, `validation_registry_internal_test.go`): parse `package ai`'s own sources (`os.ReadDir("..")`, non-test `.go`, `go/parser`) for exported `With*` funcs returning `RequestOption` — exactly **ten** verified (`WithModel`, `WithMessages`, `WithSystemInstruction`, `WithMaxOutputTokens`, `WithTemperature`, `WithTopP`, `WithStopSequences`, `WithTools`, `WithToolChoice`, `WithProviderExtension`). Three sources cross-checked pairwise: AST-declared names, the production policy table, and a per-feature witness table in the test file (closures building a request expressing the feature) — any disagreement fails, naming the member | Reflection (R-ART-020 forbids; sealed design already provides enumerators); trusting `Roles()` et al. as sole ground truth without noting why | Ground truth is the neutral package's own declarations (S-ART-072): the enumerators are themselves kept honest by `ai`'s own AST guards, and the option surface is parsed from `ai` source directly. **Bite**: an eleventh `With*` appears in the AST list with no policy entry → fail naming it (S-ART-071); a sixth vocabulary member appears in its enumerator with no entry → fail naming it (S-ART-070) |
| Policy table | Production `policy.go`: a **slice** of entries (ai's "a Layer 1 registry is never a map" precedent), each `{feature, disposition ∈ {translate, drop, refuse}, reason}`. Translation consults it (refusals fire from the production path); the walk enumerates it (R-ART-021). The extension surface is **two** inventory features — own-namespace → translate, foreign-namespace → drop — so every entry keeps exactly one disposition (S-ART-074). Cache regions → drop (AI-11.3 advisory, reason recorded); reasoning part/states → refuse; everything else → translate | Test-only table (refusals must fire in production); one composite extension entry (breaks one-disposition totality) | Dropping is always a decision with a recorded reason and a twin proof (S-ART-076); the walk is inventory-driven, never a parallel list (S-ART-078) |
| Expectation harness | Per-case struct `{name, build func() ai.Request, want string}` (raw-string inline literal); each node's table self-registers into a package-level `expectationCases` registry (AI-27's `sweepTranscripts` precedent). Total walks run over the registry: byte-exact compare per case, **positive `include_usage` assertion on every case** (S-ART-059/060/061), determinism sweep (S-ART-009). Refusal cases use a parallel `refusalCases` registry (no bytes). Gated-claim expectations slot in as new table entries once cited — the harness itself is claim-independent | Golden files / `-update` flag (NFR-ART-D forbids; blessing drift trivially) | One diff hunk per case+bytes under the chained budget; later nodes covered without editing the harness |
| Usage opt-in placement | `stream: true` + `stream_options.include_usage: true` are emitted by the **skeleton** (slice 1), so no later slice rewrites every prior expectation literal; AI-26.7 owns the total positive assertion and its mutation red (R-ART-017; AI-24 §§ 8, 13.1) | Adding the field at 26.7 (would churn every earlier expectation byte) | The obligation is assigned to 26.7 by name; 26.7 discharges the *assertion*, the bytes exist from birth |
| Credential scan | One test: `os.ReadDir(".")`, every `*_test.go` read as raw bytes, matched against credential-shaped sentinel classes only (bearer-token shapes, `sk-`-style key prefixes). Patterns are **assembled by concatenation at runtime** so the scan's own source never matches itself. Failure names file + sentinel class (S-ART-017). Hosts and model identifiers unmatched (S-ART-016) | `testdata/` walk (no tree exists); entropy heuristics (noise trains reviewers to ignore the scan) | Whole-surface subject catches setup credentials too (S-ART-015); new test files are covered without edits (S-ART-014) |
| Citation gating | All four claims are researched **once, during slice 1**, and land as a provenance section in `doc.go` (title + locator per claim) plus a citing comment beside each encoding test. Reason: the spec's earlier-of-charter-and-dependency rule makes claims 1 and 2 block AI-26.2 (slice 2) and claims 3 and 4 block AI-26.4 (slice 3) — deferring per-node buys nothing. An unobtainable citation halts the chain after slice 1, recorded, never waived (R-ART-001, S-ART-004); a contradiction escalates (S-ART-005) | Per-node lazy citation (the earlier-of rule collapses the laziness); in-repo restatement (S-ART-001 forbids) | No expectation encoding a gated claim is authored before its citation lands |

## Data Flow

    ai.Request ──► Translate(req)
                     ├─ policy consult (policy.go table)
                     │    └─ refuse(feature) ─► ai.PreStreamFailure + ErrUnsupportedCapability (no bytes)
                     └─ appender (body.go): fixed field order
                          model → messages(+parts) → tools → tool_choice → options
                          → stream/stream_options → own-namespace extension members (raw splice)
                          ──► wire bytes  (cache markers, foreign namespaces: dropped whole — twin-proven)

## File Changes

All paths under `/Users/braejan/workspace/witsaba/repositories/cachicamas-worktrees/ai-wave-4/backend/agent/src/ai/openaicompat/`. Non-test sources are single plain nouns (AI-25's convention); snake_case reserved for tests. Expectation tests are external (`package openaicompat_test`); the inventory/walk tests are internal (registry-idiom precedent: witness tables need the unexported policy table).

| File | Action | Slice | Owns |
|---|---|---|---|
| `translation.go` | Create | 1 | `Translate`, model identity, `stream`/`stream_options.include_usage` emission |
| `body.go` | Create | 1 | appender: ordered field append, `json.Marshal`-per-string leaves, raw splice |
| `doc.go` | Modify | 1 (grown 2,3,5,7,8) | wire-shape provenance section (four citations); NFR-ART-E taxonomy table; later: marker-drop reason (S-ART-024), no-merge decision (S-ART-034), dead output-limit branch (S-ART-064), reasoning hard-failure + AI-40 routing (S-ART-056) |
| `system.go` | Create | 2 | ordered system-segment rendering (claim 1 shape) |
| `tool.go` | Create | 3 | tool declarations in caller order, raw schema splice, `tool_choice` |
| `option.go` | Create | 4 | generation options with presence flags (verified: `has*` per option), own-namespace extension member-merge |
| `message.go` | Create | 5 (grown 6) | message/part rendering, order preservation, no-merge; slice 6 adds tool-role result messages, ID pass-through |
| `policy.go` | Create | 7 (grown 8) | `refuse` helper + reasoning entries; slice 8 completes the table for every inventory feature |
| `translation_test.go` | Create | 1 | harness + registries, skeleton expectation, determinism pin (labelled), model-differs case |
| `credential_scan_test.go` | Create | 1 | the scan (runtime-assembled patterns) |
| `system_segment_test.go`, `cache_marker_test.go` | Create | 2 | segment order/content; marked/unmarked twins across **every** cache region |
| `tool_test.go` | Create | 3 | caller order, schema-verbatim (distinctive whitespace), re-marshal mutation red |
| `option_test.go`, `extension_test.go` | Create | 4 | zero-vs-unset, explicit absence, opt-in total walk, foreign-namespace twins |
| `message_test.go` | Create | 5 | per-variant expectations, mixed parts, swap-differs, no-merge |
| `tool_result_test.go` | Create | 6 | tool-role shape, failed disposition, interleaved correlation, ID byte-identity |
| `reasoning_refusal_test.go` | Create | 7 | 3 states × positions refuse; sentinel-set-unchanged; error names feature |
| `feature_inventory_test.go`, `policy_walk_test.go` | Create | 8 | AST scan + three-source cross-check; exhaustive inventory-driven walk |

## Interfaces / Contracts

```go
func Translate(req ai.Request) ([]byte, error)  // pure; PreStreamFailure on refusal, nil bytes

// unexported, policy.go
type disposition uint8                          // dispTranslate | dispDrop | dispRefuse
type featurePolicy struct{ feature string; disp disposition; reason string }
func refuse(feature string) error               // the one refusal door (26.6 + 26.8.2)

// unexported, body.go — appender writes fields in declaration order only;
// appendRaw splices schema/extension bytes verbatim (never through encoding/json)
```

## Testing Strategy (RED-first; runner `make test` from `backend/agent/`, `go test -race -v ./...`)

| Node | Failing assertion before implementation |
|---|---|
| 26.1 | Skeleton request ≠ inline expectation (byte-exact, incl. `stream_options`); two models → bodies differ only there; extra-field mutation red (S-ART-008); scan red on a staged runtime-built sentinel, message names file+class, then removed; host/model literals stay green |
| 26.2 | Three-segment order + content vs expectation; reversed twins differ (S-ART-019); no-system case has no placeholder; marker twins byte-identical per region; marker-renders mutation red (S-ART-023) |
| 26.4 | Distinctive-whitespace schema verbatim vs expectation; decode/re-encode mutation red (S-ART-037); caller-order expectation; map-keyed-tools mutation staged and run repeatedly until ≥1 red recorded (S-ART-040) |
| 26.7 | Per-option field present with caller value; zero-vs-unset bodies differ (S-ART-058); omitted limit → field **absent** (S-ART-062); opt-in walk over full registry, omission mutation red (S-ART-061); own-namespace merge differs from twin, foreign twins byte-identical incl. several-at-once (S-ART-065–068) |
| 26.3 | Per-variant expectations; mixed-part message; adjacent-swap differs (S-ART-030); two/three consecutive same-role messages distinct (S-ART-031/032); merge mutation red (S-ART-033); part-kind coverage vs `PartKinds()` (S-ART-027) |
| 26.5 | One tool-role message per result, caller order; failed disposition carried (S-ART-044); interleaved 3-call correlation by identifier, position-correlation mutation red (S-ART-049); unusual-shape ID verbatim; regeneration mutation red (S-ART-047) |
| 26.6 | Refusal per state × position: `errors.Is(err, ai.ErrUnsupportedCapability)`, nil bytes; error text names reasoning; exported sentinel set unchanged (S-ART-054); drop-and-succeed mutation red (S-ART-055) |
| 26.8 | Walk fails on a staged policy-less vocabulary member and a staged `WithFake` constructor (each red recorded, then removed); every refuse-policy feature refuses naming itself; every drop-policy feature twin-proven with recorded reason; category uniform across 26.6/26.8.2 (S-ART-079); purity pin: request equals independent copy after Translate (S-ART-083) |

Staged-mutation reds follow AI-25's recorded-evidence pattern: staged in the finished implementation, red `make test` output recorded in the slice PR, reverted, suite green. `*(review)*` scenarios are discharged by recorded reviewer confirmations in `tasks.md`, never reported as red→green (NFR-ART-F).

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary. The inventory guard parses source statically in-process; production code performs no I/O at all.

## Migration / Rollout — chain slices

Additive, zero consumers (AI-28…AI-32 unstarted); rollback = revert slices in reverse order, `package ai` and `go.mod` untouched. Feature-branch chain on tracker `feat/2026-08-03-cachicamas-ai-layer1-wave-4`; slice 1 branches only after **AI-25 slice A merges**. Proposal order **confirmed**: 26.1 → 26.2 → 26.4 → 26.7 → 26.3 → 26.5 → 26.6 → 26.8 — skeleton first, then graph-independent siblings cheapest-first so harness/fixture/error conventions settle on small diffs before the largest node; 26.5/26.6 extend 26.3's rendering; 26.8's walk must cover everything prior. One refinement, not a reorder: the four citations land in slice 1 (see Citation gating) because the spec's earlier-of rule pulls claims 1/2 to slice 2 and 3/4 to slice 3 anyway. 26.3 fission trigger stands: 26.3a skeleton/order/no-merge, 26.3b part-variant expectations. Corrected total ≈ 4,050–8,600 lines against the 5,000 budget — chaining mandatory, no `size:exception`.

`Decision needed before apply: No` · `Chained PRs recommended: Yes` · `400-line budget risk: High`

## Open Questions

- [ ] The reserved extension namespace value is read from AI-25's **landed** artifact at apply time, never re-invented here (spec `R-ART-019`).
- [ ] If slice-1 research finds vendor documentation **contradicting** any of the four claims, the node halts and escalates (S-ART-005) — the chain does not proceed past slice 1 on claims 1/2.
- [ ] Whether the skeleton's single-text-message `content` renders as a JSON string or a one-element part array is settled by the same slice-1 citation task (Create chat completion reference), before the first expectation literal is authored.
