package ai_test

import (
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/cachicamas/backend/database_administrator/src/tools/agent/ai"
)

// TestUsage_NewUsage_RequiredFieldBoundaries exercises the negative-zero-max
// boundary matrix for the 3 required fields. Per AI-10 spec § "Requirement:
// Numeric validation".
func TestUsage_NewUsage_RequiredFieldBoundaries(t *testing.T) {
	t.Run("zero required is valid and absent", func(t *testing.T) {
		u, err := ai.NewUsage(0, 0, 0, nil, nil, nil)
		if err != nil {
			t.Fatalf("NewUsage(0,0,0,...) = %v, want nil", err)
		}
		if !u.IsAbsent() {
			t.Errorf("zero Usage must be absent, got IsAbsent()=false")
		}
		if err := u.Validate(); err != nil {
			t.Errorf("zero Usage.Validate() = %v, want nil", err)
		}
	})

	t.Run("negative input rejected", func(t *testing.T) {
		_, err := ai.NewUsage(-1, 0, 0, nil, nil, nil)
		assertErrorIs(t, err, ai.ErrUsageNegativeInputTokens)
	})
	t.Run("negative output rejected", func(t *testing.T) {
		_, err := ai.NewUsage(0, -1, 0, nil, nil, nil)
		assertErrorIs(t, err, ai.ErrUsageNegativeOutputTokens)
	})
	t.Run("negative total rejected", func(t *testing.T) {
		_, err := ai.NewUsage(0, 0, -1, nil, nil, nil)
		assertErrorIs(t, err, ai.ErrUsageNegativeTotalTokens)
	})

	t.Run("max input accepted", func(t *testing.T) {
		u, err := ai.NewUsage(math.MaxInt64, 0, math.MaxInt64, nil, nil, nil)
		if err != nil {
			t.Fatalf("NewUsage(MaxInt64,0,MaxInt64,...) = %v, want nil", err)
		}
		if u.InputTokens() != math.MaxInt64 {
			t.Errorf("InputTokens() = %d, want MaxInt64", u.InputTokens())
		}
	})
	t.Run("max output accepted", func(t *testing.T) {
		u, err := ai.NewUsage(0, math.MaxInt64, math.MaxInt64, nil, nil, nil)
		if err != nil {
			t.Fatalf("NewUsage(0,MaxInt64,MaxInt64,...) = %v, want nil", err)
		}
		if u.OutputTokens() != math.MaxInt64 {
			t.Errorf("OutputTokens() = %d, want MaxInt64", u.OutputTokens())
		}
	})
	t.Run("max total accepted", func(t *testing.T) {
		u, err := ai.NewUsage(0, 0, math.MaxInt64, nil, nil, nil)
		if err != nil {
			t.Fatalf("NewUsage(0,0,MaxInt64,...) = %v, want nil", err)
		}
		if u.TotalTokens() != math.MaxInt64 {
			t.Errorf("TotalTokens() = %d, want MaxInt64", u.TotalTokens())
		}
	})
}

// TestUsage_NewUsage_ProviderInvariant exercises the strict-rejection policy
// for TotalTokens < InputTokens + OutputTokens. Per AI-10 spec § "Requirement:
// Numeric validation" and design #8.
func TestUsage_NewUsage_ProviderInvariant(t *testing.T) {
	t.Run("total equals sum accepted", func(t *testing.T) {
		_, err := ai.NewUsage(10, 20, 30, nil, nil, nil)
		if err != nil {
			t.Errorf("NewUsage(10,20,30) = %v, want nil", err)
		}
	})
	t.Run("total greater than sum accepted", func(t *testing.T) {
		_, err := ai.NewUsage(10, 20, 100, nil, nil, nil)
		if err != nil {
			t.Errorf("NewUsage(10,20,100) = %v, want nil", err)
		}
	})
	t.Run("total less than sum rejected", func(t *testing.T) {
		_, err := ai.NewUsage(10, 20, 25, nil, nil, nil)
		assertErrorIs(t, err, ai.ErrUsageTotalLessThanInputs)
	})
	t.Run("total less than input alone rejected", func(t *testing.T) {
		_, err := ai.NewUsage(50, 0, 10, nil, nil, nil)
		assertErrorIs(t, err, ai.ErrUsageTotalLessThanInputs)
	})
}

