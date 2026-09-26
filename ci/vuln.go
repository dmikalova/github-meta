package ci

import "github.com/magefile/mage/sh"

// Vuln fails if govulncheck finds a vulnerability the code reaches.
// It checks the project's dependencies and the standard library it builds
// with, so a new advisory fails the next commit and gets fixed the same day.
// The weekly conformance run bumps affected modules too, as a backstop for
// projects nobody commits to (ADR 0007). It needs network access to fetch the
// vulnerability database.
func Vuln() error {
	return sh.RunV("go", "run", govulncheck, "./...")
}
