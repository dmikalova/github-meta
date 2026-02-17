## Why

The infrastructure repository has no automated deployment - all Terraform/OpenTofu changes must be applied manually. This creates risk of drift, forgotten applies, and inconsistent state. A reusable workflow would enable automated infrastructure deployment with proper versioning, similar to the existing deno-cloudrun workflow for applications.

## What Changes

- Add a new reusable GitHub Actions workflow `terramate-deploy.yaml` in github-meta
- Workflow runs `terramate run -- tofu plan` on PRs and `terramate run -- tofu apply` on main
- Semantic versioning via semantic-release for infrastructure changes
- Weekly cron schedule to detect and reconcile any drift
- Workload Identity Federation authentication (same pattern as deno-cloudrun)

## Capabilities

### New Capabilities

- `terramate-workflow`: Reusable GitHub Actions workflow for Terramate/OpenTofu infrastructure deployment with plan on PR, apply on merge, and scheduled drift detection

### Modified Capabilities

<!-- No existing capabilities are being modified -->

## Impact

- **github-meta**: New workflow file `.github/workflows/terramate-apply-all.yaml`
- **infrastructure repo**: Will add `.github/workflows/cicd.yaml` that calls the reusable workflow
- **GCP**: Uses existing Workload Identity Federation setup and `github-actions-deploy` service account that is accessible via the infra-deploy topic on the repo
- **Dependencies**: Requires Terramate and OpenTofu available in workflow runner
