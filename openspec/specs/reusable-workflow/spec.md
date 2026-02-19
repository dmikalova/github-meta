# Reusable Workflow

## ADDED Requirements

### Requirement: Workflow is reusable

The workflow SHALL be defined as a reusable workflow that other repos can call with `workflow_call`.

#### Scenario: Called from app repo

- **WHEN** an app repo workflow uses
  `dmikalova/github-meta/.github/workflows/deno-cloudrun.yaml@main`
- **THEN** the workflow executes in the context of the calling repo

### Requirement: Workflow checks out source

The workflow SHALL check out the calling repository's source code.

#### Scenario: Source available to Dagger

- **WHEN** the workflow runs
- **THEN** the app repo's source code is available at the workspace root

### Requirement: Workflow authenticates to GCP via WIF

The workflow SHALL authenticate to GCP using Workload Identity Federation with hardcoded provider
and service account.

#### Scenario: WIF authentication succeeds

- **WHEN** the workflow runs on a repo with the `mklv-deploy` topic
- **THEN** it obtains GCP credentials without stored secrets

#### Scenario: Uses deploy service account

- **WHEN** the workflow authenticates
- **THEN** it uses `github-actions-deploy@dmikalova-mklv.iam.gserviceaccount.com`

### Requirement: Workflow runs Dagger pipeline

The workflow SHALL invoke the Dagger pipeline from github-meta.

#### Scenario: Dagger module called

- **WHEN** the workflow runs
- **THEN** it calls `dagger call` with the `github.com/dmikalova/github-meta/dagger/deno` module

### Requirement: Workflow passes secrets to Dagger

The workflow SHALL pass required secrets (GITHUB\_TOKEN, DATABASE\_URL) to the Dagger pipeline.

#### Scenario: GitHub token available

- **WHEN** Dagger needs to push to GHCR
- **THEN** GITHUB\_TOKEN is available as a Dagger secret

#### Scenario: Database URL fetched for Atlas

- **WHEN** Atlas needs to apply schema changes
- **THEN** DATABASE\_URL is fetched from GCP Secret Manager using `gcloud secrets versions access`

### Requirement: Workflow triggers on main branch push

The workflow SHALL run when called by workflows that trigger on push to main.

#### Scenario: Main branch triggers deploy

- **WHEN** code is pushed to main
- **THEN** the full pipeline runs (lint, test, build, migrate, deploy)

### Requirement: Workflow uses semantic-release

The workflow SHALL run semantic-release to determine version from commits.

#### Scenario: Version determined from commits

- **WHEN** the workflow runs on main
- **THEN** semantic-release analyzes commits and determines the next version

#### Scenario: No release on non-releasable commits

- **WHEN** commits are only `chore:` or `docs:`
- **THEN** no new version is created and deploy is skipped
