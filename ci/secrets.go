package ci

import (
	"fmt"
	"strings"

	"github.com/magefile/mage/sh"
)

// Secrets scans history, staged and unstaged changes for secrets.
// It runs gitleaks with the generated .gitleaks.toml, redacting what it finds.
// Covering all three means a secret is caught before it is committed (the
// pre-commit hook runs ci:check) and cannot hide in an older commit. Every
// scan runs even after one finds a leak, so one run reports them all.
func Secrets() error {
	common := []string{"run", gitleaks, "git", "--no-banner", "--redact", "--verbose"}
	var failed []string
	for _, scan := range []struct {
		name  string
		flags []string
	}{
		{"history", nil}, // every commit reachable from HEAD
		{"staged", []string{"--pre-commit", "--staged"}}, // the index against HEAD
		{"unstaged", []string{"--pre-commit"}},           // the working tree against the index
	} {
		args := append(append(append([]string{}, common...), scan.flags...), ".")
		if err := sh.RunV("go", args...); err != nil {
			failed = append(failed, scan.name)
		}
	}
	if len(failed) > 0 {
		return fmt.Errorf("gitleaks found secrets or failed in: %s", strings.Join(failed, ", "))
	}
	return nil
}
