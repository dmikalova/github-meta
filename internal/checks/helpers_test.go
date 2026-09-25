package checks

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// testChecks is the Checks every test runs, with fake pins the fake go run
// recognises.
var testChecks = &Checks{
	Tools: Tools{
		Commitlint:   "commitlint@v0",
		Gitleaks:     "gitleaks@v0",
		GoldmarkLint: "goldmark-lint@v0",
		Misspell:     "misspell@v0",
	},
	Label:      "test ",
	FixCommand: "test fix",
}

// A goCall is one `go run` the code under test made.
type goCall struct {
	args  []string
	stdin string
}

// tool is the pinned tool a call ran.
func (c goCall) tool() string { return c.args[0] }

// fakeGo replaces `go run` for the rest of the test: fn gets each call and
// returns the tool's exit status, where -1 means the go command itself failed.
// Every other command, such as git, runs for real. It returns the calls made.
func fakeGo(t *testing.T, fn func(c goCall, stdout, stderr io.Writer) int) *[]goCall {
	t.Helper()
	var calls []goCall
	realRun := run
	run = func(cmd *exec.Cmd) error {
		if len(cmd.Args) < 2 || cmd.Args[0] != "go" || cmd.Args[1] != "run" {
			return realRun(cmd)
		}
		c := goCall{args: cmd.Args[2:]}
		if cmd.Stdin != nil {
			in, err := io.ReadAll(cmd.Stdin)
			if err != nil {
				t.Fatal(err)
			}
			c.stdin = string(in)
		}
		calls = append(calls, c)
		switch code := fn(c, cmd.Stdout, cmd.Stderr); code {
		case 0:
			return nil
		case -1:
			_, _ = io.WriteString(cmd.Stderr, "go: downloading failed\n")
			return errors.New("exit status 1")
		default:
			_, _ = fmt.Fprintf(cmd.Stderr, "exit status %d\n", code)
			return errors.New("exit status 1")
		}
	}
	t.Cleanup(func() { run = realRun })
	return &calls
}

// passGo fakes every `go run` as passing.
func passGo(t *testing.T) *[]goCall {
	t.Helper()
	return fakeGo(t, func(goCall, io.Writer, io.Writer) int { return 0 })
}

// newRepo moves the test into a new git repository on main and returns a
// function that runs git in it, failing the test when git fails.
func newRepo(t *testing.T) func(args ...string) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	t.Chdir(t.TempDir())
	g := func(args ...string) string {
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
	g("init", "-q", "-b", "main")
	return g
}

// writeFiles writes each name, creating its directories, with its content.
func writeFiles(t *testing.T, files map[string]string) {
	t.Helper()
	for name, content := range files {
		if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(name, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// wantErr fails the test unless err contains want.
func wantErr(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("err = %v, want it to contain %q", err, want)
	}
}

// captureStdout returns what fn prints to os.Stdout.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	saved := os.Stdout
	os.Stdout = w
	done := make(chan []byte)
	go func() {
		out, _ := io.ReadAll(r)
		done <- out
	}()
	defer func() {
		os.Stdout = saved
	}()
	fn()
	_ = w.Close()
	return string(<-done)
}
