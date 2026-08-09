// Package conformancetest holds the CapRetry conformance case body,
// extracted from openaicompat's own test package (design D2, AI-38,
// tasks.md Phase 5 WU6) so it can be imported — a blank import suffices
// — by any conformance test binary that wants to drive it, including the
// OpenRouter conformance bridge's UNSCOPED run.
//
// # Why extraction was necessary (R-ACR-006)
//
// The case body's init() registration (agenttest.RegisterConformanceCase)
// only ever fires inside whichever test binary actually imports the file
// that carries it. When the case body lived in openaicompat's own
// _test.go file (package openaicompat_test), only the openaicompat test
// binary ever saw it registered. An unscoped agenttest.RunConformance run
// in a DIFFERENT binary — the OpenRouter conformance bridge's, in
// particular — would therefore never execute the retry case at all, even
// with its Factory.Retry declared true: the generated CapabilityRecord's
// CAP-O-04 entry would read structurally NotExercised rather than
// Satisfied, and CompareCapabilityRecords would report a mismatch against
// a committed "Satisfied" expectation.
//
// This is a REGULAR, non-test package (no _test.go suffix) precisely so
// its init() runs as a side effect of any importer — blank or named — the
// same way a _test.go file's init() never does outside its own binary.
// Using *testing.T / testing.TB from regular Go source is an established
// pattern elsewhere in this module: agenttest.Factory.New itself takes
// testing.TB, and conformance_redaction.go's capturingTB is a
// testing.TB-implementing type living outside any _test.go file.
//
// The case body needs the *openaicompat.Client directly — the suite's
// Factory pattern only builds an ai.ModelProvider, but this case needs
// HTTP-level scripting to assert wire request count and byte-equality
// across retry attempts — so it imports openaicompat's public API only,
// exactly as it did from its previous home.
package conformancetest

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
	"github.com/cachicamas/backend/agent/src/ai/openaicompat"
)

// retryDefaultMaxAttempts mirrors internal/retry.DefaultMaxAttempts so
// this case body can assert against the same N+1 wire-request count
// without importing the internal package. The constant's single source
// of truth is internal/retry/doc.go (R-AIS-044); any drift between the
// two is caught by review against the delta spec at the archive phase.
const retryDefaultMaxAttempts = 3

// RetryOffered is the single declared source of truth every conformance
// factory that wraps *openaicompat.Client MUST read for its
// Factory.Retry field (R-ACR-006, AI-38 WU10 CRITICAL-1 remediation,
// design D2 locked decision 2): the client auto-retries per AI-35, so
// every conformance factory built around the same underlying client
// offers the identical retry behaviour and MUST declare it identically.
//
// Before this constant existed, openaicompat/bridge_test.go and
// openrouter/conformance/bridge_test.go each hard-coded their own local
// `true` literal — two independent declarations living in separate test
// binaries with no cross-binary comparison. verify-report.md's Defeat #7
// proved the gap: flipping one factory's local literal to false left
// both packages' tests exit 0, naming neither factory.
//
// Consuming this single constant instead of a local literal makes that
// drift structurally impossible: there is exactly one place in the
// module capable of declaring the value, and every conformance factory
// copies it rather than inventing its own. Each consuming package also
// carries a TestConformanceBridgeFactory_RetryDeclaration_MatchesSharedSource
// self-test asserting its own constructed Factory.Retry still
// dereferences to this exact value, so a local override that bypassed
// this constant entirely — reintroducing the disagreement — is caught
// by name rather than silently accepted (S-ACR-017, S-ACR-018).
const RetryOffered = true

func init() {
	agenttest.RegisterConformanceCase(
		"retry/auto_retry_up_to_documented_bound",
		agenttest.CapRetry,
		RetryAutoRetryUpToBoundCase,
	)
}

