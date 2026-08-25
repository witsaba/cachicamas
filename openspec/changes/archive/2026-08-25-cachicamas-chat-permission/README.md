# CH-10 — cachicamas-chat-permission

> Archive folder for the CH-10 SDD change (`cachicamas-chat-permission`).
> Closed 2026-08-25 at apply-phase end (verify-report stub; the binding verify-report lands after sdd-verify).

## Contents

| File | Source |
|------|--------|
| `README.md` | This file — pointer to all artifacts |
| `proposal.md` | `engram://sdd/cachicamas-chat-permission/proposal` (#3987, #3988) |
| `design.md` | `engram://sdd/cachicamas-chat-permission/design` (#3990, #3991) |
| `tasks.md` | `engram://sdd/cachicamas-chat-permission/tasks` (#3992) |
| `apply-progress.md` | `engram://sdd/cachicamas-chat-permission/apply-progress` (#3994) |
| `verify-report.md` | STUB — overwritten by sdd-verify |
| `archive-report.md` | Closure report + lineage |
| `specs/cachicamas-chat-permission/spec.md` | `openspec/specs/cachicamas-chat-permission/spec.md` (NEW spec) |

## Charter

CH-10 of doc 0005 (`0005:936-947`). Closes R-15 (v2 § 6 seam 2 — approval is a suspension **inside** the loop on the same stream). Depends on CH-09 (`cachicamas-chat-tool-source`). Blocks CH-11.

Acceptance (verbatim `0005:944`):
> *Given a tool call the policy defers, When the participant approves it from the browser, Then the suspended turn resumes and the tool's result reaches the transcript; and when they decline, the turn continues with the refusal recorded and no execution.*