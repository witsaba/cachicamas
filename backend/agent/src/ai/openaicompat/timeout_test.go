package openaicompat

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// newDripHandler returns a handler that writes chunks body chunks,
// flushing each one, with interval delay before every chunk after the
// first — so headers (and the first chunk) arrive promptly, and the
// remaining chunks arrive spread across roughly (chunks-1)*interval of
// wall-clock time.
func newDripHandler(chunks int, interval time.Duration) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		for i := range chunks {
			// The handler cannot usefully act on a write failure to a
			// ResponseWriter — the client side is what this test observes.
			_, _ = fmt.Fprintf(w, "chunk-%d\n", i)
			flusher.Flush()
			if i < chunks-1 {
				time.Sleep(interval)
			}
		}
	}
}

// countChunksUntilError reads newline-delimited chunks from r until EOF or
// an error, returning the count successfully read and any non-EOF error
// encountered while reading.
func countChunksUntilError(r io.Reader) (int, error) {
	scanner := bufio.NewScanner(r)
	count := 0
	for scanner.Scan() {
		count++
	}
	return count, scanner.Err()
}

// TestTimeout_NoWholeRequestCapOnAdapterBuiltClient is the milestone's
// paired-comparison proof for R-APC-004 (S-APC-017, S-APC-018, S-APC-019).
//
// It is deliberately SERIAL. Do NOT add t.Parallel() anywhere in this test
// or its subtests: Go runs a package's non-parallel tests strictly
// sequentially, which is what keeps this pair's timing free of
// interference from any other test in the package.
//
// A control client carrying a whole-request cap of 2*interval must die
// mid-read against the drip handler below, proving the fixture genuinely
// exercises the whole-request-cap footgun rather than being too fast to
// matter (S-APC-017): the handler's total span is (chunks-1)*interval,
// comfortably longer than the control's cap regardless of machine speed,
// so the control's failure is guaranteed by construction, not by luck.
// Only the failure's SHAPE is asserted — a net.Error reporting
// Timeout()==true, and strictly fewer than chunks chunks read — never a
// duration (NFR-APC-E). The adapter-built client, driven through the exact
// same handler and timing with a caller context carrying no deadline, must
// read every chunk to completion (S-APC-018, S-APC-019).
func TestTimeout_NoWholeRequestCapOnAdapterBuiltClient(t *testing.T) {
	const (
		chunks   = 5
		interval = 200 * time.Millisecond
	)

	server := httptest.NewServer(newDripHandler(chunks, interval))
	defer server.Close()

	t.Run("control client with a whole-request cap dies mid-read", func(t *testing.T) {
		control := &http.Client{Timeout: 2 * interval}

		var (
			got     int
			readErr error
		)
		resp, err := control.Get(server.URL)
		if err != nil {
			readErr = err
		} else {
			defer func() { _ = resp.Body.Close() }()
			got, readErr = countChunksUntilError(resp.Body)
		}

		if readErr == nil {
			t.Fatalf("control read all %d chunks with no error — the fixture did not exercise the whole-request cap (S-APC-017)", got)
		}
		netErr, ok := readErr.(net.Error)
		if !ok {
			t.Fatalf("control error is %T, want a net.Error (S-APC-017); err = %v", readErr, readErr)
		}
		if !netErr.Timeout() {
			t.Errorf("control error.Timeout() = false, want true (S-APC-017); err = %v", readErr)
		}
		if got >= chunks {
			t.Errorf("control read %d/%d chunks, want fewer than %d — never asserting an exact duration (S-APC-017)", got, chunks, chunks)
		}
	})

	t.Run("adapter-built client reads the full stream to completion", func(t *testing.T) {
		c, err := New(Config{
			Endpoint:   server.URL,
			Credential: NewCredential("token"),
		})
		if err != nil {
			t.Fatalf("New() error = %v, want nil", err)
		}

		// A caller context carrying no deadline: any internally imposed
		// deadline would have to originate inside the adapter itself
		// (S-APC-019).
		req, err := c.newRequest(context.Background(), nil)
		if err != nil {
			t.Fatalf("newRequest() error = %v, want nil", err)
		}
		resp, err := c.httpClient.Do(req)
		if err != nil {
			t.Fatalf("httpClient.Do() error = %v, want nil (S-APC-018/S-APC-019)", err)
		}
		defer func() { _ = resp.Body.Close() }()

		got, readErr := countChunksUntilError(resp.Body)
		if readErr != nil {
			t.Fatalf("reading body: %v, want nil — no internally imposed deadline (S-APC-018/S-APC-019)", readErr)
		}
		if got != chunks {
			t.Errorf("adapter-built client read %d/%d chunks, want all %d (S-APC-018)", got, chunks, chunks)
		}
	})
}
