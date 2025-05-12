#!/bin/bash
# CI helper script to fix module path issues

# Create a local go.mod for testing in CI
cat > go.mod.tmp << EOL
module github.com/navicore/mcpterm-go

go 1.23.0

require (
	github.com/spf13/cobra v1.9.1
	github.com/charmbracelet/bubbletea v1.3.5
	github.com/charmbracelet/lipgloss v1.1.1-0.20250404203927-76690c660834
	github.com/charmbracelet/bubbles v0.21.0
	github.com/charmbracelet/glamour v0.10.0
	github.com/aws/aws-sdk-go-v2 v1.36.3
	github.com/aws/aws-sdk-go-v2/config v1.29.14
	github.com/aws/aws-sdk-go-v2/service/bedrockruntime v1.30.0
	github.com/atotto/clipboard v0.1.4
)
EOL

# Use the temporary module file for CI
mv go.mod.tmp go.mod

# Tidy and download dependencies
go mod tidy
echo "CI module fix applied"