// AG-17.2 — the exported counting-capable fixture (design.md DD7): the
// stubProviderWithTokenCounter embedding shape (provider_test.go:133-
// 142), made exported and scriptable. Embedding is *Provider — Stream
// is on the pointer receiver (fake_provider.go:74), so value embedding
// would not satisfy ai.ModelProvider at all.

package agenttest

import (
	"context"
	"slices"
	"sync"

	"github.com/cachicamas/backend/agent/src/ai"
)

// CountingProvider embeds *Provider and advertises ai.TokenCounter by
// satisfying it, and by no other means (ai-minimum-capabilities §9).
type CountingProvider struct {
	*Provider

	mu        sync.Mutex
	result    ai.TokenCount
	err       error
	countReqs []ai.Request
}

// NewCountingProvider constructs a counting-capable fake: CountTokens
// reports result for every call. scripts queue Stream calls exactly as
// NewProvider.
func NewCountingProvider(result ai.TokenCount, scripts ...Script) *CountingProvider {
	return &CountingProvider{Provider: NewProvider(scripts...), result: result}
}

// NewFailingCountingProvider constructs an ADVERTISED counter that
// declines every CountTokens call with err — the R-AMP-019
// non-conformance fixture. Two constructors, so a (result, err) pair
// can never be half-stated.
func NewFailingCountingProvider(err error, scripts ...Script) *CountingProvider {
	return &CountingProvider{Provider: NewProvider(scripts...), err: err}
}

// CountTokens implements ai.TokenCounter, capturing req under mu
// before answering with the scripted result or error.
func (p *CountingProvider) CountTokens(_ context.Context, req ai.Request) (ai.TokenCount, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.countReqs = append(p.countReqs, req)
	if p.err != nil {
		return ai.TokenCount{}, p.err
	}
	return p.result, nil
}

// CountRequests returns every request CountTokens captured, in call
// order. Fresh slice on every call — the Requests() posture
// (fake_provider.go:157-161).
func (p *CountingProvider) CountRequests() []ai.Request {
	p.mu.Lock()
	defer p.mu.Unlock()
	return slices.Clone(p.countReqs)
}

// Compile-time guards: CountingProvider must satisfy both the
// required ai.ModelProvider surface (via the embedded *Provider) and
// the optional ai.TokenCounter surface it adds.
var _ ai.ModelProvider = (*CountingProvider)(nil)
var _ ai.TokenCounter = (*CountingProvider)(nil)
