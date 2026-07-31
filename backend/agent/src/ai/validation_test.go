// Tests for AI-04 — the validation error taxonomy.
//
// The package under test is imported by its module path from the external test
// package ai_test, so every assertion below is written against exactly the
// surface a consumer in another package sees. Nothing here can reach an
// unexported field, which is the point: AI-05 through AI-13 will report through
// this taxonomy from inside package ai, and Layer 2 will inspect it from
// outside.
package ai_test

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// AI-04.2 — a wrapped failure still matches its sentinel.
//
// V-FAIL-02 requires a validation sentinel to be "matchable through at least one
// layer of wrapping, so a caller can classify a failure it received indirectly".
// The negative assertion is what stops a match-everything implementation from
// passing: matching one sentinel must exclude the others.
func TestViolation_WrappedFailure_StillMatchesItsSentinel(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		wrap func(error) error
	}{
		{"unwrapped", func(err error) error { return err }},
		{"wrapped once", func(err error) error { return fmt.Errorf("build request: %w", err) }},
		{"wrapped three times", func(err error) error {
			for range 3 {
				err = fmt.Errorf("layer: %w", err)
			}
			return err
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.wrap(ai.Invalid(ai.ErrEmpty))

			if !errors.Is(err, ai.ErrEmpty) {
				t.Errorf("errors.Is(err, ErrEmpty) = false, want true")
			}
			if errors.Is(err, ai.ErrMalformed) {
				t.Errorf("errors.Is(err, ErrMalformed) = true, want false")
			}
		})
	}

	// *(pin)* — green from birth. The rule classes are a closed set, so a
	// failure matches exactly one of them. It fails the day a later milestone
	// appends a class by wrapping an existing one, which would make errors.Is
	// ambiguous for every consumer.
	t.Run("a failure matches exactly one rule class", func(t *testing.T) {
		t.Parallel()

		for _, class := range ruleClasses {
			err := fmt.Errorf("build request: %w", ai.Invalid(class, ai.At("model")))

			matched := 0
			for _, other := range ruleClasses {
				if errors.Is(err, other) {
					matched++
				}
			}
			if matched != 1 {
				t.Errorf("a failure of class %q matched %d rule classes, want exactly 1", class, matched)
			}
		}
	})
}

// ruleClasses mirrors the package's own closed registry. It is written out here
// rather than exported from the package: a consumer never needs to iterate the
// set, and exporting it to satisfy a test would widen the surface AI-40 freezes.
var ruleClasses = []error{
	ai.ErrEmpty,
	ai.ErrNotInVocabulary,
	ai.ErrOutOfRange,
	ai.ErrMalformed,
	ai.ErrUnresolvedReference,
}

// AI-04.2 — a failure carrying positional context is extracted by errors.As,
// and the extracted value names the position unambiguously.
//
// V-FAIL-03 requires the position to name "which message, which content index,
// which tool" without becoming a second, parallel failure vocabulary — hence a
// single errors.As target. The position is read programmatically, step by step,
// rather than by parsing the rendered message: a consumer that had to parse text
// would be doing exactly what V-FAIL-06 records as the failure mode of an open
// classification.
func TestViolation_PositionalContext_IsExtractedByErrorsAs(t *testing.T) {
	t.Parallel()

	type step struct {
		name    string
		index   int
		indexed bool
	}

	cases := []struct {
		name string
		at   []ai.Step
		want []step
	}{
		{
			name: "a field with no index",
			at:   []ai.Step{ai.At("model")},
			want: []step{{name: "model"}},
		},
		{
			name: "the content of a message",
			at:   []ai.Step{ai.AtIndex("messages", 2), ai.AtIndex("content", 0)},
			want: []step{{"messages", 2, true}, {"content", 0, true}},
		},
		{
			name: "one tool of the declared set",
			at:   []ai.Step{ai.AtIndex("tools", 3)},
			want: []step{{"tools", 3, true}},
		},
		{
			name: "a field of one tool",
			at:   []ai.Step{ai.AtIndex("tools", 3), ai.At("schema")},
			want: []step{{"tools", 3, true}, {name: "schema"}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := fmt.Errorf("validate request: %w", ai.Invalid(ai.ErrEmpty, tc.at...))

			var v *ai.Violation
			if !errors.As(err, &v) {
				t.Fatalf("errors.As did not reach the violation through one wrapper")
			}

			path := v.Path()
			if len(path) != len(tc.want) {
				t.Fatalf("len(path) = %d, want %d", len(path), len(tc.want))
			}
			for i, want := range tc.want {
				if got := path[i].Name(); got != want.name {
					t.Errorf("step %d: Name() = %q, want %q", i, got, want.name)
				}
				index, indexed := path[i].Index()
				if indexed != want.indexed {
					t.Errorf("step %d: Index() reported indexed = %t, want %t", i, indexed, want.indexed)
				}
				if indexed && index != want.index {
					t.Errorf("step %d: Index() = %d, want %d", i, index, want.index)
				}
			}
		})
	}

	// A failure is passed upward and logged; a consumer that appends to the
	// position it received must not be able to rewrite the failure's own.
	t.Run("the returned position is a copy", func(t *testing.T) {
		t.Parallel()

		v := ai.Invalid(ai.ErrEmpty, ai.AtIndex("messages", 2))

		tampered := v.Path()
		tampered[0] = ai.At("tampered")

		if got := v.Path()[0].Name(); got != "messages" {
			t.Errorf("Path()[0].Name() = %q after mutating a previously returned position, want %q", got, "messages")
		}
	})
}

