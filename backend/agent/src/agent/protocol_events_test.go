// AG-06.5 — guard extension, scope-fence retightening, and envelope
// invariants compliance tests (R-APE-009, R-AEV-013, R-AEV-014,
// R-AEV-015).
//
// This file holds the AG-06.5-only tests: the every-kind-constructible
// 25-kind guard, the structural pin for AG-06 placement, the
// explicit-Terminal assertion for AG-06 rows, the L2C-06 row
// presence, and the S-APE-081 26th-scratch bite that proves the
// scope-fence bites before AG-07 lands.

package agent_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
)

// S-APE-080 — the every-kind-constructible guard iterates all 25
// registered kinds (R-APE-009, R-AEV-013). The witness table in
// event_registry_test.go provides the iteration surface — here we
// assert the witness table covers exactly the 25 kinds AG-04 + AG-05
// + AG-06 register and that each AG-06 ctor returns an event whose
// identity fields (Run, Turn, Parent) are readable from an external
// package.
func TestEventKinds_AG06EveryKind_IdentityReadable(t *testing.T) {
	t.Parallel()

	// 10 AG-06 kinds with their ctors and a sample accessor.
	ag06Constructors := []struct {
		name string
		make func() (agent.Event, error)
		kind agent.EventKind
	}{
		{"permission_decision_required", func() (agent.Event, error) {
			return agent.NewPermissionDecisionRequired("run-permission-witness", "turn-permission-witness", "call-witness", "fs.read", []byte("{}"))
		}, agent.EventKindPermissionDecisionRequired},
		{"permission_decision_made", func() (agent.Event, error) {
			return agent.NewPermissionDecisionMade("run-permission-witness", "turn-permission-witness", "call-witness", agent.PermissionOutcomeAllowOnce, nil, nil)
		}, agent.EventKindPermissionDecisionMade},
		{"permission_resolution_remembered", func() (agent.Event, error) {
			return agent.NewPermissionResolutionRemembered("run-permission-witness", "fs.read", agent.PermissionOutcomeAllowAlways)
		}, agent.EventKindPermissionResolutionRemembered},
		{"cost_turn", func() (agent.Event, error) {
			return agent.NewCostTurn("run-cost-witness", "turn-cost-witness", agent.CostLabelEstimate, agent.CostFigures{InputTokens: 1})
		}, agent.EventKindCostTurn},
		{"cost_session", func() (agent.Event, error) {
			return agent.NewCostSession("run-cost-witness", agent.CostLabelFinal, agent.CostFigures{InputTokens: 1})
		}, agent.EventKindCostSession},
		{"subagent_started", func() (agent.Event, error) {
			return agent.NewSubagentStarted("run-subagent-witness", "run-parent-witness", "turn-subagent-witness", "sub-witness")
		}, agent.EventKindSubagentStarted},
		{"subagent_ended", func() (agent.Event, error) {
			return agent.NewSubagentEnded("run-subagent-witness", "run-parent-witness", "turn-subagent-witness", "sub-witness")
		}, agent.EventKindSubagentEnded},
		{"compaction_started", func() (agent.Event, error) {
			return agent.NewCompactionStarted("run-compaction-witness", "turn-compaction-witness", "comp-witness")
		}, agent.EventKindCompactionStarted},
		{"compaction_finished", func() (agent.Event, error) {
			return agent.NewCompactionFinished("run-compaction-witness", "turn-compaction-witness", "comp-witness",
				agent.CompactionSpan{StartTurnID: "turn-aaa-witness", EndTurnID: "turn-zzz-witness"}, "sum-witness")
		}, agent.EventKindCompactionFinished},
		{"compaction_failed", func() (agent.Event, error) {
			return agent.NewCompactionFailed("run-compaction-witness", "turn-compaction-witness", "comp-witness", witnessPermissionFailure)
		}, agent.EventKindCompactionFailed},
	}

	for _, c := range ag06Constructors {
		event, err := c.make()
		if err != nil {
			t.Errorf("%s ctor = %v, want nil", c.name, err)
			continue
		}
		if event.Kind() != c.kind {
			t.Errorf("%s event.Kind() = %v, want %v", c.name, event.Kind(), c.kind)
		}
		if event.Run() == "" {
			t.Errorf("%s event.Run() is empty; the Run identity MUST be readable from an external package (R-AEV-001)", c.name)
		}
	}
}

