//go:build ignore
// +build ignore

// Command task provides a simple task runner for Go projects.
// Run it with: go run task.go [command]
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// config contains project-wide variables
var config = struct {
	ProjectName string
	BinaryName  string
	BuildDir    string
	MainPackage string
	Verbose     bool
}{
	ProjectName: "mcpterm-go",
	BinaryName:  "mcpterm",
	BuildDir:    "build",
	MainPackage: ".",
	Verbose:     true,
}

// commands maps command names to their implementations
var commands = map[string]func(){
	"build":     cmdBuild,
	"run":       cmdRun,
	"test":      cmdTest,
	"lint":      cmdLint,
	"fmt":       cmdFmt,
	"clean":     cmdClean,
	"all":       cmdAll,
	"help":      cmdHelp,
	"tools":     cmdTools,
	"fmt-check": cmdFmtCheck,
}

func main() {
	if len(os.Args) < 2 {
		cmdBuild()
		return
	}

	cmd := os.Args[1]
	if fn, ok := commands[cmd]; ok {
		fn()
	} else {
		fmt.Printf("Unknown command: %s\n", cmd)
		cmdHelp()
		os.Exit(1)
	}
}

// Utility functions
func run(name string, args ...string) {
	if config.Verbose {
		fmt.Printf("Running: %s %s\n", name, strings.Join(args, " "))
	}
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		fmt.Printf("Command failed: %s %s\n", name, strings.Join(args, " "))
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

func runGo(args ...string) {
	run("go", args...)
}

// Command implementations
func cmdBuild() {
	fmt.Printf("Building %s...\n", config.ProjectName)
	runGo("build", "-v", "-o", config.BinaryName, config.MainPackage)
}

func cmdRun() {
	cmdBuild()
	fmt.Printf("Running %s...\n", config.ProjectName)
	runGo("run", ".")
}

func cmdTest() {
	fmt.Println("Running tests...")
	runGo("test", "-v", "./...")
}

func cmdLint() {
	// Check if golangci-lint is installed
	_, err := exec.LookPath("golangci-lint")
	if err == nil {
		fmt.Println("Running golangci-lint...")
		run("golangci-lint", "run")
	} else {
		fmt.Println("golangci-lint not found, falling back to go vet...")
		runGo("vet", "./...")
		fmt.Println("For more comprehensive linting, please install golangci-lint")
	}
}

func cmdFmt() {
	fmt.Println("Formatting code...")
	// Check if goimports is available
	_, err := exec.LookPath("goimports")
	if err == nil {
		run("goimports", "-w", ".")
	} else {
		runGo("fmt", "./...")
	}
}

func cmdFmtCheck() {
	fmt.Println("Checking if code is formatted...")
	cmd := exec.Command("go", "fmt", "./...")
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("Error checking format: %v\n", err)
		os.Exit(1)
	}
	if len(output) > 0 {
		fmt.Println("Code is not formatted, run 'go run task.go fmt'")
		os.Exit(1)
	}
	fmt.Println("Code is properly formatted")
}

func cmdClean() {
	fmt.Println("Cleaning...")
	runGo("clean")
	os.Remove(config.BinaryName)
	os.RemoveAll(config.BuildDir)

	// Handle platform-specific executables
	if runtime.GOOS == "windows" {
		os.Remove(config.BinaryName + ".exe")
	}
}

func cmdAll() {
	cmdFmt()
	cmdBuild()
	cmdTest()
	cmdLint()
}

func cmdTools() {
	fmt.Println("Installing development tools...")
	runGo("install", "github.com/golangci/golangci-lint/cmd/golangci-lint@latest")
	runGo("install", "golang.org/x/tools/cmd/goimports@latest")
}

func cmdHelp() {
	fmt.Println("Available commands:")
	fmt.Println("  go run task.go           - Build the application")
	fmt.Println("  go run task.go build     - Build the application")
	fmt.Println("  go run task.go run       - Build and run the application")
	fmt.Println("  go run task.go test      - Run tests")
	fmt.Println("  go run task.go lint      - Run linters")
	fmt.Println("  go run task.go fmt       - Format code")
	fmt.Println("  go run task.go fmt-check - Check if code is formatted")
	fmt.Println("  go run task.go clean     - Clean build artifacts")
	fmt.Println("  go run task.go all       - Format, build, test, and lint")
	fmt.Println("  go run task.go tools     - Install development tools")
	fmt.Println("  go run task.go help      - Display this help message")
}
