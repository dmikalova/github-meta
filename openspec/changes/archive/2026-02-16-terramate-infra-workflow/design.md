# Design

## Context

The infrastructure repository (`dmikalova/infrastructure`) has no CI/CD — all OpenTofu changes are
applied manually via `tofu apply`. The repo uses Terramate to organize stacks across multiple
providers (GCP, GitHub, Namecheap, Supabase). Workload Identity Federation is already configured
with a `github-actions-infra` service account bound to the infrastructure repo, with broad IAM
roles. The `github-meta` repo provides reusable GitHub Actions workflows (e.g., `deno-cloudrun.yaml`
for app deployments). GCP provider configurations impersonate the `tofu-ci` service account via
`impersonate_service_account`. Some stacks are tagged `manual` (e.g., Namecheap) to indicate they
should not run in CI. The repo already has a Terramate script `apply` that runs `tofu init`

- `tofu apply`.

## Goals / Non-Goals

**Goals:**

- Automated `tofu apply -auto-approve` on push to main
- Scheduled weekly apply (Monday midnight UTC) to reconcile drift
- Manual workflow dispatch for on-demand runs
- Reusable workflow in github-meta, callable from infrastructure repo
- WIF authentication using the existing `github-actions-infra` service account
- SOPS Age key stored in GCP Secret Manager (accessible to infra SA only, not app SAs)
- Exclude stacks tagged `manual` from CI runs
- Terramate `cicd` script for CI-specific apply with auto-approve
- Semantic versioning of infrastructure changes

**Non-Goals:**

- PR-based plan/review workflow (direct merge to main)
- Managing `manual`-tagged stacks in CI (e.g., Namecheap requires IP whitelisting)
- Parallel stack execution (Terramate handles ordering)
- Notification integrations (Slack, email, etc.)

## Decisions

### 1. Reusable workflow in github-meta

**Decision**: Create `.github/workflows/terramate-apply-all.yaml` as a `workflow_call` reusable
workflow, following the same pattern as `deno-cloudrun.yaml`.

**Alternatives considered**:

- *Workflow in the infrastructure repo directly*: Simpler but breaks the centralized workflow
  pattern. All reusable workflows live in github-meta.
- *Composite action*: Less flexible — can't define full jobs, permissions, or concurrency.

**Rationale**: Consistent with the existing pattern. Infrastructure repo calls the reusable workflow
with `secrets: inherit`, same as email-unsubscribe calls deno-cloudrun.

### 2. WIF auth with service account impersonation chain

**Decision**: The workflow authenticates as `github-actions-infra` via WIF. Since GCP providers are
configured to `impersonate_service_account = tofu-ci`, the `github-actions-infra` SA needs
`roles/iam.serviceAccountTokenCreator` on `tofu-ci` to impersonate it. This preserves existing
provider configuration without changes.

**Alternatives considered**:

- *Change provider config to use `github-actions-infra` directly*: Would require updating all
  generated `_terraform.tf` files and break local development where developers use `tofu-ci`
  impersonation.
- *Give `github-actions-infra` all the same roles as `tofu-ci` and remove impersonation*: Duplicates
  role management and diverges local vs CI behavior.

**Rationale**: The impersonation chain (`WIF → github-actions-infra → tofu-ci`) keeps provider
config stable for both local and CI usage. Only one IAM binding needs to be added.

### 3. Auto-apply on push to main

**Decision**: The workflow runs `terramate run --no-tags manual -- tofu apply
-auto-approve` on push
to main. No PR-based plan step — changes are merged directly to main and applied automatically.

**Alternatives considered**:

- *Plan on PR, apply on merge*: Adds a review step but this repo uses direct-to-main workflow.
- *Apply requires manual approval*: More safety but adds friction for a single-maintainer repo.

**Rationale**: For a personal infrastructure repo with a direct-to-main workflow, automatic apply is
appropriate. Plan output is visible in the Actions log for debugging.

### 4. Exclude `manual`-tagged stacks

**Decision**: The workflow uses `terramate run --no-tags manual` to exclude stacks tagged with
`manual`. This is the Terramate-native way to filter stacks.

**Alternatives considered**:

- *`disable = true` on stacks*: Terramate doesn't support disable as a default exclusion mechanism.
  The `manual` tag is more explicit and flexible.
- *Path-based exclusion*: Fragile and doesn't scale as new providers are added.

**Rationale**: Tags are the Terramate-native filtering mechanism. The `manual` tag clearly
communicates intent — stacks that require manual intervention (e.g., IP whitelisting) are excluded
from automated runs.

### 5. SOPS Age key in GCP Secret Manager, managed by OpenTofu

**Decision**: The Age private key for SOPS decryption is stored in GCP Secret Manager as a
`sops-age-key` secret. The secret is managed in OpenTofu — the value is read from
`secrets/age.sops.json` via the SOPS provider (`keys_file_base64`, base64-decoded). This creates a
bootstrap requirement: the first `tofu apply` must be run locally (where the user has the Age key),
after which CI can fetch the key from Secret Manager and decrypt SOPS files itself. The workflow
retrieves the key via `gcloud secrets versions access` and exports it as `SOPS_AGE_KEY`. The app
deploy SA (`github-actions-deploy`) does not have Secret Manager access, so app repos cannot access
the Age key.