// RetryAutoRetryUpToBoundCase proves CAP-O-04 (R-CNF-019, S-CNF-069..075):
// a retryable pre-stream failure is auto-retried up to the documented
// bound with byte-identical request replay, a non-retryable failure and a
// partial-output boundary are never retried, and the declared-absent
// half is reported rather than silent (R-CNF-004). Exported so both
// openaicompat's own thin driver test and any importing conformance
// binary's init() registration (this file's own init(), above) can reach
// it.
func RetryAutoRetryUpToBoundCase(t *testing.T, f agenttest.Factory) {
	t.Helper()
	retryOffered := f.Retry != nil && *f.Retry

	t.Run("retryable_pre_stream_retried_up_to_bound", func(t *testing.T) {
		if !retryOffered {
			t.Skipf("factory declares %v not offered; case skipped, recorded absent (R-CNF-004)", agenttest.CapRetry)
		}
		agenttest.RequireNoGoroutineLeak(t, func() {
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				requests.Add(1)
				if requests.Load() < int32(retryDefaultMaxAttempts)+1 {
					w.Header().Set("Retry-After", "0")
					w.WriteHeader(http.StatusTooManyRequests)
					_, _ = io.WriteString(w, `{"error":{"type":"rate_limit_error"}}`)
					return
				}
				confRetryStreamOK(t, w)
			}))
			defer server.Close()

			client, err := openaicompat.New(openaicompat.Config{
				Endpoint:   server.URL,
				Credential: openaicompat.NewCredential("conformance-retry-token"),
			})
			if err != nil {
				t.Fatalf("New error = %v, want nil", err)
			}
			ch, streamErr := client.Stream(t.Context(), confRetryRequest(t))
			if streamErr != nil {
				t.Fatalf("Stream error = %v, want nil", streamErr)
			}
			confRetryDrainAll(t, ch)
			if got, want := requests.Load(), int32(retryDefaultMaxAttempts+1); got != want {
				t.Fatalf("wire requests = %d, want %d", got, want)
			}
		})
	})

	t.Run("terminal_category_never_retried", func(t *testing.T) {
		if !retryOffered {
			t.Skipf("factory declares %v not offered; case skipped, recorded absent (R-CNF-004)", agenttest.CapRetry)
		}
		var requests atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			requests.Add(1)
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = io.WriteString(w, `{"error":{"type":"invalid_api_key"}}`)
		}))
		defer server.Close()

		client, err := openaicompat.New(openaicompat.Config{
			Endpoint:   server.URL,
			Credential: openaicompat.NewCredential("conformance-retry-token"),
		})
		if err != nil {
			t.Fatalf("New error = %v, want nil", err)
		}
		_, streamErr := client.Stream(t.Context(), confRetryRequest(t))
		if streamErr == nil {
			t.Fatal("Stream error = nil, want authentication failure")
		}
		if got := requests.Load(); got != 1 {
			t.Fatalf("wire requests = %d, want 1", got)
		}
		var failure *ai.Failure
		if !errors.As(streamErr, &failure) {
			t.Fatalf("errors.As(%T) did not find *ai.Failure", streamErr)
		}
		if failure.Category() != ai.FailureCategoryAuthentication || failure.Retryable() {
			t.Fatalf("failure = category=%v retryable=%v, want authentication/non-retryable", failure.Category(), failure.Retryable())
		}
	})

	t.Run("partial_output_boundary_no_retry", func(t *testing.T) {
		if !retryOffered {
			t.Skipf("factory declares %v not offered; case skipped, recorded absent (R-CNF-004)", agenttest.CapRetry)
		}
		var requests atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			requests.Add(1)
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(w, "data: {\"id\":\"chatcmpl-conf-retry\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\"},\"finish_reason\":null}]}\n\n")
			_, _ = io.WriteString(w, "data: {\"id\":\"chatcmpl-conf-retry\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"partial\"},\"finish_reason\":null}]}\n\n")
			_, _ = io.WriteString(w, "data: {\"error\":{\"type\":\"rate_limit_error\"}}\n\n")
		}))
		defer server.Close()

		client, err := openaicompat.New(openaicompat.Config{
			Endpoint:   server.URL,
			Credential: openaicompat.NewCredential("conformance-retry-token"),
		})
		if err != nil {
			t.Fatalf("New error = %v, want nil", err)
		}
		ch, streamErr := client.Stream(t.Context(), confRetryRequest(t))
		if streamErr != nil {
			t.Fatalf("Stream error = %v, want carrier handover", streamErr)
		}
		events := confRetryDrainAll(t, ch)
		if got := requests.Load(); got != 1 {
			t.Fatalf("wire requests = %d, want 1 after partial output", got)
		}
		var failure *ai.Failure
		for _, event := range events {
			if candidate, ok := event.ErrorPayload(); ok {
				failure = candidate
				break
			}
		}
		if failure == nil {
			t.Fatal("stream emitted no terminal typed failure after partial output")
		}
		if !failure.PartialOutput() {
			t.Fatal("terminal failure PartialOutput() = false, want true")
		}
	})

	t.Run("byte_identical_replay", func(t *testing.T) {
		if !retryOffered {
			t.Skipf("factory declares %v not offered; case skipped, recorded absent (R-CNF-004)", agenttest.CapRetry)
		}
		agenttest.RequireNoGoroutineLeak(t, func() {
			var requests atomic.Int32
			var mu sync.Mutex
			var bodies [][]byte
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Errorf("ReadAll(request body) error = %v", err)
				}
				mu.Lock()
				bodies = append(bodies, append([]byte(nil), body...))
				mu.Unlock()
				w.Header().Set("Retry-After", "0")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = io.WriteString(w, `{"error":{"type":"rate_limit_error"}}`)
			}))
			defer server.Close()

			client, err := openaicompat.New(openaicompat.Config{
				Endpoint:   server.URL,
				Credential: openaicompat.NewCredential("conformance-retry-token"),
			})
			if err != nil {
				t.Fatalf("New error = %v, want nil", err)
			}
			request := confRetryRequest(t)
			original, translateErr := openaicompat.Translate(request)
			if translateErr != nil {
				t.Fatalf("Translate error = %v, want nil", translateErr)
			}
			_, streamErr := client.Stream(t.Context(), request)
			if streamErr == nil {
				t.Fatal("Stream error = nil, want exhaustion error")
			}
			if got, want := requests.Load(), int32(retryDefaultMaxAttempts+1); got != want {
				t.Fatalf("wire requests = %d, want %d", got, want)
			}
			mu.Lock()
			snapshot := append([][]byte(nil), bodies...)
			mu.Unlock()
			if len(snapshot) != retryDefaultMaxAttempts+1 {
				t.Fatalf("recorded bodies = %d, want %d", len(snapshot), retryDefaultMaxAttempts+1)
			}
			for i, body := range snapshot {
				if !bytes.Equal(body, original) {
					t.Fatalf("attempt %d body differs from original bytes", i+1)
				}
				if !bytes.Equal(body, snapshot[0]) {
					t.Fatalf("attempt %d body differs from attempt 1", i+1)
				}
			}
		})
	})

	t.Run("attempt_count_and_final_cause_in_chain", func(t *testing.T) {
		if !retryOffered {
			t.Skipf("factory declares %v not offered; case skipped, recorded absent (R-CNF-004)", agenttest.CapRetry)
		}
		var requests atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			requests.Add(1)
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = io.WriteString(w, `{"error":{"type":"rate_limit_error"}}`)
		}))
		defer server.Close()

		client, err := openaicompat.New(openaicompat.Config{
			Endpoint:   server.URL,
			Credential: openaicompat.NewCredential("conformance-retry-token"),
		})
		if err != nil {
			t.Fatalf("New error = %v, want nil", err)
		}
		_, streamErr := client.Stream(t.Context(), confRetryRequest(t))
		if streamErr == nil {
			t.Fatal("Stream error = nil, want exhaustion error")
		}
		if got, want := requests.Load(), int32(retryDefaultMaxAttempts+1); got != want {
			t.Fatalf("wire requests = %d, want %d", got, want)
		}
		var failure *ai.Failure
		if !errors.As(streamErr, &failure) {
			t.Fatalf("errors.As(%T) did not find *ai.Failure", streamErr)
		}
		if failure.Category() != ai.FailureCategoryRateLimit {
			t.Fatalf("final failure category = %v, want rate-limit", failure.Category())
		}
	})

	t.Run("cap_retry_absent_reported_not_silent", func(t *testing.T) {
		if f.Retry == nil {
			t.Fatalf("factory.Retry is nil; S-CNF-006 should have failed construction")
		}
		if *f.Retry {
			t.Skipf("factory declares %v offered; the absent scenario is not exercised (R-CNF-004)", agenttest.CapRetry)
		}
		record := agenttest.NewCapabilityRecordForTest("S-CNF-074")
		entry, ok := record.Entry(agenttest.CapRetry)
		if !ok {
			t.Fatal("CapRetry entry is missing from the record (R-CNF-017)")
		}
		if entry.Standing != agenttest.StandingOptional {
			t.Errorf("CapRetry standing = %v, want %v (AI-03 §6)", entry.Standing, agenttest.StandingOptional)
		}
		record.SetOutcomeForTest(agenttest.CapRetry, agenttest.OutcomeAbsent)
		got, _ := record.Entry(agenttest.CapRetry)
		if got.Outcome != agenttest.OutcomeAbsent {
			t.Errorf("CapRetry outcome = %v, want %v (R-CNF-004 absent declaration is a conclusion, never silent)", got.Outcome, agenttest.OutcomeAbsent)
		}
	})

	t.Run("factory_nil_retry_defect", func(t *testing.T) {
		got := agenttest.FactoryDefectForTest(f)
		if f.Retry == nil && got == "" {
			t.Fatal("FactoryDefectForTest returned empty against a nil Retry declaration; want a defect naming CapRetry (R-CNF-002, S-CNF-006)")
		}
	})
}

