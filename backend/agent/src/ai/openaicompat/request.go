package openaicompat

import (
	"context"
	"net/http"
)

// newRequest builds an HTTP request against the client's base endpoint
// joined with segments, carrying the client's credential as a Bearer
// authorization header (R-APC-001, R-APC-006, R-APC-013).
//
// It is unexported: AI-25 sends no request as adapter behavior other than
// the AI-25.3 viability probe, and AI-26 is the seam that will drive this
// method with a real request body.
func (c *Client) newRequest(ctx context.Context, segments ...string) (*http.Request, error) {
	target := joinRequestPath(c.base, segments...)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.credential.bearer())
	return req, nil
}