// TestUsage_NewUsage_OverflowSum covers input+output wrapping. When the sum
// would overflow int64, validation MUST detect it before comparison and
// return ErrUsageOverflow. Per AI-10 spec § "Requirement: Numeric validation".
func TestUsage_NewUsage_OverflowSum(t *testing.T) {
	_, err := ai.NewUsage(math.MaxInt64, 1, math.MaxInt64, nil, nil, nil)
	assertErrorIs(t, err, ai.ErrUsageOverflow)

	// Even with total = MaxInt64 (would otherwise satisfy the invariant
	// as <= comparison), the sum-overflow check runs first and wins.
	_, err = ai.NewUsage(math.MaxInt64, 1, 1, nil, nil, nil)
	assertErrorIs(t, err, ai.ErrUsageOverflow)
}

// TestUsage_NewUsage_OptionalFields covers pointer-aliasing safety, absent vs
// present distinction, and the cacheR > InputTokens consistency rule. Per
// AI-10 spec § "Requirement: Counts and presence".
func TestUsage_NewUsage_OptionalFields(t *testing.T) {
	t.Run("all nil accepted and absent", func(t *testing.T) {
		u, err := ai.NewUsage(10, 20, 30, nil, nil, nil)
		if err != nil {
			t.Fatalf("setup = %v", err)
		}
		if val, ok := u.CacheReadTokens(); ok {
			t.Errorf("CacheReadTokens present = %d, want absent", val)
		}
		if val, ok := u.CacheWriteTokens(); ok {
			t.Errorf("CacheWriteTokens present = %d, want absent", val)
		}
		if val, ok := u.ReasoningTokens(); ok {
			t.Errorf("ReasoningTokens present = %d, want absent", val)
		}
		if u.IsAbsent() {
			t.Errorf("non-zero Usage with all-nil details must NOT be absent (got IsAbsent()=true)")
		}
	})

	t.Run("explicit zero detail is present and zero", func(t *testing.T) {
		zero := int64(0)
		u, err := ai.NewUsage(10, 20, 30, &zero, &zero, &zero)
		if err != nil {
			t.Fatalf("setup = %v", err)
		}
		if val, ok := u.CacheReadTokens(); !ok || val != 0 {
			t.Errorf("CacheReadTokens = (%d, %v), want (0, true)", val, ok)
		}
		if val, ok := u.CacheWriteTokens(); !ok || val != 0 {
			t.Errorf("CacheWriteTokens = (%d, %v), want (0, true)", val, ok)
		}
		if val, ok := u.ReasoningTokens(); !ok || val != 0 {
			t.Errorf("ReasoningTokens = (%d, %v), want (0, true)", val, ok)
		}
		if u.IsAbsent() {
			t.Error("non-zero Usage with explicit-zero details must NOT be absent")
		}
	})

	t.Run("set value preserved verbatim", func(t *testing.T) {
		cr, cw, rs := int64(5), int64(7), int64(11)
		u, err := ai.NewUsage(20, 30, 50, &cr, &cw, &rs)
		if err != nil {
			t.Fatalf("setup = %v", err)
		}
		if val, ok := u.CacheReadTokens(); !ok || val != 5 {
			t.Errorf("CacheReadTokens = (%d, %v), want (5, true)", val, ok)
		}
		if val, ok := u.CacheWriteTokens(); !ok || val != 7 {
			t.Errorf("CacheWriteTokens = (%d, %v), want (7, true)", val, ok)
		}
		if val, ok := u.ReasoningTokens(); !ok || val != 11 {
			t.Errorf("ReasoningTokens = (%d, %v), want (11, true)", val, ok)
		}
	})

	t.Run("negative cache read collapsed to overflow", func(t *testing.T) {
		neg := int64(-1)
		_, err := ai.NewUsage(10, 20, 30, &neg, nil, nil)
		assertErrorIs(t, err, ai.ErrUsageOverflow)
	})

	t.Run("cache read exceeding input rejected", func(t *testing.T) {
		cr := int64(20)
		_, err := ai.NewUsage(10, 0, 10, &cr, nil, nil)
		assertErrorIs(t, err, ai.ErrUsageInconsistentCacheTokens)
	})

	t.Run("cache read equal to input accepted", func(t *testing.T) {
		cr := int64(10)
		_, err := ai.NewUsage(10, 0, 10, &cr, nil, nil)
		if err != nil {
			t.Errorf("NewUsage cacheR == input rejected: %v, want nil", err)
		}
	})
}

