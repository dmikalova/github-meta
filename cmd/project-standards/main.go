// Command project-standards runs the shared, language-independent checks in
// projects that do not use Go as their task runner (ADR 0002): Markdown,
// spelling, secrets, commit messages and drift of the generated configs. Go
// projects get the same checks from the ci package's mage targets.
//
// Usage:
//
//	project-standards <command>
//
// The commands are:
//
//	check     run every check, modifying nothing
//	fix       apply the autofixes and regenerate the generated configs
//	markdown  lint Markdown
//	spell     check spelling
//	secrets   scan history, staged and unstaged changes for secrets
//	commits   lint commit messages not yet on the default branch
//	drift     check the generated configs are up to date
//	version   print the version
//
// It runs in the project root. The tools run with `go run pkg@version`, pinned
// by the ci package, so Go must be installed; the project itself needs no
// go.mod. Without one, the Go-only generated configs are neither written nor
// checked, and spelling only checks text files. CI_COMMIT_RANGE, as
// "<from>..<to>", makes commits lint exactly that range.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/dmikalova/project-standards/ci"
	"github.com/dmikalova/project-standards/internal/checks"
)

// version is the release version, set by goreleaser with -ldflags.
var version = "dev"

func main() {
	c := &checks.Checks{
		Tools:      ci.SharedTools,
		Label:      "project-standards ",
		FixCommand: "project-standards fix",
	}
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, c))
}

// A checker is the shared checks the commands run.
type checker interface {
	Markdown() error
	MarkdownFix() error
	Spell(write bool) error
	Secrets() error
	Commits() error
	Drift() error
	WriteGenerated() error
}

// A command is one check the check command runs.
type command struct {
	name string
	run  func(checker) error
}

// checkCommands are the checks, in the order the check command runs them.
var checkCommands = []command{
	{"markdown", checker.Markdown},
	{"spell", func(c checker) error { return c.Spell(false) }},
	{"secrets", checker.Secrets},
	{"commits", checker.Commits},
	{"drift", checker.Drift},
}

const usage = `Usage: project-standards <command>

Runs the shared, language-independent checks in the current directory, which
must be the project root.

Commands:
  check     run every check, modifying nothing
  fix       apply the autofixes and regenerate the generated configs
  markdown  lint Markdown with goldmark-lint
  spell     check spelling with misspell
  secrets   scan history, staged and unstaged changes with gitleaks
  commits   lint commit messages not yet on the default branch
  drift     check the generated configs are up to date
  version   print the version
`

// run runs the command args name and returns the exit status: 0 on success, 1
// when a check fails and 2 for a usage error.
func run(args []string, stdout, stderr io.Writer, c checker) int {
	fs := flag.NewFlagSet("project-standards", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { _, _ = io.WriteString(stderr, usage) }
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return 2
	}
	var err error
	switch name := fs.Arg(0); name {
	case "check":
		err = check(stdout, c)
	case "fix":
		err = fix(c)
	case "version":
		_, _ = fmt.Fprintln(stdout, version)
	case "help":
		_, _ = io.WriteString(stdout, usage)
	default:
		i := indexCommand(name)
		if i < 0 {
			_, _ = fmt.Fprintf(stderr, "project-standards: unknown command %q\n\n%s", name, usage)
			return 2
		}
		err = checkCommands[i].run(c)
	}
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "project-standards:", err)
		return 1
	}
	return 0
}

func indexCommand(name string) int {
	for i, cmd := range checkCommands {
		if cmd.name == name {
			return i
		}
	}
	return -1
}

// check runs every check, even after one fails, so one run reports every
// finding.
func check(stdout io.Writer, c checker) error {
	var failed []string
	for _, cmd := range checkCommands {
		if err := cmd.run(c); err != nil {
			_, _ = fmt.Fprintf(stdout, "project-standards %s: %v\n", cmd.name, err)
			failed = append(failed, cmd.name)
		}
	}
	if len(failed) > 0 {
		return fmt.Errorf("failed: %s", strings.Join(failed, ", "))
	}
	_, _ = fmt.Fprintln(stdout, "ALL GREEN")
	return nil
}

// fix regenerates the generated configs first, so the fixers run with them,
// then applies goldmark-lint's and misspell's fixes. Issues a fixer cannot fix
// are left for check to report.
func fix(c checker) error {
	for _, step := range []func() error{
		c.WriteGenerated,
		c.MarkdownFix,
		func() error { return c.Spell(true) },
	} {
		if err := step(); err != nil {
			return err
		}
	}
	return nil
}
