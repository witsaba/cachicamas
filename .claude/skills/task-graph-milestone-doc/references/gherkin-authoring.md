# Gherkin authoring for task-graph leaves

Every behavior in a v2 milestone document is written in Gherkin (Given/When/Then), because a
scenario is the one phrasing that is simultaneously language-agnostic, testable, and executable as
a Strict-TDD RED test. A leaf whose behavior cannot be phrased as a scenario is not a leaf yet —
decompose it or reclassify it.

## The scenario → TDD mapping (the reason Gherkin is mandatory)

Each scenario becomes **exactly one test**, driven through the full cycle in order:

1. **RED** — write the test verbatim from the scenario; watch it fail for the stated reason.
2. **Implementation** — the minimum that satisfies the scenario.
3. **GREEN** — the test passes; the leaf's evidence-gate command passes.
4. **Refactor** — performance, clean code, and the idioms of the implementation language, under
   green tests. The scenario never changes during refactor.
5. **Review** — the milestone's review pass (native review / adversarial), after all scenarios of
   the leaf are green.

Because the document is language-agnostic, the scenario states *observable behavior* and the SDD
cycle of the milestone decides how it is spelled in the implementation language.

## Leaf anatomy (v2)

A `[leaf]` node body contains, in order:

- **Scenarios:** 1–7, in a fenced ```gherkin block (preferred; renderable and extractable) or as
  `- **Scenario:** Given … When … Then …` bullets. More than 7 is a split trigger.
- **Depends on:** bare node ids (the edge list — see the DAG contract in `method.md`).
- **Out of scope:** what keeps this leaf exclusive of its siblings.
- **Split if:** the pre-declared fission trigger, where foreseeable.
- ***(pin)*** marker on any scenario that is a regression pin rather than new behavior.

```gherkin
Feature: stream sequencing            # the leaf's title, restated as a capability

Scenario: two concurrent streams stay contiguous
  Given two streams opened from the same provider
  When both stream to completion concurrently under the race detector
  Then each stream observes sequence numbers 1..N with no gap

Scenario: a cancelled stream still terminates exactly once
  Given a stream in mid-delivery
  When the consumer cancels the context
  Then exactly one terminal event is observed
  And no producer goroutine remains after the stream ends
```

## Style rules

- **Observable behavior only.** "Add a mutex" is illegal; the scenario above about contiguous
  sequences under concurrency is the legal form of the same requirement.
- **One When per scenario.** If you need two Whens, you have two scenarios.
- **Domain vocabulary from the boundary section** — never type names, field names, signatures,
  framework names, or language keywords. The wording must survive a change of implementation
  language unmodified.
- **Given** states context that already holds; **When** is the single action under test; **Then**
  (+ `And`/`But`) states only what a test can assert. No "should work correctly", no "handles
  errors gracefully" — name the observable outcome.
- **Scenario Outline + Examples** for property-like cases with a value table; prefer it over
  copy-pasted near-identical scenarios.
- **Error paths are scenarios too**, and they follow the happy-path scenarios of the same leaf or
  live in a later sibling leaf — never before the walking skeleton.

## Non-leaf nodes

- `[guard]` — one scenario proving the guard **bites**: Given a scratch violation, When the check
  runs, Then it fails naming the violation; plus the green run. The bite is the RED.
- `[decision]` — no scenarios; closes by a recorded artifact answering every closing-checklist
  question. The questions themselves are still phrased so a reader can verify each answer.
- `[mechanical]` — no scenarios; closes by recorded objective check evidence (build output, diff
  scan). If you find yourself writing Given/When/Then for it, it was a leaf mislabeled.
- `[compound]` — no scenarios of its own; its exit check is one line, and its scope is exactly the
  union of its children (100 % rule).

## Anti-patterns (each observed in real reviews)

| Smell | Fix |
| --- | --- |
| Scenario names an implementation artifact ("the mutex", "the goroutine") | Restate as what a test observes from outside |
| Then asserts an internal state no test can reach | Find the observable consequence, or move the claim to a `[guard]` |
| One scenario per code path of an anticipated design | Scenarios follow *requirements*; the design does not exist yet |
| A leaf whose scenarios span two capabilities | Split — scenarios are the 100 %-rule unit at leaf depth |
| Gherkin used for a mechanical scan | Reclassify as `[mechanical]` with check evidence |