// TestUsage_IsAbsent_Matrix pins IsAbsent across the 6-field zero-vs-non-zero
// matrix. Per AI-10 spec § "Requirement: Counts and presence".
func TestUsage_IsAbsent_Matrix(t *testing.T) {
	zero := int64(0)
	one := int64(1)
	cases := []struct {
		name string
		u    ai.Usage
		want bool
	}{
		{"zero value literal", ai.Usage{}, true},
		{"input only", mustUsage(t, 1, 0, 1, nil, nil, nil), false},
		{"output only", mustUsage(t, 0, 1, 1, nil, nil, nil), false},
		{"total only", mustUsage(t, 0, 0, 1, nil, nil, nil), false},
		{"cache read present (even at zero)", mustUsage(t, 0, 0, 0, &zero, nil, nil), false},
		{"cache write present (even at zero)", mustUsage(t, 0, 0, 0, nil, &zero, nil), false},
		{"reasoning present (even at zero)", mustUsage(t, 0, 0, 0, nil, nil, &zero), false},
		{"cache read set non-zero", mustUsage(t, 5, 0, 5, &one, nil, nil), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.u.IsAbsent(); got != tc.want {
				t.Errorf("Usage{...}.IsAbsent() = %v, want %v (usage=%+v)", got, tc.want, tc.u)
			}
		})
	}
}

// TestUsage_RequiredAccessorsRoundTrip verifies that the 3 required-field
// accessors return the verbatim value supplied to the constructor. Per AI-10
// spec § "Requirement: Counts and presence".
func TestUsage_RequiredAccessorsRoundTrip(t *testing.T) {
	u, err := ai.NewUsage(11, 22, 33, nil, nil, nil)
	if err != nil {
		t.Fatalf("setup = %v", err)
	}
	if u.InputTokens() != 11 {
		t.Errorf("InputTokens() = %d, want 11", u.InputTokens())
	}
	if u.OutputTokens() != 22 {
		t.Errorf("OutputTokens() = %d, want 22", u.OutputTokens())
	}
	if u.TotalTokens() != 33 {
		t.Errorf("TotalTokens() = %d, want 33", u.TotalTokens())
	}
}

// TestUsage_NewUsage_InvalidInputReturnsZero verifies the construction-error
// contract: a failing NewUsage returns zero Usage and an error. Per AI-10
// spec § "Requirement: Counts and presence" ("failure MUST return zero
// Usage and an error").
func TestUsage_NewUsage_InvalidInputReturnsZero(t *testing.T) {
	u, err := ai.NewUsage(-1, -1, -1, nil, nil, nil)
	if err == nil {
		t.Fatal("NewUsage(-1,-1,-1) error = nil, want non-nil")
	}
	if !reflect.DeepEqual(u, ai.Usage{}) {
		t.Errorf("NewUsage error path returned non-zero Usage: %#v", u)
	}
	if !u.IsAbsent() {
		t.Errorf("failed-construction Usage must be absent, got IsAbsent()=false")
	}
}

