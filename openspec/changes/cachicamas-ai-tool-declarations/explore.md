# Explore — tool declarations

> **Change**: `cachicamas-ai-tool-declarations`
> **Milestone**: AI-08 — Define tool declarations
> **Nodes**: AI-08.1 `[leaf]` · AI-08.2 `[leaf]` · AI-08.3 `[leaf]`
> **Phase**: explore
> **Project**: cachicamas (witsaba)
> **Date**: 2026-07-31
> **Driver**: braejan
> **Depends on**: AI-04, AI-05
> **Blocks**: AI-10, AI-18, AI-26

---

## 1. What this milestone is, in one paragraph

Layer 1 must be able to carry, byte-faithfully and in a stable order, the description of the tools a model may call — and nothing more. A tool declaration is a name, a description and the bytes of an argument schema. A tool set is the ordered collection of them one request offers. A tool choice is the instruction constraining whether and which tool the model may call. All three cross the model API, which is why they are Layer 1's at all; none of them is executed, resolved, or checked against a schema here, which is why the milestone is small.

## 2. What already exists, and what it settles

### 2.1 The code

`backend/agent/src/ai/` holds four production files after AI-04 and AI-05:

| File | What it settles for this milestone |
| --- | --- |
| `validation.go` | The whole failure vocabulary. Five sentinels, one `*Violation`, `At`/`AtIndex`, `Rule` + `FirstFailure`. This milestone reports every rule through it and defines no error of its own |
| `role.go` | The reusable closed-vocabulary pattern: `uint8` + `iota+1`, an indexed name table, an enumeration over the **constant space**, an exact parser, an exhaustiveness pin. AI-08.3's tool choice is the second consumer of that pattern |
| `message.go` | Two habits worth copying rather than re-deriving: copy the sequence in *and* copy it out, and return the zero value on a failed construction so an ignored error cannot be mistaken for success |
| `doc.go` | Records trap 1 in the package's own documentation. This milestone is the first that has to live inside it |

`import_boundary_test.go` and `src/agenttest/import_compile_test.go` are the AI-00 guards; both must still pass, and `go.mod` must still carry zero requires.

### 2.2 The vocabulary — four terms AI-08 implements rather than invents

| Term | Register clause this milestone must satisfy |
| --- | --- |
| `V-REQ-12` tool declaration | "name, description, schema bytes … **not** an executable, not a resolution target, not a permission subject" |
| `V-REQ-13` schema bytes | "carried byte-faithfully. Layer 1 transports it and never validates it against a meta-schema" |
| `V-REQ-14` tool set | "**ordered, deterministically iterable** … the set is a cache prefix (`V-REQ-25`); a non-deterministic order silently invalidates a cache" |
| `V-REQ-15` tool choice | "constraining whether and which tool the model may call, **validated against the declared tool set**" |

Register § 3's "applying the trap to a new case" is the test this milestone applies twice: schema **bytes** cross the API and are ours; a parsed schema, or a judgement about whether it is a well-formed JSON Schema, is not.

### 2.3 What the register does *not* settle, and this milestone must

1. **What a legal tool name looks like.** `V-REQ-12` says a declaration has a name; doc 0002 deliberately leaves the character rules to the SDD ("a name outside the documented character rules"). § 3.1 works it.
2. **Which rule class an illegal name violates.** AI-04 has five classes and no guidance on which one a malformed name belongs to. § 3.2.
3. **What "deterministic iteration" means concretely** — a property of the type, or a property everyone has to remember. § 3.3.
4. **How a payload-carrying member fits AI-05's payload-free vocabulary pattern.** § 3.4. This is the milestone's only real design question.

## 3. The real design tensions

### 3.1 The tool-name character rules — permissive or intersective

Two defensible positions.

**Permissive.** Accept anything non-empty; let the provider reject what it dislikes. Cheap, and it never blocks a caller whose provider accepts something we did not anticipate.

**Intersective.** Accept only what every provider a v1 adapter could plausibly target accepts. More work, and it can reject a name one provider would have taken.

The intersective position wins here, for a reason specific to *names* rather than to validation in general: **a tool name is not merely sent, it comes back**. The model answers with a tool call naming the tool, and Layer 2 resolves that name to an implementation. An adapter that had to rewrite an illegal name on the way out would have to un-rewrite it on the way in — a mapping table AI-26 would own and AI-30 would have to invert, for no benefit. The alternative to rejecting the name here is a network round trip that ends in a provider 400, or worse, a silent adapter-local mangling. Doc 0002's phrase for the seam is exactly this: "caught client-side".

The primary-source anchor is Anthropic's published constraint, verified for this change: a tool `name` **"Must match the regex `^[a-zA-Z0-9_-]{1,64}$`"** ([tool-use/define-tools](https://platform.claude.com/docs/en/agents-and-tools/tool-use/define-tools)). Other major vendors publish the same alphabet and the same 64-byte ceiling; at least one additionally requires the first character to be a letter or an underscore.

`design.md` § 3 fixes the exact rule and states which half is verified from a primary source and which half is a deliberate narrowing.

### 3.2 Which sentinel an illegal name reports

AI-04's decision record is explicit that a sentinel names a **rule class**, not a field. Applying that here splits what looks like one rule into two:

- A name longer than the ceiling is a value outside a **documented bound** → `ErrOutOfRange`.
- A name containing a character outside the alphabet is a value **not well-formed for its documented encoding** → `ErrMalformed`.

They are different facts with different fixes ("shorten it" versus "spell it differently"), which is the same argument `role.go` makes for splitting `ErrEmpty` from `ErrNotInVocabulary` in `ParseRole`. Doc 0002's test list names one item ("a name outside the documented character rules"); the length case is therefore a **discovered case appended** to AI-08.1's list, per the living-graph clause's rule 3.

