package ci

import (
	"bytes"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestModulePath(t *testing.T) {
	tests := []struct{ gomod, want string }{
		{"module github.com/dmikalova/vex\n\ngo 1.27\n", "github.com/dmikalova/vex"},
		{"// comment\nmodule \"example.com/q\" // trailing\n", "example.com/q"},
		{"module example.com/c // comment\n", "example.com/c"},
		{"go 1.27\n", ""},
		{"module\n", ""},
	}
	for _, tt := range tests {
		if got := modulePath([]byte(tt.gomod)); got != tt.want {
			t.Errorf("modulePath(%q) = %q, want %q", tt.gomod, got, tt.want)
		}
	}
}

func TestTidyTestOutput(t *testing.T) {
	setModulePrefix()
	modulePrefix = "github.com/dmikalova/project-standards/"
	raw := "ok  \tgithub.com/dmikalova/project-standards/internal/config\t0.4s\tcoverage: 100.0% of statements\n" +
		"?   \tgithub.com/dmikalova/project-standards/cmd/project-standards\t[no test files]\n" +
		"--- FAIL: TestX (0.00s)\n" +
		"FAIL\tgithub.com/dmikalova/project-standards/ci\t0.1s\n"
	want := "ok  internal/config        0.4s           cov 100.0%\n" +
		"?   cmd/project-standards  no test files\n" +
		"--- FAIL: TestX (0.00s)\n" +
		"FAIL  ci  0.1s\n"
	if got := tidyTestOutput(raw); got != want {
		t.Errorf("tidyTestOutput =\n%s\nwant\n%s", got, want)
	}
}

func TestMatchGlob(t *testing.T) {
	tests := []struct {
		pattern, name string
		want          bool
	}{
		{"**/go.sum", "go.sum", true},
		{"**/go.sum", "tools/go.sum", true},
		{"**/go.sum", "go.sum.bak", false},
		{"docs/**", "docs/adr/0001.md", true},
		{"docs/**", "readme/docs.md", false},
		{"docs/*.md", "docs/a.md", true},
		{"docs/*.md", "docs/adr/a.md", false},
		{"**/testdata/**", "internal/generate/testdata/x.golden", true},
		{"LICENSE", "LICENSE", true},
		{"LICENSE", "docs/LICENSE", false},
		{"[", "[", false},
	}
	for _, tt := range tests {
		if got := matchGlob(tt.pattern, tt.name); got != tt.want {
			t.Errorf("matchGlob(%q, %q) = %v, want %v", tt.pattern, tt.name, got, tt.want)
		}
	}
}

func TestChunks(t *testing.T) {
	got := chunks([]string{"a", "b", "c", "d", "e"}, 2)
	want := [][]string{{"a", "b"}, {"c", "d"}, {"e"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("chunks = %v, want %v", got, want)
	}
	if got := chunks(nil, 2); got != nil {
		t.Errorf("chunks(nil) = %v, want nil", got)
	}
}

func TestParseCoverFunc(t *testing.T) {
	out := "a.go:1:\tF\t100.0%\na.go:9:\tG\t50.0%\ntotal:\t(statements)\t75.0%\n"
	pct, short, err := parseCoverFunc(out)
	if err != nil || int(pct) != 75 || len(short) != 1 || !strings.Contains(short[0], "G") {
		t.Errorf("parseCoverFunc = %v, %v, %v", pct, short, err)
	}
	if _, _, err := parseCoverFunc("total: x\n"); err == nil {
		t.Error("parseCoverFunc accepted a bad percent")
	}
	if _, _, err := parseCoverFunc(""); err == nil {
		t.Error("parseCoverFunc accepted an empty report")
	}
}

// TestMageVersionMatchesGoMod keeps the mage a project runs with `go run` and
// the mage library ci imports on the same version.
func TestMageVersionMatchesGoMod(t *testing.T) {
	data, err := os.ReadFile("../go.mod")
	if err != nil {
		t.Fatal(err)
	}
	_, version, _ := strings.Cut(mage, "@")
	if !bytes.Contains(data, []byte("github.com/magefile/mage "+version)) {
		t.Errorf(
			"go.mod does not require github.com/magefile/mage %s, the version in tools.go",
			version,
		)
	}
}

// TestTargetDocs checks what `mage -l` shows: every exported function is a
// target, and its first doc sentence, minus the leading name, is the synopsis
// mage prints after the padded target name. That line must fit in 80 columns.
func TestTargetDocs(t *testing.T) {
	names, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	var synopsis doc.Package
	targets := map[string]string{}
	longest := 0
	for _, name := range names {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fset, name, nil, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range file.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Recv != nil || !fn.Name.IsExported() {
				continue
			}
			targets[fn.Name.Name] = synopsis.Synopsis(fn.Doc.Text())
			longest = max(longest, len(fn.Name.Name))
		}
	}
	// mage -l prints "  ci:<name>", pads names to the longest plus four, then
	// the synopsis without its first word.
	indent := len("  ci:") + longest + 4
	for name, synopsis := range targets {
		rest, ok := strings.CutPrefix(synopsis, name+" ")
		if !ok {
			t.Errorf("target %s: doc must start with its name, got %q", name, synopsis)
			continue
		}
		if indent+len(rest) > 80 {
			t.Errorf(
				"target %s: `mage -l` line is %d columns, over 80: %q",
				name,
				indent+len(rest),
				rest,
			)
		}
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
