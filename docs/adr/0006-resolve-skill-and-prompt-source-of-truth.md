# ADR 0006: Resolve the skill and prompt source-of-truth split

> Status: **Proposed** (2026-07-30, change `cachicamas-agent-skill-prompt-sources`)
> Author: cachicamas SDD pipeline (parent orchestrator)
> Amends: [ADR 0004](./0004-adopt-tau-3-layer-agentic-architecture.md) — supersedes the *Skills*
> and *Custom prompts* rows of its Layer → location mapping
> Companion: [ADR 0005](./0005-promote-agent-stack-to-own-module.md) — that ADR defines the
> module boundary and the `SkillSource` / `PromptSource` **seam**; this one decides the **policy**
> Review source: adversarial architecture review of 2026-07-30 (Engram `obs #2243`, finding S7)

---

## Resolved TOC (jump links)

- [D1 — Two stores, one resolver](#d1--two-stores-one-resolver) — the shape of the decision
- [D2 — The ports and their implementations](#d2--the-ports-and-their-implementations)
- [D3 — Precedence and shadowing](#d3--precedence-and-shadowing) — which one wins
- [D4 — The accepted parser duplication](#d4--the-accepted-parser-duplication) — and its guard

## Full TOC

- [Context](#context)
- [Decision](#decision)
  - [D1 — Two stores, one resolver](#d1--two-stores-one-resolver)
  - [D2 — The ports and their implementations](#d2--the-ports-and-their-implementations)
  - [D3 — Precedence and shadowing](#d3--precedence-and-shadowing)
  - [D4 — The accepted parser duplication](#d4--the-accepted-parser-duplication)
- [Alternatives rejected](#alternatives-rejected)
- [Consequences](#consequences)
- [References](#references)

---

## Context

[ADR 0004](./0004-adopt-tau-3-layer-agentic-architecture.md) maps *Skills* to
`.cachicamas/skills/` + `~/.cachicamas/skills/` and *Custom prompts* to a Layer 3 template loader —
both as filesystem artifacts, both inherited from the source narrative.

**Both concepts already ship in cachicamas, in Postgres, with a UI.** ADR 0004 does not mention
either implementation.

| Layer | Skills | Prompts |
| --- | --- | --- |
| Schema | `migration/sql/20260717120000_skills.sql` — `skill`, `skill_revision`; the `body` column comment reads *"SKILL.md file content (YAML frontmatter + markdown body) … Frontmatter MUST contain name and description matching the row"* | `prompt`, `prompt_revision` |
| Domain | `domain/skill.go` — `ParseFrontmatter`, `LockStepCheck`, `skillNameRegex = ^[a-z0-9]+(-[a-z0-9]+)*$` *"matches the agentskills.io spec"*, `MaxSkillDescriptionLen = 1024`, `MaxSkillBodyLen = 524288`, reserved words `anthropic` / `claude` | `domain/prompt.go` |
| Application | `application/skill_service.go` | `application/prompt_service.go` |
| Infrastructure | `infrastructure/postgres/skills/` | `infrastructure/postgres/prompts/` |
| HTTP | `interfaces/http/skill_handler.go` — 7 routes (`POST/GET/PATCH/DELETE /skills`, `GET /skills/:name`, `GET /skills/:name/revisions`, `POST /skills/:name/revisions/:n/restore`) | equivalent prompt routes |
| UI | `frontend/src/routes/settings/skills/` — "Skill Studio" | `frontend/src/routes/settings/prompts/` |
| Spec | — | `openspec/specs/prompts/spec.md` — *"first-class, versioned, slug-addressable **LLM prompt bodies** stored in PostgreSQL"* |

This is not a naming collision. It is the **same concept, in the same format** — agentskills.io
SKILL.md, YAML frontmatter with `name` and `description` — with two sources of truth and no stated
relationship between them.

Worse, ADR 0004's own dependency rule blocked the obvious resolution: reading skills from Postgres
would have required Layer 3 to import `domain`, `application` and `infrastructure`. ADR 0005's
module boundary makes that impossible by construction rather than by convention, which forces this
decision to be made explicitly instead of drifting.

---

## Decision

### D1 — Two stores, one resolver

**Neither store is authoritative over the other. Conflict is resolved by an ordered chain, never by
synchronisation.**

- **Postgres is the system of record for org-shared, versioned, audited skills and prompts.**
  Reachable through `database_administrator`'s existing routes. Skill Studio remains the authoring
  surface. **Nothing about the current implementation changes.**
- **The filesystem is the system of record for repo-local and user-local skills and prompts** —
  `.cachicamas/skills/`, `~/.cachicamas/skills/`, `.cachicamas/prompts/`. It is **not a cache of
  Postgres**.
- **The agent never writes back to Postgres.** All sources are read-only from the agent's
  perspective. Authoring happens in Skill Studio or in a text editor, never as a side effect of a
  session.

The last point is what keeps this from becoming a sync problem. A cache needs invalidation, a
mirror needs reconciliation, and a bidirectional store needs conflict resolution. A read-only
ordered chain needs none of those.

### D2 — The ports and their implementations

| Port | Location | Implementations |
| --- | --- | --- |
| `SkillSource` (read-only, context-aware) | `backend/agent/src/coding/skill/` | `skill/fsource` — filesystem · `skill/httpsource` — the org catalog over HTTP · `skill/chain` — ordered fan-in |
| `PromptSource` | `backend/agent/src/coding/prompt/` | the same three |

Both ports are defined in **Layer 3**, which is where resource loading belongs under ADR 0004's
original split and ADR 0005 § D1 row 3. Layer 2 never learns that skills exist; it receives a
system prompt and a tool list.

**`httpsource` depends only on `net/http` and the wire JSON — never on `database_administrator`'s
Go types.** That is what keeps ADR 0005 § D1's arrow one-way. The cost is real and is stated here
rather than discovered later: the wire shape becomes a cross-module contract with **no compiler
check**. It is guarded the same way § D4 guards the parser — a checked-in golden pair plus an
opt-in contract test.

`httpsource` is also **optional at runtime**. A developer running the CLI against a repo with no
cachicamas backend reachable gets the filesystem sources and a degraded-but-working session; an
unreachable catalog is a warning on the session stream, not a startup failure.

### D3 — Precedence and shadowing

| Precedence | Source | Rationale |
| --- | --- | --- |
| 1 — wins | repo `.cachicamas/skills/` | A developer editing a skill on a branch must override the org's copy without a round trip through Skill Studio |
| 2 | user `~/.cachicamas/skills/` | Personal overrides beat org defaults |
| 3 | org HTTP catalog (Postgres) | The shared default |

A name present in more than one source is **not an error**. The chain resolves it by precedence and
emits a **shadowing event** on the session stream, carrying the name, the winning source, and the
shadowed ones.

That event is not decoration. ADR 0004's *Positive* consequences promise an "honest TUI sidebar"
that "can enumerate exactly what the LLM receives." A silent precedence rule would break that
promise the first time a stale local skill shadowed a corrected org one — the sidebar would show a
skill name that is truthful and a skill body that is not.

### D4 — The accepted parser duplication

ADR 0005 § D1 forbids `backend/agent` from importing `domain/skill.go`. `ParseFrontmatter` and
`LockStepCheck` implement the SKILL.md frontmatter contract, so **a second implementation of that
contract will exist in `backend/agent/src/coding/skill/`** — roughly 60 lines.

**This duplication is accepted deliberately.** Extracting a third shared Go module to hold 60 lines
of frontmatter parsing costs more than it saves: a new `go.mod`, a fourth place to run `make test`,
a version-skew surface between three consumers, and a release step for every parser tweak.

**It is accepted only because it is guarded.** Without a guard it is a latent bug, and a
particularly unpleasant one: the failure mode is a skill that validates in Skill Studio and
silently fails to load in the CLI, or the reverse — no error, no log, just a capability that is
mysteriously absent. The guard is:

- **A shared golden corpus** at `docs/architecture/testdata/skillmd/` — valid documents, and
  invalid ones paired with the sentinel each parser must return. Every case exercises a rule the
  two parsers must agree on: the name regex, the 1024-character description cap, the 524288-byte
  body cap, the reserved words, rune-versus-byte counting, and frontmatter/row lock-step.
- **Both modules' tests read that corpus**, from the repo root, and a checksum test fails if the
  corpus changes without both test suites being re-run.

The corpus is the contract. Neither parser is the reference implementation; the files are.

---

## Alternatives rejected

**Postgres as the only source.** Would make the CLI unusable offline and unusable against a repo
whose skills are still on a branch — the single most common authoring case. Also forces a backend
round trip into agent startup.

**Filesystem as the only source, with Skill Studio exporting to disk.** Discards the versioning,
revision history, and audit trail that `skill_revision` already provides, and leaves the shipped
Skill Studio with no consumer.

**Sync the two stores.** Requires invalidation, conflict resolution, and a writer on the agent
side. Every one of those is a new failure mode, and none of them buys anything the precedence chain
does not already provide.

**Extract a shared `skillmd` Go module.** The honest alternative to § D4, and the closest call in
this ADR. Rejected on cost: a fourth module and a release step for 60 lines. Revisit if the parser
grows a third consumer or exceeds roughly 200 lines — at that point the corpus stops being cheaper.

---

## Consequences

### Positive

- **Skill Studio and the shipped Postgres implementation survive untouched.** This ADR adds a
  consumer; it does not migrate or deprecate anything.
- **The CLI works offline and works on a branch.** Local skills win, so the edit-test loop is a
  file save rather than a round trip.
- **The one-way module arrow is preserved** without giving up access to org-shared resources,
  because the coupling is HTTP and JSON rather than a Go import.
- **The precedence rule is visible.** Shadowing is an event, so the TUI sidebar's honesty promise
  survives contact with two stores.

### Negative / risk

- **Two implementations of the SKILL.md contract exist**, and they will drift the moment someone
  changes one without the other. The golden corpus is the only thing standing between that and a
  silent capability mismatch. If the corpus is allowed to rot, this decision becomes a bug.
- **The `/skills` wire shape becomes a cross-module contract with no compiler check.** A field
  rename in `skill_handler.go`'s response breaks the agent at runtime, not at build time.
- **Three sources multiply the "why is this skill not loading?" support surface.** The shadowing
  event mitigates it; it does not remove it.
- **`httpsource` needs credentials** to reach an authenticated `/skills` route. Session
  authentication for the CLI is out of scope here and is a real prerequisite for precedence
  level 3 to work at all.

### Neutral

- Prompts follow skills exactly. They are a separate port and a separate precedence chain over the
  same three source kinds, with no shared code beyond the chain pattern.
- Project instructions (`AGENTS.md`, `.cachicamas/`, `.agents/`) remain **filesystem only** and are
  unaffected. They are repo documents, not an org-shared versioned resource, and nothing in the
  backend models them.
- The decision does not depend on which store a given team actually uses. A team that never opens
  Skill Studio gets levels 1 and 2; a team that authors only in Skill Studio gets level 3.

---

## References

- [ADR 0004 — Adopt 3-Layer Agentic Architecture](./0004-adopt-tau-3-layer-agentic-architecture.md)
  — the Skills and Custom prompts rows this ADR supersedes
- [ADR 0005 — Promote the agent stack to the `backend/agent` Go module](./0005-promote-agent-stack-to-own-module.md)
  — defines the module boundary that forces this decision, and the § D1 rule that forbids importing
  `domain/skill.go`
- `backend/database_administrator/src/domain/skill.go` — the shipped SKILL.md contract
  (`ParseFrontmatter`, `LockStepCheck`, the caps and the reserved words)
- `backend/database_administrator/src/interfaces/http/skill_handler.go` — the 7 routes
  `httpsource` consumes
- `openspec/specs/prompts/spec.md` — the shipped prompts capability
- [agentskills.io specification](https://agentskills.io/specification) — the format both parsers
  implement
- Adversarial architecture review, 2026-07-30 — Engram `obs #2243`, finding S7