// TestUsage_ValidationOrder_FirstFailureWins exercises the first-failure
// semantics: when multiple fields violate the rules, the first violation in
// the spec order (input, output, total, cache-read, cache-write, reasoning,
// overflow, total-invariant) is returned. Per AI-10 spec § "Scenario:
// Invalid numeric matrix".
func TestUsage_ValidationOrder_FirstFailureWins(t *testing.T) {
	neg := int64(-1)
	// Both input and total negative — input wins.
	_, err := ai.NewUsage(-1, 0, -1, nil, nil, nil)
	assertErrorIs(t, err, ai.ErrUsageNegativeInputTokens)

	// Both output and total negative — output wins (input is OK).
	_, err = ai.NewUsage(0, -1, -1, nil, nil, nil)
	assertErrorIs(t, err, ai.ErrUsageNegativeOutputTokens)

	// Cache-read negative + total-invariant violated — cache-read wins (overflow
	// category, since cache-read is optional).
	_, err = ai.NewUsage(10, 20, 5, &neg, nil, nil)
	assertErrorIs(t, err, ai.ErrUsageOverflow)
}

// TestUsage_Sentinels_AreAllDistinct verifies the 6 Usage sentinels are
// pairwise distinguishable via errors.Is (no aliasing). Per AI-10 spec §
// "Requirement: Numeric validation" (pairwise errors.Is-distinguishable).
func TestUsage_Sentinels_AreAllDistinct(t *testing.T) {
	sentinels := usageSentinels()
	if len(sentinels) != 6 {
		t.Fatalf("Usage sentinel count = %d, want 6", len(sentinels))
	}
	for _, s := range sentinels {
		if s == nil {
			t.Fatal("Usage sentinel is nil")
		}
		if !errors.Is(s, s) {
			t.Errorf("sentinel %v must satisfy errors.Is(err, err)", s)
		}
		if !strings.HasPrefix(s.Error(), "ai: ") {
			t.Errorf("sentinel %q lacks ai: prefix", s)
		}
	}
	for i := range sentinels {
		for j := range sentinels {
			if i != j && errors.Is(sentinels[i], sentinels[j]) {
				t.Errorf("sentinels %v and %v alias via errors.Is", sentinels[i], sentinels[j])
			}
		}
	}
}

// TestUsage_NoMarshalOrClone verifies, via reflection, that Usage exposes
// no wire-format or cloning methods. AI-10 keeps Usage as a pure value type;
// AI-11 owns marshaling. Per AI-10 design #11 and spec § "Requirement: Layer
// and scope boundary".
func TestUsage_NoMarshalOrClone(t *testing.T) {
	assertNoMethods(t, reflect.TypeOf(ai.Usage{}), "MarshalJSON", "UnmarshalJSON", "MarshalText", "UnmarshalText", "Clone")
}

// TestUsage_NotContentPart verifies, via reflection, that Usage does NOT
// implement ContentPart (no Kind() method). Per AI-10 spec § "Requirement:
// Layer and scope boundary".
func TestUsage_NotContentPart(t *testing.T) {
	if _, ok := reflect.TypeOf(ai.Usage{}).MethodByName("Kind"); ok {
		t.Error("Usage must not implement ContentPart (no Kind() method)")
	}
}

// TestMaxUsageTokenCount_Value pins the constant value. The design names
// MaxUsageTokenCount = math.MaxInt64.
func TestMaxUsageTokenCount_Value(t *testing.T) {
	if ai.MaxUsageTokenCount != math.MaxInt64 {
		t.Errorf("MaxUsageTokenCount = %d, want math.MaxInt64 (%d)", ai.MaxUsageTokenCount, math.MaxInt64)
	}
}

// usageSentinels returns the 6 Usage sentinel errors in canonical declaration
// order. Helper to power the pairwise-distinctness assertion.
func usageSentinels() []error {
	return []error{
		ai.ErrUsageNegativeInputTokens,
		ai.ErrUsageNegativeOutputTokens,
		ai.ErrUsageNegativeTotalTokens,
		ai.ErrUsageOverflow,
		ai.ErrUsageTotalLessThanInputs,
		ai.ErrUsageInconsistentCacheTokens,
	}
}

// mustUsage constructs a Usage or fails the test. Used only by the IsAbsent
// matrix where the inputs are guaranteed valid by construction.
func mustUsage(t *testing.T, in, out, total int64, cacheR, cacheW, reason *int64) ai.Usage {
	t.Helper()
	u, err := ai.NewUsage(in, out, total, cacheR, cacheW, reason)
	if err != nil {
		t.Fatalf("mustUsage setup error = %v", err)
	}
	return u
}
