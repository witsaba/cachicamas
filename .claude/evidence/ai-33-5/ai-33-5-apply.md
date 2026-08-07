# AI-33.5 evidence — Resource discipline: drain-before-close + full-package leak check (R-AIS-033, R-AIS-038)

## RED (stream.go with original defer body — no drain)

Without the drain, the test fails as expected. With chunked encoding (forced by Flush()), the transport sees "body not fully consumed" and discards the keep-alive slot after every non-fully-consumed body. Over 50 repeats × 2 streams/repeat = 100 attempts, the ConnState counter recorded **6 distinct TCP connections** for text and **2** for tool-call.

```
--- FAIL: TestAI33_5_DrainBeforeClose_OnCompletionPath_Text (0.10s)
    a_i-33_5a_test.go:373: TCP connections across all repeats = 6, want 1 — drain-before-close did NOT free the keep-alive slot for reuse (R-AIS-033 / S-1): without drain, the transport closes the connection after every non-fully-consumed body
--- FAIL: TestAI33_5_DrainBeforeClose_OnCompletionPath_ToolCall (0.10s)
    a_i-33_5a_test.go:416: TCP connections across all repeats = 2, want 1 — drain-before-close did NOT free the keep-alive slot for reuse on the tool-call path (R-AIS-033 / S-1)
```

The drain tests proved RED genuinely — verified by reverting stream.go's defer chain back to `_ = resp.Body.Close()` only, re-running, observing fail, then re-applying the drain.

## GREEN (with the drain — single deferred io.Copy + Body.Close)

```
=== RUN   TestAI33_5_DrainBeforeClose_OnCompletionPath_Text
--- PASS: TestAI33_5_DrainBeforeClose_OnCompletionPath_Text (0.10s)
=== RUN   TestAI33_5_DrainBeforeClose_OnCompletionPath_ToolCall
--- PASS: TestAI33_5_DrainBeforeClose_OnCompletionPath_ToolCall (0.09s)
=== RUN   TestAI33_5_FullPackageLeakCheck
=== RUN   TestAI33_5_FullPackageLeakCheck/normal_completion_text
=== RUN   TestAI33_5_FullPackageLeakCheck/normal_completion_tool
=== RUN   TestAI33_5_FullPackageLeakCheck/pre_headers_cancel_text
=== RUN   TestAI33_5_FullPackageLeakCheck/pre_headers_cancel_tool
=== RUN   TestAI33_5_FullPackageLeakCheck/between_frames_cancel_text
=== RUN   TestAI33_5_FullPackageLeakCheck/between_frames_cancel_tool
=== RUN   TestAI33_5_FullPackageLeakCheck/blocked_send_abandonment_text
=== RUN   TestAI33_5_FullPackageLeakCheck/blocked_send_abandonment_tool
=== RUN   TestAI33_5_FullPackageLeakCheck/after_completion_cancel_text
=== RUN   TestAI33_5_FullPackageLeakCheck/after_completion_cancel_tool
--- PASS: TestAI33_5_FullPackageLeakCheck (0.74s)
    --- PASS: TestAI33_5_FullPackageLeakCheck/normal_completion_text (0.07s)
    --- PASS: TestAI33_5_FullPackageLeakCheck/normal_completion_tool (0.08s)
    --- PASS: TestAI33_5_FullPackageLeakCheck/pre_headers_cancel_text (0.07s)
    --- PASS: TestAI33_5_FullPackageLeakCheck/pre_headers_cancel_tool (0.07s)
    --- PASS: TestAI33_5_FullPackageLeakCheck/between_frames_cancel_text (0.07s)
    --- PASS: TestAI33_5_FullPackageLeakCheck/between_frames_cancel_tool (0.08s)
    --- PASS: TestAI33_5_FullPackageLeakCheck/blocked_send_abandonment_text (0.07s)
    --- PASS: TestAI33_5_FullPackageLeakCheck/blocked_send_abandonment_tool (0.08s)
    --- PASS: TestAI33_5_FullPackageLeakCheck/after_completion_cancel_text (0.07s)
    --- PASS: TestAI33_5_FullPackageLeakCheck/after_completion_cancel_tool (0.08s)
PASS
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat	2.279s
```

## Full suite — `make test`

```
ok  	github.com/cachicamas/backend/agent/src/agenttest	2.041s
ok  	github.com/cachicamas/backend/agent/src/ai	3.369s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat	20.498s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter	2.039s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter/conformance	1.778s
?   	github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter/conformance/fixtures	[no test files]
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter/smoke	2.290s
FAIL count: 0
```

## make lint

```
go vet ./...
bin/golangci-lint run --config=.golangci.yml ./...
0 issues.
```

## go.mod unchanged (R-STK-009)

```
(no changes — `wc -l backend/agent/go.mod` = 3, zero requires)
```

## Production diff

```
 backend/agent/src/ai/openaicompat/stream.go | 19 ++++++++++++++++++-
 1 file changed, 18 insertions(+), 1 deletion(-)
```

Production logic: +1 line (the existing `defer func() { _ = resp.Body.Close() }()` body changed to `defer func() { _, _ = io.Copy(io.Discard, resp.Body); _ = resp.Body.Close() }()`). The +18 includes 14 lines of header doc comment (reviewer-facing, cites R-AIS-033, capture.go:117-122 mirror, R-ATS-003, R-STK-009) and the refactored defer body.

## Aggregate diff (5 subnodes vs main)

```
 backend/agent/src/ai/openaicompat/a_i-33_1_test.go | 226 +++++++++++
 backend/agent/src/ai/openaicompat/a_i-33_2_test.go | 390 +++++++++++++++++++
 backend/agent/src/ai/openaicompat/a_i-33_3_test.go | 279 ++++++++++++++
 backend/agent/src/ai/openaicompat/a_i-33_4_test.go | 322 ++++++++++++++++
 backend/agent/src/ai/openaicompat/a_i-33_5a_test.go | 423 +++++++++++++++++++++
 backend/agent/src/ai/openaicompat/a_i-33_5b_test.go | 260 +++++++++++++
 backend/agent/src/ai/openaicompat/stream.go        |  19 +-
 7 files changed, 1918 insertions(+), 1 deletion(-)
```

## Commits

```
83eef31 test(openaicompat): add AI-33.5 full-package leak check over every exit path (R-AIS-038)
99fef5a fix(openaicompat): add drain-before-close to run() defer chain (R-AIS-033)
```

Two commits per the work-unit-commits skill: each has one clear purpose (drain impl + drain tests in 99fef5a; full-suite leak check in 83eef31). Split into chained PRs (33.5a + 33.5b) per tasks.md line 173's threshold (`wc -l a_i-33_5_test.go ≥ 400` triggered the split).

## Evidence file SHA256s

- `ai-33-5-red-green.txt`: 73d01db087e0a9a568f56ae9e9f438172188eafa8d1744984c1c180de8045e17
- `ai-33-5-full-suite.txt`: 2dda5a245ccb46d63199fd4aa57c5255732cd62fb2f450f17dd7a9a64713cead
- `ai-33-5-lint.txt`: 66d9a3373b26e70b4206ef2aab426698da81f1f718bd3e1c6bb58b06ca3eb38a

The primary evidence file (the focused test command + result for this subnode) is `ai-33-5-red-green.txt`.
