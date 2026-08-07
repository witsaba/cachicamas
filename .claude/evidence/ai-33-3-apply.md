# AI-33.3 evidence — truly-abandoned consumer + cancellation drops bare (R-AIS-036 / R-CNF-012)

## RED (naive 100ms window — consumer not truly abandoned)
```
--- FAIL: TestAI33_3_TextStream_AbandonedThenCancelled (0.10s)
    R-AIS-036 / S-1 (text): first receive after the abandonment window returned error, want a closed channel
--- FAIL: TestAI33_3_ToolCallStream_AbandonedThenCancelled (0.10s)
    R-AIS-036 / S-2 (tool-call): first receive after the abandonment window returned error, want a closed channel
```

## GREEN (window > emitFailureSendBound) — go test -race -run TestAI33_3
```
=== RUN   TestAI33_3_TextStream_AbandonedThenCancelled
--- PASS: TestAI33_3_TextStream_AbandonedThenCancelled (5.50s)
=== RUN   TestAI33_3_ToolCallStream_AbandonedThenCancelled
--- PASS: TestAI33_3_ToolCallStream_AbandonedThenCancelled (5.50s)
=== RUN   TestAI33_3_AbandonedThenCancelled_LeakFree
--- PASS: TestAI33_3_AbandonedThenCancelled_LeakFree (0.09s)
=== RUN   TestAI33_3_AbandonedNeverCancelledPathNotAsserted
    a_i-33_3_test.go:278: scanned 4 AI-33 test file(s), 13 test declaration(s): abandoned-never-cancelled remains unasserted, on the record (S-CNF-031, R-STK-010)
--- PASS: TestAI33_3_AbandonedNeverCancelledPathNotAsserted (0.00s)
PASS
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat	12.581s
```

## S-3 guard bite-proof (scratch violation, then removed)
```
--- FAIL: TestAI33_3_AbandonedNeverCancelledPathNotAsserted (0.00s)
    a_i-33_9_scratch_test.go declares TestAI33_3_ScratchAbandonedNeverCancelledLeaks: the abandoned-never-cancelled path must not be asserted
```

## Full suite — make test
```
ok  	github.com/cachicamas/backend/agent/src/agenttest	2.022s
ok  	github.com/cachicamas/backend/agent/src/ai	3.217s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat	19.236s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter	(cached)
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter/conformance	(cached)
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter/smoke	(cached)
FAIL count: 0
```

## make lint
```
0 issues.
```

## go.mod unchanged (R-STK-009)
```
(no changes)
```
