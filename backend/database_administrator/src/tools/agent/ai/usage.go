package ai

import (
	"errors"
	"math"
)

// MaxUsageTokenCount is the inclusive upper bound on every present Usage
// token count. The value is math.MaxInt64 (the maximum value an int64
// can hold) so the bound is forward-safe: any positive int64 value a
// vendor could report is admitted in v1, while the contract remains
// anchored on a named constant for future tightenings. Per AI-10 design #7.
const MaxUsageTokenCount = math.MaxInt64

// ErrUsageNegativeInputTokens is returned when InputTokens is negative.
// Per AI-10 spec § "Requirement: Numeric validation" (first-failure
// order: input).
var ErrUsageNegativeInputTokens = errors.New("ai: negative input token count")

// ErrUsageNegativeOutputTokens is returned when OutputTokens is negative.
// Per AI-10 spec § "Requirement: Numeric validation" (first-failure
// order: output).
var ErrUsageNegativeOutputTokens = errors.New("ai: negative output token count")

// ErrUsageNegativeTotalTokens is returned when TotalTokens is negative.
// Per AI-10 spec § "Requirement: Numeric validation" (first-failure
// order: total).
var ErrUsageNegativeTotalTokens = errors.New("ai: negative total token count")

// ErrUsageOverflow is the catch-all for "value out of [0, MaxUsageTokenCount]"
// on optional detail fields (cache-read, cache-write, reasoning) and for
// input+output sum wrapping. Per AI-10 design reconciliation note
// ("Negative cache-read/cache-write/reasoning collapse to ErrUsageOverflow")
// and spec § "Requirement: Numeric validation" (first-failure order:
// cache-read, cache-write, reasoning, overflow).
var ErrUsageOverflow = errors.New("ai: token count exceeds MaxUsageTokenCount or negative optional detail")

// ErrUsageTotalLessThanInputs is returned when TotalTokens < InputTokens + OutputTokens.
// Per AI-10 spec § "Requirement: Numeric validation" (first-failure
// order: total invariant — runs LAST so it does not mask earlier
// field-level violations).
var ErrUsageTotalLessThanInputs = errors.New("ai: total token count less than input plus output")

// ErrUsageInconsistentCacheTokens is returned when an explicitly-present
// CacheReadTokens exceeds InputTokens (a provider invariant: cache reads
// cannot exceed fresh inputs). Per AI-10 spec § "Requirement: Numeric
// validation" (first-failure order: cache-read — runs after the cache-read
// overflow check).
var ErrUsageInconsistentCacheTokens = errors.New("ai: cache read tokens exceed input tokens")

// Usage is the provider-neutral value type for normalized token usage
// reported at the end of a model completion. AI-10 commits three required
// counts (input, output, total) and three optional counts (cache-read,
// cache-write, reasoning). Per AI-10 spec § "Requirement: Counts and presence".
//
// Vocabulary: see AI-00 § usage.
//
// The fields are unexported so the constructor NewUsage is the only path
// that produces valid values. Reconstruct to extend; Usage is immutable.
// Usage is struct-valued (not a pointer), so var u Usage is a usable zero
// value that Validate accepts as the "absent" sentinel. This mirrors the
// GenerationOptions{} zero-value-valid precedent from AI-09.
//
// Wire-format note: JSON marshaling is owned by AI-11 (event envelope).
// Usage exposes no MarshalJSON / UnmarshalJSON / MarshalText /
// UnmarshalText on its own. Per AI-10 design #11.
type Usage struct {
	inputTokens      int64
	outputTokens     int64
	totalTokens      int64
	cacheReadTokens  *int64
	cacheWriteTokens *int64
	reasoningTokens  *int64
}

// NewUsage constructs a Usage after validating every present count. It
// returns a zero Usage with the first validation error. Validation order
// is fixed by the spec (per AI-10 § "Requirement: Numeric validation"):
//
//  1. input < 0                            → ErrUsageNegativeInputTokens
//  2. output < 0                           → ErrUsageNegativeOutputTokens
//  3. total < 0                            → ErrUsageNegativeTotalTokens
//  4. cacheRead present and < 0            → ErrUsageOverflow
//  5. cacheWrite present and < 0           → ErrUsageOverflow
//  6. reasoning present and < 0            → ErrUsageOverflow
//  7. input + output sum wraps int64       → ErrUsageOverflow
//  8. total < input + output               → ErrUsageTotalLessThanInputs
//
// The cacheR > InputTokens consistency check (ErrUsageInconsistentCacheTokens)
// runs as part of step 4 after the negative check, so an unset/zero
// cacheR never trips it.
func NewUsage(inputTokens, outputTokens, totalTokens int64, cacheRead, cacheWrite, reasoning *int64) (Usage, error) {
	u := Usage{
		inputTokens:      inputTokens,
		outputTokens:     outputTokens,
		totalTokens:      totalTokens,
		cacheReadTokens:  cacheRead,
		cacheWriteTokens: cacheWrite,
		reasoningTokens:  reasoning,
	}
	if err := u.Validate(); err != nil {
		return Usage{}, err
	}
	return u, nil
}

