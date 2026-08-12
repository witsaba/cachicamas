# Exploration — AG-03 package scaffold and boundary guards

> Milestone AG-03 (Layer 2 Wave 1), doc `docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md` lines 329-409. Artifact store: openspec only (Engram unavailable this session).

## 1. The AI-00 precedent

`openspec/specs/agent-module-scaffold/spec.md` is AI-00's live spec (Layer 1's module scaffold). Key finding: **AI-00 has only two guards — forward (R-AGM-005, AI-00.3) and reverse (R-AGM-006, AI-00.4). It has no no-ambient-authority guard of its own.**

- AI-00's spec explicitly defers ambient-authority scanning to a later, adapter-scoped milestone: *"AI-25.2's ambient-authority scan over the adapter package"* (spec.md:227), also stated in doc 0002 at AI-00.3's Out-of-scope field (0002:352).
- R-AGM-005 (forward guard): deny-by-default allowlist over `go list -deps -test`, three groups (stdlib / own-module / third-party), named forbidden-prefix list, closed on 3 recorded bites (S-AGM-044/045/046).
- R-AGM-006 (reverse guard): `database_administrator/src/domain/imports_test.go`, closed on 1 recorded bite (S-AGM-055).
- "Method — SDD milestone rules" (cited by both AI-00 and AG-03) is **in doc 0003 itself** (0003:114-125), not doc 0002 as the milestone text might suggest — it cross-references doc 0002's node grammar / leaf anatomy / split triggers sections under different heading names (e.g. `## Rules for every future SDD milestone` at 0002:125).
- Normative source for AG-03.2/AG-03.3, doc 0003:121-122:
  > "The import guard is Layer 1's `go list`-allowlist mechanism (`backend/agent/src/ai/import_boundary_test.go`), retargeted..."
  > "The no-ambient-authority guard is an AST scan in the style doc 0002 establishes at AI-25."

So AG-03.2 retargets AI-00.3's mechanism; **AG-03.3 has no AI-00 precedent** — its real precedent is AI-25.2, a different, adapter-scoped guard (see §7).

## 2. The actual guard code

**Forward guard** — `backend/agent/src/ai/import_boundary_test.go` (515 lines, package `ai_test`):
- Enumerates the import closure via `go list -deps -test -f '{{if not .Standard}}{{.ImportPath}}{{end}}'` (`listNonStdlibDeps`, line 366) — not AST parsing, not literal string matching. Stdlib excluded via the toolchain's own `.Standard` field (a "no dot in first segment" heuristic was explicitly rejected, since vendored `golang.org/x/...` paths inside `go list std` would misclassify).
- `-test` output synthesizes three shapes (`pkg [pkg.test]`, `pkg_test [pkg.test]`, `pkg.test`) that `normalizeListedPackage` (line 406) strips before matching.
- Forbidden prefixes are checked **before** the allowlist (`matchForbidden`, line 342) — order matters, otherwise `.../agent/src/agent` would match the allowed `.../agent` own-module prefix.
- Bite proof: a `_test.go` file; deliberate-violation bites (S-AGM-044/045/046) were done via a scratch file, `go test` run, failure recorded, file removed — no permanent bite fixture stays in the repo. Evidence gate is `make test` (`go test -race -v ./...`), not `go generate` or a separate lint target.
- Second, narrower guard in the same file: `TestRequestPath_DependencyClosure_ContainsNoNetworkOrFilesystemPackage` (line 454, AI-10.4) asserts a *different* path's full closure excludes `net`, `net/http`, `os`, `io/fs`. Its own comment (432-453) explains why an import-closure scan is the wrong tool for a true ambient-authority guard.

**No-ambient-authority guard** — does not exist for AI-00/Layer 1 as a package-wide guard. Real precedent is AI-25.2: `backend/agent/src/ai/openaicompat/ambient_authority_test.go` and its copy `backend/agent/src/ai/openaicompat/openrouter/ambient_authority_test.go`. See §7 — a `go/ast` call-site scan, not an import-closure scan.

## 3. The "doc-guard byte-suffix convention" — does not exist as a named, worked mechanism

Repo-wide search for "byte-suffix" (case-insensitive) finds exactly one relevant hit, and it is a parenthetical example, not a specification:

> `docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md:155` — *"**Guard leaf** | `[guard]` | A mechanical check (import scan, AST scan, byte-suffix scan) that must keep failing forever when violated."*

AG-03.1's clause — *"The doc comment carries the doc-guard byte-suffix convention doc 0002's guards establish, so later milestones append guarded paragraphs the same way"* — cites something that, as written in doc 0002, is a one-line example inside a table cell, not an established pattern with rules. The same phrasing already existed verbatim in the archived v1 of doc 0003 (`archive/0003-...-v1.md:227`), so the ambiguity predates the v2 restructure — it is not a transcription error introduced this wave.

Closest **actual working precedent** for a doc-comment content guard: `backend/agent/src/ai/openaicompat/openrouter/conformance/doc_matrix_guard_test.go` (AI-40.2, R-L2H-004) — reads `src/ai/doc.go`'s raw bytes via `os.ReadFile`, regex-matches tab-indented `// CAP-...` lines, diffs them against a committed Go expectation table. Proves the doc comment's *content* stays byte-accurate to a source of truth, but it's a line-pattern scan, not literally a trailing-bytes "suffix" check.

