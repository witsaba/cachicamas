// AI-10 — the normalized request: the one value a provider receives.

package ai

import (
	"bytes"
	"slices"
	"strconv"
	"strings"
)

// Request is the complete provider-neutral description of one model call
// (V-REQ-20).
type Request struct {
	model     string
	messages  []Message
	system    SystemInstruction
	hasSystem bool
	options   requestDraft
}

// rolePermittedKinds is the table AI-10.3 item 3 lands: which content-part
// kinds a role may carry (design.md § 5.1, R-AMR-011).
//
// Indexed by Role rather than a map, for AI-04's reason: nothing in this
// package may let an unordered iteration decide anything. A later loosening
// is one element added to one role's slice; design.md § 5.2 records why the
// table starts strict rather than permissive — loosening a cell is additive,
// tightening one breaks a caller who shipped.
var rolePermittedKinds = [][]PartKind{
	RoleUser:      {PartKindText},
	RoleAssistant: {PartKindText, PartKindReasoning, PartKindToolCall},
	RoleTool:      {PartKindToolResult},
}

// roleAllowsKind reports whether a role may carry a content part of the given
// kind.
//
// A role with no table entry — including the zero Role, which no constructed
// message carries — permits nothing. That folds "not a role" into "permits no
// kind" rather than panicking or special-casing it, and it costs nothing
// because [Message.Role] can never in fact return one: [NewMessage] rejects a
// role outside the vocabulary before a message exists to ask this question of.
func roleAllowsKind(role Role, kind PartKind) bool {
	if int(role) >= len(rolePermittedKinds) {
		return false
	}
	return slices.Contains(rolePermittedKinds[role], kind)
}

// toolCallLocation is one tool call's identity and its position in the
// request's ordered messages and content.
type toolCallLocation struct {
	id                      string
	messageIndex, partIndex int
}

// duplicateToolCallRule reports the first tool-call identity that repeats an
// earlier one, positioned at the second occurrence (design.md § 7,
// R-AMR-012).
//
// Uniqueness is a precondition of [unresolvedToolResultRule] below, which
// resolves a result to a call by identity — a rule that resolves by a
// non-unique key is not a rule — and that is why this rule is checked first.
//
// Calls are collected into one ordered slice first, and the duplicate scan
// then walks it exactly as [NewToolSet] walks a tool set: linear and nested,
// for AI-04's reason. Nothing in this package may let an unordered iteration
// decide anything, and the reported duplicate is always the lowest index
// whose identity repeats an earlier one.
func duplicateToolCallRule(messages []Message) *Violation {
	var calls []toolCallLocation
	for i, message := range messages {
		for j, part := range message.Content() {
			if call, ok := part.ToolCall(); ok {
				calls = append(calls, toolCallLocation{id: call.ID(), messageIndex: i, partIndex: j})
			}
		}
	}
	for i, call := range calls {
		for _, earlier := range calls[:i] {
			if earlier.id == call.id {
				return Invalid(ErrDuplicate, AtIndex("messages", call.messageIndex), AtIndex("content", call.partIndex))
			}
		}
	}
	return nil
}

// unresolvedToolResultRule reports the first tool result whose call identity
// correlates to no tool call anywhere in the request (design.md § 6,
// R-AMR-012).
//
// "Anywhere in the request" is deliberate and not "earlier in the request":
// whether a call must precede the result that correlates to it is
// unlanded — design.md § 6.3 pins the non-decision by test rather than
// deciding it here. A tool call with no matching result is not checked by
// this rule at all: it is the ordinary mid-turn state, and repairing it is
// V-OUT-02's, one layer up.
func unresolvedToolResultRule(messages []Message) *Violation {
	for i, message := range messages {
		for j, part := range message.Content() {
			result, ok := part.ToolResult()
			if !ok {
				continue
			}
			if !anyToolCallHasID(messages, result.CallID()) {
				return Invalid(ErrUnresolvedReference, AtIndex("messages", i), AtIndex("content", j))
			}
		}
	}
	return nil
}

// anyToolCallHasID reports whether some tool call anywhere in the ordered
// messages carries the given identity.
func anyToolCallHasID(messages []Message, id string) bool {
	for _, message := range messages {
		for _, part := range message.Content() {
			if call, ok := part.ToolCall(); ok && call.ID() == id {
				return true
			}
		}
	}
	return false
}

