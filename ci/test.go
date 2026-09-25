package ci

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/magefile/mage/sh"
)

// Test runs every package's tests and prints an aligned report.
func Test() error {
	return goTest("./...")
}

// Cover fails if a CoverGates area is below 100% statement coverage.
// It lists each short area's functions that are not fully covered. With no
// gates set it passes.
func Cover() error {
	if len(CoverGates) == 0 {
		fmt.Println("ci:cover: no coverage gates set (ci.CoverGates)")
		return nil
	}
	// Profiles go to a temporary directory, never the tree, so Check stays
	// read-only.
	dir, err := os.MkdirTemp("", "ci-cover-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	var failed []string
	for i, g := range CoverGates {
		if err := g.run(filepath.Join(dir, strconv.Itoa(i)+".out")); err != nil {
			fmt.Println(err)
			failed = append(failed, g.Name)
		}
	}
	if len(failed) > 0 {
		return fmt.Errorf("coverage below 100%%: %s", strings.Join(failed, ", "))
	}
	return nil
}

// run measures one gate, writing its profile to the given path, and reports
// whether it is short of 100%.
func (g CoverGate) run(profile string) error {
	if g.Name == "" || g.Test == "" || g.Count == "" {
		return fmt.Errorf("coverage gate %+v needs a Name, Test and Count", g)
	}
	// go test's per-package lines report the coverpkg's coverage against other
	// packages' tests, which reads as noise next to the real figure below, so the
	// output is only shown when the run fails.
	out, err := sh.Output("go", "test", g.Test, "-coverpkg="+g.Count, "-coverprofile="+profile)
	if err != nil {
		fmt.Print(tidyTestOutput(out))
		return err
	}
	out, err = sh.Output("go", "tool", "cover", "-func="+profile)
	if err != nil {
		return err
	}
	pct, short, err := parseCoverFunc(out)
	if err != nil {
		return err
	}
	fmt.Printf("%-10s %.1f%%\n", g.Name+":", pct)
	if pct < 100 {
		fmt.Printf("%s functions below 100%%:\n", g.Name)
		for _, l := range short {
			fmt.Println("  " + l)
		}
		return fmt.Errorf("%s coverage %.1f%% is below 100%%", g.Name, pct)
	}
	return nil
}

// parseCoverFunc reads `go tool cover -func` output: it returns the total
// percentage from the last line and every function line not at 100%.
func parseCoverFunc(out string) (pct float64, short []string, err error) {
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	total := lines[len(lines)-1]
	fields := strings.Fields(total)
	if len(fields) == 0 {
		return 0, nil, errors.New("empty coverage report")
	}
	pct, err = strconv.ParseFloat(strings.TrimSuffix(fields[len(fields)-1], "%"), 64)
	if err != nil {
		return 0, nil, fmt.Errorf("parsing coverage percent from %q: %w", total, err)
	}
	for _, l := range lines[:len(lines)-1] {
		if !strings.HasSuffix(l, "100.0%") {
			short = append(short, l)
		}
	}
	return pct, short, nil
}
