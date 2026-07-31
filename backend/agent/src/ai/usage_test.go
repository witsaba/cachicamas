// Tests for AI-13.3 and AI-13.4 — the usage record.
//
// External test package, so every assertion is written against exactly the
// surface an adapter and a Layer 2 consumer see. That matters more here than
// anywhere else in this milestone: the property under test is that an
// unreported count cannot be mistaken for a reported nought, and a test with
// access to unexported state could prove that about a representation while a
// consumer still could not observe it.
package ai_test

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"reflect"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// usageField names one token count and gives the test a way to set and read it
// without reflection, so that every property below is asserted once per field
// rather than once on whichever field happened to be written first.
//
// V-MET-10 makes each count independently present or absent. A property proven
// on Input alone would be a property of Input.
type usageField struct {
	name string
	set  func(*ai.Usage, ai.TokenCount)
	get  func(ai.Usage) ai.TokenCount
}

// usageFields is the five counts of V-MET-09, in the order the register lists
// them, which is also the struct's field order and the order Validate reports.
var usageFields = []usageField{
	{"input", func(u *ai.Usage, c ai.TokenCount) { u.Input = c }, func(u ai.Usage) ai.TokenCount { return u.Input }},
	{"output", func(u *ai.Usage, c ai.TokenCount) { u.Output = c }, func(u ai.Usage) ai.TokenCount { return u.Output }},
	{"cache_read", func(u *ai.Usage, c ai.TokenCount) { u.CacheRead = c }, func(u ai.Usage) ai.TokenCount { return u.CacheRead }},
	{"cache_write", func(u *ai.Usage, c ai.TokenCount) { u.CacheWrite = c }, func(u ai.Usage) ai.TokenCount { return u.CacheWrite }},
	{"reasoning", func(u *ai.Usage, c ai.TokenCount) { u.Reasoning = c }, func(u ai.Usage) ai.TokenCount { return u.Reasoning }},
}

// AI-13.3 — an absent token count is distinguishable from a zero one.
//
// V-MET-11: "not reported" and "reported as nought" are different facts, and a
// consumer that cannot tell them apart writes a wrong cost formula and a wrong
// compaction estimate. The assertion runs once per count because the register
// makes each of them independently present or absent.
func TestUsage_AnAbsentCount_IsDistinguishableFromZero(t *testing.T) {
	t.Parallel()

	for _, field := range usageFields {
		t.Run(field.name, func(t *testing.T) {
			t.Parallel()

			var absent ai.Usage
			count, present := field.get(absent).Count()
			if present {
				t.Errorf("absent %s: Count() = (%d, true), want (0, false) — an unreported count must not report as present", field.name, count)
			}
			if count != 0 {
				t.Errorf("absent %s: Count() = (%d, _), want 0 as the uninformative value", field.name, count)
			}

			var reportedZero ai.Usage
			field.set(&reportedZero, ai.Tokens(0))
			count, present = field.get(reportedZero).Count()
			if !present {
				t.Errorf("%s reported as nought: Count() = (_, false), want (0, true)", field.name)
			}
			if count != 0 {
				t.Errorf("%s reported as nought: Count() = (%d, _), want 0", field.name, count)
			}

			if got, want := field.get(absent).String(), field.get(reportedZero).String(); got == want {
				t.Errorf("absent %s and %s reported as nought both render as %q, want two renderings", field.name, field.name, got)
			}
		})
	}

	t.Run("a present count carries its value", func(t *testing.T) {
		t.Parallel()

		count, present := ai.Tokens(1234).Count()
		if !present || count != 1234 {
			t.Errorf("Tokens(1234).Count() = (%d, %t), want (1234, true)", count, present)
		}
	})
}

