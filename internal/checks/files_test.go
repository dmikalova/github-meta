package checks

import (
	"errors"
	"os"
	"os/exec"
	"reflect"
	"slices"
	"testing"
)

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
	got := Chunks([]string{"a", "b", "c", "d", "e"}, 2)
	want := [][]string{{"a", "b"}, {"c", "d"}, {"e"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Chunks = %v, want %v", got, want)
	}
	if got := Chunks(nil, 2); got != nil {
		t.Errorf("Chunks(nil) = %v, want nil", got)
	}
}

func TestProjectFiles(t *testing.T) {
	g := newRepo(t)
	writeFiles(t, map[string]string{
		".gitignore":      "ignored.txt\n",
		"a.md":            "a",
		"docs/b.md":       "b",
		"gone.txt":        "c",
		"ignored.txt":     "d",
		"untracked.go":    "package x",
		"vendor/skip.txt": "e",
	})
	g("add", ".gitignore", "a.md", "docs/b.md", "gone.txt")
	if err := os.Remove("gone.txt"); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("a.md", "link.md"); err != nil {
		t.Fatal(err)
	}
	got, err := ProjectFiles([]string{"vendor/**"})
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(got)
	want := []string{".gitignore", "a.md", "docs/b.md", "untracked.go"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ProjectFiles = %v, want %v", got, want)
	}
}

func TestProjectFilesOutsideRepository(t *testing.T) {
	t.Chdir(t.TempDir())
	t.Setenv("GIT_CEILING_DIRECTORIES", os.TempDir())
	_, err := ProjectFiles(nil)
	wantErr(t, err, "git ls-files: exit status")
	if _, ok := errors.AsType[*exec.ExitError](err); !ok {
		t.Errorf("err = %v, want it to wrap the git command's exit error", err)
	}
}
