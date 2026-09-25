package checks

import (
	"fmt"
	"strings"

	"github.com/dmikalova/project-standards/internal/config"
	"github.com/dmikalova/project-standards/internal/generate"
)

// markdownGlob is every Markdown file. goldmark-lint expands it itself, and the
// generated config's gitignore setting drops ignored paths.
const markdownGlob = "**/*.md"

// Markdown lints Markdown with goldmark-lint, fixing nothing.
func (c *Checks) Markdown() error {
	_, err := GoRun(c.Tools.GoldmarkLint, "--no-cache", markdownGlob)
	return err
}

// MarkdownFix applies goldmark-lint's fixes. Issues it cannot fix are left for
// Markdown to report.
func (c *Checks) MarkdownFix() error {
	status, err := GoRun(c.Tools.GoldmarkLint, "--no-cache", "--fix", markdownGlob)
	return c.TolerateFindings("goldmark-lint", 1, status, err)
}

// Spell runs misspell over the project's files with the tools.misspell settings
// from mklv.config.json, writing corrections when write is set.
//
// In a Go project, Go files are checked in go mode, which only reads comments,
// so a correction never renames an identifier, and everything else is checked
// as text. Without a go.mod only the text files are checked: Go files there
// belong to no module, and text mode would rewrite their identifiers.
func (c *Checks) Spell(write bool) error {
	p, err := config.Load(".")
	if err != nil {
		return err
	}
	s, err := generate.MisspellSettings(p)
	if err != nil {
		return err
	}
	goModule := GoModule()
	files, err := ProjectFiles(s.Exclude)
	if err != nil {
		return err
	}
	var goFiles, textFiles []string
	for _, f := range files {
		switch {
		case !strings.HasSuffix(f, ".go"):
			textFiles = append(textFiles, f)
		case goModule:
			goFiles = append(goFiles, f)
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
		for _, chunk := range Chunks(set.files, 500) {
			args := append([]string{c.Tools.Misspell, "-source", set.source}, flags...)
			status, err := GoRun(append(args, chunk...)...)
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
		return fmt.Errorf("misspell found misspellings (run %s to correct them)", c.FixCommand)
	}
	if goModule {
		fmt.Printf("%s: checked %d Go and %d text files\n",
			c.label("spell"), len(goFiles), len(textFiles))
	} else {
		fmt.Printf("%s: checked %d text files\n", c.label("spell"), len(textFiles))
	}
	return nil
}