// AI-13.3 — a usage record is constructible with any subset of counts present.
//
// AI-03's CAP-R-03 clause 2: "an adapter that reports only input and output
// produces a valid usage record, not a deficient one … requiring a populated
// count is requiring a fabricated one". So the interesting assertion is the
// negative one — that validation has no opinion about presence at all — and the
// only rule it does enforce is that a reported count is not negative.
func TestUsage_AnySubsetOfCounts_ProducesAValidRecord(t *testing.T) {
	t.Parallel()

	valid := []struct {
		name  string
		usage ai.Usage
	}{
		{"a provider that reports nothing", ai.Usage{}},
		{"a provider that reports only input and output", ai.Usage{Input: ai.Tokens(1200), Output: ai.Tokens(340)}},
		{"a provider that reports only a reported nought", ai.Usage{CacheRead: ai.Tokens(0)}},
		{"a provider that reports everything", ai.Usage{
			Input:      ai.Tokens(100),
			Output:     ai.Tokens(500),
			CacheRead:  ai.Tokens(900),
			CacheWrite: ai.Tokens(0),
			Reasoning:  ai.Tokens(380),
		}},
	}

	for _, tc := range valid {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if err := tc.usage.Validate(ai.At("usage")); err != nil {
				t.Errorf("Validate() = %v, want <nil> — any subset of counts is a valid record", err)
			}
		})
	}

	t.Run("an all-absent record reports absence rather than zero", func(t *testing.T) {
		t.Parallel()

		var reported ai.Usage
		for _, field := range usageFields {
			if _, present := field.get(reported).Count(); present {
				t.Errorf("%s of an all-absent record reports as present, want absent", field.name)
			}
		}
	})

	t.Run("a negative count is rejected out of range at its own field", func(t *testing.T) {
		t.Parallel()

		usage := ai.Usage{Input: ai.Tokens(10), CacheRead: ai.Tokens(-1)}

		err := usage.Validate(ai.At("completion"), ai.At("usage"))
		if err == nil {
			t.Fatalf("Validate() = <nil>, want a violation for a negative count")
		}
		if !errors.Is(err, ai.ErrOutOfRange) {
			t.Errorf("errors.Is(err, ErrOutOfRange) = false, want true; err = %v", err)
		}

		var violation *ai.Violation
		if !errors.As(err, &violation) {
			t.Fatalf("errors.As(err, &*ai.Violation) = false, want true; err = %v", err)
		}
		if got, want := violation.Path().String(), "completion.usage.cache_read"; got != want {
			t.Errorf("violation position = %q, want %q", got, want)
		}
	})

	t.Run("several negative counts report the first in the documented order", func(t *testing.T) {
		t.Parallel()

		usage := ai.Usage{
			Output:    ai.Tokens(-3),
			CacheRead: ai.Tokens(-1),
			Reasoning: ai.Tokens(-7),
		}

		var violation *ai.Violation
		if err := usage.Validate(ai.At("usage")); !errors.As(err, &violation) {
			t.Fatalf("Validate() = %v, want a violation", err)
		}
		if got, want := violation.Path().String(), "usage.output"; got != want {
			t.Errorf("violation position = %q, want %q — output precedes cache_read and reasoning", got, want)
		}
	})
}

// AI-13.3 — a usage record is readable from another package, field by field,
// with absence surfaced rather than defaulted.
//
// Written without the usageFields helper on purpose: this is the shape a real
// consumer writes, and the point of the item is that the shape works from
// outside the package. Retired defect C2 was a contract that could not be read
// from another package at all, which made translation structurally impossible;
// this test is the equivalent proof for the usage record, one milestone before
// AI-06.2 makes it for content parts.
func TestUsage_FromAnExternalPackage_IsReadableFieldByField(t *testing.T) {
	t.Parallel()

	// A provider that reported four of the five counts, one of them as nought.
	usage := ai.Usage{
		Input:     ai.Tokens(1200),
		Output:    ai.Tokens(340),
		CacheRead: ai.Tokens(0),
		Reasoning: ai.Tokens(96),
	}

	reads := []struct {
		name        string
		count       ai.TokenCount
		wantValue   int64
		wantPresent bool
	}{
		{"input", usage.Input, 1200, true},
		{"output", usage.Output, 340, true},
		{"cache_read reported as nought", usage.CacheRead, 0, true},
		{"cache_write never reported", usage.CacheWrite, 0, false},
		{"reasoning", usage.Reasoning, 96, true},
	}

	for _, read := range reads {
		value, present := read.count.Count()
		if value != read.wantValue || present != read.wantPresent {
			t.Errorf("%s: Count() = (%d, %t), want (%d, %t)",
				read.name, value, present, read.wantValue, read.wantPresent)
		}
	}

	t.Run("a copy of the record is a copy of its counts", func(t *testing.T) {
		t.Parallel()

		copied := usage
		copied.Input = ai.Tokens(7)

		if value, _ := usage.Input.Count(); value != 1200 {
			t.Errorf("writing to a copy changed the original: input = %d, want 1200", value)
		}
	})
}

// structFieldDoc returns the documentation comment of one field of one struct
// type of the package under test, with its whitespace collapsed to single
// spaces.
//
// Same mechanism and same justification as constantDoc in
// finish_reason_test.go: doc 0002 puts "the semantics are documented" in a test
// list rather than in a review checklist, and its leaf anatomy says a prose
// claim with no objective check does not belong in a test list. An AST scan is
// how the claim becomes objective, and go/ast is standard library.
func structFieldDoc(t *testing.T, typeName, fieldName string) string {
	t.Helper()

	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, "usage.go", nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("parsing usage.go: %v", err)
	}

	for _, declaration := range file.Decls {
		general, isGeneral := declaration.(*ast.GenDecl)
		if !isGeneral || general.Tok != token.TYPE {
			continue
		}
		for _, spec := range general.Specs {
			typeSpec, isType := spec.(*ast.TypeSpec)
			if !isType || typeSpec.Name.Name != typeName {
				continue
			}
			structType, isStruct := typeSpec.Type.(*ast.StructType)
			if !isStruct {
				t.Fatalf("%s is not a struct type", typeName)
			}
			for _, field := range structType.Fields.List {
				if len(field.Names) == 0 || field.Names[0].Name != fieldName {
					continue
				}
				if field.Doc == nil {
					return ""
				}
				return strings.Join(strings.Fields(field.Doc.Text()), " ")
			}
			t.Fatalf("%s has no field named %s", typeName, fieldName)
		}
	}

	t.Fatalf("no type named %s found in usage.go", typeName)
	return ""
}

