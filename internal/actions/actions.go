// Package actions updates the GitHub Actions a repository's workflows and
// composite actions use to their latest releases (ADR 0007). A reference to a
// major tag, such as actions/checkout@v6, moves to the latest release's major
// tag; a reference to a fuller version, such as ariga/setup-atlas@v0.2.1,
// moves to the latest release's tag. Branch, SHA and local references are left alone,
// and so is any action whose latest release is older than the one in use.
package actions

import (
	"cmp"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/dmikalova/project-standards/internal/conform"
)

// A Resolver looks up an action repository's releases.
type Resolver interface {
	// LatestRelease returns the tag of owner/repo's latest release.
	LatestRelease(repo string) (string, error)
	// HasTag reports whether owner/repo has the tag.
	HasTag(repo, tag string) (bool, error)
}

// Report is what Update changed and what it could not resolve.
type Report struct {
	// Changed lists the files written, relative to the repository root, sorted.
	Changed []string `json:"changed"`
	// Updates lists each action moved, as "owner/repo old → new", sorted.
	Updates []string `json:"updates"`
	// Findings lists the actions that could not be resolved.
	Findings []conform.Finding `json:"findings"`
}

// rule names the findings Update reports.
const rule = "actions"

// usesLine matches a `uses: action@ref` line, keeping everything around the
// ref so a rewrite changes nothing else.
var usesLine = regexp.MustCompile(`^(\s*-?\s*uses:\s*["']?)([^@\s"']+)@([^\s"'#]+)(.*)$`)

// version matches a major tag (v6) or a fuller version (v0.3, v0.2.1).
var version = regexp.MustCompile(`^v(\d+)((?:\.\d+){0,2})$`)

// Update rewrites the action references in dir's workflows and composite
// actions and reports what changed. Each repository is resolved once.
func Update(dir string, r Resolver) (*Report, error) {
	files := workflowFiles(dir)
	u := &updater{resolver: r, resolved: map[string]resolution{}}
	report := &Report{Changed: []string{}, Updates: []string{}, Findings: []conform.Finding{}}
	updates := map[string]bool{}
	reported := map[conform.Finding]bool{}
	for _, rel := range files {
		path := filepath.Join(dir, rel)
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		lines := strings.Split(string(data), "\n")
		changed := false
		for i, line := range lines {
			m := usesLine.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			action, ref := m[2], m[3]
			repo, ok := actionRepo(action)
			if !ok {
				continue
			}
			next, finding := u.next(repo, ref)
			if finding != "" {
				f := conform.Finding{Rule: rule, Path: rel, Message: finding}
				if !reported[f] {
					reported[f] = true
					report.Findings = append(report.Findings, f)
				}
				continue
			}
			if next == ref {
				continue
			}
			lines[i] = m[1] + action + "@" + next + m[4]
			updates[fmt.Sprintf("%s %s → %s", repo, ref, next)] = true
			changed = true
		}
		if !changed {
			continue
		}
		if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
			return nil, err
		}
		report.Changed = append(report.Changed, rel)
	}
	for update := range updates {
		report.Updates = append(report.Updates, update)
	}
	slices.Sort(report.Updates)
	return report, nil
}

// workflowFiles returns the workflow and composite action files under dir's
// .github, relative to dir, sorted.
func workflowFiles(dir string) []string {
	var files []string
	for _, pattern := range []string{
		".github/workflows/*.yml", ".github/workflows/*.yaml",
		".github/actions/*/action.yml", ".github/actions/*/action.yaml",
	} {
		// The patterns are fixed and valid, so Glob cannot fail.
		matches, _ := filepath.Glob(filepath.Join(dir, pattern))
		for _, m := range matches {
			rel, _ := filepath.Rel(dir, m) // m is under dir
			files = append(files, rel)
		}
	}
	slices.Sort(files)
	return files
}

// actionRepo returns the owner/repo of an action reference, or false for a
// local (./) or Docker action.
func actionRepo(action string) (string, bool) {
	if strings.HasPrefix(action, "./") || strings.HasPrefix(action, "docker://") {
		return "", false
	}
	parts := strings.Split(action, "/")
	if len(parts) < 2 {
		return "", false
	}
	return parts[0] + "/" + parts[1], true
}

// updater resolves each repository's latest release once.
type updater struct {
	resolver Resolver
	resolved map[string]resolution
}

// resolution is a repository's latest release, or why it has none.
type resolution struct {
	tag     string
	finding string
}

// next returns the ref to move ref to, which is ref itself when it is already
// the latest or is not a version, or a finding when the latest release cannot
// be used.
func (u *updater) next(repo, ref string) (string, string) {
	current := version.FindStringSubmatch(ref)
	if current == nil {
		return ref, "" // a branch or SHA: pinned on purpose
	}
	latest, ok := u.resolved[repo]
	if !ok {
		latest = u.latest(repo)
		u.resolved[repo] = latest
	}
	if latest.finding != "" {
		return "", latest.finding
	}
	release := version.FindStringSubmatch(latest.tag)
	if cmp.Compare(semver(release), semver(current)) <= 0 {
		return ref, ""
	}
	if current[2] != "" {
		return latest.tag, ""
	}
	major := "v" + release[1]
	has, err := u.resolver.HasTag(repo, major)
	if err != nil {
		return "", fmt.Sprintf("%s: %v", repo, err)
	}
	if !has {
		return "", fmt.Sprintf("%s: the latest release %s has no %s tag to move to",
			repo, latest.tag, major)
	}
	return major, ""
}

// latest looks up repo's latest release.
func (u *updater) latest(repo string) resolution {
	tag, err := u.resolver.LatestRelease(repo)
	if err != nil {
		return resolution{finding: fmt.Sprintf("%s: %v", repo, err)}
	}
	if version.FindStringSubmatch(tag) == nil {
		return resolution{finding: fmt.Sprintf(
			"%s: the latest release %s is not a vX, vX.Y or vX.Y.Z version", repo, tag)}
	}
	return resolution{tag: tag}
}

// semver orders a version match: a major tag sorts after every release of
// that major (v6 is at least v6.9.9), so v6 only moves to a newer major.
func semver(m []string) int {
	major, _ := strconv.Atoi(m[1]) // the pattern matched digits
	if m[2] == "" {
		return major*1_000_000_000 + 999_999_999
	}
	var minor, patch int
	_, _ = fmt.Sscanf(m[2], ".%d.%d", &minor, &patch) // v0.3 leaves patch 0
	return major*1_000_000_000 + minor*1_000_000 + patch
}
