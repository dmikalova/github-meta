package main

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/dmikalova/project-standards/internal/actions"
	"github.com/dmikalova/project-standards/internal/conform"
)

// noUpdate is an update-actions that must not run.
func noUpdate() (*actions.Report, error) {
	return nil, errors.New("update-actions ran")
}

// fakeChecker records the checks run and fails those named in fail.
type fakeChecker struct {
	ran  []string
	fail map[string]bool
}

func (f *fakeChecker) do(name string) error {
	f.ran = append(f.ran, name)
	if f.fail[name] {
		return errors.New(name + " failed")
	}
	return nil
}

func (f *fakeChecker) Markdown() error                 { return f.do("markdown") }
func (f *fakeChecker) MarkdownFix() error              { return f.do("markdown-fix") }
func (f *fakeChecker) Secrets() error                  { return f.do("secrets") }
func (f *fakeChecker) Commits() error                  { return f.do("commits") }
func (f *fakeChecker) CommitMessage(path string) error { return f.do("commit-msg:" + path) }
func (f *fakeChecker) Drift() error                    { return f.do("drift") }

func (f *fakeChecker) WriteGenerated() error { return f.do("write-generated") }

func (f *fakeChecker) Spell(write bool) error {
	if write {
		return f.do("spell-fix")
	}
	return f.do("spell")
}

func TestRun(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		fail   string
		status int
		ran    []string
		stdout string
		stderr string
	}{
		{
			name:   "check runs every check",
			args:   []string{"check"},
			ran:    []string{"markdown", "spell", "secrets", "commits", "drift"},
			stdout: "ALL GREEN\n",
		},
		{
			name:   "check keeps going after a failure",
			args:   []string{"check"},
			fail:   "spell",
			status: 1,
			ran:    []string{"markdown", "spell", "secrets", "commits", "drift"},
			stdout: "project-standards spell: spell failed\n",
			stderr: "project-standards: failed: spell\n",
		},
		{
			name: "fix regenerates, then fixes",
			args: []string{"fix"},
			ran:  []string{"write-generated", "markdown-fix", "spell-fix"},
		},
		{
			name:   "fix stops at a failure",
			args:   []string{"fix"},
			fail:   "markdown-fix",
			status: 1,
			ran:    []string{"write-generated", "markdown-fix"},
			stderr: "project-standards: markdown-fix failed\n",
		},
		{name: "one check", args: []string{"commits"}, ran: []string{"commits"}},
		{
			name:   "one failing check",
			args:   []string{"drift"},
			fail:   "drift",
			status: 1,
			ran:    []string{"drift"},
			stderr: "project-standards: drift failed\n",
		},
		{name: "version", args: []string{"version"}, stdout: "dev\n"},
		{name: "help", args: []string{"help"}, stdout: usage},
		{name: "-h", args: []string{"-h"}, stderr: usage},
		{name: "no command", status: 2, stderr: usage},
		{name: "two commands", args: []string{"check", "fix"}, status: 2, stderr: usage},
		{name: "unknown flag", args: []string{"-x"}, status: 2},
		{
			name:   "unknown command",
			args:   []string{"lint"},
			status: 2,
			stderr: "project-standards: unknown command \"lint\"\n\n" + usage,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &fakeChecker{fail: map[string]bool{tt.fail: true}}
			var stdout, stderr bytes.Buffer
			noConform := func() (*conform.Report, error) {
				t.Error("conform ran")
				return nil, nil
			}
			noNotes := func(string, string) (string, error) {
				t.Error("changelog ran")
				return "", nil
			}
			if got := run(
				tt.args,
				&stdout,
				&stderr,
				c,
				noConform,
				noUpdate,
				noNotes,
			); got != tt.status {
				t.Errorf("status = %d, want %d", got, tt.status)
			}
			if !reflect.DeepEqual(c.ran, tt.ran) {
				t.Errorf("ran %v, want %v", c.ran, tt.ran)
			}
			if stdout.String() != tt.stdout {
				t.Errorf("stdout = %q, want %q", stdout.String(), tt.stdout)
			}
			if tt.stderr != "" && stderr.String() != tt.stderr {
				t.Errorf("stderr = %q, want %q", stderr.String(), tt.stderr)
			}
		})
	}
}

