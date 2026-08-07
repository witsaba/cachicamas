// Package retry implements the Layer 1 pre-stream retry discipline.
//
// DefaultMaxAttempts = 3 is the retry budget: a logical call may issue at
// most N+1 = 4 wire requests when every pre-stream failure is retryable.
// The composed ceiling for Layer 2 is harness attempts × Layer 1 attempts.
// AG-15.2 (doc 0003, line 718) consumes this documented Layer 1 multiplier.
//
// Loop must be called before a stream carrier is handed to a consumer. It
// retries only failures observed before that handover; mid-stream failures
// are delivered through the stream's terminal event and never reach this
// package.
package retry