// NewRequest constructs a normalized request.
//
// The rules are checked in the order written, per V-FAIL-04, and the order is
// part of the contract. design.md § 4 carries the whole table, including the
// rows later leaves insert; the rows this constructor holds today are:
//
//  1. The model identity is present, else [ErrEmpty] at "model". Whitespace-only
//     is folded into emptiness: a name made of spaces names nothing, and the
//     alternative is a provider round trip that fails for a reason the caller
//     could have been told about here.
//  2. There is at least one message, else [ErrEmpty] at "messages". A nil
//     sequence and an empty sequence are one fact and report one failure.
//  3. Every message was constructed, else [ErrEmpty] at "messages[i]", checked
//     in index order so the first failing element is the one reported. The
//     detector is the message's own identity — message.go documents
//     [MessageID.IsZero] as existing for exactly this question — and not its
//     role or content, because those are what a forged value would set.
//  4. Every message's content satisfies AI-06's rules, reported beneath this
//     request's position — "messages[2].content[0]" rather than "content[0]".
//
// Rule 4 is one call to AI-06's own validator with a request-shaped prefix, and
// it is the reason this file holds no per-kind content logic at all: no text
// bound, no reasoning-token rule, no tool-call argument rule. AI-06's decision
// record states the principle as one rule set, two callers, and names the
// failure mode of the alternative — a constructor that checks and a boundary
// that does not. Every content kind added after this milestone is validated at
// request depth on the day it is added, with no edit here.
//
// Today the rule cannot fire through the exported surface, because NewMessage
// validates content and Part is sealed, so a constructed message cannot carry
// an invalid part. That is defence in depth on purpose. AI-12 rebuilds requests
// and an adapter may one day produce messages by another path; the rule that is
// already there costs one call and removes the class of bug where a second
// producer silently skips the first producer's checks.
//
// Only emptiness is decidable about a model identity in this layer. The
// identity is not checked against a catalog, a price table or a selection
// policy: the register's § 6.3 places an unrecognised model on the provider side
// of the failure boundary, because recognition cannot be decided from the
// request alone.
//
// On failure the zero Request is returned, so a caller that ignored the error
// cannot mistake the result for a constructed request.
//
// The message sequence is copied, so a caller may reuse the slice it passed.
//
// The rule list itself is [requestDraft.rules] (§ 2.2 of the design):
// NewRequest seeds a draft from its two parameters, applies opts, then calls
// FirstFailure(draft.rules()...) and freezes — the same three steps
// [Request.With] performs after seeding from an existing request instead.
func NewRequest(model string, messages []Message, opts ...RequestOption) (Request, error) {
	draft := requestDraft{model: model, messages: slices.Clone(messages)}
	for _, opt := range opts {
		if opt != nil {
			opt(&draft)
		}
	}
	if err := FirstFailure(draft.rules()...); err != nil {
		return Request{}, err
	}
	return draft.freeze(), nil
}

