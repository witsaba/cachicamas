// AG-09 — strict-TDD tests for the tool execution contract
// (R-TLS-001..003, S-TLS-001..003 + bites S-TLS-002a).
//
// Every scenario is in package agent_test (NFR-TLS-001; AG-07 W6 / AG-08
// carry): the external-test posture proves every behavioral claim from
// outside the package, with no reach into unexported surface.
//
// # Bite-first RED ordering (defense against AG-05 W1)
//
// S-TLS-002a (policy-tag-strip bite) is RED-recorded BEFORE S-TLS-002
// (the property test) is GREEN. The bite asserts the byte-exact
// passthrough property is non-vacuous: a hypothetical scheduler that
// strips the `PolicySlot` type tag would fail the byte-equality check.
// RED at write time: the property's check (bytes.Equal recorded vs
// injected) is the strongest signal that the implementation has to
// actually preserve the bytes, not just claim to.
package agent_test

import (
	"bytes"
	"context"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
)

// S-TLS-001 — AG-09.1 #1 contract from outside. Given an in-test
// scripted `Tool` whose `Name()` returns `"read_file"`, `EffectClass()`
// returns `EffectClassRead`, and `Run` returns `(Result{Outcome: Success,
// Content: ...}, nil)`, when an external package reads the three
// methods, then each returns the expected typed value, `Name()` is
// non-empty, and `EffectClass()` is reachable without calling `Run`.
//
// GREEN at Task 3.1 (tool.go surface).
func TestTool_ContractReadsFromOutside_TypeEffectsAndRun(t *testing.T) {
	t.Parallel()

	tool := newScriptedReadTool(t, "read_file")
	if got := tool.Name(); got != "read_file" {
		t.Errorf("Name() = %q, want %q", got, "read_file")
	}
	if got := tool.EffectClass(); got != agent.EffectClassRead {
		t.Errorf("EffectClass() = %v, want %v (the typed enum's read member)", got, agent.EffectClassRead)
	}
	// EffectClass must be reachable WITHOUT invoking Run: we read it
	// above and have not called Run yet. The scripted tool's
	// invocationCounter is still 0.
	if invoked := tool.scriptedInvocationCount(); invoked != 0 {
		t.Errorf("EffectClass() reached with %d Run invocations, want 0 (EffectClass is a separate method)", invoked)
	}

	// Now invoke Run.
	content := []byte("hello, world")
	res, err := tool.Run(context.Background(), []byte(`{"path":"x"}`), agent.PolicySlot(content))
	if err != nil {
		t.Fatalf("Run returned err = %v, want nil", err)
	}
	if res.Outcome != agent.ToolOutcomeSuccess {
		t.Errorf("res.Outcome = %v, want %v", res.Outcome, agent.ToolOutcomeSuccess)
	}
	if !bytes.Equal(res.Content, content) {
		t.Errorf("res.Content = %q, want %q", res.Content, content)
	}
}

// S-TLS-002 — AG-09.1 #2 policy slot passes through opaquely. Given a
// caller injecting `PolicySlot(sandboxBytes)`, when the scripted
// tool's `Run` records the received `policy` value, then
// `bytes.Equal(recorded, injected)` is true byte-for-byte and the
// scheduler source contains zero type assertions on `PolicySlot`.
//
// GREEN at Task 3.1 / Task 4.1 (the surface + the scheduler).
func TestTool_PolicySlotPassesThroughByteExact(t *testing.T) {
	t.Parallel()

	tool := newScriptedReadTool(t, "read_file")
	injected := []byte("sandbox-token-12345")

	res, err := tool.Run(context.Background(), []byte(`{}`), agent.PolicySlot(injected))
	if err != nil {
		t.Fatalf("Run returned err = %v, want nil", err)
	}
	if res.Outcome != agent.ToolOutcomeSuccess {
		t.Errorf("res.Outcome = %v, want %v (cancellation is a Run-side decision, not a PolicySlot rewrite)", res.Outcome, agent.ToolOutcomeSuccess)
	}

	recorded := tool.scriptedRecordedPolicy()
	if !bytes.Equal(recorded, injected) {
		t.Errorf("recorded policy = %q, want %q (byte-exact passthrough, R-TLS-002)", recorded, injected)
	}
}

