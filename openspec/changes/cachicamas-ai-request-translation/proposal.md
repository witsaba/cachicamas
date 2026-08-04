# Proposal: AI-26 — Translate normalized requests to wire requests

> **Change**: `cachicamas-ai-request-translation` · **Milestone**: AI-26 (doc 0002, Wave 4)
> **Nodes**: 8 — AI-26.1 … AI-26.7 `[leaf]`, AI-26.8 (`26.8.1 [decision]`, `26.8.2 [leaf]`)
> **Phase**: proposal · **Date**: 2026-08-03 · **Project**: cachicamas (witsaba)
> **Depends on**: AI-10, AI-11, AI-12 (shipped) · AI-25 (in flight, same wave) · **Blocks**: AI-28 … AI-32, AI-38
> **Input**: Engram `sdd/cachicamas-ai-request-translation/explore`, `sdd/.../reasoning-disposition`, `sdd/cachicamas-ai-layer-1/wave-4-preflight`; AI-24's merged `openspec/changes/cachicamas-ai-first-provider-decision/decision.md` §§ 6, 7, 8, 12, 13
> **Revision**: 2026-08-03, after coordinator rulings — typed error settled, doc 0002 amendments found already merged, fixture convention reconciled with AI-27

---

## Intent

AI-25 builds a configured client that *can* reach the vendor. Nothing yet turns an `ai.Request` into
a body it can send. AI-26 is the single place where Layer 1's neutral vocabulary meets one vendor's
schema — and therefore the single place where a caller-expressible feature can disappear without
anyone noticing.

Three properties make this more than marshalling:

1. **Silent loss is the default failure mode.** A translator that meets an input it has no wire field
   for will, unless designed otherwise, emit a body missing it and succeed. The caller gets a
   plausible answer computed from less than it asked for. The charter's exit check — *no expressible
   request feature can be silently dropped* — exists because this is the milestone where that
   happens.
2. **Determinism cannot be proven the obvious way.** "Translate twice, compare bytes" is
   probabilistic: Go re-randomizes map-range start per range-statement *execution*, so a map-ordering
   bug can pass by luck. The charter's "across process runs" is only proven by a checked-in golden
   compared across independent `go test` invocations.
3. **This repo has no wire-fixture convention.** Grepping `backend/agent` for `golden`, `-update`,
   `.golden` returns zero matches (`src/ai/testdata/{constructed,handrolled}` is AI-06.3's
   compile-boundary seal proof, not fixtures). AI-26 and AI-27 both need one, and both independently
   arrived at **inline literals** — adopted adapter-wide (see *Approach*).

## Scope

### In Scope

- **AI-26.1 — Wire skeleton `[leaf]`**: model identity and body scaffold; the determinism harness;
  the fixture-wide credential-sentinel scan.
- **AI-26.2 — System segments and cache markers `[leaf]`**: segment order preserved; cache markers
  **dropped whole**, proven by a marked/unmarked twin producing byte-identical fixtures. Item 3
  (marker-cap refusal) has **no subject** — see *Amendments*.
- **AI-26.3 — Messages and content parts `[leaf]`**: role and part mapping, order preservation, and
  the **no-merging** decision — two consecutive same-role messages render two distinct wire message
  objects. *Largest node; pre-declared fission trigger below.*
- **AI-26.4 — Tools, deterministic `[leaf]`**: declaration order is caller order; `Tool.Schema()`
  bytes pass through unmodified; cross-run byte-identity proven by golden.
- **AI-26.5 — Tool results and identifiers `[leaf]`**: result messages and correlation. Item 2
  (synthetic ID minting) has **no subject** — folded into items 1 and 3 as a no-op pin.
- **AI-26.6 — Reasoning replay `[leaf]`**: **typed refusal**, per the settled coordinator ruling.
- **AI-26.7 — Options, limits, escape hatch `[leaf]`**: generation options; the adapter's own
  provider-extension namespace merged, foreign namespaces enumerated but never inspected. Item 2
  (mandatory-limit default) is a **deliberate no-op** — see *Amendments*. **Plus the usage opt-in
  obligation** (below), which AI-24 § 13.1 assigns to this node by name.
