//go:build tools
// +build tools

package tools

import (
	// These imports are used by the go.mod file to track tool dependencies
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "golang.org/x/tools/cmd/goimports"
)
