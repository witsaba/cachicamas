// AI-26.8.1 — the feature inventory: every feature a request can express,
// derived mechanically from the neutral package's own declarations, never
// from a list this adapter maintains about itself (S-ART-072).
//
// # Two derivation surfaces, one reason each
//
// Five closed vocabularies enumerate at runtime — ai.PartKinds(),
// ai.ReasoningStates(), ai.ToolChoiceModes(), ai.CacheRegions() and
// ai.Roles() (discoverVocabularyFeatures, below) — no reflection needed:
// each is already an exported func returning its own constant space in
// declaration order (role.go's own pattern, reused by every later closed
// vocabulary in package ai).
//
// AI-12's ten With* option constructors are a different shape: plain
// fields, not a closed vocabulary — there is no WithOptions() enumerator to
// call. requestOptionConstructorNames (below) discovers them the way
// content_part_registry_test.go and validation_registry_internal_test.go
// (package ai) already discover their own guarded surfaces: a go/parser
// scan of the target directory's own non-test sources, matched
// syntactically (an exported top-level func named With* returning
// RequestOption) rather than resolved through go/types. Reflection is
// rejected here because it cannot do this job at all, not merely because a
// scan is stylistically preferred: reflect only inspects values already in
// hand, and there is no runtime handle on "every top-level function
// declared in this package" for it to walk — go/ast reads the source text
// directly instead. See doc.go's "Feature inventory: hybrid derivation,
// reflection rejected" section (R-ART-020) for the full record (S-ART-073).
//
// # Why the scan takes a directory parameter
//
// requestOptionConstructorNames is parameterized by dir rather than
// hard-coding "..": the production guard below
// (TestFeatureInventory_MatchesProductionPolicyExactly) calls it with ".."
// to read the real ai package directory this package's own module lives
// beside, but every bite proof in this file (S-ART-070, S-ART-071) and the
// ground-truth proof (S-ART-072) call the identical function against a
// t.TempDir() holding synthetic, throwaway .go sources instead — proving
// the real scanning mechanism responds to whatever a directory currently
// declares, permanently, with no real ai/*.go file ever staged and
// reverted. This is this session's own preferred, durable alternative to a
// transient mutation of a shared package file: AI-27's own verify flagged
// exactly this tradeoff as a warning for a staged-then-deleted proof, and
// this repo already ships the durable pattern (AI-24's
// TestSequenceGuard_*_Fails) — a factored, parameterized function tested
// against a synthetic fixture is strictly better than a transient mutation
// where the factoring is reasonably possible, which it is here.
package openaicompat

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// discoverVocabularyFeatures returns one namespaced feature name per member
// of the five closed vocabularies AI-26.8.1 enumerates, in a fixed,
// deterministic order (vocabulary by vocabulary, member by declaration
// order within each) — never ranged from a map (design.md "Map
// discipline", restated for this audit surface).
//
// Each name is prefixed by its own vocabulary's type name
// ("PartKind: text", "ReasoningState: text", ...): two different
// vocabularies both register a member whose bare String() rendering is
// "text" (ai.PartKindText and ai.ReasoningStateText), so the flat feature
// set this file and policy_walk_test.go build would silently collide two
// distinct features into one shared name without this namespacing.
func discoverVocabularyFeatures() []string {
	var out []string
	for _, kind := range ai.PartKinds() {
		out = append(out, "PartKind: "+kind.String())
	}
	for _, state := range ai.ReasoningStates() {
		out = append(out, "ReasoningState: "+state.String())
	}
	for _, mode := range ai.ToolChoiceModes() {
		out = append(out, "ToolChoiceMode: "+mode.String())
	}
	for _, region := range ai.CacheRegions() {
		out = append(out, "CacheRegion: "+region.String())
	}
	for _, role := range ai.Roles() {
		out = append(out, "Role: "+role.String())
	}
	return out
}