// rules returns the request's rule list, in the documented order (V-FAIL-04,
// design.md § 3). [NewRequest] and [Request.With] both call
// FirstFailure(d.rules()...), so the two doors share one rule set in one
// order — R-REX-005's "derive-time validation is construction's validation"
// made structural rather than a discipline to re-check on every future rule.
//
// AI-12.1 (design.md § 2.2) is the extraction that makes this a method: every
// rule reads draft state, including the two — model and messages — that
// [NewRequest] previously closed over as parameters. The three cross-region
// free functions [duplicateToolCallRule], [unresolvedToolResultRule] and
// [anyToolCallHasID] still take []Message as a parameter; only their call
// sites here changed, from the parameter to d.messages.
func (d requestDraft) rules() []Rule {
	return []Rule{
		func() *Violation {
			if strings.TrimSpace(d.model) == "" {
				return Invalid(ErrEmpty, At("model"))
			}
			return nil
		},
		func() *Violation {
			if len(d.messages) == 0 {
				return Invalid(ErrEmpty, At("messages"))
			}
			return nil
		},
		func() *Violation {
			for i, message := range d.messages {
				if message.ID().IsZero() {
					return Invalid(ErrEmpty, AtIndex("messages", i))
				}
			}
			return nil
		},
		func() *Violation {
			for i, message := range d.messages {
				if violation := validateContent(Path{AtIndex("messages", i)}, message.Content()); violation != nil {
					return violation
				}
			}
			return nil
		},
		func() *Violation {
			if d.hasSystem && d.system.IsZero() {
				return Invalid(ErrEmpty, At("system"))
			}
			return nil
		},
		func() *Violation {
			for i, message := range d.messages {
				for j, part := range message.Content() {
					if !roleAllowsKind(message.Role(), part.Kind()) {
						return Invalid(ErrMisplaced, AtIndex("messages", i), AtIndex("content", j))
					}
				}
			}
			return nil
		},
		func() *Violation { return duplicateToolCallRule(d.messages) },
		func() *Violation { return unresolvedToolResultRule(d.messages) },
		func() *Violation {
			if !d.hasToolChoice {
				return nil
			}
			return violationOf(d.toolChoice.ValidateAgainst(d.tools))
		},
		d.cacheBoundaryCapRule(d.messages),
		d.boundsRule(),
	}
}

// freeze builds the immutable [Request] a validated draft describes.
//
// It is the ONE place a non-zero Request is built (design.md § 2.2.1): both
// [NewRequest] and [Request.With] end "FirstFailure(draft.rules()...);
// return draft.freeze(), nil", and nothing else in this package constructs a
// non-zero Request. That symmetry is what closes a trap the two-copy layout
// below would otherwise open on the very first derivation.
//
// [Request] stores the system region twice: the top-level
// Request.system/hasSystem pair, which [Request.SystemInstruction] reads, and
// requestDraft.system/hasSystem inside Request.options. A With that applied
// options and returned Request{options: draft} without also re-deriving the
// top-level pair would silently revert whatever WithSystemInstruction had
// just set — a bug no test could catch before this milestone, because no
// earlier test derived a request at all. Model and messages get the same
// two-copy layout for the identical reason RequestOption needs them
// reachable at all (design.md § 4), and freeze is what keeps every copy in
// sync: because it is the only constructor, the top-level fields and the
// options field can never disagree.
func (d requestDraft) freeze() Request {
	return Request{
		model:     d.model,
		messages:  d.messages,
		system:    d.system,
		hasSystem: d.hasSystem,
		options:   d,
	}
}

// With derives a new request from r by applying opts to a copy of r's draft
// (V-REQ-29).
//
// The three steps are [NewRequest]'s own, with one seed swapped: copy the
// draft — here from r.options rather than from two parameters — apply opts
// in order (last-wins, per [RequestOption]'s contract), then
// FirstFailure(draft.rules()...) and freeze. Deriving therefore runs the
// identical rule set, in the identical order, that constructing does
// (R-REX-005): there is no second, weaker validation path, because there is
// no second implementation of validation to begin with.
//
// r is a value receiver and is never written to, so r is left observably
// unmodified by any derivation, successful or not (R-REX-001). On failure the
// zero Request is returned, matching [NewRequest].
//
// Every region unsupplied by opts carries r's own value forward unchanged,
// because the seed is r.options itself — copying a struct of scalars, one
// string, and slice headers whose backing arrays are never mutated in place
// anywhere in this package. Every region that IS supplied copies on the way
// in, through the same options that build a constructed request
// ([WithMessages], [WithStopSequences], …), so mutating a slice passed to an
// option after deriving cannot reach the derived request (S-REX-005).
func (r Request) With(opts ...RequestOption) (Request, error) {
	draft := r.options
	for _, opt := range opts {
		if opt != nil {
			opt(&draft)
		}
	}
	if err := FirstFailure(draft.rules()...); err != nil {
		return Request{}, err
	}
	return draft.freeze(), nil
}

// WithModel sets the request's model identity, reachable through
// [RequestOption] so [Request.With] can replace it (design.md § 4).
//
// Applied inside [NewRequest], it is last-wins over the constructor's own
// model parameter, exactly as re-applying any other option is: this is not a
// second rule, it is [RequestOption]'s documented last-wins contract read at
// the one region a caller might not expect it to reach
// (TestNewRequest_WithModelBesideTheParameter_LastApplicationWins).
func WithModel(model string) RequestOption {
	return func(d *requestDraft) { d.model = model }
}