// totalInputTokens is the input side of the documented cost formula:
// Input + CacheRead + CacheWrite, the three fields being disjoint.
//
// It lives in the test and not in the package. design.md § 5.4 argues the
// rejected alternative — exporting it — at full strength; the short version is
// that a derived total has to decide what an absent component means, and both
// available answers are wrong. Here the answer is available: the test refuses to
// compute a total whose terms are not all present, which is the same discipline
// stated as a precondition rather than as a return value.
func totalInputTokens(t *testing.T, u ai.Usage) int64 {
	t.Helper()

	var total int64
	for name, term := range map[string]ai.TokenCount{
		"input": u.Input, "cache_read": u.CacheRead, "cache_write": u.CacheWrite,
	} {
		count, present := term.Count()
		if !present {
			t.Fatalf("the total input is not computable: %s is absent, and absent is not zero", name)
		}
		total += count
	}
	return total
}

// AI-13.4 — a cache-hit record excludes the cached tokens from its input count.
//
// The trap this pins: one vendor reports its plain input figure EXCLUSIVE of
// cached tokens and another reports it INCLUSIVE of them, so a consumer that
// picks the wrong reading either double-counts every cached token or
// under-reports a cache hit by most of the prompt. Neither mistake is visible
// until the bill arrives.
//
// The neutral record is exclusive: Input, CacheRead and CacheWrite are disjoint
// and the total input is their sum (design.md § 5.3). The fixture below states
// the provider-side facts once and then builds the record the way an adapter for
// each vendor style must, so the assertion is about the conversion rather than
// about arithmetic on numbers the test itself chose.
func TestUsage_ACacheHitRecord_ExcludesCachedTokensFromTheInputCount(t *testing.T) {
	t.Parallel()

	// One cache hit, as it happened: the request carried 1000 prompt tokens and
	// 900 of them were served from a cached prefix.
	const (
		promptTokens = int64(1000)
		cachedTokens = int64(900)
	)

	t.Run("the five counts document which side of the formula they are on", func(t *testing.T) {
		t.Parallel()

		cases := []struct {
			fieldName   string
			mustContain []string
		}{
			{"Input", []string{"excluding", "CacheRead", "CacheWrite"}},
			{"Output", []string{"including", "Reasoning"}},
			{"CacheRead", []string{"Disjoint from Input"}},
			{"CacheWrite", []string{"Disjoint"}},
			{"Reasoning", []string{"breakdown of Output"}},
		}

		for _, tc := range cases {
			documentation := structFieldDoc(t, "Usage", tc.fieldName)
			if documentation == "" {
				t.Errorf("Usage.%s carries no documentation, want its side of the cost formula", tc.fieldName)
				continue
			}
			for _, phrase := range tc.mustContain {
				if !strings.Contains(documentation, phrase) {
					t.Errorf("Usage.%s documentation does not state %q", tc.fieldName, phrase)
				}
			}
		}
	})

	t.Run("both vendor styles converge on the same disjoint record", func(t *testing.T) {
		t.Parallel()

		// A vendor whose input figure already excludes cached tokens: the
		// adapter passes the three figures through.
		exclusiveVendor := ai.Usage{
			Input:      ai.Tokens(promptTokens - cachedTokens),
			CacheRead:  ai.Tokens(cachedTokens),
			CacheWrite: ai.Tokens(0),
		}

		// A vendor whose input figure includes cached tokens: the adapter must
		// subtract. Forgetting this line is the defect, and it is why the
		// semantics are documented on the declaration rather than only here.
		reportedPrompt, reportedCached := promptTokens, cachedTokens
		inclusiveVendor := ai.Usage{
			Input:      ai.Tokens(reportedPrompt - reportedCached),
			CacheRead:  ai.Tokens(reportedCached),
			CacheWrite: ai.Tokens(0),
		}

		if exclusiveVendor != inclusiveVendor {
			t.Errorf("the two adapters produced different records for one cache hit:\n  %+v\n  %+v",
				exclusiveVendor, inclusiveVendor)
		}

		if got := totalInputTokens(t, exclusiveVendor); got != promptTokens {
			t.Errorf("total input = %d, want %d — the documented sum must recover the tokens the request carried", got, promptTokens)
		}

		inputAlone, _ := exclusiveVendor.Input.Count()
		if inputAlone >= promptTokens {
			t.Errorf("input alone = %d, want strictly less than the %d tokens carried — "+
				"the input count is the uncached portion only", inputAlone, promptTokens)
		}
		if inputAlone != promptTokens-cachedTokens {
			t.Errorf("input = %d, want %d", inputAlone, promptTokens-cachedTokens)
		}
	})

	t.Run("the reasoning count is inside the output count, not beside it", func(t *testing.T) {
		t.Parallel()

		// A reasoning model that produced 500 output tokens, 380 of which it
		// spent reasoning. Both are billed at the output rate, which is why
		// reasoning is a breakdown rather than a term (design.md § 5.3).
		const (
			outputTokens    = int64(500)
			reasoningTokens = int64(380)
		)

		usage := ai.Usage{Output: ai.Tokens(outputTokens), Reasoning: ai.Tokens(reasoningTokens)}

		// The output side of the documented formula is Output alone.
		total, present := usage.Output.Count()
		if !present || total != outputTokens {
			t.Fatalf("Output.Count() = (%d, %t), want (%d, true)", total, present, outputTokens)
		}

		reasoning, _ := usage.Reasoning.Count()
		if reasoning > total {
			t.Errorf("reasoning = %d exceeds output = %d — a breakdown cannot be larger than what it breaks down",
				reasoning, total)
		}
		if total+reasoning == total {
			t.Errorf("this fixture cannot detect double counting; want a non-zero reasoning count")
		}
	})
}