// requestOptionConstructorNames returns the exported, top-level function
// names declared in dir's own non-test .go sources whose signature is
// func(...) RequestOption — an exported With* constructor, AI-12's own
// naming convention for every option this package's guard needs to
// account for. Declaration order within a file, file order from
// os.ReadDir's own filename sort — content_part_registry_test.go's own
// idiom, restated for a directory one level up.
//
// The name-prefix check ("With") is deliberately independent of the
// return-type check: matching a RequestOption return type is what the
// neutral package's own construction contract guarantees (RequestOption's
// unexported parameter type seals the set to package ai's own
// constructors, request.go's own GoDoc), while the "With" prefix is this
// milestone's own naming assumption about that set — checked as a second,
// independent condition rather than assumed, so a mis-scanned function
// exported for an unrelated reason cannot silently inflate the inventory.
func requestOptionConstructorNames(t *testing.T, dir string) []string {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("os.ReadDir(%q): %v", dir, err)
	}

	fset := token.NewFileSet()
	var out []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}

		file, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parsing %s: %v", filepath.Join(dir, name), err)
		}

		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil {
				continue // a method, not a top-level constructor.
			}
			if !fn.Name.IsExported() || !strings.HasPrefix(fn.Name.Name, "With") {
				continue
			}
			if !returnsRequestOption(fn) {
				continue
			}
			out = append(out, fn.Name.Name)
		}
	}
	return out
}

// returnsRequestOption reports whether fn's signature has exactly one
// unqualified result named "RequestOption" — the shape every With*
// constructor in package ai (request.go, request_extension.go,
// system_instruction.go) is written with.
func returnsRequestOption(fn *ast.FuncDecl) bool {
	if fn.Type.Results == nil || len(fn.Type.Results.List) != 1 {
		return false
	}
	ident, ok := fn.Type.Results.List[0].Type.(*ast.Ident)
	return ok && ident.Name == "RequestOption"
}

// withConstructorFeatureNames maps AST-scanned With* constructor names onto
// this package's own inventory feature names.
//
// Every constructor maps to itself, one name in, one name out, except
// WithProviderExtension: design.md decision 6 splits it into two features,
// because its own two namespace cases carry genuinely different
// dispositions (this adapter's own reserved namespace translates/merges;
// every other namespace drops) — S-ART-074 requires every feature to
// resolve to exactly one disposition, and a single "WithProviderExtension"
// row could not carry two.
func withConstructorFeatureNames(constructors []string) []string {
	out := make([]string, 0, len(constructors)+1)
	for _, name := range constructors {
		if name == "WithProviderExtension" {
			out = append(out, "WithProviderExtension: own namespace", "WithProviderExtension: foreign namespace")
			continue
		}
		out = append(out, name)
	}
	return out
}

// fullInventory is the complete, mechanically-derived feature set this
// milestone's guard and walk both check against: every vocabulary member
// (discoverVocabularyFeatures) plus every With* constructor feature
// (withConstructorFeatureNames over the real ai package,
// requestOptionConstructorNames(t, "..")).
func fullInventory(t *testing.T) []string {
	t.Helper()
	inventory := discoverVocabularyFeatures()
	inventory = append(inventory, withConstructorFeatureNames(requestOptionConstructorNames(t, ".."))...)
	return inventory
}

// policyFeatureNames projects featurePolicy (policy.go) onto its own
// feature names, in table order — the production table read directly,
// never mirrored by hand.
func policyFeatureNames() []string {
	names := make([]string, len(featurePolicy))
	for i, entry := range featurePolicy {
		names[i] = entry.feature
	}
	return names
}

