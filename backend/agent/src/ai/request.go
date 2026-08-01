// AI-10 — the normalized request: the one value a provider receives.

package ai

import (
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
func NewRequest(model string, messages []Message, opts ...RequestOption) (Request, error) {
	draft := requestDraft{}
	for _, opt := range opts {
		if opt != nil {
			opt(&draft)
		}
	}
	if err := FirstFailure(
		func() *Violation {
			if strings.TrimSpace(model) == "" {
				return Invalid(ErrEmpty, At("model"))
			}
			return nil
		},
		func() *Violation {
			if len(messages) == 0 {
				return Invalid(ErrEmpty, At("messages"))
			}
			return nil
		},
		func() *Violation {
			for i, message := range messages {
				if message.ID().IsZero() {
					return Invalid(ErrEmpty, AtIndex("messages", i))
				}
			}
			return nil
		},
		func() *Violation {
			for i, message := range messages {
				if violation := validateContent(Path{AtIndex("messages", i)}, message.Content()); violation != nil {
					return violation
				}
			}
			return nil
		},
		func() *Violation {
			if draft.hasSystem && draft.system.IsZero() {
				return Invalid(ErrEmpty, At("system"))
			}
			return nil
		},
		draft.boundsRule(),
	); err != nil {
		return Request{}, err
	}
	return Request{
		model:     model,
		messages:  slices.Clone(messages),
		system:    draft.system,
		hasSystem: draft.hasSystem,
		options:   draft,
	}, nil
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
type requestDraft struct {
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
