# Terramate Workflow

## Purpose

Reusable GitHub Actions workflow for applying Terramate + OpenTofu infrastructure stacks.
Authenticates to GCP via Workload Identity Federation, decrypts secrets with SOPS/Age, and runs
`terramate script run cicd` to apply all non-manual stacks.

## Requirements

### Requirement: Workflow is reusable

The workflow SHALL be defined as a reusable workflow (`workflow_call`) in github-meta at
`.github/workflows/terramate-apply-all.yaml`, callable from other repos with `secrets: inherit`.

#### Scenario: Called from infrastructure repo

- **WHEN** the infrastructure repo workflow uses
  `dmikalova/github-meta/.github/workflows/terramate-apply-all.yaml@main`
- **THEN** the workflow executes in the context of the calling repo

### Requirement: Workflow triggers on push, schedule, and manual dispatch

The caller workflow SHALL trigger on push to main, weekly cron schedule, and manual dispatch.

#### Scenario: Push to main triggers apply

- **WHEN** code is pushed to main branch
- **THEN** the workflow runs and applies all non-manual stacks

#### Scenario: Weekly schedule triggers apply

- **WHEN** the cron schedule fires at Monday midnight UTC (`0 0 * * 1`)
- **THEN** the workflow runs and applies all non-manual stacks to reconcile drift

#### Scenario: Manual dispatch triggers apply

- **WHEN** a user triggers `workflow_dispatch`
- **THEN** the workflow runs and applies all non-manual stacks

### Requirement: Workflow authenticates to GCP via WIF

The workflow SHALL authenticate to GCP using Workload Identity Federation with the
`github-actions-infra` service account.

#### Scenario: WIF authentication succeeds

- **WHEN** the workflow runs on the `dmikalova/infrastructure` repo
- **THEN** it obtains GCP credentials for `github-actions-infra` via WIF without stored secrets

#### Scenario: Impersonation chain to tofu-ci

- **WHEN** OpenTofu runs with providers configured to `impersonate_service_account = tofu-ci`
- **THEN** `github-actions-infra` impersonates `tofu-ci` via the
  `roles/iam.serviceAccountTokenCreator` role

### Requirement: Age key managed in OpenTofu via SOPS provider

The `sops-age-key` Secret Manager secret SHALL be managed in OpenTofu, with its value read from
`secrets/age.sops.json` via the SOPS provider. This bootstraps CI access after a single local apply.

#### Scenario: Secret created from SOPS-encrypted source

- **WHEN** `tofu apply` runs in the baseline stack (locally, with the Age key available)
- **THEN** the Age private key is read from `age.sops.json` (`keys_file_base64`, base64-decoded) and
  stored in Secret Manager as `sops-age-key`

#### Scenario: Key rotation via SOPS

- **WHEN** the Age key is rotated in `age.sops.json`
- **THEN** running `tofu apply` updates the Secret Manager secret version

### Requirement: Workflow retrieves SOPS Age key from GCP Secret Manager

The workflow SHALL fetch the Age private key from GCP Secret Manager and export it as `SOPS_AGE_KEY`
for SOPS decryption.

#### Scenario: Age key retrieved successfully

- **WHEN** the workflow authenticates via WIF as `github-actions-infra`
- **THEN** it retrieves the `sops-age-key` secret from GCP Secret Manager and exports `SOPS_AGE_KEY`

#### Scenario: App deploy SA cannot access Age key

- **WHEN** a workflow authenticates as `github-actions-deploy` (app SA)
- **THEN** it SHALL NOT have access to the `sops-age-key` secret

### Requirement: Workflow checks out source

The workflow SHALL check out the calling repository's source code.

#### Scenario: Source available to Terramate

- **WHEN** the workflow runs
- **THEN** the infrastructure repo's source code is available at the workspace root

### Requirement: Workflow installs Terramate and OpenTofu

The workflow SHALL install Terramate and OpenTofu CLI tools.

#### Scenario: Tools available for stack execution

- **WHEN** the workflow runs
- **THEN** `terramate` and `tofu` commands are available on PATH

### Requirement: Workflow runs Terramate cicd script excluding manual stacks

The workflow SHALL execute `terramate run --no-tags manual script run cicd` to apply all non-manual
stacks with auto-approve.

#### Scenario: All non-manual stacks applied

- **WHEN** the workflow runs the cicd script
- **THEN** every stack without the `manual` tag runs `tofu init` followed by
  `tofu apply -auto-approve`

#### Scenario: Manual-tagged stacks skipped

- **WHEN** a stack has `tags = ["manual"]` (e.g., Namecheap)
- **THEN** it is excluded from the CI run

### Requirement: Terramate cicd script exists

The infrastructure repo SHALL define a Terramate script named `cicd` that runs `tofu init` and
`tofu apply -auto-approve`.

#### Scenario: cicd script runs non-interactively

- **WHEN** `terramate script run cicd` is invoked
- **THEN** it executes `tofu init` followed by `tofu apply -auto-approve` without interactive
  prompts

#### Scenario: Existing apply script unchanged

- **WHEN** a user runs `terramate script run apply` locally
- **THEN** it still runs `tofu init` followed by `tofu apply` (interactive, no auto-approve)

### Requirement: Workflow uses concurrency control

The workflow SHALL use GitHub Actions concurrency groups to prevent parallel applies, with no
cancellation of in-progress runs.

#### Scenario: Concurrent runs queue

- **WHEN** a second workflow run is triggered while one is in progress
- **THEN** the second run waits for the first to complete (no cancellation)

#### Scenario: State lock protection

- **WHEN** only one run executes at a time
- **THEN** OpenTofu state lock conflicts are prevented

### Requirement: Workflow runs semantic-release

The workflow SHALL run semantic-release after successful apply to version infrastructure changes.

#### Scenario: Version determined from commits

- **WHEN** the workflow runs on main after a successful apply
- **THEN** semantic-release analyzes commits and creates a release if warranted

#### Scenario: No release on non-releasable commits

- **WHEN** commits are only `chore:` or `docs:`
- **THEN** no new version is created

### Requirement: GitHub provider authenticates via SOPS

The GitHub Terraform provider SHALL get its token from SOPS (`secrets/github.sops.json`), not from
`GITHUB_TOKEN` or separate PAT.

#### Scenario: GitHub provider uses SOPS token

- **WHEN** OpenTofu runs stacks that use the GitHub provider
- **THEN** the provider reads its token from the SOPS-encrypted `github.sops.json` file

### Requirement: Plan output visible in Actions log

The workflow SHALL output plan/apply logs to the GitHub Actions log for debugging.

#### Scenario: Apply output visible

- **WHEN** `tofu apply -auto-approve` runs for each stack
- **THEN** the full output (including plan summary) is visible in the Actions log
