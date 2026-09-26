package actions

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/dmikalova/project-standards/internal/conform"
)

// fakeResolver serves latest releases and tags from maps, failing a repo
// named in errs.
type fakeResolver struct {
	latest  map[string]string
	tags    map[string]bool
	errs    map[string]bool
	tagErrs map[string]bool
	lookups []string
}

func (f *fakeResolver) LatestRelease(repo string) (string, error) {
	f.lookups = append(f.lookups, repo)
	if f.errs[repo] {
		return "", errors.New("no releases")
	}
	return f.latest[repo], nil
}

func (f *fakeResolver) HasTag(repo, tag string) (bool, error) {
	if f.tagErrs[repo] {
		return false, errors.New("lookup failed")
	}
	return f.tags[repo+"@"+tag], nil
}

// writeTree writes files under a new temporary directory and returns it.
func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func read(t *testing.T, dir, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

const workflow = `jobs:
  a:
    steps:
      - uses: actions/checkout@v6
      - name: Upload
        uses: actions/upload-artifact@v4 # keep this comment
        with:
          name: x
      - uses: "actions/download-artifact@v4"
      - uses: ariga/setup-atlas@v0.2.1
      - uses: dmikalova/project-standards/.github/actions/setup@main
      - uses: ./.github/actions/local
      - uses: docker://alpine@sha256:0123
      - uses: google-github-actions/auth@v3
      - uses: opentofu/setup-opentofu@v1
      - uses: nonsense@v1
#     uses: actions/upload-artifact@v4
`

func TestUpdate(t *testing.T) {
	dir := writeTree(t, map[string]string{
		".github/workflows/ci.yaml":      workflow,
		".github/workflows/other.yml":    "steps:\n  - uses: actions/upload-artifact@v4\n",
		".github/workflows/same.yaml":    "steps:\n  - uses: actions/checkout@v6\n",
		".github/actions/x/action.yml":   "runs:\n  steps:\n    - uses: actions/cache@v4\n",
		".github/actions/y/action.yaml":  "runs:\n  steps:\n    - uses: actions/cache@v5\n",
		".github/workflows/notes.md":     "uses: actions/cache@v1\n",
		".github/actions/z/README.md":    "uses: actions/cache@v1\n",
		".github/workflows/sub/deep.yml": "uses: actions/cache@v1\n",
	})
	r := &fakeResolver{
		latest: map[string]string{
			"actions/checkout":           "v6.0.1",
			"actions/upload-artifact":    "v7.0.1",
			"actions/download-artifact":  "v8.0.1",
			"actions/cache":              "v5.0.0",
			"ariga/setup-atlas":          "v0.3",
			"google-github-actions/auth": "v2.9.9", // older than in use
			"opentofu/setup-opentofu":    "v2.0.0",
		},
		tags: map[string]bool{
			"actions/upload-artifact@v7":   true,
			"actions/download-artifact@v8": true,
			"actions/cache@v5":             true,
		},
	}
	report, err := Update(dir, r)
	if err != nil {
		t.Fatal(err)
	}

	want := &Report{
		Changed: []string{
			".github/actions/x/action.yml",
			".github/workflows/ci.yaml",
			".github/workflows/other.yml",
		},
		Updates: []string{
			"actions/cache v4 → v5",
			"actions/download-artifact v4 → v8",
			"actions/upload-artifact v4 → v7",
			"ariga/setup-atlas v0.2.1 → v0.3",
		},
		Findings: []conform.Finding{{
			Rule: rule, Path: ".github/workflows/ci.yaml",
			Message: "opentofu/setup-opentofu: the latest release v2.0.0 has no v2 tag to move to",
		}},
	}
	if !reflect.DeepEqual(report, want) {
		t.Errorf("report = %+v\nwant     %+v", report, want)
	}

	got := read(t, dir, ".github/workflows/ci.yaml")
	for _, line := range []string{
		"      - uses: actions/checkout@v6\n",
		"        uses: actions/upload-artifact@v7 # keep this comment\n",
		`      - uses: "actions/download-artifact@v8"` + "\n",
		"      - uses: ariga/setup-atlas@v0.3\n",
		"      - uses: dmikalova/project-standards/.github/actions/setup@main\n",
		"      - uses: google-github-actions/auth@v3\n",
		"      - uses: opentofu/setup-opentofu@v1\n",
		"#     uses: actions/upload-artifact@v4\n",
	} {
		if !strings.Contains(got, line) {
			t.Errorf("ci.yaml lacks %q:\n%s", line, got)
		}
	}
	if read(t, dir, ".github/workflows/notes.md") != "uses: actions/cache@v1\n" {
		t.Error("a non-workflow file was rewritten")
	}

	// Each repository is looked up once, however many times it is used.
	seen := map[string]int{}
	for _, repo := range r.lookups {
		seen[repo]++
		if seen[repo] > 1 {
			t.Errorf("%s looked up %d times", repo, seen[repo])
		}
	}
}

func TestUpdateFindings(t *testing.T) {
	dir := writeTree(t, map[string]string{
		".github/workflows/a.yaml": "- uses: a/fails@v1\n- uses: a/fails@v1\n" +
			"- uses: b/branch-release@v1\n- uses: c/tag-lookup@v1\n",
		".github/workflows/b.yaml": "- uses: a/fails@v1\n",
	})
	r := &fakeResolver{
		latest:  map[string]string{"b/branch-release": "nightly", "c/tag-lookup": "v2.0.0"},
		errs:    map[string]bool{"a/fails": true},
		tagErrs: map[string]bool{"c/tag-lookup": true},
	}
	report, err := Update(dir, r)
	if err != nil {
		t.Fatal(err)
	}
	want := []conform.Finding{
		{Rule: rule, Path: ".github/workflows/a.yaml", Message: "a/fails: no releases"},
		{
			Rule:    rule,
			Path:    ".github/workflows/a.yaml",
			Message: "b/branch-release: the latest release nightly is not a vX, vX.Y or vX.Y.Z version",
		},
		{Rule: rule, Path: ".github/workflows/a.yaml", Message: "c/tag-lookup: lookup failed"},
		{Rule: rule, Path: ".github/workflows/b.yaml", Message: "a/fails: no releases"},
	}
	if !reflect.DeepEqual(report.Findings, want) {
		t.Errorf("findings = %+v\nwant       %+v", report.Findings, want)
	}
	if len(report.Changed) != 0 || len(report.Updates) != 0 {
		t.Errorf("report = %+v, want nothing changed", report)
	}
}

func TestUpdateNoGitHubDir(t *testing.T) {
	report, err := Update(t.TempDir(), &fakeResolver{})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Changed)+len(report.Updates)+len(report.Findings) != 0 {
		t.Errorf("report = %+v, want empty", report)
	}
}

func TestUpdateReadError(t *testing.T) {
	dir := writeTree(t, map[string]string{".github/workflows/dir.yaml/x": ""})
	if _, err := Update(dir, &fakeResolver{}); err == nil {
		t.Error("Update read a directory as a workflow")
	}
}

func TestUpdateWriteError(t *testing.T) {
	dir := writeTree(t, map[string]string{".github/workflows/a.yaml": "- uses: a/b@v1\n"})
	path := filepath.Join(dir, ".github/workflows/a.yaml")
	if err := os.Chmod(path, 0o444); err != nil {
		t.Fatal(err)
	}
	if f, err := os.OpenFile(path, os.O_WRONLY, 0); err == nil {
		_ = f.Close()
		t.Skip("running as a user that can write read-only files")
	}
	r := &fakeResolver{
		latest: map[string]string{"a/b": "v2.0.0"},
		tags:   map[string]bool{"a/b@v2": true},
	}
	if _, err := Update(dir, r); err == nil {
		t.Error("Update ignored a failed write")
	}
}