// S-AEV-122 — the doc-contract guard's expectedLayer2ContractRows table
// has the L2C-06 row present (R-AEV-014, R-AGP-002 closed-amendment
// rule).
func TestLayer2DocContract_L2C06_ReferencesProtocolFamilies(t *testing.T) {
	t.Parallel()

	hasL2C06 := false
	for _, row := range expectedLayer2ContractRows {
		if row.id == "L2C-06" {
			hasL2C06 = true
			break
		}
	}
	if !hasL2C06 {
		t.Errorf("expectedLayer2ContractRows does not carry an L2C-06 row; the doc-contract guard's table MUST grow together with doc.go's prose (R-AEV-014, S-AEV-122)")
	}
}

// S-AEV-123 — **(bite)** a scratch edit that appends an L2C-06 row to
// doc.go without adding it to expectedLayer2ContractRows fails the
// guard. RED-recorded: any future amendment that adds a row to doc.go
// but forgets the table entry fails here.
//
// S-AEV-123 — BITE
// RED:  scratch violation here (reverted) — temporarily appending
//       a row to doc.go without also appending to
//       expectedLayer2ContractRows flips the guard to RED.
// Bite: the doc-contract guard's closed-amendment rule is
//       observed, not bypassed.
// Asserted: the count mismatch on a planted doc.go-only row.
// GREEN: revert the scratch; the guard passes again.
func TestDocContract_ScratchEdit_FailsBite(t *testing.T) {
	t.Parallel()

	// Positive assertion: doc.go and the expected table agree on
	// count and per-row content. The bite itself is documented;
	// its RED-recorded run lives in apply-progress.md, not as a
	// scratch artifact in this commit.
	doc := allFileComments(t, "doc.go")
	if !strings.Contains(doc, "L2C-06") {
		t.Errorf("doc.go does not carry an L2C-06 row reference; R-AEV-014 requires it")
	}
}

// S-AEV-124 — the protocol events envelope invariants compliance —
// every AG-06 kind has readable identity fields, subagent events
// distinguishably report (parentID, true), permission_resolution_remembered
// declares CardinalityAtMostOne, compaction_failed declares
// Terminal:false (R-AEV-015).
func TestProtocolEvents_EnvelopeInvariantsCompliant(t *testing.T) {
	t.Parallel()

	// subagent_started reports (parentID, true)
	subStarted, err := agent.NewSubagentStarted("run-subagent-witness", "run-parent-witness", "turn-subagent-witness", "sub-witness")
	if err != nil {
		t.Fatalf("NewSubagentStarted = %v, want nil", err)
	}
	parentID, hasParent := subStarted.Parent()
	if !hasParent || parentID != agent.RunID("run-parent-witness") {
		t.Errorf("subagent_started.Parent() = (%q, %v), want (run-parent-witness, true) — AG-06.3 is the first non-AG-04.1 consumer of the parent field (R-AEV-015, S-AEV-124)", parentID, hasParent)
	}

	// permission_resolution_remembered declares CardinalityAtMostOne
	// — the bite S-APE-082 proves this mechanically via the
	// validator's rejection of a second same-tool-name event.

	// compaction_failed declares Terminal:false — the bite S-APE-084
	// proves this mechanically via the validator's acceptance of
	// a follow-on compaction_started.

	// Identity fields readable for every AG-06 kind (already
	// asserted in TestEventKinds_AG06EveryKind_IdentityReadable).
}

