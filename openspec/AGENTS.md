# AGENTS — cachicamas

> Authored by `sdd-init`. Read this before running any SDD phase (`sdd-explore`,
> `sdd-propose`, `sdd-spec`, `sdd-design`, `sdd-tasks`, `sdd-apply`, `sdd-verify`,
> `sdd-archive`) or any non-trivial implementation task on this repo.

## How to work in this repo

1. **Stack is Go 1.26.3 + Echo v5 + Postgres 18 + OTel.** Use the project's own
   Makefile targets instead of raw `go test` / `golangci-lint` when possible, so
   `./bin/` and race flags stay consistent:
   - `make test` — `go test -race -v ./...`
   - `make test/cover` — writes `coverage.out` in atomic mode
   - `make lint` — runs `go vet` then `golangci-lint` (auto-installs v2.9.0)
   - `make fmt` — `gofmt` + `goimports`
   - `make build` — compiles `./bin/database_administrator`
2. **Hexagonal layout is sacred — in `database_administrator` and `workspace_syncer`.** Code goes in `backend/<service>/src/{cmd,application,domain,interfaces,otel}/`. `domain/` MUST NOT import `interfaces/` or `otel/`. `application/` orchestrates `domain/` ports and is consumed by `interfaces/http/`.

   **`backend/agent` is the exception and is NOT hexagonal.** It is a *layered* module — `src/ai` (Layer 1) ← `src/agent` (Layer 2) ← `src/coding` (Layer 3) ← `src/cmd/cachicamas` — governed by [ADR 0005 § D1](../docs/adr/0005-promote-agent-stack-to-own-module.md#d1--dependency-rule-v2). `src/coding` is the coding archetype, Layer 3's first occupant (the archetype layer — [ADR 0009](../docs/adr/0009-redefine-cachicamas-as-a-multiplayer-agentic-system.md)). Applying hexagonal rules to it produces nonsense. Its own targets: `cd backend/agent && make test | lint | fmt | build`.

   **The agent module imports nothing from the other two, in any package including `cmd/`.** Go enforces this (a module cycle is a build error); the reverse direction — anything outside `application/` and `cmd/server` naming `github.com/cachicamas/backend/agent/…` — is enforced by a test and belongs on every review checklist.
3. **Observability is via OTLP env vars.** Do not hardcode endpoints. Follow the pattern in `src/otel/otel.go` and `src/otel/logging.go` (`slog` + `otelslog` bridge).
4. **Conventional commits only.** No `Co-Authored-By` trailer, no AI attribution.
5. **New top-level dependency ⇒ ADR first.** Document rationale, alternatives, rollback.

## SDD artifact layout

```
openspec/
├── project.md                ← this bootstrap (stack + conventions)
├── AGENTS.md                 ← how-to for SDD agents (this file)
├── config.yaml               ← project-specific SDD rules (already exists)
├── specs/                    ← source of truth (main specs) — populated as changes land
└── changes/
    ├── archive/              ← completed changes (YYYY-MM-DD-{change-name}/)
    └── {change-name}/        ← active change folder
        ├── proposal.md
        ├── specs/{domain}/spec.md
        ├── design.md
        ├── tasks.md
        └── verify-report.md
```

Read first: `openspec/project.md` (this directory) + `openspec/config.yaml` (rules).

## Strict TDD is on

This project enforces RED-GREEN-REFACTOR for any new behavior in the Go service.

1. **Red** — write the failing test FIRST. Test must compile and run; assertion must fail
   for the right reason.
2. **Green** — write the MINIMUM production code that makes it pass. No gold-plating.
3. **Refactor** — clean up while green. Re-run `make test` after every change.

Apply this to:
- New domain logic → test in `domain/`
- New use cases → test in `application/`
- New HTTP handlers → test in `interfaces/http/` (see `health_handler_test.go` for the pattern)
- New observability wiring → test by reading exported configuration, not by booting OTel

Do NOT skip straight to implementation, even for "obvious" fixes. The test is the spec.

## Sub-agent launch contract (when delegating)

When you (the orchestrator) launch sub-agents, pass exact `SKILL.md` paths — do NOT
pass generated summaries. Resolve from `.atl/skill-registry.md` (project-level) or
`~/.claude/skills/` (global). The relevant skills for this project:

| Skill             | Path                                                           | When to load |
| ----------------- | -------------------------------------------------------------- | ------------ |
| `go-testing`      | `/Users/braejan/.claude/skills/go-testing/SKILL.md`             | Any Go test work |
| `test-driven-development` | `/Users/braejan/.claude/skills/test-driven-development/SKILL.md` | RED-GREEN-REFACTOR phases |
| `work-unit-commits` | `/Users/braejan/.claude/skills/work-unit-commits/SKILL.md`   | Implementation, commit splitting, chained PRs |
| `chained-pr`      | `/Users/braejan/.claude/skills/chained-pr/SKILL.md`            | PRs over 400 lines, stacked PRs |
| `branch-pr`       | `/Users/braejan/.claude/skills/branch-pr/SKILL.md`             | Creating/opening PRs |
| `cognitive-doc-design` | `/Users/braejan/.claude/skills/cognitive-doc-design/SKILL.md` | Writing/validating review-facing docs |
| `skill-creator`   | `/Users/braejan/.claude/skills/skill-creator/SKILL.md`         | Authoring new SKILL.md |

Skills read full `SKILL.md` (source of truth) before task work.

## Hard rules

- **Do not modify `docker-compose.yaml` or `infra/`** unless the active change is
  explicitly about infrastructure. Init never touches these.
- **Do not modify `backend/database_administrator/src/`** unless running `sdd-apply`
  for a change that targets it.
- **Do not commit during init** — only files written are `openspec/project.md`,
  `openspec/AGENTS.md`, and (separately) the Engram observation `sdd-init/cachicamas`.
- **Do not propose new top-level Go dependencies without an ADR** — log the decision
  in Engram (topic_key `adr/<short-name>`) and reference it from the proposal.
- **Do not skip the TDD red step** — a test written after the implementation does
  not satisfy this repo's discipline.

## Substrate preservation in `backend/agent` (NFR-TLS-003)

Ten files under `backend/agent/src/agent/` form the invariant substrate for
the Layer-2 event system: `event_descriptor.go`, `stream_check.go`,
`failure.go`, `sequence.go`, `event.go`, `go.mod`, `go.sum`, `Makefile`,
`.golangci.yml`, `import_boundary_test.go`. A milestone that extends the
event or scheduler surface MUST leave these ten files byte-unchanged and
widen `TestTurn_SubstrateUntouched`'s (`loop_test.go`) exclusion filter to
cover its own new file pair instead of editing them directly — verify with
`git diff` against the list before opening a PR.

**AG-10** (`cachicamas-agent-permission-protocol`) is the first milestone to
record this pointer here; the convention itself predates it (carried
byte-clean through AG-06.1, AG-07, AG-08, AG-09) but was never previously
written into this file. Future milestones extending the same surface:
append your own one-line pointer to this section rather than assuming the
next agent already knows the list.

**AG-11** (`cachicamas-agent-turn-termination`) is the first milestone
since AG-10's own pointer above to actually modify one of these ten
files: `failure.go` gains a nil-safe `PartialOutput() bool` accessor
(R-ATT-006), for the same structural reason `R-LSK-004`/`R-APP-012`
record — the accessor set is local to `failure.go`. The release is
scoped to AG-11 only, exact-filename, and does not extend to any later
milestone without its own recorded delta.

**CH-07** (`cachicamas-chat-store-adapter`) is the first chat-archetype
milestone to admit new top-level deps in `backend/agent` — `pgx/v5` and
`pressly/goose/v3` — via the `ch07_carveout_test.go` carve-out pattern.
Nine existing `backend/agent/src/agent/*_test.go` files gain an
`isCH07GoModDrift(...)` clause so their `TestTurn_SubstrateUntouched`
checks route the go.mod drift through the helper instead of failing
outright; `import_boundary_test.go` widens its chat allowlist per
R-AGP-003. The pattern is consistent with `ch03_carveout_test.go` and
`ch04_carveout_test.go` and is documented in ADR
`0010-add-pgx-and-goose-to-backend-agent.md`. Future chat or
chat-adjacent milestones admitting new `go.mod` deps in `backend/agent`
MUST follow the same pattern (or its milestone-specific equivalent)
and SHOULD append a one-line pointer here.

**CH-08** (`cachicamas-chat-resume-in-browser`) is the first
chat-archetype milestone after CH-07 to widen the closed
`ConversationStore` port (R-CCS-010) by adding — not replacing — a
third method `List(participantID) ([]ConversationSummary, error)`, in
the same declaration; the new method is additive per R-CCS-010's
anticipatory clause, and the project keeps the additive pattern as
the binding precedent for any future widening (per R-AGP-003 / R-AGP-005).
The CH-08 read surface lands behind the existing `identityMiddleware`
via a fresh `RegisterResumeRoutes` helper, leaving the CH-03 frozen
three-route surface untouched. No file under `backend/agent/src/agent/`
is modified; the ten-file substrate list survives byte-clean. No new
top-level Go deps (CH-07's `pgx/v5` and `pressly/goose/v3` already
cover the postgres surface). The read-side wire is closed under
REQ-7 — the frontend's `ChatStreamEvent` union is preserved; new DTOs
(`ExchangeDTO`, `ConversationSummary`) live adjacent to it, never as
additions to it.

## Review checklist (for reviewers)

- [ ] reviewer can confirm the Makefile targets listed above exist and match this file
- [ ] reviewer can confirm hexagonal boundaries (domain has no `interfaces` imports)
- [ ] reviewer can confirm `sdd-apply` agents were told to load `go-testing` and `test-driven-development`
- [ ] reviewer can confirm no PR was opened or commit created during init
- [ ] reviewer can confirm `openspec/project.md` and `openspec/AGENTS.md` are the only new files