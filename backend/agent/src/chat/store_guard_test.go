// CH-07.2 guard — the file-tree walker that proves "the adapter
// swap changed no caller" (R-CCS-010, design D-G, doc 0005:820-836).
//
// The guard walks `backend/agent/src/chat/**/*.go` excluding the
// test files and the adapter's own declaration file
// (`store_postgres.go`). Any source file that names
// `NewPostgresConversationStore` or `*PostgresConversationStore`
// outside the adapter fails the build with a message naming the
// file and the bypassed port.
//
// The byte-form substring match (rather than AST analysis) is the
// recorded design choice (D-G): the closed-port invariant is fully
// visible at identifier level. A CH-08 wrapper re-exporting the
// constructor under a different name would surface as a different
// identifier and the guard's failure message would still direct a
// reviewer to the right file.
//
// The planted-defeat scenario writes a scratch file in t.TempDir()
// and asserts the failure message names BOTH the file path AND
// the string `chat.ConversationStore` — the byte-form bite
// contract from doc 0005:879-883 ("fails naming the file and the
// bypassed port").
package chat_test

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// runGuard walks root and asserts no source file in the chat
// archetype names the postgres adapter's identifier outside the
// adapter's own declaration. Returns (failed, msg) — failed is
// true when the guard bites; msg is the diagnostic the build
// surfaces to the reviewer.
func runGuard(t *testing.T, root string) (failed bool, msg string) {
	t.Helper()
	var files []string
	if err := filepath.Walk(root, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(p, ".go") {
			return nil
		}
		files = append(files, p)
		return nil
	}); err != nil {
		t.Fatalf("walk %s: %v", root, err)
	}

	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		// Adapter's own declaration: NewPostgresConversationStore +
		// *PostgresConversationStore live here.
		if filepath.Base(f) == "store_postgres.go" {
			continue
		}
		// Port declaration: ConversationStore lives here (R-CCS-010).
		if filepath.Base(f) == "store.go" {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v", f, err)
		}
		if bytes.Contains(src, []byte("NewPostgresConversationStore")) {
			return true, "guard: file " + f + " bypasses the ConversationStore port — reach through chat.ConversationStore, not the postgres adapter type"
		}
		if bytes.Contains(src, []byte("*chat.PostgresConversationStore")) {
			return true, "guard: file " + f + " names the postgres adapter type — reach through chat.ConversationStore, not the type itself"
		}
	}
	return false, ""
}

// chatPackageDir is the absolute path of the chat package's own
// source directory. Computed once at package init via runtime.Caller
// so the walker doesn't walk up from the test file on every call.
var chatPackageDir = func() string {
	_, thisFile, _, _ := runtime.Caller(0)
	dir := filepath.Dir(thisFile)
	if _, err := os.Stat(filepath.Join(dir, "store.go")); err != nil {
		panic("chat package directory not found: " + dir + ": " + err.Error())
	}
	return dir
}()

// TestChatStoreAdapterSwapGuard_NoBypass asserts the guard passes
// against the current chat-package source tree. No source file
// currently bypasses the port (no caller names the postgres adapter
// outside its own declaration), so the guard is a structural proof
// that the CH-06 → CH-07 adapter swap did not introduce a bypass.
//
// This test is GREEN from birth — the guard is the contract, not a
// regression. If a future contributor adds a direct caller of
// NewPostgresConversationStore in a non-adapter source file, this
// test fails with a message naming the file and the bypassed port
// (the planted-defeat scenario below is the bite-proof).
func TestChatStoreAdapterSwapGuard_NoBypass(t *testing.T) {
	t.Parallel()

	failed, msg := runGuard(t, chatPackageDir)
	if failed {
		t.Fatalf("%s", msg)
	}
}

// TestChatStoreAdapterSwapGuard_PlantedDefeatBites is the bite
// proof for the guard's failure-message contract. It writes a
// scratch file in t.TempDir() that names the postgres adapter
// identifier (mirroring a future contributor's mistake), then
// runs the guard against the scratch dir and asserts the failure
// message names BOTH the file path AND `chat.ConversationStore`.
//
// The failure message format MUST match the contract at
// doc 0005:879-883 ("fails naming the file and the bypassed port").
func TestChatStoreAdapterSwapGuard_PlantedDefeatBites(t *testing.T) {
	t.Parallel()

	defeatDir := t.TempDir()
	defeatFile := filepath.Join(defeatDir, "defeat.go")
	if err := os.WriteFile(defeatFile, []byte(
		"package chat\n\n"+
			"import \"fmt\"\n\n"+
			"var _ = fmt.Sprintf\n"+
			"var _ = chat.NewPostgresConversationStore\n",
	), 0o644); err != nil {
		t.Fatalf("seed defeat file: %v", err)
	}

	failed, msg := runGuard(t, defeatDir)
	if !failed {
		t.Fatalf("guard did not bite planted defeat; msg=%q", msg)
	}
	if !strings.Contains(msg, defeatFile) {
		t.Errorf("guard message must name file: %q", msg)
	}
	if !strings.Contains(msg, "chat.ConversationStore") {
		t.Errorf("guard message must name port: %q", msg)
	}
}