// sentinelBody is a value that could only have come from caller content: a
// message body, an argument payload, a credential. Every test that asserts the
// redaction posture plants this exact string and then proves its absence.
const sentinelBody = "CACHICAMA-SENTINEL-BODY-9f3a"

// AI-04.2 — the rendered message never carries a content body, an argument
// payload, or a credential.
//
// V-FAIL-13 puts the redaction posture here rather than at AI-36: "the posture
// starts at the first thing in the package that formats caller data — a
// validation failure — not at the hardening milestone." A validation failure is
// that first thing, and it has exactly two dynamic inputs: the structural name
// and the rule it was built from. Both are asserted below.
func TestViolation_RenderedMessage_CarriesNoCallerContent(t *testing.T) {
	t.Parallel()

	// The message has to be worth rendering at all, or "carries no caller
	// content" is satisfied by rendering nothing.
	t.Run("the message names its rule class and its position", func(t *testing.T) {
		t.Parallel()

		got := ai.Invalid(ai.ErrEmpty, ai.AtIndex("messages", 2), ai.At("content")).Error()

		for _, want := range []string{ai.ErrEmpty.Error(), "messages[2]", "content"} {
			if !strings.Contains(got, want) {
				t.Errorf("Error() = %q, want it to contain %q", got, want)
			}
		}
	})

	cases := []struct {
		name string
		err  error
	}{
		{
			name: "a content body supplied as a structural name",
			err:  ai.Invalid(ai.ErrEmpty, ai.At(sentinelBody)),
		},
		{
			name: "an over-long structural name",
			err:  ai.Invalid(ai.ErrEmpty, ai.At(strings.Repeat(sentinelBody, 8))),
		},
		{
			name: "an argument payload supplied as a structural name",
			err:  ai.Invalid(ai.ErrMalformed, ai.AtIndex("tools", 1), ai.At(`{"api_key":"`+sentinelBody+`"}`)),
		},
		{
			name: "a rule error whose own text carries a content body",
			err:  ai.Invalid(fmt.Errorf("rejected %s: %w", sentinelBody, ai.ErrEmpty), ai.At("model")),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := tc.err.Error()

			// Not merely the whole body: a prefix of a secret is still a
			// secret, which is why a name that fails the filter is replaced
			// whole rather than truncated.
			for n := len(sentinelBody); n >= 4; n-- {
				if strings.Contains(got, sentinelBody[:n]) {
					t.Fatalf("Error() = %q, want it to contain no prefix of the sentinel body (found %q)",
						got, sentinelBody[:n])
				}
			}
			if !strings.Contains(got, ai.ErrEmpty.Error()) && !strings.Contains(got, ai.ErrMalformed.Error()) {
				t.Errorf("Error() = %q, want it to still name its rule class", got)
			}
		})
	}

	// Rendering from the registered class must not cost matchability: the rule
	// the failure was built from stays reachable by errors.Is.
	t.Run("a rule error carrying a body is still matchable", func(t *testing.T) {
		t.Parallel()

		err := ai.Invalid(fmt.Errorf("rejected %s: %w", sentinelBody, ai.ErrEmpty), ai.At("model"))

		if !errors.Is(err, ai.ErrEmpty) {
			t.Errorf("errors.Is(err, ErrEmpty) = false, want true")
		}
	})
}