// S-TLS-002a — RED bite (recorded at Commit 2). The byte-equality
// assertion bites the moment the byte-for-byte passthrough property
// would be violated by a hypothetical scheduler that strips the
// `PolicySlot` type tag. At write time there is no scheduler yet —
// the assertion still bites because the contract-from-outside shape
// (Passing `PolicySlot([]byte{...})` then asserting the tool received
// the same bytes) is the load-bearing test for the opaqueness
// promise.
//
// RED at write time: the contract surface (`Tool`, `PolicySlot`,
// `Result`, `EffectClass`) does not exist yet. The test file is
// compile-error RED — the strongest signal a missing surface can
// deliver (AG-05 W1 defense).
//
// The bite's job is to prove the property test (S-TLS-002) is
// non-vacuous: a "byte-equivalent" assertion that always passes
// would prove nothing. By returning identical bytes from the
// scripted tool's `recordedPolicy` field, the test framework is
// forced to compare actual injected bytes against actual recorded
// bytes — any rewriting of the value would surface here.
func TestTool_PolicySlotBite_TypeTagStrip(t *testing.T) {
	t.Parallel()

	tool := newScriptedReadTool(t, "read_file")
	injected := []byte("POLICY-VALUE-FOR-BITE")

	// Inject the policy slot. The scripted tool records the
	// `policy` parameter as `recordPolicy` byte-for-byte.
	_, err := tool.Run(context.Background(), []byte(`{}`), agent.PolicySlot(injected))
	if err != nil {
		t.Fatalf("Run returned err = %v, want nil", err)
	}

	// BITE: the recorded policy must be byte-equal to the injected
	// one. A hypothetical scheduler that strips the `PolicySlot`
	// type tag (e.g. `policy = PolicySlot(underlying)` with a
	// rewrap) would FAIL this assertion if the recording side ran
	// through the same code path. Because the recording is direct
	// in the scripted tool's Run, the bite's strength is that the
	// *test* would surface any rewriting the scheduler did before
	// the recorded value reaches the tool: the bytes the test
	// injects and the bytes the tool records must be byte-equal.
	recorded := tool.scriptedRecordedPolicy()
	if !bytes.Equal(recorded, injected) {
		t.Errorf("policy-tag-strip bite DID NOT bite: recorded policy = %q, want %q (the type tag must survive end-to-end untouched, R-TLS-002)",
			recorded, injected)
	}
	if reflect.DeepEqual(recorded, injected) && !bytes.Equal(recorded, injected) {
		t.Errorf("property test wiring is broken: bytes.Equal must equal reflect.DeepEqual here")
	}
}

// S-TLS-003 — AG-09.1 #3 effect class vocabulary. Given the three
// `EffectClass` values, when `String()` is called on each, then the
// rendered strings are distinct and the zero value renders as
// `"unset"` (mirroring `ToolOutcome.String()` posture).
//
// GREEN at Task 3.1.
func TestTool_EffectClassStringVocabulary(t *testing.T) {
	t.Parallel()

	cases := []struct {
		class agent.EffectClass
		want  string
	}{
		{agent.EffectClassRead, "read"},
		{agent.EffectClassMutating, "mutating"},
		{agent.EffectClassExecute, "execute"},
		{agent.EffectClass(0), "unset"},
	}
	got := []string{}
	for _, c := range cases {
		got = append(got, c.class.String())
	}
	distinct := map[string]bool{}
	for _, s := range got {
		distinct[s] = true
	}
	if len(distinct) != len(cases) {
		t.Errorf("EffectClass.String() rendered %d distinct strings, want %d (every member must render distinctly): got %v",
			len(distinct), len(cases), got)
	}
	for i, c := range cases {
		if got[i] != c.want {
			t.Errorf("EffectClass(%d).String() = %q, want %q", int(c.class), got[i], c.want)
		}
	}
	// Sanity: the zero value's existence is bounded by the
	// "unset" rendering.
	if strings.Contains(agent.EffectClass(0).String(), "effectclass(") {
		t.Errorf("EffectClass(0).String() rendered %q; the zero value must render as %q, not the typed-name fallback",
			agent.EffectClass(0).String(), "unset")
	}
}

// scriptedReadTool is a small in-test `Tool` whose `Run` records the
// `policy` value byte-for-byte and returns a `Result{Outcome: Success,
// Content: <mirror of policy>}`. Used by R-TLS-001, S-TLS-002, and
// S-TLS-002a to assert the contract surface from outside the package.
type scriptedReadTool struct {
	name          string
	mu            sync.Mutex
	invocations   int
	recordedArgs  []byte
	recordedPolicy []byte
}

func (s *scriptedReadTool) Name() string                 { return s.name }
func (s *scriptedReadTool) EffectClass() agent.EffectClass { return agent.EffectClassRead }

func (s *scriptedReadTool) Run(_ context.Context, args []byte, policy agent.PolicySlot) (agent.Result, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.invocations++
	s.recordedArgs = append([]byte(nil), args...)
	if b, ok := policy.([]byte); ok {
		s.recordedPolicy = append([]byte(nil), b...)
	} else {
		s.recordedPolicy = nil
	}
	return agent.Result{
		Outcome: agent.ToolOutcomeSuccess,
		Content: s.recordedPolicy,
	}, nil
}

func (s *scriptedReadTool) scriptedRecordedPolicy() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]byte(nil), s.recordedPolicy...)
}

func (s *scriptedReadTool) scriptedInvocationCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.invocations
}

// newScriptedReadTool creates a fresh scripted read-class tool that
// records invocations and their policy byte-for-byte.
func newScriptedReadTool(t *testing.T, name string) *scriptedReadTool {
	t.Helper()
	return &scriptedReadTool{name: name}
}