- **The usage opt-in trap — AI-24 § 13.1, assigned to AI-26.7.** Every request this adapter builds
  must set `stream_options.include_usage: true`. Left unset, the vendor returns an empty `usage`
  that is *legal-shaped* and would misread as `CAP-R-03 = absent` — illegal for a required
  capability, and an adapter defect wearing a vendor limitation's clothes. AI-28.3 and AI-31.2 hold
  the response-side halves; **the request-side half is AI-26.7's and is in scope here.**
- **AI-26.8 — Unsupported-feature policy**: `26.8.1` the feature inventory `[decision]`;
  `26.8.2` the exhaustive walk `[leaf]` that fails on any inventoried feature lacking a policy entry.
- The first golden-fixture convention in this repo, deliberately shaped to generalize adapter-wide.

### Out of Scope

- **Sending anything.** Translation is pure: neutral request in, wire bytes out. No client, no
  network, no `httptest`. AI-25 owns construction; AI-27 owns decoding.
- Response/stream direction entirely (AI-27 … AI-31), and error mapping (AI-32).
- **The capability verdict.** AI-26.6's refusal and AI-24's response-side finding both *indicate*
  toward recording `CAP-O-01` absent; **AI-29.0 still owns that verdict** and this change does not
  pre-empt it.
- **Layer 2's fallback.** Stripping reasoning parts before replaying history is the consumer's job
  (AI-03 standing rule 4). Not built here, and **not a defect**.
- Any edit to `package ai`. The neutral surface is read-only; every accessor AI-26 needs is already
  public (verified).
- Migrating AI-25's tests to the fixture convention.
- **Deferred but related**: the two Wave-2 carryovers (`CheckEmit` rule 4 failure-path gap, the
  redacting `*Failure.GoString()`) are **no longer unassigned** — AI-24 § 13.3 assigns them to a new
  milestone **AI-41** (leaves 41.1/41.2, `Blocks: AI-36`), scheduled Wave 5. Do not absorb them here.
- **The `docs/architecture/milestones/0002-*.md` amendments.** Already merged by AI-24's apply — see
  *Amendments* below. Landing them again would duplicate blockquotes under the same headings.

## Capabilities

### New Capabilities

- `ai-request-translation`: what each neutral region maps to on the wire; which features are
  translated, which are deliberately dropped, and which refuse; the determinism guarantee and how it
  is proven; the golden-fixture and credential-scan conventions; the feature-inventory mechanism.

### Modified Capabilities

- **None.** `ai-cache-breakpoints` (AI-11.3's advisory contract), `ai-content-parts`,
  `ai-reasoning-content`, `ai-model-request`, `ai-tool-declarations`, `ai-tool-messages`,
  `ai-request-extension-points` and `ai-provider-errors` are **cited by identifier and exercised,
  never modified**. AI-26.2 does not change AI-11.3 — it supplies the wire-level proof of a contract
  AI-11.3 already states.

## doc 0002 amendments — ALREADY MERGED upstream, not this change's work

**AI-24's apply landed all three dated blockquotes** under AI-26.2, AI-26.5 and AI-26.7. They are
merged upstream fact and are **out of scope here**: re-landing them would produce duplicate
blockquotes under the same headings. The table below records what AI-26 *inherits*, matching AI-24
`decision.md` § 12's AI-26 row and § 6's divergence answers.

Three items lose their *subject* under the OpenAI-compatible choice. A never-firing conditional is
**not** a superseded claim: each item's text remains in force for a future adapter and carries a
dated *not-applicable / deliberate-no-op* blockquote, never a strikethrough.

| Item | Why it has no subject | What is built instead |
|---|---|---|
| **26.2 item 3** — markers exceed the vendor cap → refuse | Caching is automatic; there is no cap | Item 2's *other* branch fires **unconditionally**: markers dropped whole. Proof = marked/unmarked twin, byte-identical fixtures (mirrors `cache_boundary_test.go`'s neutral-level marked-twin pattern) |
| **26.5 item 2** — mint synthetic tool-call IDs | The vendor assigns IDs | A no-op pin folded into items 1 and 3: the wire tool-call-id is always exactly the neutral call's identifier, byte-for-byte. No separate empty test |
| **26.7 item 2** — mandatory output limit → documented default | `max_tokens` is optional; the IF-guard never fires | Assert the **negative**: omitting the option means the field is **absent** from the golden body (explicit absence, not "equals a default"), plus package documentation stating why the branch is intentionally dead here |

