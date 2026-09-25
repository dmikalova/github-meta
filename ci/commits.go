package ci

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
	return shared.Commits()
}
