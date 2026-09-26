# 3. Fix and check are separate, and every commit is green

This decision records the two modes of the shared gate and where each one runs.

## Context

vex's `mage check` formats files, runs `go fix` and runs both linters with `--fix`, and CI ran that same command. A CI run could therefore pass by fixing code in its throwaway checkout, not by finding the code correct. Separately, the shared lefthook config runs `go-fmt`, `go-vet` and `go-test` hooks next to `mage check`, even though a comment says they stand down when magefiles exist.

## Decision

**`ci:fix` applies every autofix. `ci:check` only verifies, and it fails on anything the fixers would change.**

- **`ci:check` covers** formatting (as a diff check), build, vet, lint, markdown, spelling, secrets, commit messages, known vulnerabilities (govulncheck), tests and coverage gates. Vulnerabilities fail the next commit, so they get fixed the same day; the weekly conformance run bumps affected modules as a backstop for projects nobody commits to.
- **CI and diatom run `ci:check`.**
- **The documented local validator is `mage ci:fix && mage ci:check`** (in non-Go projects, the binary's equivalent).
- **pre-commit runs the full check**, so every commit is green and breakage shows up at the earliest point. Test suites are expected to stay fast. A slow suite should split its long-running parts into a separate target, not weaken the gate.
- **The separate Go hooks in the shared lefthook config are removed**, so the gate is the only place checks are defined.
