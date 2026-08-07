package openaicompat

import (
	"context"
	"net/http"
	"time"
)

// executeOnce performs one complete pre-stream attempt. It rebuilds the
// request from the immutable body for every call so retry.Loop can replay the
// exact wire bytes without sharing a consumed request reader.
func (c *Client) executeOnce(ctx context.Context, body []byte) (*http.Response, error) {
	httpReq, err := c.newRequest(ctx, body, "chat", "completions")
	if err != nil {
		return nil, preStreamTransportFailure(ctx, err)
	}
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, preStreamTransportFailure(ctx, err)
	}
	if failure := mapResponse(resp, time.Now()); failure != nil {
		return nil, failure
	}
	return resp, nil
}
