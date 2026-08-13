// Package agent is Layer 2 of the cachicamas agent stack. This file
// adds AG-07's walking skeleton: the one-turn function `Turn`, the
// walking-skeleton options struct `TurnOptions`, and the per-call
// identity minters the loop uses to satisfy R-LSK-002 (two sequential
// turns share nothing).
//
// AG-07 is the first milestone where Layer 2 emits events from a live
// loop: AG-04/05/06's stream-validator tests fed the validator hand-built
// events, and AG-07 owns the producer side of AG-01's carrier decision
// (per `openspec/changes/cachicamas-agent-loop-skeleton/design.md`).
// The single public surface is `Turn`: a function form, not a value, so
// AG-13's later `Harness` can wrap it without changing the signature
// (D1a) and so two `Turn(...)` invocations share nothing by construction
// (D1a, R-LSK-002). The function mints a fresh `runID`, `turnID` and
// `LaneStamper` per call (D1a), translates each provider event into a
// bracket on the Layer 2 envelope, and emits the resulting events on
// `sink`, closing it before returning.
//
// # Substrate preservation
//
// AG-07 changes are limited to this file and `loop_test.go`. The
// substrate's envelope, descriptor vocabulary, validator, run / turn /
// message / tool / permission / cost / delegation / compaction families,
// and every-kind-constructible guard stay byte-unchanged (R-LSK-004,
// NFR-LSK-003): 4th consecutive "substrate untouched" milestone.
//
// # Walking-skeleton scope
//
// The function is the thinnest end-to-end path. Tools, hooks, errors
// beyond typed pass-through, permission, retry, context-check, cost
// events are AG-08…AG-18; value-form `Harness` is AG-13; multi-turn
// state is AG-21. AG-07 emits one assistant turn, end to end.
package agent

import (
	"context"

	"github.com/cachicamas/backend/agent/src/ai"
)

// TurnOptions is the options struct for a single assistant turn.
// Walking-skeleton scope: trivial/zero fields; AG-08 introduces hooks,
// AG-09 introduces tool opts, AG-11 introduces retry opts.
type TurnOptions struct {
	// Model is the model identifier passed to the provider. Empty = provider default.
	Model string
	// MaxTokens is the optional max-tokens budget. Zero = provider default.
	MaxTokens int
}

// Turn runs one assistant turn end-to-end and emits the resulting events
// to sink. The function is stateless across calls: two Turn(...)
// invocations share nothing by construction (no closure captures, no
// shared LaneStamper — each turn mints a fresh one, R-LSK-002). The
// function closes sink before returning.
//
// R-LSK-001..005 / S-LSK-001..007 — see
// openspec/specs/agent-loop-skeleton/spec.md.
func Turn(
	_ context.Context,
	_ ai.ModelProvider,
	_ string,
	_ []ai.Message,
	_ TurnOptions,
	sink chan<- *Event,
) (ai.Message, ai.FinishReason, error) {
	// Phase-1 stub closes the sink so consumers never strand a
	// producer. The real implementation lands at Phase 2 (S-LSK-001
	// GREEN) — this stub's only job today is to prove the file
	// compiles and the substrate stays untouched.
	close(sink)
	return ai.Message{}, 0, nil
}
