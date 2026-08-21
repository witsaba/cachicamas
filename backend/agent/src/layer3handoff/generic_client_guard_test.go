// AG-23 — the generic-client boundary's mechanical half (R-L3H-010,
// S-L3H-044..047). Nothing in the kit or the proof may name a concept
// that only one kind of application would need. Capability leaks are
// import-caught (the guard's fifth check, agent-package-scaffold's
// R-AGP-003): this test is the OTHER half — a vocabulary leak, which
// imports nothing and so is import-invisible.
//
// The scan reads .go source bytes whole, comments included, because a
// comment teaching the wrong vocabulary is the thing a later application
// copies. Its own needles are therefore assembled at run time
// (concatenated, never spelled as one contiguous literal anywhere in
// this source unit's own bytes) — a scanner that read its own kind of
// source and carried a literal needle would fail on itself, which is
// exactly the trap S-L3H-046 exists to name. This comment itself is
// written without using any of the denied terms, for the same reason
// the two scanned trees' own doc comments must be.
//
// This one source unit is itself excluded, by exact name, from
// import_boundary_test.go's own zero-I/O scan (agent-package-scaffold's
// amended R-AGP-003, check 5, layer2SiblingTreeSelfExcludedFile) —
// discovered during apply, not planned at design time. Its own standard-
// library reads are verification tooling over already-committed source
// at build time, not part of the kit's or the proof's own RUNTIME
// surface, exactly the category import_boundary_test.go's own reads over
// Layer 2's production source already are for that check.
package layer3handoff_test

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// l3hGuardConcat joins two halves at run time. Every denied term below
// is built through this call, never written whole as a literal in this
// source unit's own bytes.
func l3hGuardConcat(a, b string) string { return a + b }

// l3hGuardDeniedVocabulary is R-L3H-010's own closed needle set: eight
// concepts specific to one kind of application (a coding assistant,
// 0003:2152), never a bare substring that would false-positive on
// ordinary prose about streams, runs and events. Word-boundary matching
// (below) is what keeps a word like "legitimate" or "digit" from
// tripping the fifth entry.
func l3hGuardDeniedVocabulary() []string {
	return []string{
		l3hGuardConcat("fi", "le"),
		l3hGuardConcat("sh", "ell"),
		l3hGuardConcat("termin", "al"),
		l3hGuardConcat("edit", "or"),
		l3hGuardConcat("g", "it"),
		l3hGuardConcat("reposit", "ory"),
		l3hGuardConcat("filesys", "tem"),
		l3hGuardConcat("director", "y"),
	}
}

// l3hGuardWordBoundaryPatterns compiles one case-insensitive,
// letter-boundary-anchored pattern per denied term. A LETTER boundary,
// not regexp's own `\b` (which treats an underscore as a word
// character): tool names in this module join two lowercase dictionary
// words with an underscore, so an underscore MUST count as a boundary
// too, or S-L3H-045's own planted shape would slip through un-caught.
// The boundary requirement is therefore "no letter immediately
// adjacent", satisfied by start/end of input or any non-letter (digit,
// underscore, space, punctuation) — this still excludes an ordinary
// prose word like "legitimate" or "digit", where a letter sits on both
// sides of the embedded needle.
func l3hGuardWordBoundaryPatterns() map[string]*regexp.Regexp {
	out := make(map[string]*regexp.Regexp, 8)
	for _, needle := range l3hGuardDeniedVocabulary() {
		out[needle] = regexp.MustCompile(`(?i)(^|[^a-zA-Z])` + regexp.QuoteMeta(needle) + `([^a-zA-Z]|$)`)
	}
	return out
}

// l3hGuardTreeDirs resolves both new sibling trees' own locations
// relative to THIS source unit's own position, via runtime.Caller(0) —
// the same resolution AD-2's own check 5 and layer2AgentPackageDir use.
func l3hGuardTreeDirs(t *testing.T) map[string]string {
	t.Helper()
	_, thisSource, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed to report this source unit's own path")
	}
	layer3handoffDir := filepath.Dir(thisSource)
	srcRoot := filepath.Dir(layer3handoffDir)
	return map[string]string{
		"apptest":       filepath.Join(srcRoot, "apptest"),
		"layer3handoff": layer3handoffDir,
	}
}

// l3hGuardSourceUnits lists every .go source unit's absolute location
// directly inside dir (no nested trees) — production and test units
// alike, since the discipline binds prose in both (R-L3H-010).
func l3hGuardSourceUnits(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("os.ReadDir(%q) error = %v, want nil", dir, err)
	}
	var out []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") {
			continue
		}
		out = append(out, filepath.Join(dir, name))
	}
	return out
}

// l3hGuardViolation names one hit: the source unit's own base name and
// the denied term found in it.
type l3hGuardViolation struct {
	base   string
	needle string
}

// l3hGuardScan scans every .go source unit directly inside dir against
// every denied term, returning every hit found. Exported to this test's
// own package scope (unexported, package-private) rather than buried
// inside the test function, so an apply-time bite can call it directly
// against a scratch location without duplicating the mechanism.
func l3hGuardScan(t *testing.T, dir string) []l3hGuardViolation {
	t.Helper()
	units := l3hGuardSourceUnits(t, dir)
	if len(units) == 0 {
		// AG-23 (sdd-verify round 1, W-10): the vacuity floor is PER
		// TREE, mirroring agent-package-scaffold's own check 5
		// (S-AGP-044) and this same guard's own tree-resolution floor
		// one source unit over — a renamed, emptied or relocated tree
		// MUST fail by naming ITS OWN location, not pass because the
		// loop's non-empty tree map (checked once, above the loop) says
		// nothing about any ONE tree's own source-unit count.
		t.Fatalf("no .go source unit found directly inside %q; the vocabulary scan would pass vacuously for this tree", dir)
	}
	patterns := l3hGuardWordBoundaryPatterns()
	var violations []l3hGuardViolation
	for _, path := range units {
		raw, rerr := os.ReadFile(path)
		if rerr != nil {
			t.Fatalf("os.ReadFile(%q) error = %v, want nil", path, rerr)
		}
		for needle, pattern := range patterns {
			if pattern.Match(raw) {
				violations = append(violations, l3hGuardViolation{base: filepath.Base(path), needle: needle})
			}
		}
	}
	return violations
}

// S-L3H-044, S-L3H-046. Given the merged change, when the vocabulary
// scan runs over both new trees, then it passes; and because this
// source unit itself lives directly inside the layer3handoff tree, that
// same scan already covers it — so a literal denied term anywhere in
// this very source unit, including inside the needle-construction
// helpers above, would fail this same test. The scan therefore does not
// trip on itself, and that is checked rather than merely claimed.
func TestGenericClientBoundary_VocabularyScan_PassesOverBothTreesIncludingItself(t *testing.T) {
	t.Parallel()

	dirs := l3hGuardTreeDirs(t)
	if len(dirs) == 0 {
		t.Fatal("l3hGuardTreeDirs returned zero trees; the scan would pass vacuously")
	}
	for name, dir := range dirs {
		if violations := l3hGuardScan(t, dir); len(violations) > 0 {
			t.Errorf("tree %q carries denied vocabulary: %+v", name, violations)
		}
	}
}
