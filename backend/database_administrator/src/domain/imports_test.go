// Package domain_test — architectural invariant test.
//
// Spec S-PR-X5: the domain package MUST NOT import github.com/jackc/pgx.
// pgx stays inside infrastructure/postgres/. This test shells out to
// `go list -deps` and asserts the dependency graph is clean.
package domain_test

import (
	"os/exec"
	"strings"
	"testing"
)

func TestDomainLayer_DoesNotImportPgx(t *testing.T) {
	t.Parallel()
	out, err := runGoListDeps("github.com/cachicamas/backend/database_administrator/src/domain")
	if err != nil {
		t.Fatalf("go list -deps failed: %v", err)
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "github.com/jackc/pgx") {
			t.Errorf("domain package must not import pgx; found: %q", line)
		}
	}
}

// runGoListDeps runs `go list -deps` on the given import path and
// returns stdout. The helper is in the test file (not in a regular
// .go file) because no production code path should depend on it.
func runGoListDeps(importPath string) (string, error) {
	cmd := exec.Command("go", "list", "-deps", importPath)
	var stdout strings.Builder
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return stdout.String(), nil
}