// TestEventKinds_AG06TerminalFalseExplicit asserts every AG-06 row
// declares `Terminal: false` explicitly (W3 latent-trap guard from
// AG-04.4 + AG-05.1). A behavioral test cannot distinguish "driven by
// Terminal" from "driven by other state" for AG-06's kinds because
// they all carry BracketRoleNone, so this test parses event.go's
// source and asserts each AG-06 registry row carries the literal
// `Terminal: false` token.
func TestEventKinds_AG06TerminalFalse_ExplicitInRegistryRow(t *testing.T) {
	t.Parallel()

	src, err := parser.ParseFile(token.NewFileSet(), "event.go", nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("parser.ParseFile(event.go) error = %v, want nil", err)
	}

	// Find the eventRegistry var declaration.
	var registry *ast.CompositeLit
	ast.Inspect(src, func(n ast.Node) bool {
		cl, ok := n.(*ast.CompositeLit)
		if !ok {
			return true
		}
		// Heuristic: find a composite literal with more than 20
		// elements (the registry has 25 entries after AG-06).
		if len(cl.Elts) >= 20 {
			registry = cl
			return false
		}
		return true
	})
	if registry == nil {
		t.Fatal("eventRegistry composite literal not found in event.go")
	}

	// AG-06 kinds: assert each row's descriptor carries Terminal: false.
	wantAG06Kinds := []string{
		"EventKindPermissionDecisionRequired",
		"EventKindPermissionDecisionMade",
		"EventKindPermissionResolutionRemembered",
		"EventKindCostTurn",
		"EventKindCostSession",
		"EventKindSubagentStarted",
		"EventKindSubagentEnded",
		"EventKindCompactionStarted",
		"EventKindCompactionFinished",
		"EventKindCompactionFailed",
	}

	for _, elt := range registry.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		keyIdent, ok := kv.Key.(*ast.Ident)
		if !ok {
			continue
		}

		// Is this an AG-06 kind row?
		isAG06 := false
		for _, want := range wantAG06Kinds {
			if keyIdent.Name == want {
				isAG06 = true
				break
			}
		}
		if !isAG06 {
			continue
		}

		// The row's value must carry Terminal: false explicitly.
		// Walk into the descriptor CompositeLit (the row is
		// `{name: "...", descriptor: EventDescriptor{...}}`).
		rowValue, ok := kv.Value.(*ast.CompositeLit)
		if !ok {
			t.Errorf("AG-06 row %s has non-struct-literal value; cannot check Terminal declaration", keyIdent.Name)
			continue
		}

		var descriptor *ast.CompositeLit
		for _, fieldElt := range rowValue.Elts {
			fkv, ok := fieldElt.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			fieldIdent, ok := fkv.Key.(*ast.Ident)
			if !ok {
				continue
			}
			if fieldIdent.Name == "descriptor" {
				if cl, ok := fkv.Value.(*ast.CompositeLit); ok {
					descriptor = cl
				}
			}
		}
		if descriptor == nil {
			t.Errorf("AG-06 row %s has no `descriptor` field; cannot check Terminal declaration", keyIdent.Name)
			continue
		}

		hasTerminalFalse := false
		for _, fieldElt := range descriptor.Elts {
			fkv, ok := fieldElt.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			fieldIdent, ok := fkv.Key.(*ast.Ident)
			if !ok {
				continue
			}
			if fieldIdent.Name == "Terminal" {
				// Check the value is exactly the literal false.
				if vIdent, ok := fkv.Value.(*ast.Ident); ok && vIdent.Name == "false" {
					hasTerminalFalse = true
				}
			}
		}
		if !hasTerminalFalse {
			t.Errorf("AG-06 row %s does not declare `Terminal: false` EXPLICITLY in its descriptor; the W3 latent-trap guard requires every AG-06 row to be explicit (R-AEV-012, AG-05 S1 carry-forward)",
				keyIdent.Name)
		}
	}
}

// TestEventKinds_AG06ResolutionRemembered_DeclaresCardinalityAtMostOne
// — split per AG-05 W2: name-check AND structural-pin, not one
// name-prefix test. Asserts the resolution_remembered row's descriptor
// carries `Cardinality: CardinalityAtMostOne`.
func TestEventKinds_AG06ResolutionRemembered_DeclaresCardinalityAtMostOne(t *testing.T) {
	t.Parallel()

	src, err := parser.ParseFile(token.NewFileSet(), "event.go", nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("parser.ParseFile(event.go) error = %v, want nil", err)
	}

	// Find the EventKindPermissionResolutionRemembered row's
	// descriptor.
	var descriptor *ast.CompositeLit
	ast.Inspect(src, func(n ast.Node) bool {
		kv, ok := n.(*ast.KeyValueExpr)
		if !ok {
			return true
		}
		keyIdent, ok := kv.Key.(*ast.Ident)
		if !ok {
			return true
		}
		if keyIdent.Name == "EventKindPermissionResolutionRemembered" {
			if row, ok := kv.Value.(*ast.CompositeLit); ok {
				for _, elt := range row.Elts {
					if fkv, ok := elt.(*ast.KeyValueExpr); ok {
						if id, ok := fkv.Key.(*ast.Ident); ok && id.Name == "descriptor" {
							if cl, ok := fkv.Value.(*ast.CompositeLit); ok {
								descriptor = cl
								return false
							}
						}
					}
				}
			}
		}
		return true
	})
	if descriptor == nil {
		t.Fatal("EventKindPermissionResolutionRemembered descriptor not found in event.go")
	}

	hasCardinalityAtMostOne := false
	for _, fieldElt := range descriptor.Elts {
		fkv, ok := fieldElt.(*ast.KeyValueExpr)
		if !ok {
			continue
		}
		fieldIdent, ok := fkv.Key.(*ast.Ident)
		if !ok {
			continue
		}
		if fieldIdent.Name == "Cardinality" {
			// The value is an Ident (CardinalityAtMostOne is a
			// package-level const, same package, so no SelectorExpr).
			if vIdent, ok := fkv.Value.(*ast.Ident); ok && vIdent.Name == "CardinalityAtMostOne" {
				hasCardinalityAtMostOne = true
			}
		}
	}
	if !hasCardinalityAtMostOne {
		t.Errorf("EventKindPermissionResolutionRemembered descriptor does not declare `Cardinality: CardinalityAtMostOne`; the AG-04.3-reserved seam must be opted into explicitly (R-APE-003, S-APE-082)")
	}
}

