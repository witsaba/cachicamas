# Spec — the Layer 3 readiness contract

> **Change**: `cachicamas-agent-layer3-handoff` · **AG-23** (`docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md:2095-2155`) · Wave 6 · **Layer 2's exit — the surface freezes here**
> **Capability**: `agent-layer3-handoff` (**new** — full spec; no MODIFIED/REMOVED/RENAMED inside this file)
> **Requirement IDs**: `R-L3H-0NN` · **Scenario IDs**: `S-L3H-0NN`
> **Prefix freedom, re-verified at mint time this phase**: `rg 'L3H|R-L3H|S-L3H' openspec/` → **1 file, `openspec/changes/cachicamas-agent-layer3-handoff/proposal.md`**, i.e. this change's own folder. **Zero** occurrences under `openspec/specs/` and **zero** under `openspec/changes/archive/`. None of the 24 taken prefixes (AGE, AGO, AGM, AEV, CNH, APE, AGP, ATT, DEL, AGS, HKS, HIS, AGV, CST, AMT, CAN, RUN, RTY, CMP, CTX, LSK, APP, TLS, PRH) collides.
> **Format**: RFC 2119 + Given/When/Then per `openspec/config.yaml` `rules.specs`. Every scenario is independently verifiable.
> **Precedent, one layer down**: `openspec/specs/ai-layer2-handoff/spec.md` (AI-40) — the same milestone shape for Layer 1's exit.
> **Binding decisions**: proposal `D-1`…`D-8` as **corrected by** design `AD-1`…`AD-8`. Where the two disagree, the design binds: it corrected the proposal twice (the determinism claim, `AD-3`; the stale-row count, `AD-6`).
> **Sibling deltas in this change** (each owns an amendment this capability depends on, and none of them is restated here): `agent-package-scaffold` (`R-AGP-003` — the guard extension), `agent-run-driver` (`R-RUN-014` — the forwarder's bounded exit), `agent-hook-taxonomy` (`R-HKS-010` — the W-6 freeze disposition), `agent-v1-scope` (`R-AGS-016` — the declined removal).
> **Composes, cited and never modified here**: `agent-event-envelope` (`stream_check.go`'s exported validator, whose own requirement already names this milestone as its consumer), `agent-history` (seeded history and transcript entries), `ai-fake-provider` (Layer 1's scripted provider), `agent-permission-protocol`, `agent-tool-scheduler`.

---

## Purpose

Layer 2 is feature-complete and **unconsumed**. Every helper a Layer 3 application would need in order to
test against the harness — the scripted permission policy, the no-op policy, the scripted tool — is
declared inside a `_test.go` file, which the Go toolchain makes unimportable from any other package at
all. So the claim *"Layer 3 can build on this"* is today **unfalsifiable**: nothing outside Layer 2's own
package has ever built a harness.

This capability states what MUST be true for Layer 2 to declare itself handed off: an external package
that is neither Layer 2 nor its Layer 1 substrate builds a harness from injected fakes and drives the
seven acceptance capabilities in one run; the substrate that consumption needs ships as an importable,
deterministic kit; runnable examples are executed by the ordinary test command; the v1 surface is
enumerated, frozen and free of any concept specific to one kind of application; the completion checklist
is walked to its closing nodes; the deferred capabilities are registered with post-v1 paths; and every
obligation an earlier milestone forwarded **to** AG-23 is resolved in place, because there is no
milestone after this one to forward it to again.

## Definitions

- **The consumer proof** — the external-package test standing in for a future Layer 3 application in miniature.
- **The kit** — the importable, non-test-file scripting substrate: a scriptable permission policy, a scriptable tool, and a drain helper that delegates validation wholesale to the already-exported stream validator.
- **The guard mechanism** — Layer 2's single, pre-existing, deny-by-default import-boundary guard, extended (never cloned) by this change's `agent-package-scaffold` delta.
- **The compatibility statement** — the durable `decision.md` artifact carrying the frozen-surface enumeration, the checklist walk, the known-limitations register, and the resolutions of `R-L3H-009`.
- **The generic-client boundary** — the rule that nothing in the proof, the kit, the examples or the statement may name a concept that only one kind of application would need (`0003:112`, `0003:2101`, `0003:2152`).
- **A minted identity** — a run or turn identifier produced by a process-global counter, and therefore not reproducible across two runs in the same process.

---

## Requirements

### R-L3H-001 — A third-party consumer builds a harness from injected fakes and drives all seven acceptance capabilities in one run

An external test package that is **neither** Layer 2's own package **nor** its Layer 1 substrate package
MUST construct a harness **through the public surface only**, from injected fakes, and MUST exercise, in
one ordered run of stages over shared state: (1) construction from injected fakes; (2) a multi-turn
conversation with tool execution; (3) a permission suspension resolved by script; (4) an interrupt;
(5) a resumed prompt after that interrupt; (6) a second harness constructed over the first's transcript
through the seeded-history route; (7) a full drain of every run's event stream, validated by the exported
stream validator with a clean report.

The proof MUST NOT reference any Layer 2 internal, MUST NOT rely on a helper declared in a Layer 2 test
file, and MUST NOT construct its stages in parallel — the transcript hand-off from stage 6 is ordered, and
a parallel stage would make the hand-off's own ordering unobserved rather than proven.

The proof's driver MUST be a single test whose stages are named for the capability each one discharges, so
a reader can map the charter's acceptance sentence onto the shipped stages without inference.

#### Scenarios

- **S-L3H-001** — Given the consumer proof's source, when its package clause and directory are read, then it is an external test package living in a directory that is neither Layer 2's package directory nor the Layer 1 substrate's, and it is not nested inside either.
- **S-L3H-002** — Given the consumer proof, when it runs uncached under `cd backend/agent && go clean -testcache && make test` (`make test` itself is `go test -race -v ./...` — the shipped Makefile carries no `-count=1`, and the module proves an uncached run by clearing the test cache first, not by a flag the Makefile does not have), then it passes, and its stage names enumerate the seven capabilities above, one stage per capability.
- **S-L3H-003** — Given stage 2, when its drained stream is read, then it carries more than one turn bracket and at least one tool execution with its result, and the drain's report is clean.
- **S-L3H-004** — Given stage 3, when the scripted policy queues a deferring verdict followed by its resolution, then the drained stream carries the suspension and its resolution in that order, and the queue is fully consumed at the stage's end.
- **S-L3H-005** — Given stage 4, when the run is interrupted while a turn is held at a test gate, then the stream drains to close and the run's terminal outcome attributes closure to the interrupt rather than to completion.
- **S-L3H-006** — Given stage 5, when a further prompt is driven on the same harness value after the interrupt, then it produces its own complete run bracket and the drain's report is clean.
- **S-L3H-007** — Given stage 6, when the first harness's transcript entries are read through the public accessors and used to seed a **second** harness, then the second harness completes a run whose stream validates clean, and no Layer 2 internal was touched to move the transcript.
- **S-L3H-008** — Given the whole proof, when its source is searched for a reference to a helper declared in a Layer 2 test file, then none exists — such a reference would not compile, and the scenario records that this is what makes the sufficiency claim falsifiable rather than asserted.

### R-L3H-002 — Zero vendor imports and zero I/O are proven by the guard mechanism, and the guard is shown to bite

The proof's and the kit's zero-vendor-import property MUST be established by **the** guard mechanism —
Layer 2's single deny-by-default import-boundary guard — rather than by human inspection or by a second,
duplicated allowlist living inside either new tree. The zero-I/O and no-wall-clock property over the two
new trees MUST be established by a source-level scan of those trees' own files, **including their test
files**, that denies the process, network, system-call, filesystem and wall-clock families by name.

The guard's own extension is owned by this change's `agent-package-scaffold` delta and MUST NOT be
restated as a second normative rule here. What this requirement adds is the **closing condition**: the
extension closes on **bite proof**, not on green. A widened pattern set or a widened allowlist that has
not been shown to still fail on a planted violation has proven nothing.

The **production** closure MUST remain unwidened: neither new tree may become admissible to Layer 2's
production allowlist, so Layer 2 production provably cannot import the kit or the proof.

#### Scenarios

- **S-L3H-009** — Given the merged change, when the guard's test-closure check runs, then both new trees are inside its swept pattern set and every dependency they carry resolves to an admitted entry.
- **S-L3H-010** — Given the merged change, when the guard's production-closure check runs, then it passes **without** admitting either new tree, and a source file placed in Layer 2 production that imports either one FAILS the guard naming the unadmitted path and the deny-by-default rule. Recorded, then removed.
- **S-L3H-011** — **(bite)** Given a scratch file planted in each new tree importing the wall-clock package, when the source-level scan runs, then it FAILS naming the file and the family, once per tree. Recorded, then removed.
- **S-L3H-012** — **(bite)** Given a scratch file planted in each new tree importing a process or filesystem package, when the source-level scan runs, then it FAILS naming the file and the family, once per tree. Recorded, then removed.
- **S-L3H-013** — Given the merged diff, when it is searched for a second, self-contained import guard inside either new tree, then none exists; the one guard was extended, and the merged diff contains no scratch violation file.

### R-L3H-003 — The kit is importable from an external package and scripts the four inputs a consumer must control

The kit MUST ship as **production** source — not in a test file, and not behind a build tag — in a package
that an external test in another directory can import. It MUST let a consumer script, without touching any
Layer 2 internal: **provider turns**, **tool results**, **permission decisions**, and **interrupts**.

The scripted permission policy MUST resolve queued verdicts in order and MUST NOT re-implement the
permission gate; the scripted tool MUST record its invocations and arguments for inspection; the drain
helper MUST delegate validation **wholesale** to the already-exported stream validator and MUST NOT
re-implement any part of it. A kit that re-implements the behaviour it exists to exercise proves the
re-implementation, not the surface.

The kit MUST NOT be built by widening the Layer 1 substrate package: that package is frozen byte-unchanged
by a shipped fence, and any implementation of a Layer 2 interface placed inside it would additionally fail
Layer 1's own import guard, which denies Layer 2 by name.

An exhausted verdict queue MUST NOT wedge a run: the policy MUST fall back to a stated default **and**
latch an inspectable exhaustion flag, so a consumer asserts full consumption rather than discovering
starvation as a hang.

#### Scenarios

- **S-L3H-014** — Given the kit's package, when its files are listed, then none of its exported members is declared in a test file and none is behind a build tag; and given an external test package in another directory, when it imports the kit, then it compiles.
- **S-L3H-015** — Given the kit's scripted policy and scripted tool, when each is assigned to the Layer 2 interface it claims to satisfy at compile time, then the assignment compiles — the interface conformance is a build-time pin, not a comment.
- **S-L3H-016** — Given a verdict queue of two entries, when three permission decisions are requested, then the first two resolve in queue order, the third resolves to the stated default, the exhaustion flag reads true, and the run does not wedge.
- **S-L3H-017** — Given the drain helper's source, when it is read, then it blocks until the sink closes, preserves event order, and returns the exported validator's own report unmodified — no re-implemented validation rule appears in it.
- **S-L3H-018** — Given the merged diff, when the Layer 1 substrate package is compared to the merge base, then it is byte-unchanged and the kit ships in a package of its own.

### R-L3H-004 — The kit is deterministic, and determinism is stated as structural equality modulo minted identities

Determinism MUST be established by **two independent mechanisms**, not by a claim in a comment:

1. **Statically** — no direct wall-clock import exists anywhere in either new tree, test files included, so
   a clock is unreferenceable from those trees' own code (`R-L3H-002`'s scan).
