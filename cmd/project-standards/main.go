// Command project-standards runs the shared, language-independent checks in
// projects that do not use Go as their task runner (ADR 0002): Markdown,
// spelling, secrets, commit messages and drift of the generated configs. Go
// projects get the same checks from the ci package's mage targets. It also
// runs conformance (ADR 0007) in any project.
//
// Usage:
//
//	project-standards <command>
//	project-standards conform [-json]
//	project-standards changelog [-from ref] [-to ref]
//	project-standards commit-msg <file>
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
//	conform   write every standard file that drifted, and report what needs a human
//	changelog print release notes for the commits in from..to
//	commit-msg lint a commit message file, for the commit-msg hook
//	version   print the version
//
// It runs in the project root. The tools run with `go run pkg@version`, pinned
// by the ci package, so Go must be installed; the project itself needs no
// go.mod. Without one, the Go-only generated configs are neither written nor
// checked, and spelling only checks text files. CI_COMMIT_RANGE, as
// "<from>..<to>", makes commits lint exactly that range.
//
// conform exits 0 whether or not it changed anything: findings are reports, not
// failures. It exits 1 only when conform itself fails. With -json it prints
// {"changed": [paths], "findings": [{"rule", "path", "message"}]}.
//
// changelog prints Markdown release notes for the commits in from..to (from
// every commit when from is empty), grouped by Conventional Commit type. The
// cicd workflow uses it for GitHub releases and the Discord post.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/dmikalova/project-standards/ci"
	"github.com/dmikalova/project-standards/internal/changelog"
	"github.com/dmikalova/project-standards/internal/checks"
	"github.com/dmikalova/project-standards/internal/conform"
)

// version is the release version, set by goreleaser with -ldflags.
var version = "dev"

func main() {
	c := &checks.Checks{
		Tools:      ci.SharedTools,
		Label:      "project-standards ",
		FixCommand: "project-standards fix",
	}
	conformHere := func() (*conform.Report, error) {
		return conform.Run(conform.Options{Dir: ".", Govulncheck: ci.Govulncheck})
	}
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, c, conformHere, notesHere))
}

// notesHere renders the release notes for from..to in the current directory,
// linking commits when origin is a GitHub remote.
func notesHere(from, to string) (string, error) {
	git := func(args ...string) (string, error) {
		out, err := exec.Command("git", args...).Output()
		if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
			err = fmt.Errorf("%w: %s", err, string(bytes.TrimSpace(exitErr.Stderr)))
		}
		return string(out), err
	}
	commits, err := changelog.Commits(git, from, to)
	if err != nil {
		return "", err
	}
	remote, _ := git("remote", "get-url", "origin") // no remote leaves SHAs unlinked
	return changelog.Render(commits, changelog.CommitURL(remote)), nil
}

// A changeloger renders the release notes for the commits in from..to.
type changeloger func(from, to string) (string, error)

// A conformer conforms the project in the current directory.
type conformer func() (*conform.Report, error)

// A checker is the shared checks the commands run.
type checker interface {
	Markdown() error
	MarkdownFix() error
	Spell(write bool) error
	Secrets() error
	Commits() error
	CommitMessage(path string) error
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
       project-standards conform [-json]
       project-standards changelog [-from ref] [-to ref]
       project-standards commit-msg <file>

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
  conform   write every standard file that drifted, and report what needs a
            human; -json prints the report as JSON
  changelog print Markdown release notes for the commits in from..to, grouped
            by Conventional Commit type; an empty -from starts at the root
  commit-msg lint the commit message in <file> with commitlint, for lefthook's
            commit-msg hook
  version   print the version
`

// run runs the command args name and returns the exit status: 0 on success, 1
// when a check fails and 2 for a usage error.
func run(
	args []string,
	stdout, stderr io.Writer,
	c checker,
	conformHere conformer,
	notes changeloger,
) int {
	fs := flag.NewFlagSet("project-standards", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { _, _ = io.WriteString(stderr, usage) }
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if fs.NArg() == 0 || fs.NArg() > 1 && !takesArgs[fs.Arg(0)] {
		fs.Usage()
		return 2
	}
	var err error
	switch name := fs.Arg(0); name {
	case "conform":
		return runConform(fs.Args()[1:], stdout, stderr, conformHere)
	case "changelog":
		return runChangelog(fs.Args()[1:], stdout, stderr, notes)
	case "commit-msg":
		if fs.NArg() != 2 {
			fs.Usage()
			return 2
		}
		err = c.CommitMessage(fs.Arg(1))
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

// takesArgs are the commands that take arguments after their name.
var takesArgs = map[string]bool{"conform": true, "changelog": true, "commit-msg": true}

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

// runConform runs conform with its flags and prints the report. Findings do not
// fail it.
func runConform(args []string, stdout, stderr io.Writer, conformHere conformer) int {
	fs := flag.NewFlagSet("project-standards conform", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { _, _ = io.WriteString(stderr, usage) }
	asJSON := fs.Bool("json", false, "print the report as JSON")
	if err := fs.Parse(args); err != nil || fs.NArg() > 0 {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		if err == nil {
			fs.Usage()
		}
		return 2
	}
	report, err := conformHere()
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "project-standards conform:", err)
		return 1
	}
	if *asJSON {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		// A Report is strings and slices of strings, which always encode.
		_ = enc.Encode(report)
		return 0
	}
	printReport(stdout, report)
	return 0
}

// runChangelog prints the release notes for -from..-to.
func runChangelog(args []string, stdout, stderr io.Writer, notes changeloger) int {
	fs := flag.NewFlagSet("project-standards changelog", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() { _, _ = io.WriteString(stderr, usage) }
	from := fs.String("from", "", "the previous release; empty starts at the root commit")
	to := fs.String("to", "HEAD", "the release")
	if err := fs.Parse(args); err != nil || fs.NArg() > 0 {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		if err == nil {
			fs.Usage()
		}
		return 2
	}
	out, err := notes(*from, *to)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "project-standards changelog:", err)
		return 1
	}
	_, _ = io.WriteString(stdout, out)
	return 0
}

// printReport prints the files changed, then the findings.
func printReport(w io.Writer, r *conform.Report) {
	if len(r.Changed) == 0 {
		_, _ = fmt.Fprintln(w, "conform: no files changed")
	} else {
		_, _ = fmt.Fprintln(w, "conform: changed:")
		for _, f := range r.Changed {
			_, _ = fmt.Fprintln(w, "  "+f)
		}
	}
	if len(r.Findings) == 0 {
		_, _ = fmt.Fprintln(w, "conform: no findings")
		return
	}
	_, _ = fmt.Fprintln(w, "conform: findings that need a human:")
	for _, f := range r.Findings {
		_, _ = fmt.Fprintf(w, "  [%s] %s: %s\n", f.Rule, f.Path, f.Message)
	}
}
