# 6. One cicd workflow for every project

This decision records how CI/CD is shared across projects in every language.

## Context

There was one reusable workflow per kind of project: `go-cloudrun.yaml`, `deno-cloudrun.yaml` (a Dagger pipeline) and `terramate-apply-all.yaml`. A project that changed language or deployment target had to switch workflows, and a Go CLI such as diatom had no workflow at all. Every project already calls its workflow from `.github/workflows/cicd.yaml`.

## Decision

**One reusable workflow, `cicd.yaml`, serves every project. It works out which languages the project uses from its files, and takes the project's `kind` from `mklv.config.json`.**

1. **Setup:** installs the detected toolchains and authenticates to GCP through WIF.
2. **Check:** runs each language's own check (`mage ci:check`, `deno task check`, `tofu` and terramate validation), plus the shared checks through the project-standards binary.
3. **Release**, on main: svu computes the next version from Conventional Commits and tags it.
4. **Deploy**, on main, depending on `kind`:
   - **cli**: goreleaser publishes binaries
   - **cloudrun**: builds the image and deploys to Cloud Run
   - **infra**: terramate apply
   - **library**: tag only
5. **Notify:** every deploy, whether it passed or failed, is posted to the Discord `#deploys` channel through a webhook. The webhook URL is stored with SOPS in infrastructure, synced to Secret Manager, and read through WIF.

The per-kind workflows and the Deno Dagger pipeline are replaced. Each project's `cicd.yaml` caller is a fixed file that the conformance bot keeps up to date (ADR 0007).
