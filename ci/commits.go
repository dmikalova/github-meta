package ci

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/magefile/mage/sh"
)

// Commits lints commit messages not yet on the default branch.
// By default the range is from the merge-base with the default branch to HEAD.
// With no such commits, or no default branch to compare with (a shallow clone,
// a repository without a remote), it passes. Merge commits are skipped.
//
// CI sets CI_COMMIT_RANGE to "<from>..<to>" to lint exactly that range, such as
// the commits a push to the default branch added, which the default range
// never sees. A from of all zeros, which a forge sends for the first push of a
// new branch, lints from the merge-base of to with the default branch. When
// the variable is set, a range that cannot be resolved is an error, not a
// skip: the clone must hold both ends, so CI needs full history.
//
// Only the branch's own commits are linted, never the default branch's
// history: those were linted when they were made, and a project adopting the
// standard should not fail on commits it can no longer reword.
func Commits() error {
	revRange, err := commitRange(os.Getenv(commitRangeEnv))
	if err != nil {
		return err
	}
	if revRange == "" {
		return nil // Skipped; commitRange said why.
	}
	revs, err := git("rev-list", "--no-merges", "--reverse", revRange)
	if err != nil {
		return fmt.Errorf("listing commits in %s: %w", revRange, err)
	}
	shas := strings.Fields(revs)
	if len(shas) == 0 {
		fmt.Printf("ci:commits: no commits in %s to lint\n", revRange)
		return nil
	}
	args := []string{"run", commitlint, "lint"}
	if _, err := os.Stat(commitlintConfig); err == nil {
		args = append(args, "--config", commitlintConfig)
	}
	var failed []string
	for _, sha := range shas {
		msg, err := git("log", "-1", "--format=%B", sha)
		if err != nil {
			return err
		}
		cmd := exec.Command("go", args...)
		cmd.Stdin = strings.NewReader(msg + "\n")
		var out bytes.Buffer
		cmd.Stdout, cmd.Stderr = &out, &out
		if err := cmd.Run(); err != nil {
			fmt.Printf("%s %s\n%s", sha[:12], firstLine(msg), out.String())
			failed = append(failed, sha[:12])
		}
	}
	fmt.Printf("ci:commits: linted %d commit(s) in %s\n", len(shas), revRange)
	if len(failed) > 0 {
		return fmt.Errorf("commit messages failed commitlint: %s", strings.Join(failed, ", "))
	}
	return nil
}

// commitlintConfig is the generated config commitlint lints with.
const commitlintConfig = ".commitlint.yaml"

// commitRangeEnv names the variable CI sets to the range a push added, as
// "<from>..<to>". The unified cicd workflow sets it from
// ${{ github.event.before }}..${{ github.sha }}.
const commitRangeEnv = "CI_COMMIT_RANGE"

// commitRange returns the git revision range Commits lints. With env set it is
// that range, a zero from resolved to the merge-base of to with the default
// branch. Without it, it is merge-base..HEAD, or "" (after saying why) when
// there is no default branch or merge-base to compare with.
func commitRange(env string) (string, error) {
	if env != "" {
		from, to, err := parseCommitRange(env)
		if err != nil {
			return "", fmt.Errorf("%s: %w", commitRangeEnv, err)
		}
		if from == "" {
			base, err := defaultBranch()
			if err != nil {
				return "", fmt.Errorf(
					"%s=%s starts at the zero commit: %w",
					commitRangeEnv,
					env,
					err,
				)
			}
			if from, err = git("merge-base", base, to); err != nil {
				return "", fmt.Errorf("%s=%s: no merge-base between %s and %s", commitRangeEnv,
					env, base, to)
			}
		}
		return from + ".." + to, nil
	}
	base, err := defaultBranch()
	if err != nil {
		fmt.Println("ci:commits: skipped:", err)
		return "", nil
	}
	mb, err := git("merge-base", base, "HEAD")
	if err != nil {
		fmt.Printf("ci:commits: skipped: no merge-base between %s and HEAD\n", base)
		return "", nil
	}
	return mb + "..HEAD", nil
}

// parseCommitRange splits "<from>..<to>". A from of all zeros, git's null
// object ID, is returned as "" so the caller lints from the merge-base. The
// three-dot form is rejected: it means the symmetric difference, which would
// lint commits the push did not add.
func parseCommitRange(s string) (from, to string, err error) {
	s = strings.TrimSpace(s)
	if strings.Contains(s, "...") {
		return "", "", fmt.Errorf("%q: want <from>..<to>, not the three-dot form", s)
	}
	from, to, ok := strings.Cut(s, "..")
	if !ok || from == "" || to == "" || strings.Contains(to, "..") {
		return "", "", fmt.Errorf("%q: want <from>..<to>", s)
	}
	if strings.Trim(from, "0") == "" {
		from = ""
	}
	return from, to, nil
}

// defaultBranch returns the ref of the default branch to compare HEAD with:
// the remote's HEAD when it is known, otherwise the first of origin/main,
// origin/master, main and master that exists.
func defaultBranch() (string, error) {
	ref, err := git("symbolic-ref", "--quiet", "--short", "refs/remotes/origin/HEAD")
	if err == nil && ref != "" {
		return ref, nil
	}
	for _, ref := range []string{"origin/main", "origin/master", "main", "master"} {
		if _, err := git("rev-parse", "--verify", "--quiet", ref+"^{commit}"); err == nil {
			return ref, nil
		}
	}
	return "", errors.New("no default branch found (origin/HEAD, main or master)")
}

// git runs a git command quietly and returns its trimmed output.
func git(args ...string) (string, error) {
	return sh.Output("git", args...)
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(s, "\n")
	return line
}