**This is a real spec gap, not something to resolve by assumption** — flagged to the orchestrator/user rather than invented.

## 4. Current backend/agent layout (verified, not assumed)

- `backend/agent/src/agent/` — confirmed does not exist.
- `backend/agent/src/coding/`, `backend/agent/src/cmd/` — confirmed absent.
- `backend/agent/src/ai/` — exists (large tree, incl. `openaicompat/` and `openaicompat/openrouter/`).
- `backend/agent/src/agenttest/` — exists, direct sibling of `src/ai/`, with `sweep/` and `tracetest/` subpackages confirmed.
- `backend/agent/src/handoff/` — exists (`doc.go`, `handoff_test.go`). Grep for `src/handoff` across the module finds it named only in a forbidden-prefix comment (`import_boundary_test.go`) and a doc comment reference (`agenttest/doc.go`) — **no `.go` file actually imports it**, confirming the milestone's claim rather than assuming it.
- Top-level module files: `.gitignore`, `.golangci.yml`, `Makefile`, `README.md`, `go.mod`, `go.sum`.

## 5. ADR 0005 §D1/D2/D3

`docs/adr/0005-promote-agent-stack-to-own-module.md`:

- **§D1** (row 2, `:166`): Layer 2 (`.../agent/src/agent`) MAY import Layer 1 `src/ai`, Go stdlib, OTel API. MUST NOT import Layer 3, `cmd`, any other module, OTel SDK, **`os` environment reads, the filesystem, `net/http`** — *"Layer 2 performs no I/O of its own, it calls `ModelProvider` and `Tool`."* Direct ADR source for AG-03.3's charter clause.
- **§D2** (`:198-228`): `backend/agent/src/agent/` = Layer 2 (`cachicamas_agent`). Load-bearing constraints: `src/agenttest/` must stay a direct sibling of `src/ai/` (Guard C resolves `../ai/provider.go` via `runtime.Caller(0)`); `src/tools/tools.go` must not move/rename.
- **§D3 table** (`:237-244`), exact permitted OTel API paths per layer:
  - `go.opentelemetry.io/otel` (global getter), `/trace`, `/attribute`, `/codes`, `/metric` — permitted for L1/L2/L3/cmd.
  - `go.opentelemetry.io/otel/sdk/…`, `…/exporters/…` — cmd only.
  - `go.opentelemetry.io/contrib/bridges/otelslog` — L3/cmd only.
  - `backend/database_administrator/src/otel` — nowhere.

**Nuance for AG-03.2's design**: the ADR permits root `otel` and `otel/metric` for Layer 2, but L1's actually-shipped guard (`import_boundary_test.go:127-149`) deliberately **excludes** root `otel` (L1 takes an injected `TracerProvider`, never the global getter) and doesn't include `otel/metric` at all (L1 only imports `trace`, `attribute`, `codes`, plus forced transitives `semconv`, `xxhash/v2`). Design must decide: full ADR-permitted set, or L1's narrower actually-used subset — and since AG-03 charter is "Out of scope: any event, loop, or harness behavior" (i.e. zero OTel usage yet), whether to admit any OTel entries at all before they're used.

## 6. The "measured free of direct network access, 2026-08-11" claim

Traces to doc 0003's own Research digest table, **not** to AG-00 or AG-01's archived `decision.md` (checked both — neither mentions network access or `net/http`):

> `docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md:81` — *"`backend/agent/src/ai`'s transitive import closure is free of direct network access in both the production and test closures — but the vendor subtree `src/ai/openaicompat/**` and `src/ai/internal/retry` do reach it."* Source: *"Measured 2026-08-11: `go list -deps` and `go list -deps -test` over the shipped module."*

This is a point-in-time manual measurement recorded in doc 0003's own prose (its Phase-1 pre-implementation audit), not a live mechanical check — nothing currently guards against it becoming false. **AG-03.2's apply phase should re-run `go list -deps` / `go list -deps -test` over `backend/agent/src/ai/...` fresh**, not trust the recorded number as still true.

## 7. Existing ambient-authority guard precedent — both adapter-scoped, neither is AI-00's

- `backend/agent/src/ai/openaicompat/ambient_authority_test.go` — AI-25.2's own guard (doc comment: *"This file is AI-25.2's [guard] node (R-APC-008)"*), scoped to package `openaicompat`.
- `backend/agent/src/ai/openaicompat/openrouter/ambient_authority_test.go` — verbatim-mechanism copy applied to the `openrouter` sub-package.

