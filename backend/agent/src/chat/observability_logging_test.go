// Fix C — observability coverage for the chat turn's three log sites:
//
//   - Conversation.Send:        "chat turn started"
//   - chat/projection.project:  "chat turn failed"
//   - chat/http.HandleOpenTurn: "chat turn opened"
//
// All three log records carry the participant id; the failure log
// also carries the typed category from ai.Failure. Without these
// records `docker logs cachicamas-agent-chat` is silent on upstream
// provider failures, leaving an operator with no way to distinguish
// upstream 4xx, upstream 5xx, network timeout, or a classification
// bug.

package chat_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"sync"
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
	"github.com/cachicamas/backend/agent/src/chat"
)

// jsonLogger is a tiny slog.Handler that writes JSON records into a
// bytes.Buffer the test can assert against. It mirrors the
// orchestrator's spec for "slog.New(slog.NewJSONHandler(&buf, ...))"
// — JSON form is the production-shaped wire format, so the test
// parses structured records rather than scraping free-form text.
type jsonLogger struct {
	mu  *sync.Mutex
	buf *bytes.Buffer
}

func newJSONLogger() *jsonLogger {
	return &jsonLogger{mu: &sync.Mutex{}, buf: &bytes.Buffer{}}
}

func (j *jsonLogger) Enabled(context.Context, slog.Level) bool { return true }

func (j *jsonLogger) Handle(_ context.Context, r slog.Record) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	rec := map[string]any{
		"level":   r.Level.String(),
		"message": r.Message,
	}
	r.Attrs(func(a slog.Attr) bool {
		rec[a.Key] = a.Value.Any()
		return true
	})
	enc, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	j.buf.Write(enc)
	j.buf.WriteByte('\n')
	return nil
}

func (j *jsonLogger) WithAttrs(_ []slog.Attr) slog.Handler { return j }
func (j *jsonLogger) WithGroup(_ string) slog.Handler      { return j }

// recordsOfKind returns the JSON-decoded records whose Message field
// equals message. The test asserts on the typed record shape rather
// than on substring matches so a future change to the surrounding
// log line can't silently break the regression guard.
func recordsOfKind(t *testing.T, raw *bytes.Buffer, message string) []map[string]any {
	t.Helper()
	var out []map[string]any
	for _, line := range strings.Split(raw.String(), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var rec map[string]any
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			t.Fatalf("log line is not JSON: %q (err=%v)", line, err)
		}
		if rec["message"] == message {
			out = append(out, rec)
		}
	}
	return out
}

// TestObservability_TurnStartedLoggedOnSend — when Send is invoked,
// the structured logger receives exactly one "chat turn started"
// record carrying the participant id and prompt byte count.
func TestObservability_TurnStartedLoggedOnSend(t *testing.T) {
	t.Parallel()

	logger := newJSONLogger()
	provider := agenttest.NewProvider(scriptTextResponse(t, 1, []string{"hi"}, ai.FinishReasonStop))
	conv, err := chat.NewConversation(chat.Config{
		Provider:      provider,
		Logger:        slog.New(logger),
		Store:         chat.NewMemoryConversationStore(),
		ParticipantID: "obs-alice",
		ToolSource:    chat.FromAgentRegistry(agent.NewMapRegistry(nil)),
		PermissionPolicy: chat.NewDefaultPermissionPolicy(nil),
	})
	if err != nil {
		t.Fatalf("chat.NewConversation returned %v, want nil", err)
	}

	out, err := conv.Send(context.Background(), "hello there")
	if err != nil {
		t.Fatalf("Send returned %v, want nil", err)
	}
	events := drainWire(t, out)
	if !endsWithTurnEnd(events) {
		t.Fatalf("terminal event = %#v, want chat.TurnEnd", events[len(events)-1])
	}

	recs := recordsOfKind(t, logger.buf, "chat turn started")
	if len(recs) != 1 {
		t.Fatalf("chat turn started records = %d, want exactly 1 (Fix C: every Send must log the start)", len(recs))
	}
	if recs[0]["participant_id"] != "obs-alice" {
		t.Errorf("participant_id = %v, want %q", recs[0]["participant_id"], "obs-alice")
	}
	if got, ok := recs[0]["prompt_bytes"].(float64); !ok || int(got) != len("hello there") {
		t.Errorf("prompt_bytes = %v, want %d", recs[0]["prompt_bytes"], len("hello there"))
	}
}

// TestObservability_TurnFailedLogsTypedCategory — when the upstream
// provider fails, the projection goroutine records exactly one
// "chat turn failed" line carrying the typed category from
// ai.Failure. This is the discriminator that lets an operator tell
// upstream 4xx (FailureCategoryAuthentication / Authorization) from
// upstream 5xx (FailureCategoryUnavailable) from network timeout
// (FailureCategoryTimeout) from classification bugs (any other
// category). Without this log line `docker logs cachicamas-agent-chat`
// is silent on the wire's `event: error` frame.
func TestObservability_TurnFailedLogsTypedCategory(t *testing.T) {
	t.Parallel()

	logger := newJSONLogger()

	const wantCategory = ai.FailureCategoryUnavailable
	script := scriptTextThenFailure(t, false, true, wantCategory)
	provider := agenttest.NewProvider(script)

	conv, err := chat.NewConversation(chat.Config{
		Provider:      provider,
		Logger:        slog.New(logger),
		Store:         chat.NewMemoryConversationStore(),
		ParticipantID: "obs-bob",
		ToolSource:    chat.FromAgentRegistry(agent.NewMapRegistry(nil)),
		PermissionPolicy: chat.NewDefaultPermissionPolicy(nil),
	})
	if err != nil {
		t.Fatalf("chat.NewConversation returned %v, want nil", err)
	}

	out, err := conv.Send(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Send returned %v, want nil", err)
	}
	events := drainWire(t, out)
	terminal := events[len(events)-1]
	if _, ok := terminal.(chat.Error); !ok {
		t.Fatalf("terminal event = %#v, want chat.Error (script emits a mid-stream failure)", terminal)
	}

	recs := recordsOfKind(t, logger.buf, "chat turn failed")
	if len(recs) != 1 {
		t.Fatalf("chat turn failed records = %d, want exactly 1 (Fix C: every failed turn must log the category)", len(recs))
	}
	if recs[0]["participant_id"] != "obs-bob" {
		t.Errorf("participant_id = %v, want %q", recs[0]["participant_id"], "obs-bob")
	}
	if recs[0]["category"] != wantCategory.String() {
		t.Errorf("category = %v, want %q (Fix C: typed category is the failure discriminator)", recs[0]["category"], wantCategory.String())
	}
	if recs[0]["level"] != "ERROR" {
		t.Errorf("level = %v, want ERROR (Fix C: failure logs at error level so an alerting pipeline can route them)", recs[0]["level"])
	}
}