// TestUsageListsEveryCheck keeps the usage text in step with the checks.
func TestUsageListsEveryCheck(t *testing.T) {
	for _, cmd := range checkCommands {
		if !strings.Contains(usage, "\n  "+cmd.name+" ") {
			t.Errorf("usage does not list %s", cmd.name)
		}
	}
}

func TestConform(t *testing.T) {
	report := &conform.Report{
		Changed: []string{".gitignore", "LICENSE"},
		Findings: []conform.Finding{
			{Rule: "readme", Path: "README.md", Message: "missing"},
		},
	}
	empty := &conform.Report{Changed: []string{}, Findings: []conform.Finding{}}
	tests := []struct {
		name   string
		args   []string
		report *conform.Report
		err    error
		status int
		stdout string
		stderr string
	}{
		{
			name:   "text report",
			args:   []string{"conform"},
			report: report,
			stdout: "conform: changed:\n  .gitignore\n  LICENSE\n" +
				"conform: findings that need a human:\n  [readme] README.md: missing\n",
		},
		{
			name:   "nothing to do",
			args:   []string{"conform"},
			report: empty,
			stdout: "conform: no files changed\nconform: no findings\n",
		},
		{
			name:   "json report",
			args:   []string{"conform", "-json"},
			report: report,
			stdout: `{
  "changed": [
    ".gitignore",
    "LICENSE"
  ],
  "findings": [
    {
      "rule": "readme",
      "path": "README.md",
      "message": "missing"
    }
  ]
}
`,
		},
		{
			name:   "empty json report",
			args:   []string{"conform", "-json"},
			report: empty,
			stdout: "{\n  \"changed\": [],\n  \"findings\": []\n}\n",
		},
		{
			name:   "failure",
			args:   []string{"conform"},
			err:    errors.New("git log failed"),
			status: 1,
			stderr: "project-standards conform: git log failed\n",
		},
		{name: "-h", args: []string{"conform", "-h"}, stderr: usage},
		{name: "unknown flag", args: []string{"conform", "-x"}, status: 2},
		{name: "extra argument", args: []string{"conform", "x"}, status: 2, stderr: usage},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			conformHere := func() (*conform.Report, error) { return tt.report, tt.err }
			if got := run(
				tt.args,
				&stdout,
				&stderr,
				&fakeChecker{},
				conformHere,
				noUpdate,
				func(string, string) (string, error) { return "", nil },
			); got != tt.status {
				t.Errorf("status = %d, want %d", got, tt.status)
			}
			if stdout.String() != tt.stdout {
				t.Errorf("stdout = %q, want %q", stdout.String(), tt.stdout)
			}
			if tt.stderr != "" && stderr.String() != tt.stderr {
				t.Errorf("stderr = %q, want %q", stderr.String(), tt.stderr)
			}
		})
	}
}

func TestChangelog(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		notes    string
		err      error
		status   int
		stdout   string
		stderr   string
		wantFrom string
		wantTo   string
	}{
		{
			name:     "defaults",
			args:     []string{"changelog"},
			notes:    "### Features\n",
			stdout:   "### Features\n",
			wantFrom: "",
			wantTo:   "HEAD",
		},
		{
			name:     "range",
			args:     []string{"changelog", "-from", "v1.2.3", "-to", "abc"},
			notes:    "No changes.\n",
			stdout:   "No changes.\n",
			wantFrom: "v1.2.3",
			wantTo:   "abc",
		},
		{
			name:     "failure",
			args:     []string{"changelog", "-from", "nope"},
			err:      errors.New("bad revision"),
			status:   1,
			stderr:   "project-standards changelog: bad revision\n",
			wantFrom: "nope",
			wantTo:   "HEAD",
		},
		{name: "-h", args: []string{"changelog", "-h"}, stderr: usage},
		{name: "unknown flag", args: []string{"changelog", "-x"}, status: 2},
		{name: "extra argument", args: []string{"changelog", "x"}, status: 2, stderr: usage},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			var gotFrom, gotTo string
			notes := func(from, to string) (string, error) {
				gotFrom, gotTo = from, to
				return tt.notes, tt.err
			}
			noConform := func() (*conform.Report, error) {
				t.Error("conform ran")
				return nil, nil
			}
			if got := run(
				tt.args,
				&stdout,
				&stderr,
				&fakeChecker{},
				noConform,
				noUpdate,
				notes,
			); got != tt.status {
				t.Errorf("status = %d, want %d", got, tt.status)
			}
			if stdout.String() != tt.stdout {
				t.Errorf("stdout = %q, want %q", stdout.String(), tt.stdout)
			}
			if tt.stderr != "" && stderr.String() != tt.stderr {
				t.Errorf("stderr = %q, want %q", stderr.String(), tt.stderr)
			}
			if gotFrom != tt.wantFrom || gotTo != tt.wantTo {
				t.Errorf(
					"notes(%q, %q), want notes(%q, %q)",
					gotFrom,
					gotTo,
					tt.wantFrom,
					tt.wantTo,
				)
			}
		})
	}
}

