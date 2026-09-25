package conform

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"golang.org/x/mod/modfile"
)

// newProject creates a git repository holding files, with one commit dated in
// each of years, and returns its directory.
func newProject(t *testing.T, files map[string]string, years ...int) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	dir := t.TempDir()
	git(t, dir, "", "init", "-q", "-b", "main")
	writeFiles(t, dir, files)
	for i, y := range years {
		if i == 0 {
			git(t, dir, "", "add", "-A")
		}
		date := fmt.Sprintf("%d-06-01T12:00:00+0000", y)
		git(t, dir, date, "commit", "-q", "--allow-empty", "-m", fmt.Sprintf("chore: %d", y))
	}
	return dir
}

// git runs git in dir, with the author and committer date set when date is
// not empty, failing the test when git fails.
func git(t *testing.T, dir, date string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(cmd.Environ(),
		"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@example.com",
		"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@example.com",
		"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1")
	if date != "" {
		cmd.Env = append(cmd.Env, "GIT_AUTHOR_DATE="+date, "GIT_COMMITTER_DATE="+date)
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(bytes.TrimSpace(out))
}

// writeFiles writes each slash-separated name under dir, creating its
// directories.
func writeFiles(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, content := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// readFile returns a project file's contents, failing the test if it cannot.
func readFile(t *testing.T, dir, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(name)))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// fakeGo is a Runner that runs git for real and fakes the go commands, which
// need the network. It keeps go.mod and go.sum consistent the way go get and
// go mod tidy would, deterministically, so a second run changes nothing.
type fakeGo struct {
	t *testing.T
	// vulns is what govulncheck prints.
	vulns string
	// outputs replaces the output of a call, keyed by the call as "name args".
	outputs map[string]string
	// fail makes the failNth (from 1) call starting with it fail.
	fail    string
	failNth int
	// after runs after each call, with the call.
	after func(call string)
	// calls records every call, as "name args".
	calls []string
}

func (f *fakeGo) run(dir, name string, args ...string) ([]byte, error) {
	call := name + " " + strings.Join(args, " ")
	f.calls = append(f.calls, call)
	if f.after != nil {
		defer f.after(call)
	}
	if f.fail != "" && strings.HasPrefix(call, f.fail) {
		n := 0
		for _, c := range f.calls {
			if strings.HasPrefix(c, f.fail) {
				n++
			}
		}
		if n == max(f.failNth, 1) {
			return nil, errors.New(call + ": failed")
		}
	}
	if out, ok := f.outputs[call]; ok {
		return []byte(out), nil
	}
	if name == "git" {
		return Exec(dir, name, args...)
	}
	switch args[0] {
	case "get":
		f.get(dir, args[1:])
	case "mod":
		f.tidy(dir)
	case "run":
		return []byte(f.vulns), nil
	default:
		f.t.Fatalf("unexpected go command %q", call)
	}
	return nil, nil
}

// get sets each module@version's requirement in go.mod, taking @latest as
// v1.9.9.
func (f *fakeGo) get(dir string, mods []string) {
	path := filepath.Join(dir, "go.mod")
	data, err := os.ReadFile(path)
	if err != nil {
		f.t.Fatal(err)
	}
	mf, err := modfile.Parse("go.mod", data, nil)
	if err != nil {
		f.t.Fatal(err)
	}
	for _, m := range mods {
		mod, version, _ := strings.Cut(m, "@")
		if version == "latest" {
			version = "v1.9.9"
		}
		if err := mf.AddRequire(mod, version); err != nil {
			f.t.Fatal(err)
		}
	}
	mf.Cleanup()
	out, err := mf.Format()
	if err != nil {
		f.t.Fatal(err)
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		f.t.Fatal(err)
	}
}

// tidy writes go.sum as one line per requirement in go.mod.
func (f *fakeGo) tidy(dir string) {
	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		f.t.Fatal(err)
	}
	mf, err := modfile.Parse("go.mod", data, nil)
	if err != nil {
		f.t.Fatal(err)
	}
	var lines []string
	for _, r := range mf.Require {
		lines = append(lines, r.Mod.Path+" "+r.Mod.Version+" h1:fake=\n")
	}
	slices.Sort(lines)
	sum := strings.Join(lines, "")
	if err := os.WriteFile(filepath.Join(dir, "go.sum"), []byte(sum), 0o644); err != nil {
		f.t.Fatal(err)
	}
}

// goCalls returns the go commands f ran.
func (f *fakeGo) goCalls() []string {
	var calls []string
	for _, c := range f.calls {
		if strings.HasPrefix(c, "go ") {
			calls = append(calls, c)
		}
	}
	return calls
}

// conformWith runs conform on dir with f.
func conformWith(t *testing.T, dir string, f *fakeGo) (*Report, error) {
	t.Helper()
	f.t = t
	return Run(Options{Dir: dir, Govulncheck: "govulncheck@v1", Run: f.run})
}

// mustConform runs conform on dir with f, failing the test on an error.
func mustConform(t *testing.T, dir string, f *fakeGo) *Report {
	t.Helper()
	r, err := conformWith(t, dir, f)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// wantErr fails the test unless err contains want.
func wantErr(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Errorf("err = %v, want it to contain %q", err, want)
	}
}

// rules lists the rule and path of each finding, as "rule path".
func rules(r *Report) []string {
	var out []string
	for _, f := range r.Findings {
		out = append(out, f.Rule+" "+f.Path)
	}
	return out
}
