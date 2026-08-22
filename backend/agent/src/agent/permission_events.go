// AG-06.1 — the permission family (R-APE-001, R-APE-002, R-APE-003,
// VL2-EVT-06).
//
// Permission events record the model's "may this call proceed?" loop:
// [NewPermissionDecisionRequired] opens one, [NewPermissionDecisionMade]
// answers it (with a typed [PermissionOutcome]), and
// [NewPermissionResolutionRemembered] records a fact about future calls
// the session log needs or it cannot explain why a later call was never
// asked (R-APE-003, G1 in doc 0001 § 7).
//
// # Typed outcome (R-APE-002)
//
// [PermissionOutcome] is a closed typed enum with four values:
// [PermissionOutcomeAllowOnce], [PermissionOutcomeAllowAlways],
// [PermissionOutcomeDeny], and [PermissionOutcomeModifyInput]. The zero
// value is NOT a member; the decision_made constructor rejects it so
// there is no route to a decision_made event with no outcome
// (S-APE-010). The ModifyInput outcome carries the modified arguments
// as a payload field (S-APE-011); the Deny outcome carries a typed
// [Failure] (S-APE-012, R-AEV-008).
//
// # Per-tool cardinality (R-APE-003)
//
// The resolution_remembered kind declares `CardinalityAtMostOne` on its
// descriptor row, exercising the AG-04.3-reserved seam
// (`event_descriptor.go:103-120`) for the first time. The validator's
// existing rule engine rejects a second same-tool-name event with the
// duplicate rule class (S-APE-082).
//
// # Per-family file split (AD-6, AG-05 precedent)
//
// Payload + ctors + kind constants + EventDescriptor rows are
// co-located, mirroring message_text.go and tool_event.go. The same
// package owns the EventKind block (event.go) and the eventRegistry
// rows — this file's constants are referenced by event.go's iota
// continuation; a new kind declared here is wired through by the same
// commit.

package agent

import (
	"strconv"

	"github.com/cachicamas/backend/agent/src/ai"
)

// PermissionOutcome is the typed outcome vocabulary a permission decision
// reaches (R-APE-002, 0003:638-642). Four members — AllowOnce,
// AllowAlways, Deny, ModifyInput — closed enum, zero not a member.
//
// Distinct from a free-form string: the validator, the session log, and
// the test suite all see four typed values, not a parser's inference.
type PermissionOutcome uint8

const (
	// _ PermissionOutcome = iota skips zero so it is NOT a member —
	// the constructor rejects it (S-APE-010).
	_ PermissionOutcome = iota

	// PermissionOutcomeAllowOnce is the outcome allowing this call only,
	// no policy effect.
	PermissionOutcomeAllowOnce

	// PermissionOutcomeAllowAlways is the outcome allowing this call and
	// remembering the rule for future calls of the same tool.
	PermissionOutcomeAllowAlways

	// PermissionOutcomeDeny is the outcome rejecting the call. Carries a
	// typed [Failure] (R-AEV-008, S-APE-012).
	PermissionOutcomeDeny

	// PermissionOutcomeModifyInput is the outcome allowing the call with
	// modified arguments. Carries the modified arguments as a payload
	// field (S-APE-011).
	PermissionOutcomeModifyInput

	// permissionOutcomeLimit is the first value outside the vocabulary.
	// It must stay last.
	permissionOutcomeLimit
)

// String renders the outcome, or "unset"/"permissionoutcome(N)" for a
// non-member, mirroring RunOutcome.String's posture.
func (o PermissionOutcome) String() string {
	if o >= permissionOutcomeLimit {
		return "permissionoutcome(" + strconv.FormatUint(uint64(o), 10) + ")"
	}
	switch o {
	case PermissionOutcomeAllowOnce:
		return "allow_once"
	case PermissionOutcomeAllowAlways:
		return "allow_always"
	case PermissionOutcomeDeny:
		return "deny"
	case PermissionOutcomeModifyInput:
		return "modify_input"
	case 0:
		return "unset"
	default:
		return "permissionoutcome(" + strconv.FormatUint(uint64(o), 10) + ")"
	}
}