// WithMessages sets the request's ordered messages, reachable through
// [RequestOption] so [Request.With] can replace them (design.md § 4).
//
// Applied inside [NewRequest], it is last-wins over the constructor's own
// messages parameter, matching [WithModel]. The sequence is copied on the way
// in, matching [NewRequest]'s own parameter: a caller that mutates the slice
// after deriving must not be able to reach the derived request (S-REX-005).
func WithMessages(messages ...Message) RequestOption {
	return func(d *requestDraft) { d.messages = slices.Clone(messages) }
}

// Model returns the neutral name of the model the request targets (V-REQ-21).
func (r Request) Model() string { return r.model }

// Messages returns the request's ordered messages, in the order it was
// constructed with.
//
// The result is a fresh slice on every call, matching [Message.Content] and
// [ToolSet.Tools]: a consumer that rewrites what it received must not be able
// to rewrite the request, and two consumers must not be able to observe each
// other.
//
// The sequence is copied, not the messages in it. A message carries no mutable
// state, so the distinction costs nothing — but it is a property of the message
// rather than of the request, and AI-05 keeps it.
func (r Request) Messages() []Message { return slices.Clone(r.messages) }

// RequestOption is one optional region or generation option of a request.
//
// The type is sealed by the compiler rather than by review: its parameter type
// is unexported, so a consumer in another package cannot write a value of it.
// The set of options is therefore exactly this package's constructors — the
// same sealing move AI-06 made with the content-part payload, one dimension
// smaller.
//
// An option is total. It never fails, never validates and never allocates a
// violation, because every rule runs once, in NewRequest, in the documented
// order. An option that could fail would be a second validation site, which is
// the "constructor that checks and a boundary that does not" failure AI-06's
// decision record names.
//
// Applying the same option twice is last-wins, deliberately: that is what lets
// AI-12 express a per-request override as re-application rather than as a new
// mechanism.
type RequestOption func(*requestDraft)

// requestDraft accumulates the optional regions before validation.
//
// It is unexported so nothing settable reaches the exported surface. Each
// optional value is paired with a flag rather than carrying a sentinel, so
// absence is structural and "set to zero" is a different fact from "not set".
//
// AI-12.1 widens it with model and messages (§ 2.2), the two regions
// [NewRequest] previously took only as parameters. Widening is what makes
// [requestDraft.rules] a zero-argument method and [Request.With] possible at
// all: both doors seed one draft — [NewRequest] from its parameters, [With]
// from r.options — apply options, then share one rule slice and one freeze.
type requestDraft struct {
	model    string
	messages []Message

	maxOutputTokens    int
	hasMaxOutputTokens bool

	temperature    float64
	hasTemperature bool

	topP    float64
	hasTopP bool

	stopSequences    []string
	hasStopSequences bool

	system    SystemInstruction
	hasSystem bool

	tools    ToolSet
	hasTools bool

	toolChoice    ToolChoice
	hasToolChoice bool
}

// WithMaxOutputTokens sets the maximum number of tokens the model may generate.
func WithMaxOutputTokens(tokens int) RequestOption {
	return func(d *requestDraft) { d.maxOutputTokens, d.hasMaxOutputTokens = tokens, true }
}

// WithTemperature sets the sampling temperature.
func WithTemperature(temperature float64) RequestOption {
	return func(d *requestDraft) { d.temperature, d.hasTemperature = temperature, true }
}

// WithTopP sets the nucleus-sampling probability mass.
func WithTopP(topP float64) RequestOption {
	return func(d *requestDraft) { d.topP, d.hasTopP = topP, true }
}

// WithStopSequences sets the sequences whose generation stops the response.
func WithStopSequences(sequences ...string) RequestOption {
	return func(d *requestDraft) { d.stopSequences, d.hasStopSequences = slices.Clone(sequences), true }
}