// AI-04.3 — a value violating several rules reports the first in the documented
// order, identically across runs and under the race detector.
//
// V-FAIL-04 requires the order to be documented and deterministic. AI-04.1
// decided the reporting policy that goes with it: the first violation, not all
// of them, because a joined error matches a sentinel when any member matches and
// that takes back everything the per-class sentinel decision bought.
//
// Determinism is proven by repetition and by concurrency rather than by
// inspection: "no map iteration decides which rule fires" is a claim about
// behavior under a scheduler.
func TestFirstFailure_SeveralViolatedRules_ReportsTheFirstInOrder(t *testing.T) {
	t.Parallel()

	// Rules 2, 3 and 5 are violated. The list records which rules ran, so
	// "stops at the first violation" is a claim about evaluation and not only
	// about the result.
	orderedRules := func(evaluated *[]int) []ai.Rule {
		violate := func(n int, class error) ai.Rule {
			return func() *ai.Violation {
				*evaluated = append(*evaluated, n)
				return ai.Invalid(class, ai.AtIndex("messages", n))
			}
		}
		pass := func(n int) ai.Rule {
			return func() *ai.Violation {
				*evaluated = append(*evaluated, n)
				return nil
			}
		}
		return []ai.Rule{
			pass(1),
			violate(2, ai.ErrNotInVocabulary),
			violate(3, ai.ErrEmpty),
			pass(4),
			violate(5, ai.ErrMalformed),
		}
	}

	const wantMessage = "messages[2]: value is outside a closed vocabulary"

	t.Run("reports the first violated rule and stops there", func(t *testing.T) {
		t.Parallel()

		var evaluated []int

		err := ai.FirstFailure(orderedRules(&evaluated)...)

		if err == nil {
			t.Fatalf("FirstFailure() = <nil>, want a violation")
		}
		if !errors.Is(err, ai.ErrNotInVocabulary) {
			t.Errorf("FirstFailure() = %v, want the second rule's class", err)
		}
		if got := err.Error(); got != wantMessage {
			t.Errorf("FirstFailure().Error() = %q, want %q", got, wantMessage)
		}
		if want := []int{1, 2}; !slices.Equal(evaluated, want) {
			t.Errorf("rules evaluated = %v, want %v — evaluation must be lazy and must stop", evaluated, want)
		}
	})

	// The typed-nil trap, pinned once here so it is not reproduced at every
	// call site in AI-05 through AI-13: a nil *Violation placed in an error
	// interface is not a nil error.
	t.Run("no violated rule yields a genuinely absent failure", func(t *testing.T) {
		t.Parallel()

		cases := map[string][]ai.Rule{
			"an empty rule list": nil,
			"one passing rule":   {func() *ai.Violation { return nil }},
			"a nil rule entry":   {nil, func() *ai.Violation { return nil }},
		}
		for name, rules := range cases {
			if err := ai.FirstFailure(rules...); err != nil {
				t.Errorf("%s: FirstFailure() = %v (type %T), want a nil error", name, err, err)
			}
		}
	})

	t.Run("repeated evaluation agrees", func(t *testing.T) {
		t.Parallel()

		for run := range 1000 {
			var evaluated []int

			err := ai.FirstFailure(orderedRules(&evaluated)...)

			if err == nil || err.Error() != wantMessage {
				t.Fatalf("run %d: FirstFailure() = %v, want %q", run, err, wantMessage)
			}
		}
	})

	t.Run("concurrent evaluation agrees", func(t *testing.T) {
		t.Parallel()

		const goroutines = 64

		var wg sync.WaitGroup
		got := make([]string, goroutines)

		for i := range goroutines {
			wg.Add(1)
			go func() {
				defer wg.Done()

				var evaluated []int

				if err := ai.FirstFailure(orderedRules(&evaluated)...); err != nil {
					got[i] = err.Error()
				}
			}()
		}
		wg.Wait()

		for i, message := range got {
			if message != wantMessage {
				t.Errorf("goroutine %d: FirstFailure() = %q, want %q", i, message, wantMessage)
			}
		}
	})
}

