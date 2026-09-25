// Package changelog renders release notes from Conventional Commit subjects,
// grouped under the headings semantic-release's Angular preset used, so a
// release reads the same as it did before svu replaced semantic-release.
package changelog

import (
	"fmt"
	"regexp"
	"strings"
)

// A Commit is one commit in a release.
type Commit struct {
	SHA     string
	Subject string
	Body    string
}

// Git runs git with args in the project and returns its standard output.
type Git func(args ...string) (string, error)

// Field and record separators for the git log format: characters no commit
// message contains.
const (
	fieldSep  = "\x1f"
	recordSep = "\x1e"
)

// Commits lists the non-merge commits in from..to, oldest first. An empty from
// lists every commit reachable from to.
func Commits(git Git, from, to string) ([]Commit, error) {
	rev := to
	if from != "" {
		rev = from + ".." + to
	}
	out, err := git("log", "--no-merges", "--reverse",
		"--format=%H"+fieldSep+"%s"+fieldSep+"%b"+recordSep, rev)
	if err != nil {
		return nil, fmt.Errorf("listing commits in %s: %w", rev, err)
	}
	var commits []Commit
	for record := range strings.SplitSeq(out, recordSep) {
		fields := strings.SplitN(strings.TrimLeft(record, "\n"), fieldSep, 3)
		if len(fields) != 3 {
			continue
		}
		commits = append(commits, Commit{
			SHA:     fields[0],
			Subject: fields[1],
			Body:    strings.TrimSpace(fields[2]),
		})
	}
	return commits, nil
}

// remotePattern matches a GitHub remote URL over SSH or HTTPS.
var remotePattern = regexp.MustCompile(
	`^(?:git@github\.com:|https://github\.com/)([^/]+/[^/]+?)(?:\.git)?/?$`,
)

// CommitURL returns the URL prefix a commit SHA is appended to, for a GitHub
// remote URL, or "" for any other remote, which renders unlinked SHAs.
func CommitURL(remote string) string {
	m := remotePattern.FindStringSubmatch(strings.TrimSpace(remote))
	if m == nil {
		return ""
	}
	return "https://github.com/" + m[1] + "/commit/"
}

// A section is one heading of the notes, for one commit type.
type section struct {
	typ   string
	title string
}

// sections are the headings in the order they render. Commits whose type is not
// listed, or whose subject is not a Conventional Commit, go under otherTitle.
var sections = []section{
	{"feat", "Features"},
	{"fix", "Bug Fixes"},
	{"perf", "Performance Improvements"},
	{"revert", "Reverts"},
	{"refactor", "Code Refactoring"},
	{"docs", "Documentation"},
	{"style", "Styles"},
	{"test", "Tests"},
	{"build", "Build System"},
	{"ci", "Continuous Integration"},
	{"chore", "Miscellaneous Chores"},
}

const (
	breakingTitle = "⚠ BREAKING CHANGES"
	otherTitle    = "Other Changes"
	other         = ""
)

// subjectPattern matches a Conventional Commit subject: type, optional scope,
// optional ! for a breaking change, and the description.
var subjectPattern = regexp.MustCompile(`^([a-zA-Z]+)(?:\(([^)]+)\))?(!)?: (.+)$`)

// breakingPattern matches a BREAKING CHANGE footer and captures its note.
var breakingPattern = regexp.MustCompile(`(?m)^BREAKING[ -]CHANGE: (.+)$`)

// An entry is one rendered line.
type entry struct {
	scope, text, sha string
}

// Render renders commits as Markdown release notes. commitURL prefixes each
// commit's SHA to link it; empty leaves the SHAs unlinked.
func Render(commits []Commit, commitURL string) string {
	if len(commits) == 0 {
		return "No changes.\n"
	}
	known := map[string]bool{}
	for _, s := range sections {
		known[s.typ] = true
	}
	byType := map[string][]entry{}
	var breaking []entry
	for _, c := range commits {
		typ, e, isBreaking := parse(c)
		if !known[typ] {
			typ = other
		}
		byType[typ] = append(byType[typ], e)
		if isBreaking {
			note := e
			if m := breakingPattern.FindStringSubmatch(c.Body); m != nil {
				note.text = m[1]
			}
			breaking = append(breaking, note)
		}
	}

	var b strings.Builder
	write := func(title string, entries []entry) {
		if len(entries) == 0 {
			return
		}
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		fmt.Fprintf(&b, "### %s\n\n", title)
		for _, e := range entries {
			b.WriteString("* ")
			if e.scope != "" {
				fmt.Fprintf(&b, "**%s:** ", e.scope)
			}
			fmt.Fprintf(&b, "%s (%s)\n", e.text, link(e.sha, commitURL))
		}
	}
	write(breakingTitle, breaking)
	for _, s := range sections {
		write(s.title, byType[s.typ])
	}
	write(otherTitle, byType[other])
	return b.String()
}

// parse splits a commit into its type and rendered entry, and reports whether
// it is a breaking change. A subject that is not a Conventional Commit keeps
// its whole text and has type other.
func parse(c Commit) (typ string, e entry, isBreaking bool) {
	e.sha = c.SHA
	m := subjectPattern.FindStringSubmatch(c.Subject)
	if m == nil {
		e.text = c.Subject
		return other, e, breakingPattern.MatchString(c.Body)
	}
	e.scope, e.text = m[2], m[4]
	return strings.ToLower(m[1]), e, m[3] == "!" || breakingPattern.MatchString(c.Body)
}

// link renders a commit's short SHA, linked when commitURL is set.
func link(sha, commitURL string) string {
	short := sha
	if len(short) > 7 {
		short = short[:7]
	}
	if commitURL == "" {
		return short
	}
	return "[" + short + "](" + commitURL + sha + ")"
}