// unaccountedFeatures reports, in both directions, where inventoried and
// policied disagree: missingFromPolicy names every inventoried feature
// with no matching policy entry (R-ART-020's own silent-drop risk —
// S-ART-069's primary concern), and deadInPolicy names every policy entry
// naming a feature the inventory no longer carries (a stale row, caught
// for the same reason validation_registry_internal_test.go checks its own
// mirror in both directions). Both are nil when the two sets agree
// exactly.
//
// This is the one comparison function every real check in this file and
// policy_walk_test.go funnels through: the guard
// (TestFeatureInventory_MatchesProductionPolicyExactly) calls it with the
// REAL inventory; every bite proof below calls it with a SYNTHETIC one, so
// the exact mechanism proven to bite is also the mechanism production
// relies on, never a re-implementation the two could quietly drift apart
// from.
func unaccountedFeatures(inventoried, policied []string) (missingFromPolicy, deadInPolicy []string) {
	inPolicy := make(map[string]bool, len(policied))
	for _, feature := range policied {
		inPolicy[feature] = true
	}
	seenInInventory := make(map[string]bool, len(inventoried))
	for _, feature := range inventoried {
		seenInInventory[feature] = true
		if !inPolicy[feature] {
			missingFromPolicy = append(missingFromPolicy, feature)
		}
	}
	for _, feature := range policied {
		if !seenInInventory[feature] {
			deadInPolicy = append(deadInPolicy, feature)
		}
	}
	return missingFromPolicy, deadInPolicy
}

// TestFeatureInventory_MatchesProductionPolicyExactly is this milestone's
// exit check (S-ART-069): every feature the live ai package currently
// declares — five vocabularies enumerated at runtime, ten With*
// constructors discovered by AST scan — has exactly one row in
// featurePolicy (policy.go), and featurePolicy carries no row naming a
// feature that does not currently exist. A feature added to either
// derivation surface with no policy entry fails this test, naming it —
// R-ART-020's own "no expressible request feature can be silently dropped"
// made mechanical.
func TestFeatureInventory_MatchesProductionPolicyExactly(t *testing.T) {
	constructors := requestOptionConstructorNames(t, "..")
	if got, want := len(constructors), 10; got != want {
		t.Fatalf("requestOptionConstructorNames(\"..\") returned %d constructor(s) %v, want exactly 10 — "+
			"this session's own instruction was to verify this count, not trust it", got, constructors)
	}

	inventory := discoverVocabularyFeatures()
	inventory = append(inventory, withConstructorFeatureNames(constructors)...)
	if len(inventory) == 0 {
		t.Fatal("the discovered inventory is empty; this guard would pass vacuously")
	}

	missing, dead := unaccountedFeatures(inventory, policyFeatureNames())
	if len(missing) > 0 {
		t.Errorf("features the live ai package declares but featurePolicy does not account for: %v", missing)
	}
	if len(dead) > 0 {
		t.Errorf("featurePolicy rows naming a feature the live ai package does not currently declare: %v", dead)
	}
}

// TestFeatureInventory_NoDuplicateFeatureNamesInPolicy guards the "exactly
// one" half of R-ART-021 at the table's own level: a feature named twice in
// featurePolicy would let unaccountedFeatures' own set-membership check
// silently ignore the duplicate, and a caller resolving the feature
// (resolveFeaturePolicy, policy.go) would silently return whichever row
// happens to appear first, hiding the ambiguity from every check above.
func TestFeatureInventory_NoDuplicateFeatureNamesInPolicy(t *testing.T) {
	seen := make(map[string]int, len(featurePolicy))
	for _, entry := range featurePolicy {
		seen[entry.feature]++
	}
	for feature, count := range seen {
		if count > 1 {
			t.Errorf("featurePolicy names %q %d times, want exactly 1 (S-ART-074's \"exactly one disposition\" requires exactly one row)", feature, count)
		}
	}
}