2. **Dynamically** — the same script drained twice yields **structurally equal** results.

**Structural equality is defined here rather than left to a reader, because the stronger phrasing is
false.** Run and turn identities mint from process-global counters, so two identical runs **cannot**
produce byte-equal event sequences, and a requirement asserting byte equality would be falsified by the
shipped code on its first run. The dynamic claim is therefore: **identical event-kind sequences, identical
identity-independent payload projections** (outcomes, tool names, message texts, verdict order), **and both
drains reported clean by the exported validator**. Minted identities MUST be excluded from the comparison,
and the comparison MUST NOT be widened to whole event values.

Synchronization inside either new tree MUST be by test gate, channel read or channel close. **No sleep, no
timeout, no deadline, no poll, and no wall-clock ordering** may be used anywhere, in production source or
in test source.

#### Scenarios

- **S-L3H-019** — Given one script, when it is driven twice in the same process and both streams are drained, then the two event-kind sequences are equal, the two identity-independent projections are equal, and both reports are clean.
- **S-L3H-020** — Given those same two drains, when their minted run and turn identities are compared, then they **differ** — and the test asserts the difference, so a future change that made them equal (or a comparison that accidentally included them) is caught rather than silently tolerated.
- **S-L3H-021** — **(defeat)** Given the comparison widened to whole event values, when the repeat-run test runs, then it FAILS on the minted identities — proving the projection is doing the work and the assertion is not vacuous. Recorded, then reverted.
- **S-L3H-022** — Given both new trees' sources, when they are searched for a sleep, timeout, deadline, poll or elapsed-time assertion, then none exists.

