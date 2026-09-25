package main

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"
)

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

func (f *fakeChecker) Markdown() error    { return f.do("markdown") }
func (f *fakeChecker) MarkdownFix() error { return f.do("markdown-fix") }
func (f *fakeChecker) Secrets() error     { return f.do("secrets") }
func (f *fakeChecker) Commits() error     { return f.do("commits") }
func (f *fakeChecker) Drift() error       { return f.do("drift") }

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
			if got := run(tt.args, &stdout, &stderr, c); got != tt.status {
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