// PermissionDecisionRequired is the payload opening the "may this call
// proceed?" loop (VL2-EVT-06, R-APE-001). It carries the tool call
// identity (the agent-level one a session log correlates, not Layer 1's
// raw one), the tool name, and the arguments as distinct readable
// fields — the surface a frontend needs to ask the user.
type PermissionDecisionRequired struct {
	callID    string
	name      string
	arguments []byte
}

func (PermissionDecisionRequired) kind() EventKind { return EventKindPermissionDecisionRequired }

// validate reports PermissionDecisionRequired's own broken rule, or nil.
// The call identity and tool name are required; arguments MUST be
// non-empty (the same "args are bytes, syntax is the adapter's problem"
// posture tool_event.go takes — ADR 0005 § D1 row 2).
func (p PermissionDecisionRequired) validate(at ai.Path) *ai.Violation {
	if p.callID == "" {
		return ai.Invalid(ai.ErrEmpty, under(at, ai.At("call_id"))...)
	}
	if p.name == "" {
		return ai.Invalid(ai.ErrEmpty, under(at, ai.At("name"))...)
	}
	if len(p.arguments) == 0 {
		return ai.Invalid(ai.ErrEmpty, under(at, ai.At("arguments"))...)
	}
	return nil
}

// NewPermissionDecisionRequired constructs the event opening the
// permission-decision loop (R-APE-001). run and turn are required;
// callID, name and arguments are all required. Arguments MUST be
// non-empty. On failure the zero [Event] is returned.
func NewPermissionDecisionRequired(run RunID, turn TurnID, callID, name string, arguments []byte) (Event, error) {
	if run == "" {
		return Event{}, ai.Invalid(ai.ErrEmpty, ai.At("run"))
	}
	if turn == "" {
		return Event{}, ai.Invalid(ai.ErrEmpty, ai.At("turn"))
	}
	payload := PermissionDecisionRequired{
		callID:    callID,
		name:      name,
		arguments: append([]byte(nil), arguments...),
	}
	if violation := payload.validate(nil); violation != nil {
		return Event{}, violation
	}
	return Event{payload: payload, run: run, turn: turn, hasTurn: true}, nil
}

// PermissionDecisionRequired returns the event's permission-decision-
// required payload, and whether the event carries one. Called on an
// event of another kind, or on an event that was never constructed, it
// returns the zero value and false, and never panics.
func (e Event) PermissionDecisionRequired() (PermissionDecisionRequired, bool) {
	payload, ok := e.payload.(PermissionDecisionRequired)
	return payload, ok
}

// CallID returns the tool call's identity the decision is about.
func (p PermissionDecisionRequired) CallID() string { return p.callID }

// Name returns the tool's name.
func (p PermissionDecisionRequired) Name() string { return p.name }

// Arguments returns the argument bytes exactly as supplied — Layer 2
// does not enforce JSON syntax (mirrors [ToolStart.Arguments]).
func (p PermissionDecisionRequired) Arguments() []byte { return p.arguments }

// String renders the payload for a diagnostic reader, naming the type,
// the call identity and the tool name — never the arguments' content.
func (p PermissionDecisionRequired) String() string {
	return "permission_decision_required(" + p.callID + " name=" + strconv.Quote(p.name) + ")"
}

// GoString renders the payload for the %#v verb.
func (p PermissionDecisionRequired) GoString() string { return p.String() }

// PermissionDecisionMade is the payload closing the permission-decision
// loop with a typed [PermissionOutcome] (R-APE-002). The ModifyInput
// outcome carries modified arguments as a payload field; the Deny
// outcome carries a typed [Failure] (R-AEV-008). Other outcomes carry
// neither.
type PermissionDecisionMade struct {
	callID             string
	outcome            PermissionOutcome
	modifiedArguments  []byte
	failure            *Failure
}

