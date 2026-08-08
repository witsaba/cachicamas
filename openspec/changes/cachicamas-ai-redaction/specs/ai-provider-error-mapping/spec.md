# Delta for `ai-provider-error-mapping` — LANDED (trigger fired empirically)

> **Change**: `cachicamas-ai-redaction` · **Milestone**: AI-36 (doc 0002:2163–2170) · **Node**: AI-36.1 item 2 · **Wave 5 — Harden**
> **Status**: **LANDED DELTA.** Amends [`openspec/specs/ai-provider-error-mapping/spec.md`](../../../../specs/ai-provider-error-mapping/spec.md). The trigger below fired empirically during `sdd-apply` (WU-2, task 2.3/2.4a) — see the resolution note immediately below.
> **Sources**: [proposal.md](../../proposal.md) § 4 D-4 · [explore.md](../../explore.md) § 1, § 4

---

> [!IMPORTANT]
> **Resolved (sdd-apply, WU-2).** `backend/agent/src/ai/openaicompat/a_i-36_1_test.go`'s
> `TestHostileServer_EchoesAuthorizationAndBody_CredentialNeverSurfacesInAnyRendering`
> drove a hostile `httptest.Server` that echoes the request's `Authorization`
> header and body into a 200 `application/json` response, forcing
> `refuseNonStreamContentType`. Before any production fix, the test's own
> RED failure was the observed evidence: the sentinel credential surfaced
> in `unwrap-depth-1 .Error()`, `%v`, `%s` and `%+v` — the excerpt
> carried it verbatim, byte-for-byte. Per the trigger table below, this is
> the "credential IS reproduced" row: **R-AEM-019 is promoted and binding.**
> The redaction landed at `stream.go`'s `redactCredential` (called from
> `refuseNonStreamContentType`, ~24 prod lines including doc comments,
> ~10 excluding them) — the same test, re-run after the fix, is GREEN and
> now stands as the regression pin for both `S-AEM-071` and `S-AEM-072`.
> The caller's own prompt content WAS also observed echoed into the
> excerpt on the same run (`t.Log` in the test records
> `prompt-body echo observed in a rendering = true`) — per the trigger
> table's third row this is **not a defect**, recorded here as the named
> residual R-AEM-019 item 4 requires.

## Trigger condition

`R-AEM-019` becomes binding **if and only if** the hostile-server case required by `S-CNF-081` shows that the caller's **own credential** is reproduced inside the bounded response excerpt that a non-streaming content-type refusal interpolates into its rendered text.

| Empirical outcome of `S-CNF-081` | Disposition of `R-AEM-019` |
| --- | --- |
| The credential **is** reproduced in the excerpt | **Land it.** Promote this delta, land the redaction behavior, prove both scenarios. **← THIS OUTCOME FIRED.** |
| The credential is **not** reproduced | Record as not-triggered. (Not this run's outcome — recorded for completeness.) |
| The caller's **prompt content** is reproduced | **Not a defect**, and **also observed on this run**. Recorded as a named residual under `S-CNF-081`; suppressing a provider's echo of the caller's own content would defeat the excerpt's diagnostic purpose. |

## Identity

| Field | Value |
| --- | --- |
| **Capability amended** | `ai-provider-error-mapping` |
| **Type** | Delta — **one `ADDED` requirement** (`R-AEM-019`, scenarios `S-AEM-071`, `S-AEM-072`); landed after the D-4 empirical trigger fired (see the resolution note above) |
| **Numbering** | Re-verified at spec time across the whole `openspec/` tree, canonical and archived alike: maxima `R-AEM-018` / `S-AEM-070`. `R-AEM-019` and `S-AEM-071` are free. |
| **Insertion point** | After `R-AEM-018` |

---

## ADDED Requirements

### R-AEM-019 — A bounded response excerpt never reproduces the caller's own credential

A refusal that reproduces a bounded excerpt of the provider's response into its rendered text MUST NOT reproduce the caller's own credential within that excerpt. Specifically:

1. Any occurrence of the caller's credential inside a captured response excerpt MUST be removed before that excerpt becomes reachable through a rendering.
2. The excerpt MUST remain present and readable to the caller reading the terminal failure's rendered text. Removal MUST be of the credential occurrences only — suppressing the excerpt would defeat its diagnostic purpose and would violate the landed obligation that the caller can see it.
3. The excerpt's existing size bound MUST be unchanged. This requirement concerns disclosure, never size.
4. A provider echoing the caller's own **content** back in its response is a **named residual**, not a defect, and MUST be recorded in writing rather than suppressed.
5. The absence claim MUST ship a positive control.

#### Scenarios

- **S-AEM-071** — Given a provider that echoes the caller's authorization value into a non-streaming response carrying an unexpected content type, when the resulting refusal is rendered, then the bounded excerpt is present and readable but the credential appears nowhere in the rendered text; and given the same case with the removal step disabled, when the check runs, then it fails — proving the claim is falsifiable.
- **S-AEM-072** — Given that same refusal, when its excerpt is measured against the previously landed size bound, then the bound is unchanged and the excerpt remains reachable to a caller reading the terminal failure's rendered text; and given a provider echoing the caller's own content rather than the credential, when the milestone closes, then that outcome is recorded in writing as a named residual.

---

## Out of scope

| Item | Owner |
| --- | --- |
| The excerpt's size bound | **AI-32.5** — charter Out-of-scope, verbatim; unchanged by this delta |
| The obligation that the caller can read the excerpt | **Already landed** — preserved, not re-litigated |
| Suppressing a provider's echo of the caller's own content | **Declared residual** — deliberately not fixed |
| Any new module dependency | Non-goal — `go.mod`/`go.sum` byte-identical |