### R-L3H-005 — Runnable examples are compiled AND run by the ordinary test command

Package documentation MUST ship runnable examples covering exactly these four subjects, one each:
**building a harness**, **driving a run**, **consuming events**, and **handling a permission suspension**.

Each MUST be a **runnable** example — one that the test run executes and whose output the test run verifies
against a mandatory expected-output block — not a compile-only fragment and not a fenced code block in a
comment. All four MUST be compiled **and run** by `cd backend/agent && make test`, so a drift between an
example and the surface it demonstrates fails the build rather than rotting silently.

No example may print a **minted identity**: identities depend on which tests ran earlier in the same
process, so an example printing one is order-dependent and would fail for a reason that has nothing to do
with the surface it documents. Examples MUST print kind names, outcomes and texts only. No example may
require a credential, perform I/O, or order events with a sleep.

#### Scenarios

- **S-L3H-023** — Given the merged change, when the ordinary test run executes, then four runnable examples are discovered and run, each verifying its own output.
- **S-L3H-024** — Given the four examples, when their subjects are enumerated, then they are exactly building a harness, driving a run, consuming events, and handling a permission suspension — one each.
- **S-L3H-025** — Given any one example's expected output altered by one character, when the test run executes, then that example FAILS; and given the surface it demonstrates changed incompatibly, it fails to compile. Recorded, then reverted.
- **S-L3H-026** — Given the example file, when its printed values are read, then no minted run or turn identity appears in any expected-output block; and when the whole file is run in isolation and again as part of the full suite, then its output is identical in both.

