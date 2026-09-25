package ci

import (
	"context"
	"fmt"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"

	"github.com/dmikalova/project-standards/internal/generate"
)

// Fix applies every autofix and regenerates the generated configs.
// It writes the generated configs first, so the linters run with them, then
// runs go mod tidy, go fix, golangci-lint --fix, goimports (to repair the
// imports those rewrites need), the formatters, goldmark-lint --fix and
// misspell -w. Issues a fixer cannot fix are left for ci:check to
// report; Fix fails only when a tool itself fails.
func Fix() error {
	if err := writeGenerated(); err != nil {
		return err
	}
	if err := sh.RunV("go", "mod", "tidy"); err != nil {
		return err
	}
	if err := sh.RunV("go", "fix", "./..."); err != nil {
		return err
	}
	status, err := goRunStatus(golangciLint, "run", "--fix")
	if err := tolerateFindings("golangci-lint", 1, status, err); err != nil {
		return err
	}
	if err := fixImports(); err != nil {
		return err
	}
	// The formatters run after the code-changing fixers so their output is final.
	if err := sh.RunV("go", golinesArgs("-w", ".")...); err != nil {
		return err
	}
	if err := sh.RunV("go", gciArgs("write")...); err != nil {
		return err
	}
	status, err = goRunStatus(goldmarkLint, "--no-cache", "--fix", markdownGlob)
	if err := tolerateFindings("goldmark-lint", 1, status, err); err != nil {
		return err
	}
	return runMisspell(true)
}

// fixImports adds the imports golangci-lint --fix's rewrites need and drops
// ones they orphaned, over the project's Go files. gci regroups them after.
func fixImports() error {
	files, err := projectFiles(nil)
	if err != nil {
		return err
	}
	var goFiles []string
	for _, f := range files {
		if strings.HasSuffix(f, ".go") && f != generate.RuleguardPath {
			goFiles = append(goFiles, f)
		}
	}
	for _, chunk := range chunks(goFiles, 500) {
		if err := sh.RunV("go", append([]string{"run", goimports, "-w"}, chunk...)...); err != nil {
			return err
		}
	}
	return nil
}

// tolerateFindings passes a fixer's findings status, which golangci-lint and
// goldmark-lint use for "issues remain", and returns any other failure.
func tolerateFindings(tool string, findings, status int, err error) error {
	if err != nil && status == findings {
		fmt.Printf("ci:fix: %s left issues it cannot fix; ci:check reports them\n", tool)
		return nil
	}
	if err != nil {
		return fmt.Errorf("%s failed (exit status %d): %w", tool, status, err)
	}
	return nil
}

// Check verifies what ci:fix would change and runs every check.
// It never writes a file. The independent checks run in parallel; tests and
// coverage then run after them, in order, so their reports read as two clean
// blocks rather than interleaving.
func Check(ctx context.Context) error {
	mg.CtxDeps(ctx, Format, Tidy, Build, Vet, Lint, Markdown, Spell, Secrets, Commits, Drift)
	if err := Test(); err != nil {
		return err
	}
	if err := Cover(); err != nil {
		return err
	}
	fmt.Println("ALL GREEN")
	return nil
}
