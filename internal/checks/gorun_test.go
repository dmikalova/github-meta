package checks

import (
	"bytes"
	"errors"
	"io"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestGoRun(t *testing.T) {
	fakeGo(t, func(c goCall, _, _ io.Writer) int {
		switch c.tool() {
		case "pass":
			return 0
		case "findings":
			return 2
		default:
			return -1
		}
	})
	tests := []struct {
		tool    string
		status  int
		wantErr bool
	}{
		{"pass", 0, false},
		{"findings", 2, true},
		{"broken", -1, true},
	}
	for _, tt := range tests {
		status, err := GoRun(tt.tool, "arg")
		if status != tt.status || (err != nil) != tt.wantErr {
			t.Errorf("GoRun(%s) = %d, %v; want %d, error %v", tt.tool, status, err,
				tt.status, tt.wantErr)
		}
	}
}

func TestGoRunCombinedOutput(t *testing.T) {
	fakeGo(t, func(c goCall, stdout, stderr io.Writer) int {
		if stdout != stderr {
			t.Error("stdout and stderr are different writers for a combined output")
		}
		_, _ = io.WriteString(stdout, "out:"+c.stdin)
		return 0
	})
	var out bytes.Buffer
	if _, err := goRun(strings.NewReader("in"), &out, &out, "tool"); err != nil {
		t.Fatal(err)
	}
	if out.String() != "out:in" {
		t.Errorf("output = %q, want %q", out.String(), "out:in")
	}
}

func TestProgramStatus(t *testing.T) {
	tests := []struct {
		tail string
		code int
		ok   bool
	}{
		{"x.go:1: found\nexit status 2\n", 2, true},
		{"level=error msg=bad config\nexit status 3\n", 3, true},
		{"exit status 1", 1, true},
		{"go: downloading failed\n", 0, false},
		{"exit status 2\nmore output\n", 0, false},
	}
	for _, tt := range tests {
		code, ok := programStatus([]byte(tt.tail))
		if code != tt.code || ok != tt.ok {
			t.Errorf("programStatus(%q) = %d, %v; want %d, %v", tt.tail, code, ok, tt.code, tt.ok)
		}
	}
}

func TestTailWriter(t *testing.T) {
	var sink bytes.Buffer
	w := &tailWriter{w: &sink}
	long := strings.Repeat("a", 300)
	if _, err := io.WriteString(w, long+"\nexit status 4\n"); err != nil {
		t.Fatal(err)
	}
	if sink.Len() != 300+len("\nexit status 4\n") {
		t.Errorf("tailWriter dropped output: %d bytes", sink.Len())
	}
	if code, ok := programStatus(w.bytes()); !ok || code != 4 {
		t.Errorf("programStatus(tail) = %d, %v; want 4", code, ok)
	}
	if len(w.bytes()) != tailSize {
		t.Errorf("tail is %d bytes, want %d", len(w.bytes()), tailSize)
	}
}

func TestTolerateFindings(t *testing.T) {
	fail := errors.New("exit status 1")
	var err error
	out := captureStdout(t, func() { err = testChecks.TolerateFindings("lint", 1, 1, fail) })
	if err != nil || out != "test fix: lint left issues it cannot fix; the check reports them\n" {
		t.Errorf("findings: err = %v, output %q", err, out)
	}
	wantErr(t, testChecks.TolerateFindings("lint", 1, 3, fail), "lint failed (exit status 3)")
	if err := testChecks.TolerateFindings("lint", 1, 0, nil); err != nil {
		t.Errorf("success: err = %v", err)
	}
}

func TestGoModule(t *testing.T) {
	t.Chdir(t.TempDir())
	if GoModule() {
		t.Error("GoModule() = true without a go.mod")
	}
	if err := os.WriteFile("go.mod", []byte("module x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !GoModule() {
		t.Error("GoModule() = false with a go.mod")
	}
}

func TestSecrets(t *testing.T) {
	calls := passGo(t)
	if err := testChecks.Secrets(); err != nil {
		t.Fatal(err)
	}
	want := [][]string{
		{"gitleaks@v0", "git", "--no-banner", "--redact", "--verbose", "."},
		{
			"gitleaks@v0",
			"git",
			"--no-banner",
			"--redact",
			"--verbose",
			"--pre-commit",
			"--staged",
			".",
		},
		{"gitleaks@v0", "git", "--no-banner", "--redact", "--verbose", "--pre-commit", "."},
	}
	var got [][]string
	for _, c := range *calls {
		got = append(got, c.args)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("gitleaks runs = %v, want %v", got, want)
	}

	fakeGo(t, func(c goCall, _, _ io.Writer) int {
		if strings.Contains(strings.Join(c.args, " "), "--pre-commit") {
			return 1
		}
		return 0
	})
	wantErr(t, testChecks.Secrets(), "gitleaks found secrets or failed in: staged, unstaged")
}