### R-L3H-006 — The v1 surface is enumerated by capability, frozen, and free of application-specific concepts

The compatibility statement MUST enumerate the v1 surface **by capability or behaviour, never by Go
identifier**, declare it frozen as of this milestone, and state exactly what a Layer 3 application may rely
on. For **every seam** it MUST name the injection point and that seam's **v1 default**. Anything not frozen
MUST be marked **experimental** in the same enumeration, so a consumer can tell the two apart without
inference; if nothing is experimental, the statement MUST say so explicitly rather than omitting the
category.

The statement MUST be written **without reference to files, shells, skills or terminals** (`0003:2152`).
Anything one kind of application needs and a different application would not is a **leak**, and naming it
here is cheaper than discovering it at the second application.

Package documentation MUST carry a short pointer section naming that the freeze happened and giving the
statement's location. It MUST NOT duplicate the enumeration, and it MUST NOT be dressed as a new
machine-checked package-contract row: a pointer paragraph is not a falsifiable package-wide contract
clause, and the shipped contract-row guard parses rows against a committed table with a pinned index.

#### Scenarios

- **S-L3H-027** — Given the compatibility statement's surface enumeration, when each entry is read, then it is stated as a capability or behaviour, is marked either frozen or experimental, and no entry is a bare Go identifier.
- **S-L3H-028** — Given the statement, when the seams are enumerated, then every seam names its injection point and its v1 default, and no seam is listed without both.
- **S-L3H-029** — Given the statement, when a reader asks what a Layer 3 application may rely on, then the answer is stated directly and does not require reading source.
- **S-L3H-030** — Given package documentation and the statement, when both are compared, then the documentation carries a pointer and a location, not a second copy of the enumeration.
- **S-L3H-031** — Given the merged change, when the contract-row guard runs, then it passes with its committed row table and its pinned row index **unchanged** — the pointer landed as prose, not as a new row.

