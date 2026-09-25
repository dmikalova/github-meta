# 4. The toolchain is Go-native and pinned in one place

This decision records which tools the shared checks use and how their versions are managed.

## Context

The previous toolchain mixed ecosystems: commitlint and semantic-release came from npm, and typos is a Rust binary. Each needed its own installation in CI and on every machine. typos had also been frustrating: it flagged, and with `--write-changes` rewrote, text that should not have changed. vex's `typos.toml` had grown into a long list of card-name allowances.

## Decision

**Every shared tool is a Go program run with `go run pkg@version`. The version constants live only in project-standards, so updating one constant updates every project.**

| Purpose | Tool | Replaces |
|---|---|---|
| Go lint | golangci-lint v2 | — |
| Commit messages | conventionalcommit/commitlint | npm commitlint |
| Spelling | golangci/misspell | typos |
| Markdown | goldmark-lint | — |
| Secrets | gitleaks (`github.com/zricethezav/gitleaks/v8`) | — |
| Versioning | svu, plus `git tag` | semantic-release |
| Releases and binaries | goreleaser (free version) | semantic-release |

- **Switching to misspell is a deliberate trade-off.** It checks a list of known misspellings and doesn't split identifiers, so some typos inside identifiers get through. In exchange, it doesn't produce typos' false positives.
- **Dropping semantic-release** removes the `@dmikalova/semantic-release-config` npm package and most of the repository's npm setup. The version bump, tag, changelog and GitHub release are chained together in the workflow.
- **commitlint's maintenance is a known risk.** The Go port is small and still pre-1.0. Because its programmatic API is simple, replacing it would be easy.