### 3.3 Determinism has to be structural, not remembered

The failure mode `V-REQ-14` names is the nastiest kind: a map-backed tool set produces no error, no crash and no wrong answer — it produces a cache prefix that differs on every call, and the only symptom is a bill roughly ten times larger than it should be. A comment saying "keep this ordered" is not a countermeasure.

The countermeasure is that the tool set **is** a slice, holds the caller's order, and never iterates a map to decide anything. That inherits AI-04's own rule verbatim: "nothing in this package may let an unordered iteration decide anything, and a registry is where that temptation is strongest." Duplicate detection is the one place a map is tempting, and a linear scan over the ordered slice is both fast enough at realistic tool counts and deterministic about *which* duplicate it reports.

### 3.4 The payload-carrying vocabulary member — the milestone's design question

AI-05's pattern is built for payload-free members: a `uint8`, a table indexed by the constant, an enumeration over the constant space. Three of tool choice's four values fit it exactly. The fourth — "call this specific tool" — carries a name.

Three shapes were considered.

| Shape | Why not |
| --- | --- |
| Put the name on the vocabulary type itself (`type ToolChoice struct { mode uint8; name string }`, no separate mode type) | The vocabulary stops being an integer, so `iota + 1` and the indexed table both stop working, and the pin has nothing to enumerate. It discards the pattern rather than extending it |
| Make "specific" a family of values, one per declared tool | The vocabulary would stop being closed — its size would depend on the request |
| **Keep the vocabulary payload-free; give the payload its own value type that carries a member plus its payload** | — |

The third is what `design.md` adopts, and it needs exactly one extension to the pattern: the table row gains a column saying whether the member takes a payload. AI-05's `design.md` § 3.2 pre-authorises precisely that move — "Rule 2's table may carry more than a name … that is a widening of the row, not a change of shape" — so this milestone is spending an allowance the pattern already granted rather than bending it.

## 4. Prior art worth borrowing, and the parts worth refusing

- **`net/url`'s opaque-versus-parsed split.** `url.URL` keeps `RawQuery` beside the parsed form because re-encoding is lossy. Schema bytes are the same problem with none of the upside: there is nothing here to parse *for*, so the milestone keeps the bytes and skips the parsed form entirely.
- **`http.Header`'s canonicalization.** Refused. Canonicalizing a value on the way in is what makes byte fidelity untestable later; AI-26.4's cache prefix needs the bytes the caller wrote.
- **The standard library's `slices.Clone` discipline for exported sequences.** Borrowed wholesale from `message.go` — a byte slice handed out unclone is a byte slice a consumer can rewrite.

## 5. What the milestone must not do

| Excluded | Owner |
| --- | --- |
| Validating schema bytes against a JSON Schema meta-schema | Nobody in Layer 1 — `V-REQ-13`, AI-08's charter |
| Validating schema bytes as syntactically well-formed JSON | Deliberately not landed; § 8 question 2 records the deliberation |
| Executing a tool, resolving a name to an implementation, deciding whether a call may run | Layer 2 / Layer 3 — `V-OUT-04`, `V-OUT-05`, `V-OUT-16` |
| Where the tool set came from, and whether it may change between turns | Layer 3 — `V-OUT-17` |
| Attaching tools or tool choice to a request | **AI-10**, and AI-10.3 re-runs AI-08.3's cross-validation at the request boundary |
| The tool **call** a model emits, and its argument bytes | **AI-09** — a different content variant with its own byte-fidelity property |
| Cache-boundary markers on a declaration | **AI-11.1** |
| Any content part, message type or request type | AI-06, AI-05, AI-10 — this milestone adds none |

## 6. Vocabulary check

Every noun this milestone exports maps to a landed register row: tool declaration `V-REQ-12`, schema bytes `V-REQ-13`, tool set `V-REQ-14`, tool choice `V-REQ-15`. **No register amendment is required.** Three near-misses, recorded so a later reader does not re-open them:

1. **The four tool-choice members** (automatic, none, required, specific) have no rows of their own. They do not need any: `V-REQ-15` defines tool choice as "the request-level instruction constraining whether and which tool the model may call", and the four members are that definition's enumeration, not four new terms. AI-05 set the same precedent — the register has no row per role.
2. **"Tool name"** is not a row. It is a field of `V-REQ-12`, named in that row's own text.
3. **The character rules** are a *decision this change records*, not a term. They live in `design.md` § 3 and in the GoDoc, which is where a reader who hits the failure will look.

## 7. Size and shape

Forecast: three production files (`tool.go`, `tool_set.go`, `tool_choice.go`) and three external test files, plus five markdown artifacts. The production surface is small — three constructors, seven accessors, one validator, one closed vocabulary of four. The test surface is not, and cannot be: byte fidelity, order determinism under many iterations, and a four-member cross-validation matrix are each irreducible to fewer cases than are written.

Doc 0002's split trigger 4 (**prefer < 250 changed lines, reassess before 400**) is expected to fire, as it did for AI-04 and AI-05. `tasks.md` carries the reassessment. No other trigger fires: the three leaves are strictly ordered, each test list is inside the 7-item limit, and none needs a seam that does not exist.

## 8. Open questions carried into the proposal

1. **The exact name rule** — answered in `proposal.md` § "The four choices", fixed in `design.md` § 3.
2. **Whether schema bytes must be syntactically well-formed** — carried as a *deliberate non-decision*. Doc 0002 lists syntactic validation for AI-09.1's argument bytes and pointedly does not for AI-08.1's schema bytes; `design.md` § 8 records both sides and leaves the rule to whoever meets a case for it.
3. **Whether an absent tool choice is legal** — not this milestone's. AI-08.3 validates a tool-choice *value*; whether a request may omit one is a request-shape question and is AI-10.3's.
