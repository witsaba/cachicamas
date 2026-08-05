# Delta for ai-provider-text-stream

> **Change**: `cachicamas-ai-provider-completion` (AI-31) · **Phase**: spec · **Date**: 2026-08-04
> The `ai-provider-text-stream` capability is still an **in-flight, unpromoted** change
> (`openspec/changes/cachicamas-ai-provider-text-stream/specs/ai-provider-text-stream/spec.md`). This delta is
> written against that in-flight text and reconciled at archive, per the wave's recorded spec-promotion lesson.

## MODIFIED Requirements

### R-ATS-026 — The charter boundary holds as of AI-28: no reasoning, no tool calls, no usage taxonomy at that time, no new dependency

That milestone MUST NOT map reasoning content (AI-29), MUST NOT map `tool_calls` or `function_call` deltas in
either direction (AI-30), MUST NOT implement the full usage field mapping or the cumulative merge (AI-31.2),
and MUST NOT author the failure-status taxonomy (AI-32.1). `backend/agent/go.mod` MUST remain at **zero**
requires; a third-party dependency is a hard blocker, not a tradeoff.

The usage clause is **as of AI-28**. AI-31.2 has since discharged it: the per-field usage taxonomy and the
merge pin now live in the `ai-provider-completion` capability (`R-ACP-005`, `R-ACP-006`, `R-ACP-007`). The
boundary is recorded as moved, never quietly invalidated.
(Previously: `S-ATS-101` asserted the shipped usage handling carries "no per-field taxonomy that AI-31.2
owns" as a standing property; landing AI-31 makes that reading false.)

#### Scenarios

- **S-ATS-098** *[inspection]* — Given `backend/agent/go.mod` before and after that change, when the two are compared, then the file is byte-identical and its require set is empty.
- **S-ATS-099** *[inspection]* — Given the shipped production sources of that change, when a reviewer reads their imports, then every import is stdlib or an in-repo `src/ai` path.
- **S-ATS-100** *[inspection]* — Given the shipped mapping source, when a reviewer searches it, then no code path emits a reasoning event or a tool-call event, and `tool_calls`/`function_call` deltas are handled only by `R-ATS-017`'s skip rule.
- **S-ATS-101** *[inspection]* — Given the usage handling **as shipped by AI-28**, when a reviewer reads it at that revision, then it establishes presence and absent-versus-zero only, with no cumulative merge and no per-field taxonomy; and given the usage handling after AI-31, when a reviewer reads it, then the per-field taxonomy and the merge pin are present and are governed by `R-ACP-005`…`R-ACP-007` rather than by this requirement.
