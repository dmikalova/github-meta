# 7. A weekly conformance bot pushes unsupervised

This decision records how existing projects are kept in line with the standard after they have been bootstrapped.

## Context

Standards drift unless something enforces them. Enforcement could have been a CLI run by hand, a bot that opens pull requests, or a bot that pushes directly. Pull requests would leave a backlog of routine chores for the human to merge. Every project already runs its own check, and has to test itself well enough to catch breaking changes.

## Decision

**A scheduled workflow in project-standards runs weekly against every opted-in project. It writes whatever changes are needed and pushes them straight to the default branch. It doesn't open a pull request, and no human reviews the change.**

- **Opt-in:** a project opts in with the `mklv-conform` topic, set in infrastructure's Terraform. Forks are never touched.
- **Auth:** a fine-grained token with Contents and Workflows write access, stored in GCP Secret Manager and read through WIF.
- **Signing:** commits are created through the GitHub API, which signs them, so they satisfy the rulesets' signed-commit rule.
- **Check first:** the bot runs the project's check on the changed tree before committing. Pushing to main deploys `cloudrun` projects, so a project whose check fails is skipped and reported.
- **Responsibility:** if a project's own tests let a breaking change through, the project is at fault, not the bot.
- **What the bot writes:**
  - `LICENSE`, from a Go template with the years taken from the project's commit history
  - the lefthook config, which must match the template exactly
  - the `cicd.yaml` caller
  - the mage stub, for Go projects
  - generated tool configs (ADR 0005). A config file at a generated path that doesn't carry the generated header is hand-written. It is reported, not overwritten.
  - one universal `.gitignore` and one universal `.dockerignore`. They're the same for every project, so language changes never cause drift, and one-off additions go in `mklv.config.json`.
  - the project-standards version, bumped to the latest
  - `go mod tidy`
  - `govulncheck`-driven bumps of affected modules only
- **What it reports without changing:**
  - hand-edited tool configs whose overrides need moving into `mklv.config.json`, because deleting them could silently drop real exclusions
  - an invalid `mklv.config.json`
  - a missing README
- **Reporting:** each run posts a summary to the Discord `#maintenance` channel: projects pushed, unchanged, skipped because their check failed, and failed.
- **Not enforced:** `CONTEXT.md` and ADRs.