## Settled: AI-26.6 reasoning replay refuses

When a request carries a `PartKindReasoning` part — **in any of its three states, at any position** —
translation **fails with a typed unsupported-capability error**. It does not drop it, does not render
it as text, does not route it through the provider-extension channel.

This is a determination, not a preference. Three alternatives are forbidden by shipped contracts:
silent drop violates AI-26.8's exit check; rendering as text violates AI-07.1 item 2 and is
*mechanically impossible* for the redacted and token-only states, which carry no text; and the
provider-extension namespace is caller-owned by contract, not an adapter smuggling channel. The
remaining option is independently **mandated** by AI-03 standing rule 4 — *"Layer 1 never substitutes
for an absent capability… Layer 1 states the absence; the consumer owns the fallback."*

Consequences, stated plainly so no later reader misreads them:

- **Layer 2 must strip reasoning parts before replaying multi-turn history to this adapter.** That is
  the consumer-owned fallback AI-03 anticipates — **not a defect, not a gap**. Recorded three ways so
  it does not survive only as prose someone must remember to read: (a) in this change's spec, (b) in
  the adapter's package documentation, and (c) **routed as an obligation to AI-40**, which already
  publishes the Layer 2 readiness contract — **no new node**. The routing is recorded here rather
  than executed, mirroring AI-24 § 9's "routed, not fixed here" precedent.
- **Provider-swap severity: acceptable for v1, with a written note.** A transcript captured from a
  reasoning-emitting provider and replayed here **fails hard**. That is acceptable on AI-03 rule 4's
  grounds, but the adapter's package documentation must say so plainly, naming consumer-side
  stripping as the remedy. An unwritten hard failure is a support incident; a written one is a
  contract.
- AI-26.6's four-item test list collapses to **one refusal test**, effectively merging into
  AI-26.8's exhaustive-walk policy. This materially shrinks the node; the forecast reflects it.
- **Revisit trigger (living-graph clause)**: if a future vendor update adds a reasoning-carrying
  request field, AI-26.6 returns from refusal to rendering.

## Approach

**Package.** `backend/agent/src/ai/openaicompat/`, extending the package AI-25 creates. Same
conventions, same doc style, same zero-dependency posture.

**Translation is a pure function** over a neutral `ai.Request`, producing wire bytes. Design owns the
exact shape; the proposal fixes only that it takes no client and performs no I/O, so every node's
proof is a value comparison and never a network assertion.

**The neutral surface needs no reflection.** Verified: every value AI-26 reads is a plain public
accessor returning a fresh copy — no type switch over unexported types, no reflection. AI-06's fix
for the retired design's defect C2 holds in practice.

**Determinism.** Every neutral collection is already a slice in caller order, so the risk lives
entirely inside AI-26's own code — e.g. an intermediate `map[toolName]…` built before flattening
back to a slice. `encoding/json`'s map-key sorting does **not** protect against this, because the
leak happens in slice-construction order, before marshalling. Primary proof is the **checked-in
inline expectation compared across independent `go test` invocations**; the same-process
double-translate stays as a cheap pin, explicitly **not** presented as the proof.

**Fixtures — inline literals, reconciled with AI-27.** AI-27's design independently chose **inline
literals** over a golden-file tree, converging with this proposal without coordination. **Adopted as
the adapter-wide convention**, and on review the analysis favours it on its own merits, not merely
for consistency:

