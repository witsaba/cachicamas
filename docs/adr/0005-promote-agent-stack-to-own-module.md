# ADR 0005: Promote the agent stack to the `backend/agent` Go module

> Status: **Proposed** (2026-07-30, implemented by change `cachicamas-agent-module-scaffold` — see
> the § Migration amendment of 2026-07-31)
> Author: cachicamas SDD pipeline (parent orchestrator)
> Amends: [ADR 0004](./0004-adopt-tau-3-layer-agentic-architecture.md) — supersedes its
> *Layer → location mapping*, its *Dependency rule*, and its first Consequence bullet.
> The three-layer split, the loop/harness separation, and the one-way dependency
> **direction** are unchanged and remain in force.
> Companion: [ADR 0006](./0006-resolve-skill-and-prompt-source-of-truth.md) (skill/prompt policy)
> Review source: adversarial architecture review of 2026-07-30 (Engram `obs #2243`)

---

## Resolved TOC (jump links)

- [D1 — Dependency rule v2](#d1--dependency-rule-v2) — who may import whom, at module granularity
- [D2 — Location mapping v2](#d2--location-mapping-v2) — where every component lands
- [D3 — Observability boundary](#d3--observability-boundary) — OTel API yes, OTel SDK no
- [D4 — v1 scope for cross-cutting concerns](#d4--v1-scope-for-cross-cutting-concerns) — in / seam / deferred
- [Enforcement](#enforcement) — the three mechanical guards

## Full TOC

- [Context](#context)
- [What ADR 0004 got wrong](#what-adr-0004-got-wrong)
- [Decision](#decision)
  - [D1 — Dependency rule v2](#d1--dependency-rule-v2)
  - [D2 — Location mapping v2](#d2--location-mapping-v2)
  - [D3 — Observability boundary](#d3--observability-boundary)
  - [D4 — v1 scope for cross-cutting concerns](#d4--v1-scope-for-cross-cutting-concerns)
- [Enforcement](#enforcement)
- [Migration](#migration)
- [Consequences](#consequences)
- [References](#references)

---

## Context

> **Amended 2026-07-31 — the history this section reports was undone.** The Layer 1 code described
> below was **removed** on 2026-07-30, and the Layer 1 plan was rebuilt from zero
> ([doc 0002 § what changed from the retired plan](../architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md#what-changed-from-the-retired-plan)).
> Nothing described as shipped here survives, and every milestone identifier below has been remapped
> to the rebuilt plan. **The decisions this ADR records — D1 – D4 and Enforcement — are unchanged
> and in force**, and the rebuilt plan implements them: AI-00 creates `backend/agent` and both
> boundary guards exactly as § D2 and § Enforcement require. Only the narrative around the decisions
> is stale, and it is corrected rather than deleted, because the review that produced those
> decisions was performed against the code described here.
>
> **Reading rule for the whole document: every struck-through AI-NN below is a *retired-plan*
> identifier and must never be followed.** Four of them — AI-16, AI-17, AI-18 and AI-39 — name
> completely different milestones in the rebuilt plan, so a struck number resolves silently to the
> wrong place. Only the unstruck identifiers are live.

[ADR 0004](./0004-adopt-tau-3-layer-agentic-architecture.md) was accepted on 2026-07-17.
~~Since then **17 of the 39 Layer 1 milestones have shipped** (AI-00 … AI-16) into
`backend/database_administrator/src/tools/agent/ai/`~~ **Between 2026-07-17 and 2026-07-30, 17 of a
then-39-milestone Layer 1 plan were built into
`backend/database_administrator/src/tools/agent/ai/`** — roughly 38 files and 10.9k lines, with
mechanical import-boundary guards and a strict RED/GREEN/DOCS/POLISH commit discipline. **That code
was subsequently removed. The rebuilt plan is 41 milestones, AI-00 … AI-40, of which four have
shipped: AI-00 … AI-03.**

An adversarial architecture review on 2026-07-30 found that **the decomposition is sound and the
location mapping is not**. The three-layer split, the stateless-loop / stateful-harness separation,
and the one-way dependency direction are all working: Layer 1 is vendor-free, boundary-tested, and
consumable from another package today. The problems are all in *where the layers were put* and in
*what the ADR did not say*.

The root cause is single and mechanical. ADR 0004's mapping table was derived from a Python
reference in which the three layers are **top-level sibling packages**, and was then transplanted
into a subdirectory of a subdirectory of one hexagonal service without re-deriving the constraints
that move imposes. Six of the seven structural findings follow directly from that transplant.

This ADR closes findings **S1 – S6**. [ADR 0006](./0006-resolve-skill-and-prompt-source-of-truth.md)
closes **S7**. ~~The four shipped-contract defects (C1 – C4) enter code through the normal SDD
pipeline as milestones AI-40, AI-41, AI-42 and AI-18~~ **The four contract defects (C1 – C4) no
longer have corrective milestones of their own: each is prevented at the milestone that defines the
contract — C1 and C2 at AI-06, C3 at AI-14, and C4 at AI-19, which lands the error taxonomy before
AI-20 defines the interface that requires it**; see the
[Layer 1 milestones and task graph](../architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md).

---

## What ADR 0004 got wrong

### S1 — the dependency rule contradicts the promised benefit

`0004:237` promises that "any package (`cmd/`, `migration/`, `application/`) can drive an agent
session without reimplementing the loop." `0004:224-229` forbids "the rest of
`database_administrator/src/`" from importing `tools/agent/`.

You cannot consume `<-chan ai.Event` without importing the package that declares `ai.Event`. The
stated benefit and the stated rule are mutually exclusive, and as written the rule forbids the
agent from ever being wired into a binary at all — `cmd/` is the composition root.

### S2 — Layer 3 was left unbounded, which inverts the hexagon

`0004:222` gives `cachicamas_coding` a *may NOT import* cell of `—`. That is correct in the source
narrative, where the three layers are separate top-level packages and "top of the stack" is
literally true. Inside `src/`, an unconstrained Layer 3 may legally import `interfaces/http`,
`infrastructure`, `migration`, `domain` and `otel`.

Combined with S1's ban on the reverse direction, `src/tools/agent/coding/` becomes a **second
composition root buried three directories deep** — a leaf that imports the whole hexagon while the
hexagon may not import it. That is precisely the layering violation that `openspec/AGENTS.md` rule
2 ("hexagonal layout is sacred") exists to prevent.

### S3 — the `src/tools/` name was already taken

`src/tools/tools.go` is the standard Go `//go:build tools` dependency-pin idiom. Its own doc
comment says it "will be removed once the consuming code lands." A Go reader who opens `src/tools/`
expects build-tooling dependency pins and finds a coding agent.

The Layer 1 roadmap already flagged this — *"The current `tools` package is not an agent
foundation. Do not extend that file into the model adapter."* — and ADR 0004 did not.

### S4 — a CLI and a TUI were mapped into an HTTP-server module

ADR 0004 maps `tui/`, `print.go`, `catalog.toml`, slash commands, and
`~/.cachicamas/sessions/<id>.jsonl` into a module whose `Makefile` builds exactly one binary from
`cmd/server/main.go` and which ships in a container (`docker-compose.yaml:185`). TOML is used
nowhere else in the repo — configuration is environment variables per `openspec/AGENTS.md` rule 3 —
and per-developer session state under `~/.cachicamas/` has no meaning inside a containerized
service.

### S5 — the rule forbids the repo's own mandatory observability

`0004:221` lists `otel` in `cachicamas_agent`'s *may NOT import* column, and `0004:220` restricts
Layer 1 to stdlib plus vendor SDKs. Verified: `grep -rn "otel\|log/slog" src/tools/agent/` returns
zero non-test hits.

The consequence is that LLM calls — the slowest, most expensive, and most failure-prone operation
this service will ever perform — are the **only** part of an OTel-instrumented service with no
traces, no spans, and no structured logs, and trace context from `otel.Middleware` will not
propagate through the agent loop. ADR 0004 does not list this under Consequences / Negative.

### S6 — enforcement is one-sided

`ai/import_boundary_test.go`, `provider_test.go`'s AST check, and `agenttest/consumer_test.go`'s
signature guard all enforce Layer 1 purity, and they do it well. **Nothing** enforces the other
half — that `cmd/`, `application/`, `interfaces/http/` and the rest stay out of `tools/agent/`. The
rule is currently satisfied by accident, and will break silently the first time anyone wires the
agent in.

---

## Decision

Promote the three layers out of `database_administrator` into a **new sibling Go module**,
`backend/agent`, alongside `database_administrator` and `workspace_syncer`. Because Go modules
cannot form an import cycle, the move converts most of the rules below from conventions that must
be tested into invariants the compiler enforces.

### D1 — Dependency rule v2

> **Invariant.** `backend/agent` never imports `backend/database_administrator` or
> `backend/workspace_syncer`, in any package, including `cmd/`. These are separate Go modules, so a
> cycle is a build error: **Go enforces half of this rule for free.** The guards under
> [Enforcement](#enforcement) cover the other half.

| # | Package | MAY import | MUST NOT import |
| --- | --- | --- | --- |
| 1 | `github.com/cachicamas/backend/agent/src/ai` (Layer 1) | Go stdlib; `net/http`; vendor model SDKs; OpenTelemetry **API** | `…/agent/src/{agent,coding,cmd}`; any package of another backend module; OTel **SDK** or any exporter; `otelslog`; any TUI / TOML / CLI dependency |
| 2 | `…/backend/agent/src/agent` (Layer 2) | row 1; Go stdlib; OTel API | Layer 3; `cmd`; any other module; OTel SDK; `os` environment reads; the filesystem; `net/http` — Layer 2 performs no I/O of its own, it calls `ModelProvider` and `Tool` |
| 3 | `…/backend/agent/src/coding` (Layer 3) | rows 1–2; Go stdlib; OTel API; `net/http`; third-party TUI / TOML / tokenizer dependencies | `…/agent/src/cmd/…`; **any** Go package of `database_administrator` or `workspace_syncer` — it may call them over HTTP, never import them; OTel SDK |
| 4 | `…/backend/agent/src/cmd/cachicamas` (composition root) | rows 1–3; OTel **SDK** and exporters; flag/env parsing; anything | — it is `package main`, so nothing *can* import it |
| 5 | `database_administrator/src/application`, `…/src/cmd/server` | rows 1–2 **only** | `…/agent/src/coding`; `…/agent/src/cmd/…`. **Not exercised in v1** — see below |
| 6 | `database_administrator/src/domain` | unchanged | the agent module, entirely |
| 7 | every other `database_administrator/src/**` (`interfaces`, `infrastructure`, `migration`, `otel`) | unchanged | the agent module, entirely |

**Composition roots.** Exactly two, both `package main`, neither importable:

- `backend/agent/src/cmd/cachicamas/main.go` — builds providers from the catalog, builds the
  skill/prompt source chain, installs the OTel SDK, wires the harness, selects a frontend.
- `backend/database_administrator/src/cmd/server/main.go` — unchanged today; the place row 5 would
  eventually be exercised.

#### Row 5 is the honest answer to S1 — and it is not free

`database_administrator` **does not import the agent module in v1.** Row 5 records the *permitted
direction* so the rule is stated once and the type-level capability exists; it does not claim the
wiring is buildable today, because it is not.

`Dockerfile:50-56` copies only `go.mod`, `go.sum` and `src/` from build context
`./backend/database_administrator` (`docker-compose.yaml:185`). The first cross-module import
breaks `docker compose build`. Making it work costs three things together: a `replace ../agent`
directive, moving the compose build context from `./backend/database_administrator` to `./backend`,
and rewriting every `COPY` path in the Dockerfile. That is its own change with its own review.

This is not a hypothetical. `README.md:143` already promises exactly this
`replace ../database_administrator` pattern for `prd_orchestrator` — a service that does not exist
on disk. The trap is repo-wide and has already stopped one service. Recording the permitted
direction *without pricing the build cost* would replace S1 with a promise the build still cannot
keep.

### D2 — Location mapping v2

New module: `backend/agent/` → `module github.com/cachicamas/backend/agent`, `go 1.26.3`.

| Component | ADR 0004 location (superseded) | ADR 0005 location |
| --- | --- | --- |
| `cachicamas_ai` (Layer 1) | `…/database_administrator/src/tools/agent/ai/` | `backend/agent/src/ai/` |
| Layer 1 external-consumer proof | `…/src/tools/agent/agenttest/` | `backend/agent/src/agenttest/` — see the constraint below |
| `cachicamas_agent` (Layer 2) | `…/src/tools/agent/agent/` | `backend/agent/src/agent/` |
| `cachicamas_coding` (Layer 3) | `…/src/tools/agent/coding/` | `backend/agent/src/coding/` |
| CLI binary / print mode | `…/coding/print.go` | `backend/agent/src/cmd/cachicamas/` (`package main`) |
| TUI | `…/coding/tui/` | `backend/agent/src/coding/tui/` — a *library*, driven from `cmd/cachicamas` |
| Built-in tools (`read`/`write`/`edit`/`bash`) | `…/coding/tools.go` | `backend/agent/src/coding/tool/` |
| Provider catalog | `…/coding/catalog.toml` | `backend/agent/src/coding/catalog/` (embedded default via `//go:embed`) + user override at `~/.cachicamas/catalog.toml` |
| Slash commands | `…/coding/slash.go` | `backend/agent/src/coding/slash/` |
| Session persistence | `~/.cachicamas/sessions/<id>.jsonl` | unchanged — a runtime path, not a repo path |
| Skills | `.cachicamas/skills/` + `~/.cachicamas/skills/` | resolved via `SkillSource` in `backend/agent/src/coding/skill/`; the filesystem is **one of three** implementations — see [ADR 0006](./0006-resolve-skill-and-prompt-source-of-truth.md) |
| Prompts | (unmapped by ADR 0004) | `PromptSource` in `backend/agent/src/coding/prompt/`; same three implementations |
| Project instructions (`AGENTS.md`) | `AGENTS.md`, `.cachicamas/`, `.agents/` | unchanged — filesystem only; not a Postgres concept |
| Build / lint config | shared with `database_administrator` | `backend/agent/Makefile`, `backend/agent/.golangci.yml` (copy of the v2.9.0 config), `backend/agent/bin/` |
| `src/tools/tools.go` | the `//go:build tools` dependency pin | **unchanged, stays exactly where it is** |

Two constraints are pinned here deliberately, because both are currently load-bearing and
undocumented:

- **`src/agenttest/` MUST remain a direct sibling of `src/ai/`.**
  `TestModelProviderInterface_SignatureGuard` resolves `../ai/provider.go` from
  `runtime.Caller(0)`. Any other layout breaks it, with no warning anywhere in the tree. It fails
  loudly when it fails, but a future reorganisation will hit it blind.
- **`src/tools/tools.go` is not moved and not renamed.** S3 is resolved by *vacating* the name, not
  by relocating a standard-idiom file.

### D3 — Observability boundary

The rule ADR 0004 needed is not "no OTel below Layer 3." It is the standard library-instrumentation
split: **API yes, SDK no.** The OpenTelemetry API modules are dependency-light and are no-ops until
a provider is registered, so an unwired Layer 1 costs nothing at runtime; the SDK and its exporters
are process-lifecycle concerns that belong to a composition root.

| Import path | L1 `ai` | L2 `agent` | L3 `coding` | `cmd/` |
| --- | :---: | :---: | :---: | :---: |
| `go.opentelemetry.io/otel` (global getter) | ✅ | ✅ | ✅ | ✅ |
| `go.opentelemetry.io/otel/trace` | ✅ | ✅ | ✅ | ✅ |
| `go.opentelemetry.io/otel/attribute`, `/codes`, `/metric` | ✅ | ✅ | ✅ | ✅ |
| `go.opentelemetry.io/otel/sdk/…`, `…/exporters/…` | ❌ | ❌ | ❌ | ✅ |
| `go.opentelemetry.io/contrib/bridges/otelslog` | ❌ | ❌ | ✅ | ✅ |
| `backend/database_administrator/src/otel` | ❌ | ❌ | ❌ | ❌ |

**Attribute allowlist** for Layer 1 spans, following the OpenTelemetry GenAI semantic conventions:
`gen_ai.system`, `gen_ai.request.model`, `gen_ai.request.max_tokens`,
`gen_ai.response.finish_reasons`, `gen_ai.usage.input_tokens`, `…output_tokens`,
`…cache_read_tokens`, `…cache_write_tokens`, `http.response.status_code`, `retry.count`,
`stream.event_count`, `error.type`.

**Attribute denylist, absolute:** any prompt, completion, reasoning, tool-argument or tool-result
text; any HTTP header; any credential; any raw provider response body. Layer 1 owns this discipline
as milestone AI-36 (redaction). Under this rule, AI-37's acceptance clause "Layer 1 does not import
Cachicamas `otel`" stays literally true and becomes *precise* rather than accidental.

> **This section is the dependency ADR for OpenTelemetry.** Adding `go.opentelemetry.io/otel*` to
> a new `go.mod` is a new top-level dependency, which `openspec/AGENTS.md` rule 5 and `README.md`
> §8 require an ADR for. ADR 0005 § D3 pre-authorises exactly the paths in the table above and
> nothing else. Any exporter, any contrib package not listed, and any additional OTel module
> requires its own ADR.

### D4 — v1 scope for cross-cutting concerns

> **Amended 2026-07-31 — two verdicts cited code that no longer exists; the verdicts themselves are
> unchanged.** Every row of the table below stands exactly as decided. What was repaired is the
> *evidence* under it: the G10 and G13 reasons pointed at shipped files (`usage.go`, an AST
> signature guard and its behavioural scenarios) that were removed with the rest of the retired
> Layer 1 implementation on 2026-07-30. Both now state the requirement rather than the vanished
> proof, and their milestone identifiers are remapped to the rebuilt plan
> ([doc 0002 § identifier map](../architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md#what-changed-from-the-retired-plan)).

The review found thirteen concerns (G1 – G13) that the three-layer split is silent on. A dependency
rule answers "what may import what"; it does not answer permissions, sandboxing, compaction,
caching, parallelism, delegation, failover, or cost. Each is assigned a verdict here so that no
milestone has to re-litigate it. The architectural shape of each lives in
[the v2 architecture reference](../architecture/0001-cachicamas-agent-stack-v2.md); this table
carries only the verdict.

| ID | Concern | v1 verdict | Owner | Layer 1 impact |
| --- | --- | --- | --- | --- |
| G1 | Permission as a suspendable protocol on the event stream | Seam now, implement in L2 | L2 + L3 | none |
| G2 | Tool-execution sandboxing | Seam now, implement later | L3 | none — but the tool signature must have room for a policy |
| G3 | Context compaction | Seam now, implement in L2 | L2 | an **optional** provider token-counting capability, opt-in by type assertion; does **not** reopen `ModelProvider` |
| G4 | Prompt caching / cache breakpoints | **In v1, Layer 1, before any adapter** | L1 places, L2 keeps the prefix stable | breaking: the system instruction must become structured; breakpoint markers on tools and messages. `Usage` already carries the cache token fields — no work there |
| G5 | Parallel tools with deterministic result order | Seam now, implement in L2 | L2 | the tool-call ordinal must survive normalisation |
| G6 | MCP as a dynamic tool source | Seam now | L3 | none — `ToolDeclaration` already exists |
| G7 | Subagents (harness-in-a-tool) | Seam now | L2 | none |
| G8 | Typed error taxonomy, retry, failover | **In v1 — already scheduled, amended** | L1 taxonomy, L2 failover | a constructible terminal error payload (C4) plus a partial-output discriminator |
| G9 | Per-request call options + a provider escape hatch | **In v1, Layer 1, before any adapter** | L1 | the request type needs extension points and copy-on-write helpers |
| G10 | Cost / usage as first-class events | Deferred — **the Layer 1 obligation is already met** | L2 emits, L3 prices | none |
| G11 | Hook taxonomy (pre-request, pre-compact, post-turn, session-start) | Seam now | L2 + L3 | pre-request hooks require the request to be rebuildable → covered by G9 |
| G12 | Provider leakage (whole tool calls, reasoning signatures, refusal / pause stop reasons, role and alternation differences) | Split | L1 for the three contract items, adapter-local for the rest | reasoning round-trip token; two new finish reasons; delta-optional tool calls |
| G13 | Stream carrier — iterator versus channel | **Decision only; default = keep channels** | L1 | none |

Two verdicts deserve their reasoning stated, because both contradict a plausible first reading:

- **G10 needs no Layer 1 work of its own.** The usage record must carry cache-read, cache-write and
  reasoning token counts, and the milestone that defines usage already owes exactly that (AI-13.3).
  Only the Layer 2 turn/session cost event and the Layer 3 price table remain.
- **G13 must not be scheduled as implementation work.** The canonical hazard — a consumer that
  stops reading strands the producer goroutine forever — is closed by the **send discipline** rather
  than by the carrier: every send is a `select` on the stream and `ctx.Done()`, and the caller owns
  a cancellable context. The residual risk is a caller who abandons the stream *and* never cancels,
  which is a contract violation rather than a design flaw. ~~Switching carriers today would
  invalidate AI-16's AST signature guard and its behavioural scenarios, merged days ago.~~ **The
  signature guard that argument relied on was never shipped, so it constrains nothing; the guard is
  now AI-20.4's obligation and the carrier was decided on its merits by AI-02, which chose a
  receive-only channel and delegated iterator ergonomics to a carrier view in the stream test kit.**
  It is a decision milestone with a documented default, to be closed before the first adapter —
  after which the same change is roughly three times larger.

---

## Enforcement

> **Amended 2026-07-31 — the decisions in this section stand; two statements of fact in it did not.**
>
> **First, no guard was upgraded.** This section was written when a Layer 1 import test existed inside `database_administrator`. That tree was removed in the from-zero restart, so AI-00.3 and AI-00.4 built both guards from nothing. Guard A's description below should be read as a specification of the guard to build, not as a diff against a predecessor. Guard B's `src/domain` half is the only one that genuinely extended an existing file.
>
> **Second, `go list -deps` does not deliver the property this section claims for it.** The flag closes the transitive blind spot but **not** the test-import one, so as specified below Guard A would have closed only half of S6. Measured at `database_administrator/src/domain` on go1.26.3: `go list -deps` reports 2 non-stdlib packages, `go list -deps -test` reports 5 — the three it adds being the external test package, the synthesized test binary, and their closure. A Layer 1 *test file* importing a sibling backend module would have passed the guard silently.
>
> The shipped guard therefore uses **`go list -deps -test`**, and normalizes the three synthesized shapes that flag introduces (`pkg [pkg.test]`, `pkg_test [pkg.test]`, `pkg.test`) before matching — unnormalized, they are measured against the allowlist and the guard fails on its own module. It also filters the standard library with the toolchain's own `.Standard` field rather than a maintained set, because `go list std` contains vendored `golang.org/x/...` paths that appear verbatim in real dependency output. The same correction is recorded as a dated amendment on [AI-00.3 in doc 0002](../architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md#ai-003--forward-guard-layer-1-purity-guard).

Three guards. ~~two of them upgrades of guards that already exist~~ **All three were built from nothing by AI-00 — see the amendment above.**

**Guard A — forward (Layer 1 purity).** `backend/agent/src/ai/import_boundary_test.go`:
the forbidden prefix is a slice covering both other backend modules and the three
sibling layers; the allowlist targets this module's own import path; the OTel API/SDK split from § D3 is
added. The mechanism must see `.TestImports` / `.XTestImports` **and** transitive dependencies, so that a
vendor SDK which transitively pulls in a cachicamas package is also caught — ~~it moves to `go list -deps` with an
allowlist~~ **which requires `go list -deps -test`, not bare `-deps`; see the amendment above.** That
is the forward half of S6.

**Guard B — reverse (the half that never existed).** `database_administrator/src/domain/imports_test.go`
is extended to forbid `github.com/cachicamas/backend/agent/`, and a new module-scope test asserts
that no package outside `application/` and `cmd/server` names the agent module. Today the count is
zero, so the test is green from birth — which is exactly the point: it fails on the first
accidental import rather than on the first production incident.

**Guard C — structural.** `backend/agent/src/agenttest/` must remain a direct sibling of
`backend/agent/src/ai/` (see § D2). This is asserted implicitly by
`TestModelProviderInterface_SignatureGuard` and stated explicitly here so a future reorganisation
has something to read.

> **No CI exists.** `.github/workflows/` is absent; every guard above runs only via `make test`
> inside a module directory. Three modules now means three places to run it. This is recorded under
> Consequences rather than solved here.

---

## Migration

> **Amended 2026-07-31 — there was no migration.** This section planned a move of shipped code out
> of `database_administrator`. That code was **removed** on 2026-07-30 instead, and the module was
> created empty
> ([doc 0002 § what changed from the retired plan](../architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md#what-changed-from-the-retired-plan)).
> The owning milestone is now **AI-00**, which is a creation rather than a relocation, and it has
> shipped. Two consequences follow. The identifiers below are remapped. And **the rename discipline
> this section exists to state now governs no work at all** — there was nothing to `git mv`, so no
> file history was at stake and none was lost. It is kept, unstruck, as the standing rule for the
> next repository-scale move, which is the only situation it was ever about.
>
> The struck **AI-39** and **AI-17** below are retired-plan numbers; both now name unrelated
> milestones (the opt-in live smoke test, and reasoning delta events). Do not follow them.

~~Milestone **AI-39** (`cachicamas-agent-module-promotion`) owns the mechanics~~ **Milestone AI-00
(`cachicamas-agent-module-scaffold`) owns the mechanics**; see
[AI-00 in the Layer 1 milestones and task graph](../architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md#ai-00--create-the-module-and-both-boundary-guards). It is a
mechanical change with no behaviour change, and it must land before ~~AI-17~~ **AI-04 — that is,
before the first milestone that writes a contract** — so that no further milestone deepens the
current path.

The one discipline worth stating at ADR level: **the move and the import-path rewrite are separate
commits.** Committing the `git mv` with byte-identical content makes every file a 100 % similarity
rename, so `git log --follow` and `git blame` traverse the move and GitHub renders it as a rename
rather than a delete-plus-add. Combining the two commits destroys that history. **This is now a
forward-looking rule only: AI-00 moved nothing.**

A repo-root `go.work` listing all three modules is added for editor and future-CI ergonomics. **No
`replace` directive** is added to `database_administrator/go.mod`: nothing imports the agent module
yet, and a `replace` with no requirer is dead weight that also disguises the D1 row-5 build cost.

---

## Consequences

### Positive

- **The S1 contradiction disappears by construction.** "Any package can drive an agent session" and
  "the hexagon must not import the agent" are no longer both required, because the agent is no
  longer inside the hexagon. What remains is a normal, honestly-priced cross-module dependency.
- **Go enforces the important half of the rule.** A module cycle is a build error, so
  `backend/agent` importing `database_administrator` cannot happen accidentally at any layer.
- **Layer 3 gets a real ceiling.** `cachicamas_coding` can no longer reach into `interfaces/http`
  or `infrastructure`, because they are not in its module. S2 is closed without needing a test.
- **The CLI gets a home.** A `package main` under `backend/agent/src/cmd/cachicamas` is an ordinary
  Go binary with an ordinary `Makefile` target, instead of a TUI stranded inside a container image
  that runs an HTTP server.
- **The LLM call path becomes observable** without weakening any boundary, because the API/SDK
  split is the same discipline every instrumented Go library uses.
- **`src/tools/` goes back to meaning what Go readers expect.**

### Negative / risk

- **Three Makefiles, three lint configs, three places to run `make test`** — and no CI to run them
  centrally. The `.golangci.yml` copy will drift from `database_administrator`'s unless someone
  notices.
- **`go.work` becomes load-bearing for editor ergonomics.** A contributor who opens a single module
  directory without it will get a degraded `gopls` experience across module boundaries.
- **The D1 row-5 capability is stated but unbuilt.** Anyone who reads row 5 and tries to import the
  agent module from `database_administrator` will break `docker compose build` and discover the
  three-part cost the hard way. The cost is named above; it is not eliminated.
- ~~**AI-39 conflicts with everything in flight.** The import-path rewrite touches 18 files; any open
  branch under `src/tools/agent/ai/` will need a rebase. It must land on a quiet tree.~~
  **Amended 2026-07-31 — this risk did not materialise.** The retired implementation was removed
  rather than moved, so AI-00 created the module empty: no import path was rewritten and no branch
  needed a rebase.
- **A second SKILL.md parser becomes necessary**, because D1 forbids importing
  `domain/skill.go`. That duplication and its guard are the subject of
  [ADR 0006](./0006-resolve-skill-and-prompt-source-of-truth.md) and must not be left implicit.
- **Adding OpenTelemetry to a new module is a real dependency addition**, with its own supply-chain
  and version-skew surface, in a module that is otherwise stdlib-only today.

### Neutral

- The three-layer decomposition, the stateless-loop / stateful-harness separation, the event stream
  as the only inter-layer contract, and the one-way dependency direction are all **unchanged**.
  This ADR moves boxes; it does not redraw arrows.
- ~~The 17 shipped Layer 1 milestones survive the move unmodified. AI-39 is an import-path change,
  not a redesign.~~ **Amended 2026-07-31 — they did not survive.** The implementation was removed
  and Layer 1 was replanned from zero, so AI-00 creates the module empty. This ADR is unaffected:
  its decisions constrain where Layer 1 lives and what it may import, not what had already been
  built inside it.
- Whether `database_administrator` ever drives an agent session remains an open product question.
  This ADR only makes the answer expressible.

---

## References

- [ADR 0004 — Adopt 3-Layer Agentic Architecture](./0004-adopt-tau-3-layer-agentic-architecture.md)
  — the decision this one amends
- [ADR 0006 — Resolve the skill and prompt source-of-truth split](./0006-resolve-skill-and-prompt-source-of-truth.md)
  — companion, closes S7
- [0004 spike — 3-layer architecture](./references/0004-spike-3-layer-architecture.md) — the
  original external narrative, unchanged and deliberately so
- [cachicamas agent stack — hardened architecture (v2)](../architecture/0001-cachicamas-agent-stack-v2.md)
  — the architecture behind § D4's verdicts
- [Layer 1 milestones and task graph](../architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md) —
  ~~AI-39 (this migration), AI-40 … AI-47 (the review's remaining code work)~~ **AI-00 (the module,
  the § D2 layout and both § Enforcement guards) and AI-01 … AI-40 (the rest of Layer 1)**
- Adversarial architecture review, 2026-07-30 — Engram `obs #2243`, topic
  `review/2026-07-30-agent-adversarial`
- [OpenTelemetry semantic conventions for GenAI](https://opentelemetry.io/docs/specs/semconv/gen-ai/)
  — the § D3 attribute vocabulary