func TestCommitMsg(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		fail   string
		status int
		ran    []string
	}{
		{
			name: "lints the file",
			args: []string{"commit-msg", ".git/COMMIT_EDITMSG"},
			ran:  []string{"commit-msg:.git/COMMIT_EDITMSG"},
		},
		{
			name:   "failure",
			args:   []string{"commit-msg", "msg"},
			fail:   "commit-msg:msg",
			status: 1,
			ran:    []string{"commit-msg:msg"},
		},
		{name: "no file", args: []string{"commit-msg"}, status: 2},
		{name: "two files", args: []string{"commit-msg", "a", "b"}, status: 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &fakeChecker{fail: map[string]bool{tt.fail: true}}
			var stdout, stderr bytes.Buffer
			noConform := func() (*conform.Report, error) { return nil, nil }
			noNotes := func(string, string) (string, error) { return "", nil }
			if got := run(
				tt.args,
				&stdout,
				&stderr,
				c,
				noConform,
				noUpdate,
				noNotes,
			); got != tt.status {
				t.Errorf("status = %d, want %d", got, tt.status)
			}
			if !reflect.DeepEqual(c.ran, tt.ran) {
				t.Errorf("ran %v, want %v", c.ran, tt.ran)
			}
		})
	}
}

func TestUpdateActions(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		report *actions.Report
		err    error
		status int
		stdout string
		stderr string
	}{
		{
			name: "updates and findings",
			args: []string{"update-actions"},
			report: &actions.Report{
				Changed: []string{".github/workflows/a.yaml"},
				Updates: []string{"actions/cache v4 → v5"},
				Findings: []conform.Finding{
					{
						Rule:    "actions",
						Path:    ".github/workflows/a.yaml",
						Message: "a/b: no releases",
					},
				},
			},
			stdout: "update-actions: updated:\n  actions/cache v4 → v5\n" +
				"  [actions] .github/workflows/a.yaml: a/b: no releases\n",
		},
		{
			name:   "current",
			args:   []string{"update-actions"},
			report: &actions.Report{},
			stdout: "update-actions: every action is current\n",
		},
		{
			name: "json",
			args: []string{"update-actions", "-json"},
			report: &actions.Report{
				Changed:  []string{},
				Updates:  []string{"x"},
				Findings: []conform.Finding{},
			},
			stdout: "{\n  \"changed\": [],\n  \"updates\": [\n    \"x\"\n  ],\n  \"findings\": []\n}\n",
		},
		{
			name:   "failure",
			args:   []string{"update-actions"},
			err:    errors.New("boom"),
			status: 1,
			stderr: "project-standards update-actions: boom\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if got := run(
				tt.args,
				&stdout,
				&stderr,
				&fakeChecker{},
				func() (*conform.Report, error) { return nil, errors.New("conform ran") },
				func() (*actions.Report, error) { return tt.report, tt.err },
				func(string, string) (string, error) { return "", nil },
			); got != tt.status {
				t.Errorf("status = %d, want %d", got, tt.status)
			}
			if stdout.String() != tt.stdout {
				t.Errorf("stdout = %q, want %q", stdout.String(), tt.stdout)
			}
			if tt.stderr != "" && stderr.String() != tt.stderr {
				t.Errorf("stderr = %q, want %q", stderr.String(), tt.stderr)
			}
		})
	}
}
