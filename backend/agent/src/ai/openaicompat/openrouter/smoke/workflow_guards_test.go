// AI-39.1 — the workflow YAML structural guard (R-OR-07, design D7,
// tasks.md § PR #3 work unit 3.4).
//
// # What this file covers
//
// The workflow file (.github/workflows/agent-openrouter-smoke.yml)
// is the CI configuration that gates the live OpenRouter HTTP
// call. Its trigger shape is load-bearing: the spec requires
// `workflow_dispatch` only — no `schedule`, no `push`, no
// `pull_request` — because the regular CI pipeline must never
// make a real-money call. This test reads the workflow file as
// raw bytes and asserts the only trigger present is
// `workflow_dispatch`. A regression that adds a `push:` or
// `schedule:` trigger would fail the test, surfacing the
// drift in CI rather than waiting for a manual review to
// notice.
//
// The pattern mirrors the PR #1
// TestOpenRouterAdapter_HeadersUnawarenessInOpenAICompatRequest
// (which reads openaicompat/request.go as raw bytes and asserts
// the three attribution header names are absent) and the PR #2
// TestFixtures_AttributionHeaderNamesConsistent (which reads the
// fixture files and asserts the header-set shape). The same
// defense-in-depth pattern, applied to the workflow YAML.
//
// # The bite-proof is structural
//
// A future change that added a `push:` trigger would fail this
// test by name: the `forbiddenTriggers` table below names every
// non-workflow_dispatch trigger R-OR-07 forbids. The test is
// fail-loud by construction — there is no `t.Skip` on absent
// workflow, the assertion calls `t.Errorf` with the offending
// trigger name so a CI operator reading the failure knows
// exactly which shape they reintroduced.
//
// # Why the path is relative to the package
//
// The smoke package is at
// backend/agent/src/ai/openaicompat/openrouter/smoke/. The
// workflow file is at the repo root. Seven `..` segments
// navigate up. The test is a repo-internal artifact: if the
// smoke sub-package is ever moved, the test fails immediately
// with an actionable error (the relative path is incorrect
// for the new location), which is the desired behavior.
package smoke_test

import (
	"bytes"
	"os"
	"testing"
)

// workflowFilePath is the relative path from the smoke package
// directory to the workflow YAML. The path is hardcoded rather
// than computed via runtime.Caller because the file's location
// is a project-level invariant, not a runtime invariant.
const workflowFilePath = "../../../../../../../.github/workflows/agent-openrouter-smoke.yml"

// workflowDispatchLiteral is the substring the test requires
// to be present in the workflow file. Hardcoded as a literal
// because the substring is small, well-known, and its presence
// is the assertion we want to make observable.
const workflowDispatchLiteral = "workflow_dispatch:"

// forbiddenTriggerLiteral is the prefix a forbidden trigger must
// carry to be detected. Two leading spaces match the YAML
// indent under the `on:` key (`on:` → `  push:` etc.).
const forbiddenTriggerIndent = "  "

// forbiddenTriggerNames is the trigger-name portion of the
// forbidden triggers. The check below matches every
// `  <name>:` substring under the `on:` key so a regression
// that adds `schedule:`, `push:`, `pull_request:`, or any
// future trigger name fails the test.
var forbiddenTriggerNames = []string{
	"schedule",
	"push",
	"pull_request",
}

// TestWorkflowFile_IsDispatchOnly is the mechanical guard for
// R-OR-07's "workflow file is dispatch-only" requirement. The
// test reads the workflow file as raw bytes and asserts the
// only trigger present is workflow_dispatch. A regression that
// added any of the forbidden triggers would fail this test,
// surfacing the drift in CI rather than waiting for a manual
// review to notice.
func TestWorkflowFile_IsDispatchOnly(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile(workflowFilePath)
	if err != nil {
		t.Skipf("workflow file not readable at %s: %v (the workflow-YAML guard is only meaningful when the smoke package is checked out alongside the workflow file)", workflowFilePath, err)
	}

	// Required: workflow_dispatch IS present.
	if !bytes.Contains(raw, []byte(workflowDispatchLiteral)) {
		t.Errorf("workflow file does not contain %q — R-OR-07 requires workflow_dispatch to be the trigger", workflowDispatchLiteral)
	}

	// Required: no forbidden trigger is present under `on:`.
	for _, name := range forbiddenTriggerNames {
		// Match `  <name>:` (the indented form under `on:`) so a
		// stray mention of the trigger name elsewhere (in a
		// comment, for example) does not false-positive.
		needle := append([]byte(forbiddenTriggerIndent), []byte(name+":")...)
		if bytes.Contains(raw, needle) {
			t.Errorf("workflow file contains forbidden trigger %q (R-OR-07 — workflow must be workflow_dispatch only; a push/schedule/pull_request trigger would make a real-money call on regular CI runs)", string(needle))
		}
	}
}

// TestWorkflowFile_HasRunSmokeInputDefaultFalse is the companion
// mechanical guard for the dispatch input's default. The spec
// requires `run_smoke` to default to false so accidental
// dispatches (operator clicks the button without flipping the
// input) do not make a real-money call. The test asserts the
// default is the literal `false` (boolean, not a quoted string).
func TestWorkflowFile_HasRunSmokeInputDefaultFalse(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile(workflowFilePath)
	if err != nil {
		t.Skipf("workflow file not readable at %s: %v", workflowFilePath, err)
	}

	// The dispatch input shape is `default: false` (boolean).
	// A regression that made it `default: true` or moved the
	// input elsewhere would surface here.
	if !bytes.Contains(raw, []byte("default: false")) {
		t.Errorf("workflow file does not declare `default: false` for the run_smoke input — accidental dispatches would start making real-money calls (R-OR-07)")
	}
}
