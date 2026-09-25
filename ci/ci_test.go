package ci

import (
	"bytes"
	"go/ast"
	"go/doc"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
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
