//go:build tools

package devdeps

import (
	// formatting
	_ "mvdan.cc/gofumpt"

	// linter
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"

	// mocks
	_ "github.com/golang/mock/mockgen"

	// for releases
	_ "github.com/goreleaser/goreleaser"

	_ "github.com/rakyll/statik"
	_ "golang.org/x/vuln/cmd/govulncheck"
)
