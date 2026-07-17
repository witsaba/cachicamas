package ai_test

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
)

// forbiddenPrefix is the import-path prefix this package must NEVER
// reach across. AI-03 follows ADR 0004's strict dependency rule:
// Layer 1 (cachicamas_ai) may import only the Go standard library and
// vendor SDKs.
const forbiddenPrefix = "github.com/cachicamas/backend/database_administrator/src/"

// allowedImports lists import paths inside forbiddenPrefix that the
// package itself is allowed to use (currently only the package under test).
var allowedImports = map[string]bool{
	"github.com/cachicamas/backend/database_administrator/src/tools/agent/ai": true,
}

// TestPackageImportBoundary fails if the package imports any cachicamas
// path outside its own subtree. Run via `make test` in the
// database_administrator module.
func TestPackageImportBoundary(t *testing.T) {
	cmd := exec.Command("go", "list", "-f", "{{range .Imports}}{{.}}\n{{end}}", ".")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		t.Fatalf("go list failed (is the test running from the package directory?): %v", err)
	}

	for _, imp := range strings.Split(stdout.String(), "\n") {
		if imp == "" {
			continue
		}
		if strings.HasPrefix(imp, forbiddenPrefix) && !allowedImports[imp] {
			t.Errorf("forbidden import %q crosses the Layer 1 boundary (see ADR 0004 and AI-00 vocabulary)", imp)
		}
	}
}
