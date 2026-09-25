package checks

import (
	"bytes"
	"os"
	"os/exec"
	"path"
	"strings"
)

// ProjectFiles lists the files git tracks plus untracked files it does not
// ignore, minus those matching an exclude glob and any that no longer exist.
func ProjectFiles(exclude []string) ([]string, error) {
	out, err := git("ls-files", "-z", "--cached", "--others", "--exclude-standard")
	if err != nil {
		return nil, err
	}
	var files []string
	seen := map[string]bool{}
	for f := range strings.SplitSeq(out, "\x00") {
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

// Chunks splits a list so no single command line grows past the OS limit.
func Chunks(list []string, size int) [][]string {
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

// git runs a git command quietly and returns its output without the trailing
// newlines. A failure's error carries git's stderr.
func git(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := run(cmd); err != nil {
		return "", &cmdError{cmd: "git " + args[0], err: err, stderr: stderr.String()}
	}
	return strings.TrimRight(stdout.String(), "\r\n"), nil
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