// WithTools attaches a declared tool set to a request (V-REQ-14).
//
// It is independent of [WithToolChoice]. A request may declare tools without
// choosing one — that is the ordinary case, where the model decides — and the
// two flags are separate so that the difference is representable.
func WithTools(tools ToolSet) RequestOption {
	return func(d *requestDraft) { d.tools, d.hasTools = tools, true }
}

// WithToolChoice attaches a tool choice to a request (V-REQ-15).
//
// The choice is cross-validated against the request's tool set by AI-08.3's own
// method, in NewRequest, at rule 9 of the documented order. This option does not
// validate: an option is total, and a rule that ran here would be the second
// validation site AI-06's decision record names as the failure mode.
func WithToolChoice(choice ToolChoice) RequestOption {
	return func(d *requestDraft) { d.toolChoice, d.hasToolChoice = choice, true }
}

// Tools returns the request's declared tool set and whether one was applied.
//
// The second result is what keeps "declared no tools" distinct from "declared
// an empty set". They are different requests: AI-08.3's rule 2 rejects a
// non-none choice against an empty set, and a caller who applied an empty set
// deliberately is making a statement a caller who applied nothing is not.
//
// The set is returned by value and [ToolSet.Tools] copies on the way out, so a
// consumer that rewrites what it received cannot rewrite the request.
func (r Request) Tools() (ToolSet, bool) { return r.options.tools, r.options.hasTools }

// ToolChoice returns the request's tool choice and whether one was applied.
//
// Omitting the choice is not choosing [ToolChoiceNone]: the first leaves the
// decision to the model, the second forbids a call. An adapter reads the second
// result to tell them apart, which is why absence is structural here as it is
// for every other optional region.
func (r Request) ToolChoice() (ToolChoice, bool) {
	return r.options.toolChoice, r.options.hasToolChoice
}

// MaxOutputTokens returns the maximum number of tokens the model may generate,
// and whether the option was applied.
//
// The second result is what makes absence structural: no caller has to know
// that some integer means "unset", and a request that asked for nothing is a
// different request from one that asked for zero.
func (r Request) MaxOutputTokens() (int, bool) {
	return r.options.maxOutputTokens, r.options.hasMaxOutputTokens
}

// Temperature returns the sampling temperature and whether it was applied.
func (r Request) Temperature() (float64, bool) {
	return r.options.temperature, r.options.hasTemperature
}

// TopP returns the nucleus-sampling probability mass and whether it was
// applied.
func (r Request) TopP() (float64, bool) { return r.options.topP, r.options.hasTopP }

// StopSequences returns the sequences whose generation stops the response, in
// the order supplied, and whether the option was applied.
//
// The result is a fresh slice on every call, for message.go's reason: a
// consumer that rewrites what it received must not be able to rewrite the
// request, and two consumers must not be able to observe each other.
func (r Request) StopSequences() ([]string, bool) {
	if !r.options.hasStopSequences {
		return nil, false
	}
	return slices.Clone(r.options.stopSequences), true
}

