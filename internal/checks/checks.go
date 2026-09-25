// Package checks implements the shared checks every project runs whatever its
// language (ADR 0002): markdown, spelling, secrets, commit messages and drift
// of the generated configs, plus their autofixes. The ci package's mage
// targets and the project-standards binary both run them from here.
//
// Every tool is run with `go run pkg@version`. The pins are the ci package's,
// passed in as Tools, so one version list governs both callers (ADR 0004).
//
// A project is a Go project when its root holds a go.mod. Without one, the
// Go-only generated configs are neither written nor checked for drift, and
// spelling only checks text files.
package checks

import (
	"os"
	"os/exec"
)

// Tools pins the tools the shared checks run, each as a `go run`
// package@version.
type Tools struct {
	Commitlint   string
	Gitleaks     string
	GoldmarkLint string
	Misspell     string
}

// Checks runs the shared checks in the current directory, which must be the
// project root.
type Checks struct {
	// Tools pins the tools the checks run.
	Tools Tools
	// Label prefixes the name of a check in progress lines, such as "ci:" for
	// "ci:spell: checked ...".
	Label string
	// FixCommand is the command that applies the autofixes, named in errors,
	// such as "mage ci:fix".
	FixCommand string
}

// label names a check the way its caller runs it, such as "ci:spell".
func (c *Checks) label(check string) string {
	return c.Label + check
}

// GoModule reports whether the project in the current directory is a Go
// module: whether it has a go.mod at its root.
func GoModule() bool {
	_, err := os.Stat("go.mod")
	return err == nil
}

// run executes a command. Tests replace it to fake the `go run` tools.
var run = func(cmd *exec.Cmd) error { return cmd.Run() }
