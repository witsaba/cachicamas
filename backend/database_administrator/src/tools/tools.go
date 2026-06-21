// Package tools pins dependencies that are required by the build (or by
// later code in this change) but are not yet imported by any production
// file. It uses a build tag so it is not compiled into any binary — its
// sole purpose is to keep the dependency in go.mod and force a refresh
// of go.sum.
//
// This file is a standard Go pattern (used by golangci-lint and many
// other projects). It will be removed once the consuming code lands
// (i.e. once runner.go imports pressly/goose and its lock package).
//
//go:build tools
// +build tools

package tools

import (
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/pressly/goose/v3"
	_ "github.com/pressly/goose/v3/lock"
)