// Equal reports whether r and other are equal under region-wise readback
// equality (design.md § 11.2, R-AMR-016): every region compared by the
// values its own accessors return, plus, for an optional region, its
// presence flag.
//
// Message identity is excluded. [Message.Equal] already excludes it, for the
// same reason restated here at request scope: V-REQ-03 makes [MessageID]
// deliberately unforgeable and minted per construction, precisely so two
// messages built from identical inputs are distinguishable — and including
// identity would make this method false for any two independently
// constructed requests at all, which is not what "equal" means for a
// request. Two requests are equal when a provider would receive the
// identical call.
//
// Go defines no == for Request: it holds slices, and neither region compared
// through a nested value — messages, the tool set — has one either. This is
// the method a consumer writes instead. AI-10.5's round trip needs it, AI-26
// needs it, and a comparison every consumer re-derives is a comparison every
// consumer gets subtly wrong.
//
// Every comparison reads through the exported surface — Messages(),
// SystemInstruction(), Tools(), ToolChoice(), each generation option — rather
// than the unexported fields it happens to share a package with, so this
// method proves the documented equality is reachable from outside, not only
// from here.
func (r Request) Equal(other Request) bool {
	if r.model != other.model {
		return false
	}
	if !slices.EqualFunc(r.Messages(), other.Messages(), Message.Equal) {
		return false
	}

	rSystem, rHasSystem := r.SystemInstruction()
	oSystem, oHasSystem := other.SystemInstruction()
	if rHasSystem != oHasSystem {
		return false
	}
	if rHasSystem && !rSystem.Equal(oSystem) {
		return false
	}

	rTools, rHasTools := r.Tools()
	oTools, oHasTools := other.Tools()
	if rHasTools != oHasTools {
		return false
	}
	if rHasTools && !slices.EqualFunc(rTools.Tools(), oTools.Tools(), toolsEqual) {
		return false
	}

	rChoice, rHasChoice := r.ToolChoice()
	oChoice, oHasChoice := other.ToolChoice()
	if rHasChoice != oHasChoice {
		return false
	}
	if rHasChoice {
		rName, rNamesOne := rChoice.Name()
		oName, oNamesOne := oChoice.Name()
		if rChoice.Mode() != oChoice.Mode() || rNamesOne != oNamesOne || rName != oName {
			return false
		}
	}

	rMax, rHasMax := r.MaxOutputTokens()
	oMax, oHasMax := other.MaxOutputTokens()
	if rHasMax != oHasMax || rMax != oMax {
		return false
	}

	rTemperature, rHasTemperature := r.Temperature()
	oTemperature, oHasTemperature := other.Temperature()
	if rHasTemperature != oHasTemperature || rTemperature != oTemperature {
		return false
	}

	rTopP, rHasTopP := r.TopP()
	oTopP, oHasTopP := other.TopP()
	if rHasTopP != oHasTopP || rTopP != oTopP {
		return false
	}

	rStopSequences, rHasStopSequences := r.StopSequences()
	oStopSequences, oHasStopSequences := other.StopSequences()
	if rHasStopSequences != oHasStopSequences || !slices.Equal(rStopSequences, oStopSequences) {
		return false
	}

	return true
}

// toolsEqual reports whether two tool declarations carry the same name,
// description, schema bytes and marker state.
//
// A free function rather than a [Tool] method: design.md § 11.2 asks for
// tools to be compared "by their own accessors", not by a new exported
// method on a type AI-08 owns, and [Tool] cannot use == because its schema
// field is a slice.
//
// The marker comparison is AI-11.1's addition (R-ACB-004): two tool
// declarations otherwise identical but differing in cache-boundary state
// describe different requests. Unlike [Segment], which stays comparable and
// is therefore compared by == inside [SystemInstruction.Equal] for free, a
// field added to the non-comparable [Tool] is invisible here unless it is
// named — cache_boundary_test.go's TestRequest_MarkersDifferByOneMarker...
// proves the omission bites.
func toolsEqual(a, b Tool) bool {
	return a.Name() == b.Name() && a.Description() == b.Description() &&
		a.IsCacheBoundary() == b.IsCacheBoundary() && bytes.Equal(a.Schema(), b.Schema())
}

// boundsRule checks the bounds of the applied generation options that are
// decidable from the request alone (design.md § 8.1).
//
// The rules are checked in the order written, per V-FAIL-04, and every one is
// skipped when its option was not applied — an unapplied option has no value to
// be out of range.
//
//  1. Maximum output tokens is strictly positive, else [ErrOutOfRange]. Zero is
//     rejected with the negatives: a request for no output is not a request.
//  2. Temperature is not negative, else [ErrOutOfRange]. There is deliberately
//     **no upper bound**. Providers disagree on the cap — 1.0 for some, 2.0 for
//     others — so a bound right for one would be a caller-contract failure this
//     package invented for the other, rejecting a call that provider accepts.
//     A too-high temperature is a provider failure and reports through AI-19,
//     which is the same treatment the register's § 6.3 gives an unrecognised
//     model identity.
//  3. Top-p is in (0, 1], else [ErrOutOfRange]. Unlike temperature, the scale
//     here is the same everywhere, so the bound is the concept's rather than
//     one provider's.
//  4. The stop-sequence option, if applied, carries at least one sequence, else
//     [ErrEmpty] at "stopSequences", and every sequence is non-empty, else
//     [ErrEmpty] at "stopSequences[i]". An empty sequence would stop generation
//     immediately or be dropped, depending on the provider — the kind of
//     divergence this layer exists to remove. A whitespace-only sequence is
//     legal: unlike a system-instruction segment, it is a real string to match.
func (d requestDraft) boundsRule() Rule {
	return func() *Violation {
		return firstViolation(
			func() *Violation {
				if d.hasMaxOutputTokens && d.maxOutputTokens <= 0 {
					return Invalid(ErrOutOfRange, At("maxOutputTokens"))
				}
				return nil
			},
			func() *Violation {
				if d.hasTemperature && d.temperature < 0 {
					return Invalid(ErrOutOfRange, At("temperature"))
				}
				return nil
			},
			func() *Violation {
				if d.hasTopP && (d.topP <= 0 || d.topP > 1) {
					return Invalid(ErrOutOfRange, At("topP"))
				}
				return nil
			},
			func() *Violation {
				if !d.hasStopSequences {
					return nil
				}
				if len(d.stopSequences) == 0 {
					return Invalid(ErrEmpty, At("stopSequences"))
				}
				for i, sequence := range d.stopSequences {
					if sequence == "" {
						return Invalid(ErrEmpty, AtIndex("stopSequences", i))
					}
				}
				return nil
			},
		)
	}
}