Both explain why an import-closure scan (AI-00.3's mechanism) can't express ambient authority: the required `net/http` transport transitively imports `os`, so a closure scan must either admit `os` everywhere (missing a narrow `os.Getenv` call) or forbid it everywhere (false-positiving on legitimate `net/http` use). AI-25.2's corrected mechanism (doc 0002 amendment A1, at `0002:1549`) is a **call-site AST scan**:
- `go/parser.ParseFile` over each non-test `.go` file in the package directory, resolving `import` declarations to local identifiers (alias- and dot-import-aware), then `ast.Inspect` walking for `*ast.CallExpr` whose `*ast.SelectorExpr` base resolves to a forbidden package.
- Forbidden set: `os`, `os/exec`, `syscall`, `io/ioutil`.
- File selection: non-`_test.go` `.go` files only (`isAdapterSourceFile`), uniformly excludes the guard's own source file.
- Bite proof: `t.TempDir()` + a planted scratch `.go` file with a real `os.Getenv` call, scan run, assert ≥1 violation naming file/line/package, `t.Cleanup` removes it — directly reusable for AG-03.3.
- Recorded limitation (both files, explicit): no type information — a local variable/param/field literally named `os` not referring to the import would false-positive. Accepted because adding `go/types`/`x/tools/go/analysis` would add a non-stdlib dependency no milestone currently authorizes.

**Doc 0003:122 explicitly directs AG-03.3 to follow "the style doc 0002 establishes at AI-25"** — the call-site AST scan, not AI-00's import-closure guard. This matters because AG-03.2 already forbids `net/http` by name, so Layer 2 won't pull `os` in transitively the way the adapter package does — but `os` is stdlib and thus passes the forward guard's `.Standard` filter regardless, so a bare `os.Getenv` call inside Layer 2 would pass AG-03.2 silently. AG-03.3 exists specifically to close that gap.

## 8. Test conventions

- `~/.claude/skills/go-testing/SKILL.md` — generic Go testing style guide (table-driven tests, `t.TempDir()`, golden files), not specific to this repo's guard-leaf conventions.
- `backend/agent/Makefile`: `test` target (108-110) → `go test -race -v ./...`; `lint` target (132-138) → `go vet ./...` then `golangci-lint run --config=.golangci.yml ./...` pinned at v2.9.0. Matches doc 0003:120's stated evidence gate: *"one command closes a leaf — `make test` in `backend/agent/`."*
- Strict TDD is active (`openspec/config.yaml`'s `apply.tdd: true`; also cited in `agent-module-scaffold/spec.md:212-217`): guard leaves are RED-first by construction (closing condition is a recorded red run against a deliberate violation, then green); mechanical leaves (AG-03.1) are exempt from red-green but never from their Check-list evidence.

## Affected areas (not yet created)

- `backend/agent/src/agent/` — new package, doc comment (AG-03.1).
- `backend/agent/src/agent/import_boundary_test.go` (or equivalent) — forward guard (AG-03.2), retargeting `src/ai/import_boundary_test.go`'s `go list -deps -test` mechanism.
- `backend/agent/src/agent/ambient_authority_test.go` (or equivalent) — no-ambient-authority guard (AG-03.3), retargeting the AI-25.2/openrouter call-site AST scan mechanism.
- `backend/agent/src/ai/import_boundary_test.go` — **has a documented, pre-existing self-referential hazard** (lines 82-92): the L1 guard's own scanned-set pattern will start matching the new `src/agent` package the moment it's created, tripping its `{modulePath}/src/agent` forbidden-prefix row with zero real violations. The comment states this must be fixed *in the same change that creates the directory* — this is in scope for AG-03, not a separate milestone.

## Risks

1. **"Doc-guard byte-suffix convention" is underspecified** — no worked mechanism anywhere in doc 0002/0003, only a one-line generic example. Needs an explicit decision before spec/design can proceed (see orchestrator note below).
2. **AG-03.2 must fix the existing L1 guard's self-reference hazard** as part of this change — a cross-cutting edit to already-shipped code, not purely new Layer-2 code, with review-budget implications.
3. **ADR 0005 §D3's OTel path list is broader than what L1 actually ships** — needs an explicit scope decision (mirror L1's narrower subset vs. the full ADR-permitted set vs. zero OTel entries since AG-03 has no OTel usage yet).
4. **The "free of direct network access" claim is unguarded** — re-verify fresh at apply time rather than trust the recorded 2026-08-11 measurement.

## Key Learnings

1. AI-00 (the Layer 1 precedent) never had a no-ambient-authority guard; it was explicitly deferred to AI-25.2, an adapter-scoped call-site AST scan over `openaicompat`, not a Layer-1-wide guard.
2. The forward import guard (`import_boundary_test.go`) uses `go list -deps -test` with toolchain-native `.Standard` filtering, and forbidden prefixes are checked before the allowlist to avoid prefix collisions with sibling layers.
3. Doc 0002's "doc-guard byte-suffix convention" cited by AG-03.1 is only a one-line example in the node-grammar table, with no dedicated specification anywhere in the repository.
4. The "measured free of direct network access, 2026-08-11" claim originates from doc 0003's own Phase-1 research digest table, not from any AG-00 or AG-01 decision artifact, and is not backed by a live guard.
5. Creating `backend/agent/src/agent/` triggers a documented, already-known self-referential hazard in the existing Layer 1 forward guard that must be fixed in the same change per that guard's own source comment.

**Status**: done · **Next recommended**: sdd-propose, pending the byte-suffix-convention decision below.