### R-L3H-007 — Every completion-checklist row is closed and cites its closing node with merged evidence

At this change's merge, **no row of doc 0003's Layer 2 completion checklist may remain unchecked**, and the
compatibility statement MUST carry a walk in which **every** row cites the node or nodes that closed it,
together with that node's evidence.

**This requirement is deliberately scoped to the property, never to a row count.** A requirement phrased as
*"the checklist contains N rows"* or *"N rows are flipped"* goes silently false the moment a later editor
appends a row, and no test fails. The binding claim is therefore: *every row present at merge is checked,
and every row cites a closing node*. Tasks and apply MUST **re-count the unchecked rows at edit time** and
trust neither the proposal's number nor the design's.

**Self-certification is forbidden.** Every already-shipped row MUST cite **merged** evidence — the closing
milestone's archived change and/or doc 0003's own dated close amendments and merge commits — resolved
against the merge base, **never** against this change. Exactly one row legitimately cites this change's own
verify gate: AG-23's own row, which flips last. A row and its evidence edited together in one change
certify each other and prove nothing.

The walk MUST **report** each row's status as of the milestone that closed it and MUST NOT re-decide or
re-verify the underlying evidence.

#### Scenarios

- **S-L3H-032** — Given doc 0003 after this change, when its Layer 2 completion checklist is scanned for an unchecked row, then **none** is found, and the scan is expressed as a search for the unchecked marker rather than as a comparison against a fixed row count.
- **S-L3H-033** — Given the walk, when every row is read, then each cites at least one closing node, and no row is closed by a blanket sweep covering several rows with one shared citation.
- **S-L3H-034** — Given the walk, when each already-shipped row's evidence is resolved, then it resolves to an archived change, a dated close amendment, or a merge commit that exists at the merge base — and **not** to any artifact created by this change.
- **S-L3H-035** — Given AG-23's own row, when its evidence is read, then it is the only row citing this change, and it is the last row to flip.
- **S-L3H-036** — Given the walk, when a row is inspected for re-verification of an already-closed row's evidence, then none is performed — the row reports status and cites the closing node.

### R-L3H-008 — The known-limitations register is stated with a post-v1 path for each entry

The statement MUST register, as **known limitations of the frozen v1 surface**, at least: **no subagent
tool**, **failover declines**, **the never-compact default**, and **the abandoned-consumer contract
inherited from Layer 1** — each with the seam it attaches to later and its **post-v1 path**.

These entries MUST be **restated, not re-litigated**: the register reports decisions taken elsewhere and
MUST NOT reopen, weaken or re-decide any of them.

**The register is for design limitations only.** A defect — in particular a crash-class defect reachable
through a seam this milestone freezes — MUST NOT be entered in it. Such a defect is fixed, or the milestone
does not freeze the seam. Laundering a defect as a limitation would make the compatibility statement false
on the day it lands.

#### Scenarios

- **S-L3H-037** — Given the register, when its entries are read, then all four named limitations appear, each naming its attaching seam and its post-v1 path.
- **S-L3H-038** — Given the register, when each entry is compared to the decision that produced it, then the entry restates that decision without reopening or weakening it.
- **S-L3H-039** — Given the register, when it is searched for an entry describing a defect rather than a design limitation, then none is present; and given this change's own forwarder defect, when its disposition is read, then it was **fixed** (`agent-run-driver`'s `R-RUN-014`) and appears in the register nowhere.