func (PermissionDecisionMade) kind() EventKind { return EventKindPermissionDecisionMade }

// validate reports PermissionDecisionMade's own broken rule, or nil:
// the call identity is required, the outcome is a member of the closed
// vocabulary, the modified arguments are present iff the outcome is
// ModifyInput, and the typed failure is present iff the outcome is
// Deny (mirroring RunEnd's iff rule).
func (p PermissionDecisionMade) validate(at ai.Path) *ai.Violation {
	if p.callID == "" {
		return ai.Invalid(ai.ErrEmpty, under(at, ai.At("call_id"))...)
	}
	if p.outcome == 0 || p.outcome >= permissionOutcomeLimit {
		return ai.Invalid(ai.ErrNotInVocabulary, under(at, ai.At("outcome"))...)
	}
	switch p.outcome {
	case PermissionOutcomeModifyInput:
		if len(p.modifiedArguments) == 0 {
			return ai.Invalid(ai.ErrEmpty, under(at, ai.At("modified_arguments"))...)
		}
		if p.failure != nil {
			return ai.Invalid(ai.ErrMisplaced, under(at, ai.At("failure"))...)
		}
	case PermissionOutcomeDeny:
		if p.failure == nil {
			return ai.Invalid(ai.ErrEmpty, under(at, ai.At("failure"))...)
		}
		if len(p.modifiedArguments) != 0 {
			return ai.Invalid(ai.ErrMisplaced, under(at, ai.At("modified_arguments"))...)
		}
	default:
		if len(p.modifiedArguments) != 0 {
			return ai.Invalid(ai.ErrMisplaced, under(at, ai.At("modified_arguments"))...)
		}
		if p.failure != nil {
			return ai.Invalid(ai.ErrMisplaced, under(at, ai.At("failure"))...)
		}
	}
	return nil
}

// NewPermissionDecisionMade constructs the event closing the
// permission-decision loop with a typed outcome (R-APE-002). The four
// typed outcomes are exhaustive; zero is not a member. The
// ModifyInput outcome requires non-empty modifiedArguments; the Deny
// outcome requires a non-nil failure; AllowOnce/AllowAlways accept
// neither. On failure the zero [Event] is returned.
func NewPermissionDecisionMade(run RunID, turn TurnID, callID string, outcome PermissionOutcome, modifiedArguments []byte, failure *Failure) (Event, error) {
	if run == "" {
		return Event{}, ai.Invalid(ai.ErrEmpty, ai.At("run"))
	}
	if turn == "" {
		return Event{}, ai.Invalid(ai.ErrEmpty, ai.At("turn"))
	}
	payload := PermissionDecisionMade{
		callID:            callID,
		outcome:           outcome,
		modifiedArguments: append([]byte(nil), modifiedArguments...),
		failure:           failure,
	}
	if violation := payload.validate(nil); violation != nil {
		return Event{}, violation
	}
	return Event{payload: payload, run: run, turn: turn, hasTurn: true}, nil
}

// PermissionDecisionMade returns the event's permission-decision-made
// payload, and whether the event carries one. Called on an event of
// another kind, or on an event that was never constructed, it returns
// the zero value and false, and never panics.
func (e Event) PermissionDecisionMade() (PermissionDecisionMade, bool) {
	payload, ok := e.payload.(PermissionDecisionMade)
	return payload, ok
}

// CallID returns the tool call's identity the decision answers.
func (p PermissionDecisionMade) CallID() string { return p.callID }

// Outcome returns the typed [PermissionOutcome] reached (R-APE-002).
func (p PermissionDecisionMade) Outcome() PermissionOutcome { return p.outcome }

// ModifiedArguments returns the modified argument bytes for a ModifyInput
// outcome; empty for other outcomes (S-APE-011).
func (p PermissionDecisionMade) ModifiedArguments() []byte { return p.modifiedArguments }