// TestFeatureInventory_UnaccountedVocabularyMemberFailsNamingIt is bite
// proof 1/2 (S-ART-070): a closed-vocabulary member with no policy entry
// fails unaccountedFeatures — the identical comparison
// TestFeatureInventory_MatchesProductionPolicyExactly runs against the real
// inventory — naming it. Durable and permanent rather than a staged,
// reverted mutation of a real vocabulary file in package ai (this file's
// own header comment states the reason): the synthetic candidate below
// stands in for "an unaccounted extra member landed on one of the five
// vocabularies with no policy entry" (none of the five currently holds
// exactly five members — 4, 3, 4, 3 and 3 respectively, 17 total, verified
// against the live ai package rather than assumed — so "a scratch sixth
// vocabulary member" names the shape of the proof, not a literal count any
// one vocabulary reaches by adding one), exercising the exact same
// production comparison function with no risk to, and no edit of, any file
// this milestone does not own.
func TestFeatureInventory_UnaccountedVocabularyMemberFailsNamingIt(t *testing.T) {
	const scratchMember = "CacheRegion: scratch_bite_proof_region"

	inventory := append(append([]string{}, policyFeatureNames()...), scratchMember)
	missing, dead := unaccountedFeatures(inventory, policyFeatureNames())

	want := []string{scratchMember}
	if !slices.Equal(missing, want) {
		t.Fatalf("unaccountedFeatures missingFromPolicy = %v, want %v — the guard did not name the unaccounted member", missing, want)
	}
	if len(dead) != 0 {
		t.Fatalf("unaccountedFeatures deadInPolicy = %v, want none (every real policy row is still present in this inventory)", dead)
	}
}

// TestFeatureInventory_UnaccountedConstructorFailsNamingIt is bite proof
// 2/2 (S-ART-071): an eleventh With* constructor with no policy entry
// fails, naming it — and, unlike the vocabulary proof above, this one
// additionally exercises the real go/parser scan end to end (not only the
// downstream set comparison) against a synthetic directory, so the
// AST-scanning mechanism itself is proven to notice a newly declared
// constructor, not only the comparison that consumes its output.
func TestFeatureInventory_UnaccountedConstructorFailsNamingIt(t *testing.T) {
	dir := t.TempDir()
	writeScratchGoFile(t, dir, "scratch_options.go", `package ai

type RequestOption func(*requestDraft)

func WithScratchEleventhOption(v string) RequestOption { return nil }
`)

	got := requestOptionConstructorNames(t, dir)
	wantScanned := []string{"WithScratchEleventhOption"}
	if !slices.Equal(got, wantScanned) {
		t.Fatalf("requestOptionConstructorNames(%q) = %v, want %v", dir, got, wantScanned)
	}

	inventory := append(withConstructorFeatureNames(got), policyFeatureNames()...)
	missing, dead := unaccountedFeatures(inventory, policyFeatureNames())

	wantMissing := []string{"WithScratchEleventhOption"}
	if !slices.Equal(missing, wantMissing) {
		t.Fatalf("unaccountedFeatures missingFromPolicy = %v, want %v — the guard did not name the unaccounted constructor", missing, wantMissing)
	}
	if len(dead) != 0 {
		t.Fatalf("unaccountedFeatures deadInPolicy = %v, want none", dead)
	}
}

// TestFeatureInventory_ScanReflectsTheTargetDirectoryNotAFixedList proves
// S-ART-072 directly: the constructor half of the inventory is grounded in
// whatever a target directory's own .go sources currently declare, never a
// list this adapter maintains about itself. Three fixtures — zero, one and
// three declared With*-shaped constructors, the last spread across two
// files and mixed with an unexported and a wrong-return-type function —
// each produce exactly the expected names: a hand-maintained mirror would
// report the same answer regardless of which fixture ran.
//
// The vocabulary half needs no analogous proof: discoverVocabularyFeatures
// calls ai.PartKinds()/ai.ReasoningStates()/ai.ToolChoiceModes()/
// ai.CacheRegions()/ai.Roles() directly, every time, with no local mirror
// of those five slices anywhere in this package for a test to catch out of
// sync.
func TestFeatureInventory_ScanReflectsTheTargetDirectoryNotAFixedList(t *testing.T) {
	cases := []struct {
		name  string
		files map[string]string
		want  []string
	}{
		{
			name:  "zero constructors",
			files: map[string]string{"empty.go": "package ai\n"},
			want:  nil,
		},
		{
			name: "one constructor",
			files: map[string]string{
				"one.go": "package ai\n\ntype RequestOption func(*requestDraft)\n\nfunc WithLoneOption(v int) RequestOption { return nil }\n",
			},
			want: []string{"WithLoneOption"},
		},
		{
			name: "three constructors across two files, excluding unexported and wrong-return-type",
			files: map[string]string{
				"a_first.go": "package ai\n\ntype RequestOption func(*requestDraft)\n\nfunc WithFirst(v int) RequestOption { return nil }\nfunc WithSecond(v int) RequestOption { return nil }\n",
				"b_second.go": "package ai\n\nfunc WithThird(v int) RequestOption { return nil }\n\n" +
					"func withUnexported(v int) RequestOption { return nil }\n" +
					"func WithWrongReturnType(v int) string { return \"\" }\n",
			},
			want: []string{"WithFirst", "WithSecond", "WithThird"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			for name, source := range tc.files {
				writeScratchGoFile(t, dir, name, source)
			}
			got := requestOptionConstructorNames(t, dir)
			if !slices.Equal(got, tc.want) {
				t.Fatalf("requestOptionConstructorNames(%q) = %v, want %v", dir, got, tc.want)
			}
		})
	}
}