### R-L3H-009 — Every obligation an earlier milestone forwarded to AG-23 is resolved in place; none is silently dropped

AG-23 is Layer 2's **last** milestone. A carry-forward past it has nowhere to go, so every shipped
requirement that names AG-23 as the holder of a future obligation MUST be resolved in this change —
**delivered, relocated, or declined with its reason recorded** — and the resolution MUST be recorded where a
later reader will meet the obligation, not only in this change's own folder.

The obligations, enumerated so the set is closed rather than open:

| Forwarded obligation | Named at | Resolution AG-23 takes |
|---|---|---|
| Removing the singular pre-request hook field | `agent-hook-taxonomy` `R-HKS-010` consequence 2 and its non-requirements table; `agent-v1-scope` `R-AGS-016`; `agent-pre-request-hook`'s blocks note | **DECLINED and frozen into v1.** AG-23 freezes the surface and removes no exported member; a removal is a breaking change and the milestone that freezes a surface is the worst possible place to take one. The field is recorded in the statement as frozen-and-superseded with a **post-v1** removal path |
| Test-convenience wrappers in the Layer 1 substrate package | `agent-loop-skeleton`, `agent-protocol-events`, `agent-message-tool-events` out-of-scope rows | **DELIVERED, RELOCATED.** The wrappers ship as this change's kit, in a package of its own one layer up. The substrate package stays byte-unchanged, as two shipped fences require and as Layer 1's own import guard makes mandatory |
| The Layer 2 scope fence's released entry (W-6) | `agent-hook-taxonomy` `R-HKS-010` | **CLOSED on the merits** by this change's `agent-hook-taxonomy` delta: one file restored to the fence, one release made **permanent** with its reasoning |
| The forwarder panic race (W-5) | AG-22's carry-forward | **FIXED**, RED-first, by this change's `agent-run-driver` delta (`R-RUN-014`) |

A resolution recorded **only** in this change's folder is not a resolution: the archive path moves and the
next reader arrives through the shipped spec.

#### Scenarios

- **S-L3H-040** — Given every shipped requirement that names AG-23 as the holder of a future obligation, when each is resolved against this change, then each appears in the table above with a delivered, relocated, or declined verdict, and no such requirement is left with an unresolved forward reference.
- **S-L3H-041** — Given the declined removal, when the merged exported surface of Layer 2 is enumerated from an external test package, then the singular pre-request field is **still present**, the shipped exported-surface pins are unchanged, and the compatibility statement records the field as frozen-and-superseded with its post-v1 removal path.
- **S-L3H-042** — Given the relocated wrappers, when the Layer 1 substrate package is compared to the merge base, then it is byte-unchanged; and when the three out-of-scope rows that named AG-23 are resolved, then each resolves to the kit's own package, with the relocation and its reason recorded in the statement.
- **S-L3H-043** — Given a reader who arrives only through the shipped specs, when they follow each forwarded obligation, then the resolution is reachable without opening this change's folder.

### R-L3H-010 — The generic-client boundary is mechanical, and its guard does not trip on its own prose

Nothing in the proof, the kit, the examples or the compatibility statement may name a concept that only one
kind of application would need. A test that only makes sense for one kind of application is a boundary
violation of the same weight as an import violation (`0003:112`, `0003:2101`).

Enforcement MUST be **mechanical**, not a review-time intention, and MUST have **two halves**, because the
two leak classes are caught by different mechanisms:

1. **Capability leaks** are import-caught: any real process, filesystem, terminal or network reach requires
   an import from a denied family, which `R-L3H-002`'s scan denies by name.
2. **Vocabulary leaks** are import-invisible — a fixture named after an application-specific tool imports
   nothing — so a committed source scan MUST additionally search both new trees' bytes, **comments
   included**, for the denied vocabulary. The discipline binds prose as well as code: a comment teaching the
   wrong vocabulary is the thing the next application copies.