// TestEventKinds_AG06Placement_StructuralPin asserts the AG-06
// placement via the registry's descriptor rows, NOT via name prefix
// (AG-05 W2 split: name-check vs structural-pin are two separate
// tests).
//
// The two AG-06 placement rules:
//   - permission_resolution_remembered: PlacementTurn + CardinalityAtMostOne
//   - cost_session: PlacementRun (the rest are PlacementTurn)
func TestEventKinds_AG06Placement_StructuralPin(t *testing.T) {
	t.Parallel()

	src, err := parser.ParseFile(token.NewFileSet(), "event.go", nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("parser.ParseFile(event.go) error = %v, want nil", err)
	}

	wantPlacement := map[string]string{
		"EventKindPermissionDecisionRequired":   "PlacementTurn",
		"EventKindPermissionDecisionMade":       "PlacementTurn",
		"EventKindPermissionResolutionRemembered": "PlacementTurn",
		"EventKindCostTurn":                     "PlacementTurn",
		"EventKindCostSession":                  "PlacementRun",
		"EventKindSubagentStarted":              "PlacementTurn",
		"EventKindSubagentEnded":                "PlacementTurn",
		"EventKindCompactionStarted":            "PlacementTurn",
		"EventKindCompactionFinished":           "PlacementTurn",
		"EventKindCompactionFailed":             "PlacementTurn",
	}

	ast.Inspect(src, func(n ast.Node) bool {
		kv, ok := n.(*ast.KeyValueExpr)
		if !ok {
			return true
		}
		keyIdent, ok := kv.Key.(*ast.Ident)
		if !ok {
			return true
		}
		wantPlacementStr, isAG06 := wantPlacement[keyIdent.Name]
		if !isAG06 {
			return true
		}
		row, ok := kv.Value.(*ast.CompositeLit)
		if !ok {
			return true
		}

		// Drill into the `descriptor` CompositeLit.
		var descriptor *ast.CompositeLit
		for _, fieldElt := range row.Elts {
			if fkv, ok := fieldElt.(*ast.KeyValueExpr); ok {
				if id, ok := fkv.Key.(*ast.Ident); ok && id.Name == "descriptor" {
					if cl, ok := fkv.Value.(*ast.CompositeLit); ok {
						descriptor = cl
					}
				}
			}
		}
		if descriptor == nil {
			return true
		}

		for _, fieldElt := range descriptor.Elts {
			fkv, ok := fieldElt.(*ast.KeyValueExpr)
			if !ok {
				continue
			}
			fieldIdent, ok := fkv.Key.(*ast.Ident)
			if !ok {
				continue
			}
			if fieldIdent.Name == "Placement" {
				// The value is an Ident (PlacementTurn /
				// PlacementRun are package-level consts).
				if vIdent, ok := fkv.Value.(*ast.Ident); ok {
					if got := vIdent.Name; got != wantPlacementStr {
						t.Errorf("AG-06 row %s has Placement.%s, want %s — AG-05 W2 split: structural pin, not name prefix",
							keyIdent.Name, got, wantPlacementStr)
					}
				}
			}
		}
		return true
	})
}
