# 1. One repository holds every project standard

This decision records what project-standards is for and where its boundaries are.

## Context

Shared standards were spread across several places. A repository of reusable workflows held the shared lefthook config, commitlint and a semantic-release package, while each project kept its own copies of lint configs and a hand-maintained LICENSE. Two separate repositories were considered: one for the build (checks, configs, workflows) and one for enforcement (templates and conformance). But enforcement checks the build's own conventions, such as "lefthook points at the shared remote" and "no hand-edited tool configs". Keeping them apart would split one question — does this project meet the standard? — across two version streams.

## Decision

**One repository, project-standards, holds every standard a project follows regardless of language, and the automation that enforces it.**

- **The name avoids forge-specific words** such as "github" and "repo", because the standards apply to projects, not to one hosting platform.
- **The repository's name is fixed before the Go module exists** (ADR 0002). A Go module path has to match the repository name, so renaming after the module exists would force every consumer through a module-path migration.
- **Renaming the repository means updating every caller's `uses:` reference at the same time**, because GitHub doesn't redirect reusable-workflow references after a rename. Git fetches (lefthook remotes) are redirected, but callers are updated anyway.
- **Settings on GitHub itself** (rulesets, topics, secrets) stay in the infrastructure repository's Terraform.
