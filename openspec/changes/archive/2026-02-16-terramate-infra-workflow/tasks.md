## 1. GCP Secret Manager Setup

- [x] 1.1 Add `sops-age-key` Secret Manager secret resource in `gcp/infra/baseline/main.tf`, reading the Age private key from `secrets/age.sops.json` via SOPS provider (`keys_file_base64`, base64-decoded)
- [x] 1.2 Run `tofu apply` locally in `gcp/infra/baseline` and `gcp/infra/workload-identity-federation` to bootstrap (requires local Age key)

## 2. IAM Bindings

- [x] 2.1 Grant `github-actions-infra` the `roles/iam.serviceAccountTokenCreator` role on `tofu-ci` SA in the WIF stack
- [x] 2.2 Verify `github-actions-infra` has `roles/secretmanager.admin` or `roles/secretmanager.secretAccessor` for the `sops-age-key` secret

## 3. Terramate cicd Script

- [x] 3.1 Add `script "cicd"` block to `terramate.tm.hcl` in the infrastructure repo with `tofu init` + `tofu apply -auto-approve`

## 4. Reusable Workflow

- [x] 4.1 Create `.github/workflows/terramate-apply-all.yaml` in github-meta with `workflow_call` trigger
- [x] 4.2 Add job step: checkout source
- [x] 4.3 Add job step: authenticate to GCP via WIF as `github-actions-infra`
- [x] 4.4 Add job step: fetch Age key from GCP Secret Manager and export `SOPS_AGE_KEY`
- [x] 4.5 Add job step: install Terramate and OpenTofu
- [x] 4.6 Add job step: run `terramate run --no-tags manual script run cicd`
- [x] 4.7 Add job step: run semantic-release via `cycjimmy/semantic-release-action`
- [x] 4.8 Configure concurrency group with `cancel-in-progress: false`

## 5. Caller Workflow

- [x] 5.1 Create `.github/workflows/cicd.yaml` in infrastructure repo
- [x] 5.2 Configure triggers: `push` to main, `schedule` cron `0 0 * * 1`, `workflow_dispatch`
- [x] 5.3 Call `dmikalova/github-meta/.github/workflows/terramate-apply-all.yaml@main` with `secrets: inherit`

## 6. Validation

- [x] 6.1 Push to main and verify the workflow triggers and applies successfully
- [x] 6.2 Verify manual-tagged stacks are skipped in the Actions log
- [x] 6.3 Trigger `workflow_dispatch` and verify it runs
