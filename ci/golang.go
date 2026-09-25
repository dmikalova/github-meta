package ci

import (
	"context"
	"fmt"
	"strings"

	"github.com/magefile/mage/sh"
)

// Format fails if a formatter or go fix would change a Go file.
// The formatters are golines (gofmt plus long-line shortening) and gci. It
// writes nothing: each tool lists what it would change, and ci:fix applies the
// changes.
func Format() error {
	golOut, err := sh.Output("go", golinesArgs("-l", ".")...)
	if err != nil {
		return err
	}
	gciOut, err := sh.Output("go", gciArgs("list")...)
	if err != nil {
		return err
	}
	if out := strings.TrimSpace(golOut + "\n" + gciOut); out != "" {
		return fmt.Errorf("formatting needed (run mage ci:fix):\n%s", out)
	}
	// go fix -diff prints the patch and exits non-zero when a fix applies.
	if err := sh.RunV("go", "fix", "-diff", "./..."); err != nil {
		return fmt.Errorf("go fix would change files (run mage ci:fix): %w", err)
	}
	return nil
}

// Tidy fails if go mod tidy would change go.mod or go.sum.
func Tidy() error {
	return sh.RunV("go", "mod", "tidy", "-diff")
}

// Build builds every package, then runs the project's ExtraBuilds.
func Build(ctx context.Context) error {
	if err := sh.RunV("go", "build", "./..."); err != nil {
		return err
	}
	for _, build := range ExtraBuilds {
		if err := build(ctx); err != nil {
			return err
		}
	}
	return nil
}

// Vet runs go vet over every package.
func Vet() error {
	return sh.RunV("go", "vet", "./...")
}

// Lint runs golangci-lint, fixing nothing.
// It reads the generated .golangci.yaml.
func Lint() error {
	return sh.RunV("go", "run", golangciLint, "run")
}

// golinesArgs builds a `go run golines` invocation. gofmt is pinned as the base
// formatter so the result is identical whether or not goimports is on PATH.
func golinesArgs(extra ...string) []string {
	return append([]string{"run", golines, "--base-formatter=gofmt"}, extra...)
}

// gciArgs builds a `go run gci <sub>` invocation grouping imports as standard,
// third-party, then local module. localmodule reads the path from go.mod.
func gciArgs(sub string) []string {
	return []string{
		"run", gci, sub,
		"--skip-generated",
		"-s", "standard", "-s", "default", "-s", "localmodule", "--custom-order",
		".",
	}
}
