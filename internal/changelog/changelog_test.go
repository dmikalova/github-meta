package changelog

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

const url = "https://github.com/dmikalova/vex/commit/"

func TestRender(t *testing.T) {
	tests := []struct {
		name    string
		commits []Commit
		url     string
		want    string
	}{
		{
			name: "no commits",
			want: "No changes.\n",
		},
		{
			name: "groups by type in heading order, keeping commit order",
			url:  url,
			commits: []Commit{
				{SHA: "1111111aaaa", Subject: "fix: handle empty decks"},
				{SHA: "2222222bbbb", Subject: "feat: implement cards"},
				{SHA: "3333333cccc", Subject: "feat(web): add the style gallery"},
				{SHA: "4444444dddd", Subject: "ci: point at project-standards"},
			},
			want: "### Features\n\n" +
				"* implement cards ([2222222](" + url + "2222222bbbb))\n" +
				"* **web:** add the style gallery ([3333333](" + url + "3333333cccc))\n" +
				"\n### Bug Fixes\n\n" +
				"* handle empty decks ([1111111](" + url + "1111111aaaa))\n" +
				"\n### Continuous Integration\n\n" +
				"* point at project-standards ([4444444](" + url + "4444444dddd))\n",
		},
		{
			name: "every known section renders under its title",
			commits: []Commit{
				{SHA: "c1", Subject: "chore: c"},
				{SHA: "b1", Subject: "build: b"},
				{SHA: "t1", Subject: "test: t"},
				{SHA: "s1", Subject: "style: s"},
				{SHA: "d1", Subject: "docs: d"},
				{SHA: "r1", Subject: "refactor: r"},
				{SHA: "v1", Subject: "revert: v"},
				{SHA: "p1", Subject: "perf: p"},
			},
			want: "### Performance Improvements\n\n* p (p1)\n" +
				"\n### Reverts\n\n* v (v1)\n" +
				"\n### Code Refactoring\n\n* r (r1)\n" +
				"\n### Documentation\n\n* d (d1)\n" +
				"\n### Styles\n\n* s (s1)\n" +
				"\n### Tests\n\n* t (t1)\n" +
				"\n### Build System\n\n* b (b1)\n" +
				"\n### Miscellaneous Chores\n\n* c (c1)\n",
		},
		{
			name: "unknown types and non-conventional subjects go under Other Changes",
			commits: []Commit{
				{SHA: "a1", Subject: "wip: something"},
				{SHA: "a2", Subject: "Merge the thing"},
			},
			want: "### Other Changes\n\n* something (a1)\n* Merge the thing (a2)\n",
		},
		{
			name: "types are case-insensitive",
			commits: []Commit{
				{SHA: "a1", Subject: "Feat: loud"},
			},
			want: "### Features\n\n* loud (a1)\n",
		},
		{
			name: "breaking changes lead, from ! or a footer, using the footer's note",
			commits: []Commit{
				{SHA: "a1", Subject: "feat(engine)!: drop the pull chooser"},
				{
					SHA:     "a2",
					Subject: "fix: rename a flag",
					Body:    "Details.\n\nBREAKING CHANGE: -old is now -new",
				},
				{SHA: "a3", Subject: "Rework everything", Body: "BREAKING-CHANGE: all of it"},
			},
			want: "### ⚠ BREAKING CHANGES\n\n" +
				"* **engine:** drop the pull chooser (a1)\n" +
				"* -old is now -new (a2)\n" +
				"* all of it (a3)\n" +
				"\n### Features\n\n* **engine:** drop the pull chooser (a1)\n" +
				"\n### Bug Fixes\n\n* rename a flag (a2)\n" +
				"\n### Other Changes\n\n* Rework everything (a3)\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Render(tt.commits, tt.url); got != tt.want {
				t.Errorf("Render() =\n%s\nwant\n%s", got, tt.want)
			}
		})
	}
}

func TestCommits(t *testing.T) {
	out := "aaa\x1ffeat: one\x1f\x1e\n" +
		"bbb\x1ffix: two\x1fbody line\n\nBREAKING CHANGE: x\n\x1e\n"
	tests := []struct {
		name     string
		from     string
		wantArgs string
	}{
		{"range", "v1.0.0", "v1.0.0..HEAD"},
		{"from the root", "", "HEAD"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotArgs []string
			git := func(args ...string) (string, error) {
				gotArgs = args
				return out, nil
			}
			got, err := Commits(git, tt.from, "HEAD")
			if err != nil {
				t.Fatal(err)
			}
			want := []Commit{
				{SHA: "aaa", Subject: "feat: one"},
				{SHA: "bbb", Subject: "fix: two", Body: "body line\n\nBREAKING CHANGE: x"},
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("Commits() = %#v, want %#v", got, want)
			}
			if last := gotArgs[len(gotArgs)-1]; last != tt.wantArgs {
				t.Errorf("revision = %q, want %q", last, tt.wantArgs)
			}
			if !strings.Contains(strings.Join(gotArgs, " "), "--no-merges") {
				t.Errorf("args %v do not skip merges", gotArgs)
			}
		})
	}
}

func TestCommitsEmpty(t *testing.T) {
	got, err := Commits(func(...string) (string, error) { return "", nil }, "v1", "HEAD")
	if err != nil || got != nil {
		t.Errorf("Commits() = %v, %v, want nil, nil", got, err)
	}
}

func TestCommitsError(t *testing.T) {
	_, err := Commits(
		func(...string) (string, error) { return "", errors.New("bad revision") },
		"nope",
		"HEAD",
	)
	if err == nil || !strings.Contains(err.Error(), "nope..HEAD") {
		t.Errorf("Commits() error = %v, want one naming the range", err)
	}
}

func TestCommitURL(t *testing.T) {
	tests := []struct{ remote, want string }{
		{"git@github.com:dmikalova/vex.git", url},
		{"https://github.com/dmikalova/vex.git", url},
		{"https://github.com/dmikalova/vex", url},
		{"https://github.com/dmikalova/vex/\n", url},
		{"git@gitlab.com:dmikalova/vex.git", ""},
		{"", ""},
	}
	for _, tt := range tests {
		if got := CommitURL(tt.remote); got != tt.want {
			t.Errorf("CommitURL(%q) = %q, want %q", tt.remote, got, tt.want)
		}
	}
}
