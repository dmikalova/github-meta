package checks

import (
	"io"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestMarkdown(t *testing.T) {
	calls := passGo(t)
	if err := testChecks.Markdown(); err != nil {
		t.Fatal(err)
	}
	if err := testChecks.MarkdownFix(); err != nil {
		t.Fatal(err)
	}
	want := []goCall{
		{args: []string{"goldmark-lint@v0", "--no-cache", "**/*.md"}},
		{args: []string{"goldmark-lint@v0", "--no-cache", "--fix", "**/*.md"}},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Errorf("calls = %v, want %v", *calls, want)
	}

	status := 1
	fakeGo(t, func(goCall, io.Writer, io.Writer) int { return status })
	wantErr(t, testChecks.Markdown(), "exit status")
	captureStdout(t, func() {
		if err := testChecks.MarkdownFix(); err != nil {
			t.Errorf("MarkdownFix with issues left = %v, want them tolerated", err)
		}
	})
	status = 2
	wantErr(t, testChecks.MarkdownFix(), "goldmark-lint failed (exit status 2)")
}

// spellRepo is a repository holding Go and text files, with a go.mod when
// goModule is set.
func spellRepo(t *testing.T, goModule bool, config string) {
	t.Helper()
	newRepo(t)
	files := map[string]string{
		"README.md":            "readme",
		"main.go":              "package main",
		"go.sum":               "excluded by the base config",
		"docs/notes.txt":       "notes",
		"mklv.config.json":     config,
		"internal/x/x_test.go": "package x",
	}
	if goModule {
		files["go.mod"] = "module example.com/x\n"
	}
	writeFiles(t, files)
}

// misspellFiles returns the files each misspell run checked, by source.
func misspellFiles(calls []goCall) map[string][]string {
	got := map[string][]string{}
	for _, c := range calls {
		source := c.args[slices.Index(c.args, "-source")+1]
		for _, a := range c.args[1:] {
			if strings.Contains(a, ".") {
				got[source] = append(got[source], a)
			}
		}
	}
	return got
}

func TestSpellGoModule(t *testing.T) {
	spellRepo(t, true, `{"tools":{"misspell":{"ignore":["colour","cancelled"],"locale":"us"}}}`)
	calls := passGo(t)
	var err error
	out := captureStdout(t, func() { err = testChecks.Spell(false) })
	if err != nil {
		t.Fatal(err)
	}
	if out != "test spell: checked 2 Go and 4 text files\n" {
		t.Errorf("output = %q", out)
	}
	want := map[string][]string{
		"go":   {"internal/x/x_test.go", "main.go"},
		"text": {"README.md", "docs/notes.txt", "go.mod", "mklv.config.json"},
	}
	if got := misspellFiles(*calls); !reflect.DeepEqual(got, want) {
		t.Errorf("files = %v, want %v", got, want)
	}
	wantFlags := []string{"-error", "-i", "colour,cancelled", "-locale", "US"}
	for _, c := range *calls {
		if !reflect.DeepEqual(c.args[3:3+len(wantFlags)], wantFlags) {
			t.Errorf("flags = %v, want %v", c.args[3:], wantFlags)
		}
	}
}

func TestSpellWithoutGoModule(t *testing.T) {
	spellRepo(t, false, `{}`)
	calls := passGo(t)
	var err error
	out := captureStdout(t, func() { err = testChecks.Spell(true) })
	if err != nil {
		t.Fatal(err)
	}
	if out != "test spell: checked 3 text files\n" {
		t.Errorf("output = %q", out)
	}
	want := map[string][]string{"text": {"README.md", "docs/notes.txt", "mklv.config.json"}}
	if got := misspellFiles(*calls); !reflect.DeepEqual(got, want) {
		t.Errorf("files = %v, want %v", got, want)
	}
	if len(*calls) != 1 || !slices.Contains((*calls)[0].args, "-w") {
		t.Errorf("calls = %v, want one run with -w", *calls)
	}
}

func TestSpellFindings(t *testing.T) {
	spellRepo(t, true, `{}`)
	calls := fakeGo(t, func(goCall, io.Writer, io.Writer) int { return 2 })
	wantErr(
		t,
		testChecks.Spell(false),
		"misspell found misspellings (run test fix to correct them)",
	)
	if len(*calls) != 2 {
		t.Errorf("misspell ran %d times, want both sources checked", len(*calls))
	}
	// When writing, exit status 2 is a failure, not a finding.
	wantErr(t, testChecks.Spell(true), "misspell -source go failed (exit status 2)")
}

func TestSpellErrors(t *testing.T) {
	spellRepo(t, true, `{"kind":"nope"}`)
	wantErr(t, testChecks.Spell(false), `kind "nope"`)

	writeFiles(t, map[string]string{"mklv.config.json": `{"tools":{"misspell":{"locale":"AU"}}}`})
	wantErr(t, testChecks.Spell(false), `"AU" is not US or UK`)

	t.Chdir(t.TempDir())
	t.Setenv("GIT_CEILING_DIRECTORIES", t.TempDir())
	wantErr(t, testChecks.Spell(false), "git ls-files")
}