- The determinism argument is untouched. What proves cross-run byte-identity is that the expectation
  is **checked into source and compared in a fresh process** — not where it lives. An inline literal
  satisfies that exactly as a file does.
- No `-update` machinery to build. That flag is a double-edged tool here: it makes blessing a wrong
  change trivial, which is precisely the drift this milestone exists to prevent. Typing the expected
  bytes by hand is a feature, not friction.
- Under a chained review budget, expectation and case appear **together in one diff hunk**, which is
  worth more to a reviewer than a tidy `testdata/` tree.

**Consequence — the credential scan retargets.** With no `testdata/` tree to walk, the scan operates
over the adapter package's own `_test.go` sources as raw bytes. This preserves the property that
mattered — fixtures added by later nodes are covered automatically, without editing the scan — and is
mildly stronger, since it also catches a credential hardcoded in test setup rather than in a fixture.
**Credential-shaped sentinels only**: endpoint hosts and model identifiers are legitimate fixture
content, and scanning them would generate noise that trains reviewers to ignore the scan.

**Feature inventory (26.8.1) — hybrid, no reflection.** Runtime enumeration of the **five** already-
exported closed vocabularies — `PartKinds()`, `ReasoningStates()`, `ToolChoiceModes()`,
`CacheRegions()`, `Roles()` (verified present) — plus the repo's existing AST-scan-against-a-policy-
table idiom (`content_part_registry_test.go`, `validation_registry_internal_test.go`) for AI-12's
option surface, which is plain fields rather than a closed vocabulary. Ten exported `With*`
constructors are the AST scan's ground truth (verified). A new option added to `ai` without a policy
entry fails the guard — the same failure mode as "sentinel declared but unregistered". Reusing an
established idiom beats inventing one.

**No merging.** The dialect does not enforce strict role alternation, so *no merging* is the correct
behavior — pinned as a deliberate decision with a test asserting two consecutive same-role messages
produce two distinct wire message objects. **This claim is inference, not a cited vendor document
(see below).**

**Escape hatch.** The adapter reads its own reserved namespace and merges those bytes; every other
namespace is enumerated but never inspected. "Ignored whole" is proven by the same twin-comparison
shape as the cache markers: attaching a foreign namespace must leave the fixture byte-identical.

## Wire-shape claims and their provenance

*A confidently wrong wire detail propagates into every downstream golden. Each row is marked.*

| Claim | Provenance |
|---|---|
| Tool results are a distinct `role: "tool"` message | **Given** (AI-24 settled input) |
| The vendor assigns tool-call identifiers | **Given** (AI-24) |
| `max_tokens` is optional | **Given** (AI-24) |
| Caching is automatic; no marker field and no cap exist | **Given** (AI-24) |
| No signed/carried reasoning blocks in this dialect | **Given** (AI-24), corroborated by exploration's read of the OpenAI reasoning guide — `completion_tokens_details.reasoning_tokens` is an opaque integer, not a content block |
| Arguments stream in fragments keyed by `index` | **Given** (AI-24) — response-side; noted for AI-26.5's correlation symmetry only |
| Body carries `model` and a `messages` array | **Vendor-documented** (Create chat completion) — re-verify at design |
| **`stream_options.include_usage: true` must be set on every request** | **Given** — AI-24 § 8 `CAP-R-03` and § 13.1, which assign the request-side half to **AI-26.7 by name** |
| **The dialect does not enforce strict user/assistant role alternation** | ⚠️ **GATE — uncited.** AI-24 § 6's tool-result row shows the dialect admits a three-role sequence (a `role: "tool"` message inherently breaks user/assistant alternation) — *indicative, not dispositive*, because it does not establish that two **consecutive same-role** messages are accepted, which is exactly what AI-26.3 item 2 turns on |
| **System instruction as a `messages` entry vs a top-level field, and whether its role is `system` or `developer`** | ⚠️ **GATE — uncited.** Checked AI-24 §§ 6, 7, 8, 12: **not addressed anywhere.** Shapes every AI-26.2 fixture |
| **Tool-call arguments are a JSON-encoded *string*, not a nested object** | ⚠️ **GATE — partially confirmed.** AI-24 § 7 ("partial JSON string fragments, concatenated in index order to reconstruct the complete argument bytes") and § 8 `CAP-R-02` establish this **response-side**; a field carrying string fragments cannot be a nested object. **Request-side symmetry is still inference** and needs its own citation |
| **`tools[].function.parameters` carries the neutral schema verbatim** | ⚠️ **GATE — uncited.** Checked AI-24 §§ 6, 7, 8, 12: the tools *declaration* shape is **not addressed**. Load-bearing: `Tool.Schema()` is byte-exact and must never be re-marshalled |