**The guard's own non-self-tripping MUST be pinned.** Because the scan reads comments, the two new trees'
own doc comments MUST describe the rule **without using the denied words**, and the scan's needles MUST be
constructed at run time rather than written as literals in the scanned bytes. A guard that fails on its own
source teaches the next reader to delete it.

The vocabulary set MUST exclude terms that occur in ordinary prose about streams, runs and events, so the
guard denies concepts rather than syllables.

#### Scenarios

- **S-L3H-044** — Given the merged change, when the vocabulary scan runs over both new trees, then it passes; and when it is run over its **own** source file, then it also passes — the guard does not trip on itself.
- **S-L3H-045** — **(bite)** Given one fixture in the proof renamed to an application-specific tool name, when the vocabulary scan runs, then it FAILS naming the file and the term. Recorded, then reverted.
- **S-L3H-046** — Given the scan's source, when its needles are read, then each is assembled at run time rather than appearing as a literal in the scanned bytes, and a comment in place states why: the scan reads comments, so a literal needle would make the guard fail on itself.
- **S-L3H-047** — Given the compatibility statement, when it is searched for a reference to files, shells, skills or terminals, then none is present.

### R-L3H-011 — The stream is unchanged: no new event kind, no new outcome, no exported surface change

AG-23 MUST register **no** new event kind, add **no** new turn outcome member and **no** new cost label, and
MUST change **no** exported signature. The stream's committed kind count MUST hold with AG-23 registering
none, and Layer 2's exported surface MUST gain and lose **no** identifier.

This is stated as a requirement rather than assumed because this repository has a recorded drift class in
which a new member lands inside a closed sequence that another spec pins, and no test in the changing
milestone fails. AG-23's only production behaviour change is internal to one function (`R-RUN-014`).

`backend/agent/go.mod` and `go.sum` MUST be byte-identical to the merge base: the kit and the proof
introduce **zero** new module dependencies.

#### Scenarios

- **S-L3H-048** — Given the merged change, when the event-kind guard and the every-kind-constructible guard run, then both pass at the committed kind count with AG-23 registering none.
- **S-L3H-049** — Given the merged change, when Layer 2's exported surface is enumerated from an external test package, then its identifier set and its pinned method and signature arities are unchanged from the merge base.
- **S-L3H-050** — Given the merged diff, when `go.mod` and `go.sum` are compared to the merge base, then both are byte-identical and no `require` entry was added.

## Non-functional requirements

### NFR-L3H-A — Additive only, and every shipped fence stays green

This change MUST be additive: two new package directories, additive hunks, and documentation flips. It MUST
NOT relocate any existing package, MUST NOT move any file, and MUST NOT edit anything under Layer 1's tree.
Every shipped scope fence MUST pass, with the **single** recorded release this change's `agent-hook-taxonomy`
delta owns.

#### Scenarios

- **S-L3H-051** — Given the merged diff, when it is inspected for renames or moves, then none exists, and no file under Layer 1's tree is changed.
- **S-L3H-052** — Given the merged change, when every shipped scope fence runs, then each passes, and the only file released from a fence is the one this change's `agent-hook-taxonomy` delta records with its reasoning.
- **S-L3H-053** — Given the change reverted in isolation, when the module's tests run, then they pass exactly as before — nothing on the base branch depended on anything this change created.

### NFR-L3H-B — Gates and evidence

Every behavioural item MUST be taken **red → green → refactor** in order under strict TDD, with both outputs
recorded. Focused, single-package or single-test evidence MUST be recorded with **`-count=1`** and the
wall-clock duration recorded beside it. Whole-module evidence MUST instead be recorded **uncached** by
running `go clean -testcache` immediately before `make test`: the shipped `make test` target
(`go test -race -v ./...`) carries no `-count=1` of its own, and adding one would contradict
`agent-run-driver`'s own shipped pin of that exact command string. Either way, the real uncached suite for
this module is on the order of minutes, so a sub-second pass — cached or not — is a **cache artifact, not
evidence**. The milestone closes on a recorded green, uncached `cd backend/agent && go clean -testcache && make test`,
`make lint` reporting zero issues **module-wide**, and `make build` exiting zero.

