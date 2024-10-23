//go:build tools

package devdeps

import (
	// required for formatting, linting, pls.
	_ "mvdan.cc/gofumpt"

	// linter
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"

	// for releases
	_ "github.com/goreleaser/goreleaser"

	_ "golang.org/x/vuln/cmd/govulncheck"
)