### These four are a hard gate, not an advisory

**Each node's first task is to obtain and cite authoritative vendor documentation for its claim. An
uncited claim BLOCKS that node — it does not proceed on inference.** A wrong wire detail here
propagates into every downstream milestone's fixtures, which is why this is the milestone's highest
risk. Gate map: claim 1 → AI-26.3; claim 2 → AI-26.2; claim 3 → AI-26.5; claim 4 → AI-26.4.

## Settled — which typed error, and why it differs from AI-25

**AI-26.6 and AI-26.8.2 refusals use `PreStreamFailure` carrying `ErrUnsupportedCapability`**
(`FailureCategoryUnsupportedCapability`, V-FAIL-08). **AI-25 uses AI-04's `Violation`.** There is no
drift here and a later reader must not read one — they are **different failure classes**:

| | AI-25 construction fault | AI-26 refusal |
|---|---|---|
| Example | malformed endpoint, empty credential | request carries a reasoning part |
| Is the request valid? | **No request exists yet** | **Yes — perfectly valid neutrally**; AI-10/AI-12 accept it |
| What failed | the **caller's contract** | the **provider's expressiveness** |
| Class | validation fault | capability failure |
| Taxonomy | AI-04 `Violation` | AI-19 `PreStreamFailure` + `ErrUnsupportedCapability` |

Settled upstream, not locally: **AI-03's capability decision § 10.4** states the *unsupported
capability* category "is a **request-time failure** and is **not** an absent optional capability."
AI-19 supplies the designated cell, and `DeliveryPreStream` — *"returned directly, before any carrier
was handed over"* — documents exactly this position. AI-26 adds **no new sentinel**, and the choice
is uniform across AI-26.6 and AI-26.8.2.

## Affected Areas

| Area | Impact | Description |
|---|---|---|
| `backend/agent/src/ai/openaicompat/` | New (extends AI-25) | Translation, feature-policy table; package documentation for the two no-op branches, the reasoning hard-failure note, and the no-merge rationale |
| `backend/agent/src/ai/openaicompat/*_test.go` | New | Per-node proofs, inline expectations, twin comparisons, determinism pin, inventory guard, credential scan |
| `backend/agent/src/ai/**` | **Read-only** | Every accessor consumed unchanged; no edit |
| `backend/agent/go.mod` | Unchanged | Zero requires — **any dependency is a hard blocker** |
| `docs/architecture/milestones/0002-*.md` | **Unchanged — already merged by AI-24** | Out of scope; re-landing would duplicate blockquotes |
| `openspec/specs/ai-request-translation/spec.md` | New (at archive) | Live contract home; carries the Layer 2 stripping duty routed to AI-40 |

## Delivery — chained PRs

`delivery_strategy: auto-chain` · `chain_strategy: feature-branch-chain` · `review_budget_lines: 5000`.
doc 0002 already flags this milestone as *"likely over one review budget: the node boundaries below
are the planned chain points."* Corrected for this repo's confirmed 2–4× forecast undershoot:

| # | Node | Contents | Naive | Corrected |
|---|---|---|---|---|
| 1 | **26.1** | skeleton, determinism harness, credential scanner, fixture convention | ~250–350 | ~500–1,000 |
| 2 | **26.2** | system segments; markers dropped whole (twin) | ~200–260 | ~400–800 |
| 3 | **26.4** | tools, caller order, schema pass-through, cross-run proof | ~250–350 | ~500–1,000 |
| 4 | **26.7** | options, absence pin, escape hatch (foreign-namespace twin), **`include_usage` opt-in** | ~300–450 | ~600–1,300 |
| 5 | **26.3** | messages and parts, order preservation, no-merge pin | ~450–650 | ~900–2,000 |
| 6 | **26.5** | tool results, ID pass-through pin | ~150–230 | ~300–600 |
| 7 | **26.6** | reasoning refusal (**shrunk by the ruling**) | ~80–130 | ~160–350 |
| 8 | **26.8** | inventory `[decision]` + exhaustive walk `[leaf]` | ~300–500 | ~600–1,200 |
| | | **Total** | **~1,980–2,920** | **≈ 4,050–8,600** |

**Recommended order and why.** 26.1 first — nothing else compiles without it. Then the three
**graph-independent siblings** 26.2, 26.4, 26.7, cheapest-first: they exercise the harness, the
golden convention and the error typing on small, low-risk diffs *before* the largest node lands.
26.3 fifth, so it is pure content-mapping against conventions already settled and reviewed. 26.5 and
26.6 extend 26.3's part-mapping and must follow it. 26.8 last — its exhaustive walk must cover every
feature the previous seven nodes taught the translator.

*Rejected alternative*: 26.3 second (biggest first). It would freeze the golden and error-typing
conventions inside the largest, hardest-to-review diff.

**Chain-strategy cost, stated explicitly.** 26.2, 26.4 and 26.7 depend only on 26.1 and the
dependency graph permits reviewing them **in parallel**. `feature-branch-chain` forces each child to
target the immediately previous branch, so they stack linearly instead. That costs review
*wall-clock latency*, not correctness, and keeps every child diff clean — the reason the strategy was
chosen. A `stacked-pr` strategy would recover the parallelism; **changing strategy mid-wave is out of
scope** and would need the coordinator.

**Pre-declared fission trigger — 26.3.** If it exceeds budget, split at the node's own seam:
**26.3a** message/role skeleton, order preservation and the no-merge pin; **26.3b** the content-part
variant goldens.

`Decision needed before apply: No` · `Chained PRs recommended: Yes` · `400-line budget risk: High`

## Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| **R1** Four wire-shape details are uncited. A wrong one propagates into every downstream fixture | **High — the milestone's top risk** | **Hard gate**, not advisory: each node's first task is to cite vendor documentation for its claim; uncited blocks the node. Gate map in the provenance section |
| **R2** ~~AI-27 invents a different fixture convention~~ **RESOLVED** | — | AI-27's design independently chose inline literals; convergence adopted adapter-wide |
| **R3** ~~Typed-error choice contradicts AI-25~~ **RESOLVED** | — | Different failure classes, settled by AI-03 § 10.4; the distinction is recorded in the proposal so it is not later read as drift |
| **R9** The `stream_options.include_usage` obligation is silently dropped, producing an empty `usage` that misreads as `CAP-R-03 = absent` | Med | AI-24 § 13.1 names AI-26.7; it is in scope and in the success criteria. Its failure mode is legal-shaped, so it needs a positive assertion, not absence of an error |
| **R4** A map used inside translation reintroduces nondeterminism that the double-translate pin misses | Med | Golden across independent `go test` runs is the primary proof; the in-process check is documented as a pin only |
| **R5** 26.3 overshoots even the corrected forecast | Med | Fission trigger pre-declared above |
| **R6** AI-25 is still in flight; its package layout or namespace choice shifts | Med | AI-26.1 must not start before AI-25 slice A merges; the reserved extension namespace is read from AI-25's artifact, not re-invented |
| **R7** A reader misreads Layer 2's reasoning-stripping duty as an AI-26 defect | Med | Stated as a consequence in this proposal, and must be repeated verbatim in the spec and in package documentation |
| **R8** Total forecast is up to 1.7× the 5,000-line budget even chained | Med | Eight slices, each independently reviewable and revertible; per-slice `make test` green recorded in its PR |