Every planted bite named in this spec (`S-L3H-010`, `S-L3H-011`, `S-L3H-012`, `S-L3H-021`, `S-L3H-025`,
`S-L3H-045`) MUST be recorded **watched failing**, with the failure text, before being reverted. A bite
recorded as "would fail" is not recorded.

#### Scenarios

- **S-L3H-054** — Given the change's task record, when each behavioural item is walked, then it carries recorded red output, recorded green output, and a refactor note.
- **S-L3H-055** — Given the recorded evidence, when each **focused** test run is read, then it names `-count=1` and carries its wall-clock duration; when each **whole-module** `make test` run is read, then it is preceded by `go clean -testcache` and carries its wall-clock duration instead, because `make test` itself carries no `-count=1`; and no `(cached)` result is offered as evidence for either kind.
- **S-L3H-056** — Given the merged change, when the test, lint and build commands run from `backend/agent`, then the whole-module test run is green under `-race`, uncached (`go clean -testcache` immediately before), `make lint` reports zero issues **module-wide** (measured after `golangci-lint cache clean`, since a stale cache has previously masqueraded as a finding in this repo), and build exits zero.

---

## Acceptance criteria

1. An external-package test builds a harness from injected fakes through the public surface only and drives all seven acceptance capabilities in one run, every drain validating clean (`R-L3H-001`).
2. Zero vendor imports and zero I/O over both new trees, proven by **the** guard mechanism and shown to **bite** on planted violations (`R-L3H-002`).
3. The kit is importable from an external package and scripts provider turns, tool results, permission decisions and interrupts, delegating validation wholesale (`R-L3H-003`).
4. Determinism holds statically (no clock) and dynamically (repeat-run structural equality **modulo minted identities**, with the difference in identities itself asserted) (`R-L3H-004`).
5. Four runnable examples compile **and run** under the ordinary test command, printing no minted identity (`R-L3H-005`).
6. The v1 surface is enumerated by capability, frozen, experimental corners marked, every seam carrying its injection point and v1 default, with no application-specific reference (`R-L3H-006`).
7. No checklist row remains unchecked; every row cites its closing node with **merged** evidence, and only AG-23's own row cites this change (`R-L3H-007`).
8. The known-limitations register states the four named limitations with post-v1 paths, and contains no defect (`R-L3H-008`).
9. Every obligation forwarded to AG-23 is delivered, relocated, or declined with its reason recorded where a later reader meets it (`R-L3H-009`).
10. The generic-client boundary is enforced by a mechanical, non-self-tripping guard over both code and prose (`R-L3H-010`).
11. No new event kind, no exported surface change, no new dependency (`R-L3H-011`, `NFR-L3H-A`).
12. Strict TDD, `-count=1` evidence with durations, every bite watched failing (`NFR-L3H-B`).

## Traceability to the charter

| Charter node / clause | Requirement |
|---|---|
| AG-23.1 — consumer proof, seven capabilities in one run | `R-L3H-001` |
| AG-23.1 — vendor-import absence proven by the guard mechanism | `R-L3H-002` |
| AG-23.2 sc.1 — the kit is importable and scriptable | `R-L3H-003` |
| AG-23.2 sc.1 — deterministic, with no wall clocks | `R-L3H-004` |
| AG-23.2 sc.2 — examples compile **and run** | `R-L3H-005` |
| AG-23.3 item 1 — the frozen v1 surface, seams and defaults, no leak | `R-L3H-006` |
| AG-23.3 item 2 — the completion-checklist walk | `R-L3H-007` |
| AG-23.3 item 3 — the known-limitations register | `R-L3H-008` |
| `0003:112` / `0003:2101` — the generic-client boundary | `R-L3H-010` |
| Layer 2's exit: no carry-forward has anywhere to go | `R-L3H-009` |
| Layer 2's exit: the surface freezes here | `R-L3H-011`, `NFR-L3H-A` |
| `openspec/config.yaml` `apply.tdd: true` | `NFR-L3H-B` |