// String renders the request for a diagnostic reader, naming its shape and
// never its payload.
//
// The request is the highest-value leak in Layer 1: it carries the prompt, the
// model's deliberation, tool arguments, tool results and the system instruction
// in one value, and fmt prints the unexported fields of a struct it has no
// String method for. V-FAIL-13 puts the posture on the type.
//
// What renders is which regions are present and how many elements each ordered
// region holds. Not the model identity — a model name is caller data reaching a
// log, and a caller that holds the request can call [Request.Model]. Not a
// segment's text, a message's content, a stop sequence or an option's value: an
// option's value is caller-chosen and, for a stop sequence, is caller-authored
// text.
//
// An unapplied option is omitted rather than rendered as absent, so the
// rendering names what the request has instead of enumerating what it lacks.
func (r Request) String() string {
	var b strings.Builder
	b.WriteString("request(")
	if r.model != "" {
		b.WriteString("model, ")
	}
	b.WriteString(strconv.Itoa(len(r.messages)))
	b.WriteString(" messages")
	if r.hasSystem {
		b.WriteString(", ")
		b.WriteString(r.system.String())
	}
	if r.options.hasTools {
		b.WriteString(", ")
		b.WriteString(strconv.Itoa(r.options.tools.Len()))
		b.WriteString(" tools")
	}
	if r.options.hasToolChoice {
		b.WriteString(", toolChoice(")
		b.WriteString(r.options.toolChoice.Mode().String())
		b.WriteByte(')')
	}
	// AI-11.2's extension: name the boundary count, never a marked carrier's
	// payload — a count is a fact about the request's shape, the same
	// admission test every other clause here applies (R-ACB-010).
	if count := len(r.CacheBoundaries()); count > 0 {
		b.WriteString(", ")
		b.WriteString(strconv.Itoa(count))
		b.WriteString(" cache boundaries")
	}
	for _, name := range r.options.appliedNames() {
		b.WriteString(", ")
		b.WriteString(name)
	}
	b.WriteByte(')')
	return b.String()
}

// GoString renders the request for the %#v verb.
//
// It exists so the redaction posture covers every fmt verb rather than the
// three a reader thinks of, and it delegates to [Request.String] so there is one
// rendering rather than two that could drift. Every payload-carrying type in
// this package spells it the same way.
func (r Request) GoString() string { return r.String() }

// appliedNames returns the names of the applied generation options, in the
// documented order.
//
// Names only, never values: an option's value is caller-chosen, and a stop
// sequence is caller-authored text. The order is the slice's, so the rendering
// of one request is the same on every call and across processes.
func (d requestDraft) appliedNames() []string {
	var names []string
	if d.hasMaxOutputTokens {
		names = append(names, "maxOutputTokens")
	}
	if d.hasTemperature {
		names = append(names, "temperature")
	}
	if d.hasTopP {
		names = append(names, "topP")
	}
	if d.hasStopSequences {
		names = append(names, "stopSequences")
	}
	return names
}
