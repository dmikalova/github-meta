package ci

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/magefile/mage/sh"

	"github.com/dmikalova/project-standards/internal/config"
	"github.com/dmikalova/project-standards/internal/generate"
)

// markdownGlob is every Markdown file. goldmark-lint expands it itself, and the
// generated config's gitignore setting drops ignored paths.
const markdownGlob = "**/*.md"

// Markdown lints Markdown with goldmark-lint, fixing nothing.
func Markdown() error {
	return sh.RunV("go", "run", goldmarkLint, "--no-cache", markdownGlob)
}

// Spell checks spelling in Go comments and every other text file.
// It uses misspell with the tools.misspell settings from mklv.config.json and
// fixes nothing.
func Spell() error {
	return runMisspell(false)
}

// runMisspell runs misspell over the project's files, writing corrections when
// write is set. Go files are checked in go mode, which only reads comments, so
// a correction never renames an identifier; everything else is checked as text.
func runMisspell(write bool) error {
	p, err := config.Load(".")
	if err != nil {
		return err
	}
	s, err := generate.MisspellSettings(p)
	if err != nil {
		return err
	}
	files, err := projectFiles(s.Exclude)
	if err != nil {
		return err
	}
	var goFiles, textFiles []string
	for _, f := range files {
		if strings.HasSuffix(f, ".go") {
			goFiles = append(goFiles, f)
		} else {
			textFiles = append(textFiles, f)
		}
	}
	flags := []string{"-error"}
	if write {
		flags = []string{"-w"}
	}
	if len(s.Ignore) > 0 {
		flags = append(flags, "-i", strings.Join(s.Ignore, ","))
	}
	if s.Locale != "" {
		flags = append(flags, "-locale", s.Locale)
	}
	// Every chunk runs even after one has findings, so one run reports every
	// misspelling.
	found := false
	for _, set := range []struct {
		source string
		files  []string
	}{{"go", goFiles}, {"text", textFiles}} {
		for _, chunk := range chunks(set.files, 500) {
			args := append([]string{misspell, "-source", set.source}, flags...)
			status, err := goRunStatus(append(args, chunk...)...)
			if status == 2 && !write {
				// -error exits 2 on a finding. The default error would echo every
				// file name in the chunk.
				found = true
				continue
			}
			if err != nil {
				return fmt.Errorf("misspell -source %s failed (exit status %d): %w",
					set.source, status, err)
			}
		}
	}
	if found {
		return errors.New("misspell found misspellings (run mage ci:fix to correct them)")
	}
	fmt.Printf("ci:spell: checked %d Go and %d text files\n", len(goFiles), len(textFiles))
	return nil
}

// projectFiles lists the files git tracks plus untracked files it does not
// ignore, minus those matching an exclude glob and any that no longer exist.
func projectFiles(exclude []string) ([]string, error) {
	cmd := exec.Command("git", "ls-files", "-z", "--cached", "--others", "--exclude-standard")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, &cmdError{cmd: "git ls-files", err: err, stderr: stderr.String()}
	}
	var files []string
	seen := map[string]bool{}
	for f := range strings.SplitSeq(string(out), "\x00") {
		if f == "" || seen[f] || excluded(f, exclude) {
			continue
		}
		seen[f] = true
		// Skip deleted files, directories (submodules) and symlinks.
		if info, err := os.Lstat(f); err != nil || !info.Mode().IsRegular() {
			continue
		}
		files = append(files, f)
	}
	return files, nil
}

func excluded(name string, globs []string) bool {
	for _, g := range globs {
		if matchGlob(g, name) {
			return true
		}
	}
	return false
}

// matchGlob reports whether a slash-separated path matches a glob in which **
// matches any number of whole path segments, including none, and every other
// segment follows path.Match.
func matchGlob(pattern, name string) bool {
	return matchSegments(strings.Split(pattern, "/"), strings.Split(name, "/"))
}

func matchSegments(pattern, name []string) bool {
	for len(pattern) > 0 {
		if pattern[0] == "**" {
			for i := 0; i <= len(name); i++ {
				if matchSegments(pattern[1:], name[i:]) {
					return true
				}
			}
			return false
		}
		if len(name) == 0 {
			return false
		}
		if ok, err := path.Match(pattern[0], name[0]); err != nil || !ok {
			return false
		}
		pattern, name = pattern[1:], name[1:]
	}
	return len(name) == 0
}

// chunks splits a list so no single command line grows past the OS limit.
func chunks(list []string, size int) [][]string {
	var out [][]string
	for len(list) > size {
		out = append(out, list[:size])
		list = list[size:]
	}
	if len(list) > 0 {
		out = append(out, list)
	}
	return out
}

// cmdError is a failed command's error with its stderr attached.
type cmdError struct {
	cmd    string
	err    error
	stderr string
}

func (e *cmdError) Error() string {
	return e.cmd + ": " + e.err.Error() + ": " + strings.TrimSpace(e.stderr)
}

func (e *cmdError) Unwrap() error { return e.err }
