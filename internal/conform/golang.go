package conform

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/semver"
)

// ciPackage is the package whose targets a project's magefile imports.
const ciPackage = standardsModule + "/ci"

// goMod parses the project's go.mod, or returns nil when it is not a Go
// project.
func (c *conformer) goMod() (*modfile.File, error) {
	data, ok, err := c.read("go.mod")
	if err != nil || !ok {
		return nil, err
	}
	mod, err := modfile.Parse("go.mod", data, nil)
	if err != nil {
		return nil, err
	}
	if mod.Module == nil {
		return nil, errors.New("go.mod: no module directive")
	}
	return mod, nil
}

// needsMagefile reports whether a Go project has no magefiles at all, so it
// gets the stub. Magefiles that do not import the ci targets are never
// touched, since they hold project-specific targets; they are reported.
func (c *conformer) needsMagefile() bool {
	// A glob of this fixed pattern cannot be malformed.
	magefiles, _ := filepath.Glob(filepath.Join(c.dir, "magefiles", "*.go"))
	where := "magefiles"
	if _, err := os.Stat(c.path("magefile.go")); err == nil {
		magefiles = append(magefiles, c.path("magefile.go"))
		where = "magefile.go"
	}
	if len(magefiles) == 0 {
		return true
	}
	if !slices.ContainsFunc(magefiles, importsCI) {
		c.find(ruleMagefile, where, "the magefiles do not import project-standards' ci "+
			"targets; add `// mage:import ci` of "+ciPackage+" by hand, keeping the "+
			"project's own targets")
	}
	return false
}

// importsCI reports whether a Go file imports the ci package under a
// mage:import comment. A file that does not parse does not.
func importsCI(path string) bool {
	f, err := parser.ParseFile(token.NewFileSet(), path, nil,
		parser.ImportsOnly|parser.ParseComments)
	if err != nil {
		return false
	}
	for _, imp := range f.Imports {
		if imp.Path.Value != strconv.Quote(ciPackage) || imp.Doc == nil {
			continue
		}
		for _, comment := range imp.Doc.List {
			text := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
			if strings.HasPrefix(text, "mage:import") {
				return true
			}
		}
	}
	return false
}

// goFiles are the files the Go dependency steps change.
var goFiles = []string{"go.mod", "go.sum"}

// goDeps bumps project-standards to its latest release when the project
// requires it, tidies, then bumps each module govulncheck reports to its fixed
// version and tidies again.
func (c *conformer) goDeps(mod *modfile.File) error {
	before := map[string][]byte{}
	for _, f := range goFiles {
		data, _, err := c.read(f)
		if err != nil {
			return err
		}
		before[f] = data
	}
	if slices.ContainsFunc(mod.Require, func(r *modfile.Require) bool {
		return r.Mod.Path == standardsModule
	}) {
		if err := c.goCmd("get", standardsModule+"@latest"); err != nil {
			return err
		}
	}
	if err := c.goCmd("mod", "tidy"); err != nil {
		return err
	}
	out, err := c.run(c.dir, "go", "run", c.govulncheck, "-format", "json", "./...")
	if err != nil {
		return err
	}
	bumps, err := c.vulnFixes(out)
	if err != nil {
		return err
	}
	if len(bumps) > 0 {
		if err := c.goCmd(append([]string{"get"}, bumps...)...); err != nil {
			return err
		}
		if err := c.goCmd("mod", "tidy"); err != nil {
			return err
		}
	}
	for _, f := range goFiles {
		data, _, err := c.read(f)
		if err != nil {
			return err
		}
		if !bytes.Equal(before[f], data) {
			c.report.Changed = append(c.report.Changed, f)
		}
	}
	return nil
}

// goCmd runs a go command in the project, discarding its output.
func (c *conformer) goCmd(args ...string) error {
	_, err := c.run(c.dir, "go", args...)
	return err
}

// vulnFinding is the part of a govulncheck -format json finding message
// conform reads. The first trace frame is the vulnerable module.
type vulnFinding struct {
	Finding *struct {
		OSV          string `json:"osv"`
		FixedVersion string `json:"fixed_version"`
		Trace        []struct {
			Module  string `json:"module"`
			Version string `json:"version"`
		} `json:"trace"`
	} `json:"finding"`
}

// vulnFixes reads govulncheck's JSON stream and returns a module@version for
// each vulnerable module with a fix, at the highest fixed version any of its
// findings needs, sorted. Only those modules are bumped. A vulnerability with
// no fix, or in the standard library, which needs a new Go, is reported.
func (c *conformer) vulnFixes(out []byte) ([]string, error) {
	fixes := map[string]string{}
	reported := map[string]bool{}
	dec := json.NewDecoder(bytes.NewReader(out))
	for {
		var msg vulnFinding
		err := dec.Decode(&msg)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading govulncheck output: %w", err)
		}
		f := msg.Finding
		if f == nil || len(f.Trace) == 0 {
			continue
		}
		module, version := f.Trace[0].Module, f.Trace[0].Version
		var problem string
		switch {
		case module == "stdlib" || module == "toolchain":
			problem = fmt.Sprintf("Go %s is affected by %s; fixed in Go %s, so raise the go "+
				"or toolchain directive", version, f.OSV, f.FixedVersion)
		case f.FixedVersion == "":
			problem = fmt.Sprintf("%s@%s is affected by %s, which has no fixed version",
				module, version, f.OSV)
		default:
			if semver.Compare(f.FixedVersion, fixes[module]) > 0 {
				fixes[module] = f.FixedVersion
			}
			continue
		}
		if !reported[problem] {
			reported[problem] = true
			c.find(ruleVuln, "go.mod", problem)
		}
	}
	bumps := make([]string, 0, len(fixes))
	for module, version := range fixes {
		bumps = append(bumps, module+"@"+version)
	}
	slices.Sort(bumps)
	return bumps, nil
}
