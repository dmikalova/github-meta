# 2. A Go module imported by mage and published as a binary

This decision records how projects consume the shared checks.

## Context

The shared checks (lint, spelling, secrets, commit messages, markdown) are Go tools. Go projects already use mage as their task runner. Non-Go projects (Deno, Terraform) have their own native tooling. Running mage in those projects would add a `go.mod` and a `go.work` to each one, the `go.work` only to work around mage issue #364, where imported packages fail to resolve from the repository root.

## Decision

**The repository root is the Go module `github.com/dmikalova/project-standards`. Go projects import it into mage, and every other project uses a published binary.**

- **Go projects** import the `ci` package in their magefiles (`// mage:import ci`), which gives `mage ci:check`, `mage ci:fix`, `mage ci:lint`, and so on. Project-specific targets live alongside them. The module is a thin library in the project's `go.mod`. The tools it runs are invoked with `go run pkg@version` and never enter the project's dependency graph.
- **Everything else uses the `project-standards` binary**, which goreleaser builds from `cmd/project-standards` on each release. It is a small hand-written command (`check`, `fix`, one command per shared check, and `version`) over the same implementation as the `ci` targets, rather than a `mage -compile` of them, so it has only the shared checks and needs no magefile. It runs the tools with `go run` and the `ci` package's pins, so every project needs Go, and CI runs the binary itself with `go run github.com/dmikalova/project-standards/cmd/project-standards@latest`. Locally, lefthook runs it from PATH, and dotfiles installs it. Non-Go projects get no Go files.
- **Language checks use each language's own tools** (`deno task check`, `tofu` and terramate). The binary only covers the checks shared by every language.
- **Layout:**
  - `ci/`: the mage targets
  - `internal/config`: config merging (ADR 0005)
  - `cmd/project-standards`: the binary
  - `templates/`: embedded base configs and standard files
- **New projects are bootstrapped by hand.** After that, the conformance bot keeps them in line (ADR 0007).
