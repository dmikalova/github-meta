# 1. One repository holds every project standard

This decision records what project-standards (formerly github-meta) is for and where its boundaries are.

## Context

Shared standards were spread across several places. github-meta held reusable workflows, the shared lefthook config, commitlint and a semantic-release package. Each repo also kept its own copies of lint configs and a hand-maintained LICENSE. Two separate repos were considered: build-automation (the build parts) and repo-standards (templates and enforcement). But the enforcement checks the build's own conventions, such as "lefthook points at the shared remote" and "no hand-edited tool configs". Keeping them apart would split one question — does this project meet the standard? — across two version streams.

## Decision

**One repository, project-standards, holds every standard a project follows regardless of language, and the automation that enforces them.**

- **The name avoids forge-specific words** such as "github" and "repo", because the standards apply to projects, not to one hosting platform.
- **The rename from github-meta happens before the Go module exists** (ADR 0002). A Go module path has to match the repository name, so creating the module first would force every consumer through a later module-path migration.
- **Every caller's `uses:` reference has to change at the same time as the rename**, because GitHub doesn't redirect reusable-workflow references after a rename. Git fetches (lefthook remotes) are redirected, but they are updated too.
- **Settings on GitHub itself** (rulesets, topics, secrets) stay in the infrastructure repository's Terraform.