func confRetryStreamOK(t *testing.T, w http.ResponseWriter) {
	t.Helper()
	w.Header().Set("Content-Type", "text/event-stream")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, "data: {\"id\":\"chatcmpl-conf-retry\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{\"role\":\"assistant\"},\"finish_reason\":null}]}\n\n")
	_, _ = io.WriteString(w, "data: {\"id\":\"chatcmpl-conf-retry\",\"model\":\"m\",\"created\":1700000000,\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":{},\"finish_reason\":\"stop\"}]}\n\n")
	_, _ = io.WriteString(w, "data: [DONE]\n\n")
}

func confRetryRequest(t *testing.T) ai.Request {
	t.Helper()
	part, err := ai.NewText("conformance retry probe")
	if err != nil {
		t.Fatalf("ai.NewText: %v", err)
	}
	message, err := ai.NewMessage(ai.RoleUser, part)
	if err != nil {
		t.Fatalf("ai.NewMessage: %v", err)
	}
	req, err := ai.NewRequest("cachicamas-conformance-retry-model", []ai.Message{message})
	if err != nil {
		t.Fatalf("ai.NewRequest: %v", err)
	}
	return req
}

func confRetryDrainAll(t *testing.T, ch <-chan ai.Event) []ai.Event {
	t.Helper()
	deadline := time.After(5 * time.Second)
	var events []ai.Event
	for {
		select {
		case ev, ok := <-ch:
			if !ok {
				return events
			}
			events = append(events, ev)
		case <-deadline:
			t.Fatalf("confRetryDrainAll: timed out after 5s with %d event(s) so far", len(events))
			return events
		}
	}
}