// Validate re-runs the validation rules against the stored fields in
// construction order. The zero value Usage{} is valid and IsAbsent() == true.
// Validation cost is O(1).
func (u Usage) Validate() error {
	// 1. Input.
	if u.inputTokens < 0 {
		return ErrUsageNegativeInputTokens
	}
	// 2. Output.
	if u.outputTokens < 0 {
		return ErrUsageNegativeOutputTokens
	}
	// 3. Total.
	if u.totalTokens < 0 {
		return ErrUsageNegativeTotalTokens
	}
	// 4. Cache-read (if present): negative → overflow; > input → inconsistent.
	if u.cacheReadTokens != nil {
		if *u.cacheReadTokens < 0 {
			return ErrUsageOverflow
		}
		if *u.cacheReadTokens > u.inputTokens {
			return ErrUsageInconsistentCacheTokens
		}
	}
	// 5. Cache-write (if present): negative → overflow.
	if u.cacheWriteTokens != nil {
		if *u.cacheWriteTokens < 0 {
			return ErrUsageOverflow
		}
	}
	// 6. Reasoning (if present): negative → overflow.
	if u.reasoningTokens != nil {
		if *u.reasoningTokens < 0 {
			return ErrUsageOverflow
		}
	}
	// 7. Input + output sum overflow. Detect via the carry-out predicate
	// (a + b > MaxInt64) rather than letting the addition wrap, because
	// wrapped results would silently mis-classify the total invariant.
	if u.inputTokens > 0 && u.outputTokens > math.MaxInt64-u.inputTokens {
		return ErrUsageOverflow
	}
	// 8. Total invariant: TotalTokens >= InputTokens + OutputTokens.
	// This step runs LAST so it does not mask earlier field-level
	// violations (negative required, optional overflow, sum wrap).
	if u.totalTokens < u.inputTokens+u.outputTokens {
		return ErrUsageTotalLessThanInputs
	}
	return nil
}

// InputTokens returns the verbatim input token count. The zero value Usage
// returns 0; NewUsage rejects negative input at construction.
func (u Usage) InputTokens() int64 {
	return u.inputTokens
}

// OutputTokens returns the verbatim output token count.
func (u Usage) OutputTokens() int64 {
	return u.outputTokens
}

// TotalTokens returns the verbatim total token count.
func (u Usage) TotalTokens() int64 {
	return u.totalTokens
}

// CacheReadTokens returns the cache-read count and a present flag. The
// flag is true iff the caller passed a non-nil pointer to NewUsage;
// an explicit zero is present-and-zero (not absent). Per AI-10 spec §
// "Requirement: Counts and presence" (optional absence MUST differ
// from explicit zero).
func (u Usage) CacheReadTokens() (int64, bool) {
	if u.cacheReadTokens == nil {
		return 0, false
	}
	return *u.cacheReadTokens, true
}

// CacheWriteTokens returns the cache-write count and a present flag.
// Same absent-vs-explicit-zero semantics as CacheReadTokens.
func (u Usage) CacheWriteTokens() (int64, bool) {
	if u.cacheWriteTokens == nil {
		return 0, false
	}
	return *u.cacheWriteTokens, true
}

// ReasoningTokens returns the reasoning count and a present flag.
// Same absent-vs-explicit-zero semantics as CacheReadTokens.
func (u Usage) ReasoningTokens() (int64, bool) {
	if u.reasoningTokens == nil {
		return 0, false
	}
	return *u.reasoningTokens, true
}

// IsAbsent reports whether the Usage carries no information: all three
// required counts are zero AND all three optional details are unset.
// Any non-zero required count or any present optional detail makes the
// Usage non-absent. The zero-value Usage{} is absent and validates
// successfully (a meaningful "no usage reported" state, distinct from
// "usage reported with all zeros"). Per AI-10 spec § "Requirement:
// Counts and presence".
func (u Usage) IsAbsent() bool {
	return u.inputTokens == 0 &&
		u.outputTokens == 0 &&
		u.totalTokens == 0 &&
		u.cacheReadTokens == nil &&
		u.cacheWriteTokens == nil &&
		u.reasoningTokens == nil
}
