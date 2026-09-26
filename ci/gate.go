package ci

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"

	"github.com/dmikalova/project-standards/internal/checks"
	"github.com/dmikalova/project-standards/internal/generate"
)

// Fix applies every autofix and regenerates the generated configs.
// It writes the generated configs first, so the linters run with them, then
// runs go mod tidy, go fix, golangci-lint --fix, goimports (to repair the
// imports those rewrites need), the formatters, goldmark-lint --fix and
// misspell -w. Issues a fixer cannot fix are left for ci:check to
// report; Fix fails only when a tool itself fails.
func Fix() error {
	if err := shared.WriteGenerated(); err != nil {
		return err
	}
	if err := sh.RunV("go", "mod", "tidy"); err != nil {
		return err
	}
	if err := sh.RunV("go", "fix", "./..."); err != nil {
		return err
	}
	status, err := checks.GoRun(golangciLint, "run", "--fix")
	if err := shared.TolerateFindings("golangci-lint", 1, status, err); err != nil {
		return err
	}
	if err := fixImports(); err != nil {
		return err
	}
	// The formatters run after the code-changing fixers so their output is final.
	files, err := goFiles()
	if err != nil {
		return err
	}
	for _, chunk := range checks.Chunks(files, 500) {
		if err := settleGolines(chunk); err != nil {
			return err
		}
		if err := sh.RunV("go", gciArgs("write", chunk...)...); err != nil {
			return err
		}
	}
	if err := shared.MarkdownFix(); err != nil {
		return err
	}
	return shared.Spell(true)
}

// maxGolinesPasses caps how often settleGolines reruns golines, so a formatter
// that never settles fails the fix instead of hanging it.
const maxGolinesPasses = 5

// settleGolines runs golines over files until it has nothing left to change.
// It can split only part of a long expression, such as a chain of ||, in one
// pass, so a single run leaves work that ci:check then reports as unformatted.
func settleGolines(files []string) error {
	for range maxGolinesPasses {
		out, err := sh.Output("go", golinesArgs(append([]string{"-l"}, files...)...)...)
		if err != nil {
			return err
		}
		pending := strings.Fields(out)
		if len(pending) == 0 {
			return nil
		}
		if err := sh.RunV("go", golinesArgs(append([]string{"-w"}, pending...)...)...); err != nil {
			return err
		}
	}
	return fmt.Errorf("golines still changes files after %d passes", maxGolinesPasses)
}

// fixImports adds the imports golangci-lint --fix's rewrites need and drops
// ones they orphaned, over the project's Go files. gci regroups them after.
func fixImports() error {
	files, err := goFiles()
	if err != nil {
		return err
	}
	files = slices.DeleteFunc(files, func(f string) bool { return f == generate.RuleguardPath })
	for _, chunk := range checks.Chunks(files, 500) {
		if err := sh.RunV("go", append([]string{"run", goimports, "-w"}, chunk...)...); err != nil {
			return err
		}
	}
	return nil
}

// Check verifies what ci:fix would change and runs every check.
// It never writes a file. The independent checks run in parallel; tests and
// coverage then run after them, in order, so their reports read as two clean
// blocks rather than interleaving.
func Check(ctx context.Context) error {
	mg.CtxDeps(ctx, Format, Tidy, Build, Vet, Lint, Markdown, Spell, Secrets, Commits, Drift, Vuln)
	if err := Test(); err != nil {
		return err
	}
	if err := Cover(); err != nil {
		return err
	}
	fmt.Println("ALL GREEN")
	return nil
}
