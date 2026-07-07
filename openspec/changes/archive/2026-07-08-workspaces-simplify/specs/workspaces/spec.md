# Spec delta: workspaces-simplify (1:1)

> This delta narrows `openspec/specs/workspaces/spec.md` from 1:many to 1:1.
> Sections replaced in full are marked with `### REPLACED-BY-WORKSPACES-SIMPLIFY` markers;
> sections removed entirely are listed at the bottom of this file with `### REMOVED-BY-WORKSPACES-SIMPLIFY`.

The canonical 1:1 spec replaces this delta once sdd-verify confirms the refactor.

---

## R-WS-001 — Workspace creation (UNCHANGED; just field rename)

A signed-in user who has completed ownboarding can create a new workspace by providing:

- `name` (required, 3–60 chars, unique among live workspaces in the same organization)
- `repository` (required, a GitHub repo the user has access to — `owner/name` shape, validated server-side against `/user/repos`)

[Scenarios unchanged from the canonical spec — field is `repository` instead of `primary_repository`.]

---

## R-WS-002 — Workspace listing (UNCHANGED; `linked_repos_count` removed from summary)

`WorkspaceSummary` shape becomes:

```ts
{
  id: number;
  name: string;
  repository: Repository;
  created_at: string;
}
```

`linked_repos_count` is removed (always 0 in the 1:1 model; the field is no longer meaningful).

---

## R-WS-003 — Workspace detail (REPLACED)

### REPLACED-BY-WORKSPACES-SIMPLIFY

A signed-in user can fetch the full detail of a single workspace. The response shape is:

```ts
{
  id: number;
  name: string;
  repository: Repository;        // { github_id, full_name, owner, name }
  created_at: string;
  updated_at: string;
}
```

No `linked_repositories` field (gone in the 1:1 model).

[Scenarios for 200 / 404 / anon / no-org unchanged from the canonical spec.]

---

## R-WS-004 — Workspace update (UNCHANGED)

`s/name/new-name` PATCH path. `repository` cannot be changed via PATCH (locked design decision — same as the canonical spec).

---

## R-WS-005 — Workspace deletion (UNCHANGED)

Soft delete by name; partial unique index ensures name can be re-used after delete.

---

## REMOVED-BY-WORKSPACES-SIMPLIFY

The following requirement groups are removed in their entirety:

- R-WS-006: Add a linked repository to a workspace
- R-WS-007: Remove a linked repository from a workspace
- R-WS-008: List linked repositories for a workspace
- R-WS-014: GitHub repo picker used inside the workspace detail page (the picker used INSIDE the create form, R-WS-012, stays)

Plus their scenarios (S-WS-050..059, S-WS-130..134) and all derived `LinkedRepository` / `AddRepoInput` types and their endpoint implementations.
