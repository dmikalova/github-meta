package ci

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"sync"
)

// goRunStatus runs `go run <args>` with its output streamed and returns the
// tool's own exit status, 0 on success.
//
// go run exits 1 whenever the program fails, whatever the program's status,
// and reports that status only as a final "exit status N" line on stderr. That
// line is how a fixer's "issues remain" (golangci-lint 1) is told apart from a
// real failure (golangci-lint 3, a bad config). A failure of the go command
// itself, such as a module that does not download, prints no such line and
// returns err with status -1.
func goRunStatus(args ...string) (status int, err error) {
	cmd := exec.Command("go", append([]string{"run"}, args...)...)
	tail := &tailWriter{w: os.Stderr}
	cmd.Stdout, cmd.Stderr = os.Stdout, tail
	err = cmd.Run()
	if err == nil {
		return 0, nil
	}
	if code, ok := programStatus(tail.bytes()); ok {
		return code, err
	}
	return -1, err
}

var exitStatusLine = regexp.MustCompile(`(?m)^exit status (\d+)\s*\z`)

// programStatus finds the "exit status N" line go run prints last when the
// program it ran fails.
func programStatus(stderrTail []byte) (int, bool) {
	m := exitStatusLine.FindSubmatch(stderrTail)
	if m == nil {
		return 0, false
	}
	code, err := strconv.Atoi(string(m[1]))
	return code, err == nil
}

// tailWriter passes writes through and keeps the last few hundred bytes.
type tailWriter struct {
	w    io.Writer
	mu   sync.Mutex
	tail []byte
}

const tailSize = 256

func (t *tailWriter) Write(p []byte) (int, error) {
	t.mu.Lock()
	t.tail = append(t.tail, p...)
	if len(t.tail) > tailSize {
		t.tail = t.tail[len(t.tail)-tailSize:]
	}
	t.mu.Unlock()
	return t.w.Write(p)
}

func (t *tailWriter) bytes() []byte {
	t.mu.Lock()
	defer t.mu.Unlock()
	return bytes.Clone(t.tail)
}
