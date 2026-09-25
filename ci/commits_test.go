package ci

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
)

func TestParseCommitRange(t *testing.T) {
	const zero = "0000000000000000000000000000000000000000"
	tests := []struct {
		in, from, to string
		wantErr      bool
	}{
		{in: "abc123..def456", from: "abc123", to: "def456"},
		{in: "  abc..def\n", from: "abc", to: "def"},
		{in: "origin/main..HEAD", from: "origin/main", to: "HEAD"},
		{in: zero + "..def456", from: "", to: "def456"},
		{in: strings.Repeat("0", 64) + "..def", from: "", to: "def"}, // SHA-256 repositories
		{in: "0..def", from: "", to: "def"},
		{in: "0a0..def", from: "0a0", to: "def"},
		{in: "abc", wantErr: true},
		{in: "abc..", wantErr: true},
		{in: "..def", wantErr: true},
		{in: "..", wantErr: true},
		{in: "abc...def", wantErr: true},
		{in: "a..b..c", wantErr: true},
	}
	for _, tt := range tests {
		from, to, err := parseCommitRange(tt.in)
		if tt.wantErr {
			if err == nil {
				t.Errorf("parseCommitRange(%q) = %q, %q; want an error", tt.in, from, to)
			}
			continue
		}
		if err != nil || from != tt.from || to != tt.to {
			t.Errorf("parseCommitRange(%q) = %q, %q, %v; want %q, %q", tt.in, from, to, err,
				tt.from, tt.to)
		}
	}
}

// TestCommitRange resolves ranges in a real repository: main with one commit,
// and a feature branch with two more.
func TestCommitRange(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	t.Chdir(t.TempDir())
	run := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Env = append(cmd.Environ(),
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@example.com",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@example.com",
			"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return string(bytes.TrimSpace(out))
	}
	run("init", "-q", "-b", "main")
	run("commit", "-q", "--allow-empty", "-m", "feat: base")
	mainSHA := run("rev-parse", "HEAD")
	run("checkout", "-q", "-b", "feature")
	run("commit", "-q", "--allow-empty", "-m", "feat: one")
	oneSHA := run("rev-parse", "HEAD")
	run("commit", "-q", "--allow-empty", "-m", "feat: two")
	twoSHA := run("rev-parse", "HEAD")

	tests := []struct {
		name, env, want string
		wantErr         bool
	}{
		{name: "unset lints since the merge-base", env: "", want: mainSHA + "..HEAD"},
		{
			name: "explicit range is used as is",
			env:  oneSHA + ".." + twoSHA,
			want: oneSHA + ".." + twoSHA,
		},
		{
			name: "zero from lints from the merge-base",
			env:  strings.Repeat("0", 40) + ".." + twoSHA,
			want: mainSHA + ".." + twoSHA,
		},
		{name: "malformed range is an error", env: "nonsense", wantErr: true},
		{name: "zero from with unknown to is an error", env: "0000..nosuchref", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := commitRange(tt.env)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("commitRange(%q) = %q; want an error", tt.env, got)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Errorf("commitRange(%q) = %q, %v; want %q", tt.env, got, err, tt.want)
			}
		})
	}
}

func TestCommitRangeWithoutDefaultBranch(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	t.Chdir(t.TempDir())
	if out, err := exec.Command("git", "init", "-q", "-b", "trunk").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	// Unset, no default branch is a clean skip.
	if got, err := commitRange(""); err != nil || got != "" {
		t.Errorf(`commitRange("") = %q, %v; want a skip`, got, err)
	}
	// Set with a zero from, the missing default branch is an error.
	if _, err := commitRange("0000..HEAD"); err == nil {
		t.Error("commitRange with a zero from and no default branch succeeded")
	}
}
