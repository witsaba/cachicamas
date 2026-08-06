package openaicompat

import (
	"bytes"
	"context"
	"io"
	"net/http"
)

// newRequest builds an HTTP request against the client's base endpoint
// joined with segments, carrying the client's credential as a Bearer
// authorization header (R-APC-001, R-APC-006, R-APC-013), and body as its
// request body when non-nil.
//
// It is unexported. AI-25 sent no request as adapter behavior other than
// the AI-25.3 viability probe (body left nil there, as it still is); AI-28
// is the seam that first drives this method with a real request body
// (R-ATS-002). Content-Type: application/json and Accept:
// text/event-stream (dialect-conventional, fixture-pinned) are set only
// when body is non-nil — neither header describes a request that carries
// no body, and the viability probe and this package's own construction-
// fault tests call newRequest with none.
func (c *Client) newRequest(ctx context.Context, body []byte, segments ...string) (*http.Request, error) {
	target := joinRequestPath(c.base, segments...)

	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target.String(), bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", c.credential.bearer())
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "text/event-stream")
	}
	return req, nil
}