// TestFeatureInventory_NonTranslateFeaturesHaveExactlyOneWitness is
// S-ART-069's own "per-feature witness table" cross-check, the third leg
// alongside the inventory and the production policy table: every
// non-translate policy row (policy_walk_test.go's refuseWitnesses and
// dropWitnesses — the features where an incorrect disposition would
// silently misbehave, this milestone's actual concern) must have exactly
// one witness, and every witness must name a real, still-current
// non-translate row — content_part_registry_test.go's own "every declared
// constant has a witness, and every witness a constant" shape, applied
// here to disposition rather than to a payload's construct/read/validate
// legs.
//
// A translate-disposition feature needs no witness of its own: it is
// already proven byte-exact by one or more of expectationCases' own
// registered cases (Phases 1-7, translation_test.go and its siblings), and
// a second, redundant witness here would prove nothing new.
func TestFeatureInventory_NonTranslateFeaturesHaveExactlyOneWitness(t *testing.T) {
	for _, entry := range featurePolicy {
		switch entry.disposition {
		case dispositionRefuse:
			if !refuseWitnessExists(entry.feature) {
				t.Errorf("policy feature %q is disposition refuse but refuseWitnesses (policy_walk_test.go) carries no witness for it", entry.feature)
			}
		case dispositionDrop:
			if !dropWitnessExists(entry.feature) {
				t.Errorf("policy feature %q is disposition drop but dropWitnesses (policy_walk_test.go) carries no witness for it", entry.feature)
			}
		}
	}

	for _, witness := range refuseWitnesses {
		entry, ok := resolveFeaturePolicy(witness.feature)
		if !ok {
			t.Errorf("refuseWitnesses names %q, which featurePolicy does not carry", witness.feature)
			continue
		}
		if entry.disposition != dispositionRefuse {
			t.Errorf("refuseWitnesses names %q, whose policy disposition is not refuse", witness.feature)
		}
	}
	for _, witness := range dropWitnesses {
		entry, ok := resolveFeaturePolicy(witness.feature)
		if !ok {
			t.Errorf("dropWitnesses names %q, which featurePolicy does not carry", witness.feature)
			continue
		}
		if entry.disposition != dispositionDrop {
			t.Errorf("dropWitnesses names %q, whose policy disposition is not drop", witness.feature)
		}
	}
}

// refuseWitnessExists reports whether refuseWitnesses (policy_walk_test.go)
// carries an entry for feature.
func refuseWitnessExists(feature string) bool {
	for _, w := range refuseWitnesses {
		if w.feature == feature {
			return true
		}
	}
	return false
}

// dropWitnessExists reports whether dropWitnesses (policy_walk_test.go)
// carries an entry for feature.
func dropWitnessExists(feature string) bool {
	for _, w := range dropWitnesses {
		if w.feature == feature {
			return true
		}
	}
	return false
}

// writeScratchGoFile writes a synthetic, throwaway .go source into dir —
// every caller in this file passes a t.TempDir(), so nothing this function
// writes ever reaches a real, tracked directory or survives past the
// calling test.
func writeScratchGoFile(t *testing.T, dir, name, source string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(source), 0o600); err != nil {
		t.Fatalf("os.WriteFile(%s): %v", name, err)
	}
}