## Rollback Plan

Translation is additive and has **no consumer** — AI-28 … AI-32 are unstarted — so rollback is
deletion, not migration.

1. Revert slices in reverse chain order (8 → 1), or delete the translation files from
   `openaicompat/` and its `testdata/` tree. AI-25's construction code is untouched and survives.
2. `package ai` is read-only in this change, so no shipped contract can be left half-edited.
3. `go.mod` is untouched by design — no dependency state can be stranded.
4. doc 0002 needs no rollback — its amendments belong to AI-24 and are untouched by this change.
5. Re-run `make test` from `backend/agent/` to confirm AI-00 … AI-25 stay green.
6. `openspec/changes/cachicamas-ai-request-translation/` moves aside; nothing is promoted to
   `openspec/specs/` until archive, so no live contract is orphaned.

## Dependencies

- **AI-25 merged** (at least slice A): the `openaicompat` package, its documentation conventions, and
  the reserved provider-extension namespace.
- AI-10, AI-11, AI-12, AI-06, AI-07, AI-19 — all shipped in-repo and consumed unchanged.
- Go standard library only (`encoding/json`, `go/ast`, `go/parser`, `go/token`, `path/filepath`,
  `testing`). **`go.mod` must stay at zero requires — any dependency is a hard blocker**, not a
  tradeoff.
- Strict TDD; `make test` from `backend/agent/` (`go test -race -v ./...`); `make lint` clean.

## Success Criteria

- [ ] **No credential** appears in any serialized fixture, proven by one scan over the adapter
      package's own `_test.go` sources that automatically covers cases added by later nodes.
      Credential-shaped sentinels only — hosts and model identifiers are legitimate content.
- [ ] Every request sets **`stream_options.include_usage: true`**, asserted positively (AI-24 § 13.1).
- [ ] Refusals carry `PreStreamFailure` + `ErrUnsupportedCapability`, uniformly across 26.6 and
      26.8.2, and **no new sentinel is added**.
- [ ] Package documentation states that a reasoning-bearing transcript **fails hard here** and must
      be stripped consumer-side; the same duty is routed to **AI-40** and stated in the spec.
- [ ] **No expressible request feature is silently dropped**: every feature in 26.8.1's inventory has
      an explicit policy entry — translate, deliberately drop, or refuse — and 26.8.2 fails if one
      lacks it.
- [ ] Translating the same request yields **identical bytes across independent `go test`
      invocations**, proven by golden fixture; the same-process double-translate is present as a pin
      and is not presented as the proof.
- [ ] A cache-marked request and its unmarked twin produce **byte-identical** wire fixtures.
- [ ] A request carrying reasoning content **fails with a typed unsupported-capability error** in all
      three reasoning states and at any position — never dropped, never rendered as text.
- [ ] Omitting the max-output-tokens option leaves the field **absent** from the golden body, checked
      as explicit absence, with package documentation explaining why the branch is intentionally dead.
- [ ] Every wire tool-call identifier is **byte-identical** to the neutral call's identifier; no
      identifier is ever minted.
- [ ] Two consecutive same-role messages render as **two distinct wire message objects**, in order.
- [ ] A foreign provider-extension namespace leaves the fixture **byte-identical**; the adapter's own
      namespace merges.
- [ ] Tool declarations follow caller order and `Tool.Schema()` bytes pass through **unmodified**.
- [ ] All four ⚠️ wire-shape claims carry an authoritative vendor citation **before** their node is
      built; an uncited claim blocked its node rather than proceeding on inference.
- [ ] `backend/agent/go.mod` still carries **zero** `require` directives.
- [ ] `make test` green and `make lint` clean for **every slice**, recorded in that slice's PR.