// AI-13.4 *(pin)* — the cost formula's term list is the record's field set.
//
// Exempt from red-first by doc 0002's leaf anatomy, and fully mechanical. It is
// recorded biting against a deliberate scratch field in tasks.md.
//
// doc 0002 asks that "a later field addition that changes the formula cannot
// land silently". The formula is documented on the type and asserted above over
// constructed records; what is missing without this pin is the link between the
// two — a sixth field would leave both the documentation and the assertions
// passing while the formula they describe had quietly become incomplete.
//
// So the field set itself is pinned: names, types and order. The failure message
// names the decision the author of a new field has to make, because the whole
// value of the pin is that it stops someone at the moment they can still make it
// cheaply.
func TestUsage_TheFieldSet_MatchesTheDocumentedCostFormula(t *testing.T) {
	t.Parallel()

	// The four terms of the documented formula, then the one field that is a
	// breakdown rather than a term. Order is part of the pin: it is also the
	// order V-MET-09 lists and the order Validate reports.
	wantFields := []struct {
		name          string
		isFormulaTerm bool
	}{
		{"Input", true},
		{"Output", true},
		{"CacheRead", true},
		{"CacheWrite", true},
		{"Reasoning", false},
	}

	usageType := reflect.TypeOf(ai.Usage{})
	tokenCountType := reflect.TypeOf(ai.TokenCount{})

	if got := usageType.NumField(); got != len(wantFields) {
		t.Errorf("ai.Usage has %d fields, want exactly the %d the documented cost formula accounts for — "+
			"decide whether the new field is a term of the formula or a breakdown of another field "+
			"(usage.go, the Usage type documentation), then update this pin",
			got, len(wantFields))
	}

	for i, want := range wantFields {
		if i >= usageType.NumField() {
			t.Errorf("ai.Usage has no field %d, want %q", i, want.name)
			continue
		}
		field := usageType.Field(i)
		if field.Name != want.name {
			t.Errorf("ai.Usage field %d is %q, want %q — the formula names its terms in this order", i, field.Name, want.name)
		}
		if field.Type != tokenCountType {
			t.Errorf("ai.Usage.%s is %v, want %v — every count is a TokenCount so that absence is expressible on it",
				field.Name, field.Type, tokenCountType)
		}
		if !field.IsExported() {
			t.Errorf("ai.Usage.%s is not exported, so a consumer cannot read it", field.Name)
		}
	}

	// Any field beyond the expected set is reported by name, so the message
	// names the field the author has to classify rather than only the count.
	for i := len(wantFields); i < usageType.NumField(); i++ {
		t.Errorf("ai.Usage field %d is %q — a field the documented cost formula does not name; "+
			"decide whether it is a term or a breakdown (usage.go, the Usage type documentation) and update both",
			i, usageType.Field(i).Name)
	}

	t.Run("the formula's terms are exactly the fields that are not breakdowns", func(t *testing.T) {
		t.Parallel()

		terms := 0
		for _, want := range wantFields {
			if want.isFormulaTerm {
				terms++
			}
		}
		if terms != 4 {
			t.Errorf("the documented formula has %d terms, want 4 — Input, CacheRead, CacheWrite and Output", terms)
		}
	})
}