// Failure returns the typed [Failure] for a Deny outcome; nil for other
// outcomes (S-APE-012, R-AEV-008).
func (p PermissionDecisionMade) Failure() (*Failure, bool) { return p.failure, p.failure != nil }

// String renders the payload for a diagnostic reader, naming the type,
// the call identity and the outcome — never modifiedArguments' content
// nor a failure's own content.
func (p PermissionDecisionMade) String() string {
	return "permission_decision_made(" + p.callID + " outcome=" + p.outcome.String() + ")"
}

// GoString renders the payload for the %#v verb.
func (p PermissionDecisionMade) GoString() string { return p.String() }

// PermissionResolutionRemembered is the payload recording a remembered
// permission decision (R-APE-003, 0003:643-647). It carries the tool
// name and the remembered outcome — enough for the session log to
// explain why a later call was never asked.
//
// At most one event of this kind per stream per tool name — the
// descriptor declares CardinalityAtMostOne, the AG-04.3-reserved seam
// (`event_descriptor.go:103-120`), first exercised here.
type PermissionResolutionRemembered struct {
	toolName string
	outcome  PermissionOutcome
}

func (PermissionResolutionRemembered) kind() EventKind { return EventKindPermissionResolutionRemembered }

// validate reports PermissionResolutionRemembered's own broken rule, or
// nil: the tool name is required and the outcome is a member of the
// closed vocabulary.
func (p PermissionResolutionRemembered) validate(at ai.Path) *ai.Violation {
	if p.toolName == "" {
		return ai.Invalid(ai.ErrEmpty, under(at, ai.At("tool_name"))...)
	}
	if p.outcome == 0 || p.outcome >= permissionOutcomeLimit {
		return ai.Invalid(ai.ErrNotInVocabulary, under(at, ai.At("outcome"))...)
	}
	return nil
}

// NewPermissionResolutionRemembered constructs the event recording a
// remembered permission decision (R-APE-003). run is required; toolName
// and outcome are required. Outcome zero is rejected. On failure the
// zero [Event] is returned.
func NewPermissionResolutionRemembered(run RunID, toolName string, outcome PermissionOutcome) (Event, error) {
	if run == "" {
		return Event{}, ai.Invalid(ai.ErrEmpty, ai.At("run"))
	}
	payload := PermissionResolutionRemembered{toolName: toolName, outcome: outcome}
	if violation := payload.validate(nil); violation != nil {
		return Event{}, violation
	}
	return Event{payload: payload, run: run}, nil
}

// PermissionResolutionRemembered returns the event's resolution-
// remembered payload, and whether the event carries one. Called on an
// event of another kind, or on an event that was never constructed, it
// returns the zero value and false, and never panics.
func (e Event) PermissionResolutionRemembered() (PermissionResolutionRemembered, bool) {
	payload, ok := e.payload.(PermissionResolutionRemembered)
	return payload, ok
}

// ToolName returns the tool's name the decision is remembered for.
func (p PermissionResolutionRemembered) ToolName() string { return p.toolName }

// cardinalityDiscriminator makes the tool name the payload's at-most-one
// identity: R-APE-003's cardinality is per stream PER TOOL NAME, so
// resolution_remembered events for two different tools are two
// identities, while the same tool twice remains a repeat (S-APE-082).
func (p PermissionResolutionRemembered) cardinalityDiscriminator() string { return p.toolName }

// Outcome returns the remembered typed [PermissionOutcome].
func (p PermissionResolutionRemembered) Outcome() PermissionOutcome { return p.outcome }

// String renders the payload for a diagnostic reader, naming the type,
// the tool name and the outcome.
func (p PermissionResolutionRemembered) String() string {
	return "permission_resolution_remembered(" + strconv.Quote(p.toolName) + " outcome=" + p.outcome.String() + ")"
}

// GoString renders the payload for the %#v verb.
func (p PermissionResolutionRemembered) GoString() string { return p.String() }