// AI-04.3 — a validation failure never panics on any input.
//
// A caller-contract failure is constructed on a path that is already going
// wrong. A panic there replaces a diagnosable bug with an undiagnosable one, and
// it does so in the one code path every contract in this package shares. The
// table below is the input space: nothing, nil, zero values, the bounds, and
// past them.
func TestViolation_ExtremeInputs_NeverPanics(t *testing.T) {
	t.Parallel()

	deep := make([]ai.Step, 10_000)
	for i := range deep {
		deep[i] = ai.AtIndex("messages", i)
	}

	cases := []struct {
		name         string
		act          func() string
		wantContains string
		wantAbsent   string
	}{
		{
			name: "a nil failure, rendered",
			act:  func() string { var v *ai.Violation; return v.Error() },
		},
		{
			name: "a nil failure, matched and unwrapped",
			act: func() string {
				var v *ai.Violation
				return fmt.Sprint(errors.Is(v, ai.ErrEmpty), errors.Unwrap(v))
			},
		},
		{
			name: "a nil failure, position read",
			act:  func() string { var v *ai.Violation; return fmt.Sprint(len(v.Path())) },
		},
		{
			name:         "no rule",
			act:          func() string { return ai.Invalid(nil, ai.At("model")).Error() },
			wantContains: "unnamed",
		},
		{
			name:         "no rule and no position",
			act:          func() string { return ai.Invalid(nil).Error() },
			wantContains: "unnamed",
		},
		{
			name:         "a rule that is not a registered class",
			act:          func() string { return ai.Invalid(errors.New("from elsewhere"), ai.At("model")).Error() },
			wantContains: "unregistered",
			wantAbsent:   "from elsewhere",
		},
		{
			name:         "a zero-value step",
			act:          func() string { return ai.Invalid(ai.ErrEmpty, ai.Step{}).Error() },
			wantContains: "?",
		},
		{
			name:         "an empty name",
			act:          func() string { return ai.Invalid(ai.ErrEmpty, ai.At("")).Error() },
			wantContains: "?",
		},
		{
			name:       "a negative index",
			act:        func() string { return ai.Invalid(ai.ErrEmpty, ai.AtIndex("messages", -1)).Error() },
			wantAbsent: "[-",
		},
		{
			name:         "the maximum index",
			act:          func() string { return ai.Invalid(ai.ErrEmpty, ai.AtIndex("messages", math.MaxInt)).Error() },
			wantContains: strconv.Itoa(math.MaxInt),
		},
		{
			name:         "a name at the length bound",
			act:          func() string { return ai.Invalid(ai.ErrEmpty, ai.At(strings.Repeat("a", 32))).Error() },
			wantContains: strings.Repeat("a", 32),
		},
		{
			name:         "a name far beyond the length bound",
			act:          func() string { return ai.Invalid(ai.ErrEmpty, ai.At(strings.Repeat("a", 1<<20))).Error() },
			wantContains: "?",
		},
		{
			name:         "a position ten thousand steps deep",
			act:          func() string { return ai.Invalid(ai.ErrEmpty, deep...).Error() },
			wantContains: "messages[9999]",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			defer func() {
				if recovered := recover(); recovered != nil {
					t.Errorf("panicked: %v", recovered)
				}
			}()

			got := tc.act()

			if got == "" {
				t.Fatalf("act() = %q, want a non-empty result", got)
			}
			if tc.wantContains != "" && !strings.Contains(got, tc.wantContains) {
				t.Errorf("act() = %q, want it to contain %q", got, tc.wantContains)
			}
			if tc.wantAbsent != "" && strings.Contains(got, tc.wantAbsent) {
				t.Errorf("act() = %q, want it not to contain %q", got, tc.wantAbsent)
			}
		})
	}
}

// AI-04.3 — *(pin)* constructing a failure retains no reference to the offending
// content.
//
// Green from birth and exempt from red-first by doc 0002's leaf anatomy, and
// still fully mechanical. It differs from the AI-04.2 message test in what it
// enumerates: that test covers the channels as they are used today, this one
// covers them structurally — every dynamic input a rendered message has, every
// rendering path, and the position as data rather than as text.
//
// The refactor this exists to catch is "put the offending value in the message,
// it is easier to debug". It cannot be made without failing here.
func TestViolation_SentinelBody_IsNeverRetainedInTheMessage(t *testing.T) {
	t.Parallel()

	channels := []struct {
		name      string
		violation *ai.Violation
	}{
		{"the structural name of a field", ai.Invalid(ai.ErrEmpty, ai.At(sentinelBody))},
		{"the structural name of an element", ai.Invalid(ai.ErrEmpty, ai.AtIndex(sentinelBody, 2))},
		{"a deeper segment of the position", ai.Invalid(ai.ErrEmpty, ai.AtIndex("messages", 2), ai.At(sentinelBody))},
		{"a name past the length bound", ai.Invalid(ai.ErrEmpty, ai.At(strings.Repeat(sentinelBody, 4)))},
		{"the rule the failure was built from", ai.Invalid(fmt.Errorf("%s: %w", sentinelBody, ai.ErrEmpty), ai.At("model"))},
		{"a rule that is not a registered class", ai.Invalid(errors.New(sentinelBody), ai.At("model"))},
	}

	for _, channel := range channels {
		t.Run(channel.name, func(t *testing.T) {
			t.Parallel()

			renderings := []string{
				channel.violation.Error(),
				channel.violation.Path().String(),
				fmt.Sprint(channel.violation),
				fmt.Sprintf("%v", error(channel.violation)),
			}
			for _, rendered := range renderings {
				// Every prefix, not only the whole body: a truncating
				// implementation would pass a whole-string check while still
				// leaking the beginning of a credential.
				for n := len(sentinelBody); n >= 4; n-- {
					if strings.Contains(rendered, sentinelBody[:n]) {
						t.Fatalf("rendering %q retains %q", rendered, sentinelBody[:n])
					}
				}
			}

			// Not merely absent from the message: absent from the value. A
			// position that stored the body and omitted it from the message
			// would still put it in anything that logs the position.
			for i, step := range channel.violation.Path() {
				if strings.Contains(step.Name(), sentinelBody[:4]) {
					t.Errorf("step %d retains the sentinel body as data: %q", i, step.Name())
				}
			}
		})
	}
}