**Alternatives considered**:

- *GitHub Actions secret (`SOPS_AGE_KEY`)*: Would need to be added to every repo that calls the
  workflow, and is harder to rotate.
- *GCP KMS for SOPS*: Would require re-encrypting all existing secrets and changing the SOPS config.
- *Manual `gcloud secrets versions add`*: Works but the secret value wouldn't be managed in
  OpenTofu, creating drift risk.

**Rationale**: Managing the secret in OpenTofu keeps it in the IaC lifecycle — rotating the Age key
just requires updating `age.sops.json` and running apply. The bootstrap is a one-time local apply.
Only the infra SA has access, maintaining the security boundary between infra and app runners.

### 6. Terramate `cicd` script

**Decision**: Add a `cicd` Terramate script alongside the existing `apply` script. The `cicd` script
runs `tofu init` + `tofu apply -auto-approve`, while the existing `apply` script remains interactive
(no `-auto-approve`).

**Alternatives considered**:

- *Modify the existing `apply` script*: Would change local behavior.
- *Pass `-auto-approve` via workflow command line*: Less repeatable and not captured in Terramate
  config.

**Rationale**: Separating `apply` (interactive) from `cicd` (non-interactive) keeps local and CI
usage distinct. The workflow calls `terramate run --no-tags manual script run cicd`.

### 7. Semantic versioning via semantic-release

**Decision**: Use `cycjimmy/semantic-release-action` with the existing
`@dmikalova/semantic-release-config`, same as other workflows. Only runs on main branch after
successful apply.

**Alternatives considered**:

- *Skip versioning*: Loses traceability of infrastructure changes.
- *Manual tagging*: Error-prone and inconsistent.

**Rationale**: Consistent with the deno-cloudrun workflow. Provides an audit trail of infrastructure
changes via GitHub releases.

### 8. Weekly scheduled apply and manual dispatch

**Decision**: The workflow triggers on `schedule: cron: '0 0 * * 1'` (Monday midnight UTC) and
`workflow_dispatch` for manual runs. Scheduled runs apply the same way as push-triggered runs —
`terramate run --no-tags manual script run
cicd`.

**Alternatives considered**:

- *Scheduled plan-only with drift alerting*: More conservative but doesn't auto-fix drift.
- *Daily schedule*: Too frequent for a personal project.

**Rationale**: Auto-applying on schedule reconciles any drift automatically. Monday midnight UTC
catches weekend drift. Manual dispatch enables on-demand runs when needed.

### 9. Concurrency control

**Decision**: Use GitHub Actions concurrency groups to prevent parallel applies. Queued runs wait
for the current run to complete (no cancellation).

**Rationale**: Prevents state corruption from concurrent `tofu apply` runs. Unlike CI for apps,
infrastructure applies should never be cancelled mid-run.

### 10. GitHub provider token from SOPS

**Decision**: The GitHub Terraform provider gets its token from SOPS (`secrets/github.sops.json`),
same as local usage. No need for `GITHUB_TOKEN` or a separate PAT.

**Rationale**: SOPS is already set up with the GitHub token. Using the same auth mechanism in CI and
locally keeps behavior consistent.

## Risks / Trade-offs

- **Service account impersonation chain**: Adds an extra hop (`github-actions-infra → tofu-ci`). →
  Low risk since both SAs are in the same project. Add `roles/iam.serviceAccountTokenCreator`
  binding.
- **Age key in Secret Manager**: Single point of failure for SOPS decryption. → Mitigated by Secret
  Manager's access controls and audit logging. Only infra SA has access.
- **Auto-apply on push and schedule**: No manual confirmation before apply. → Acceptable for a
  single-maintainer repo. Actions log provides full plan/apply output.
- **Non-GCP providers in CI**: GitHub and Supabase tokens come from SOPS, which requires the Age key
  to be available first in the workflow.
- **Scheduled apply failures**: May encounter transient provider errors. → Review Actions log;
  failures don't affect running infrastructure.

## Migration Plan

1. **Add Secret Manager resource**: Add `google_secret_manager_secret` and
   `google_secret_manager_secret_version` for `sops-age-key` in the baseline stack, reading the
   value from `age.sops.json` via SOPS provider.
2. **Add IAM binding**: Grant `github-actions-infra` the `roles/iam.serviceAccountTokenCreator` role
   on `tofu-ci` SA in the WIF stack.
3. **Bootstrap apply locally**: Run `tofu apply` in baseline to create the Secret Manager secret
   (requires local Age key).
4. **Add `cicd` Terramate script**: Add to `terramate.tm.hcl` in the infrastructure repo.
5. **Create reusable workflow**: Add `terramate-apply-all.yaml` to github-meta.
6. **Create caller workflow**: Add `.github/workflows/cicd.yaml` to infrastructure repo calling the
   reusable workflow.
7. **Push to main**: Verify the workflow can fetch the Age key, decrypt SOPS, and apply
   successfully.
