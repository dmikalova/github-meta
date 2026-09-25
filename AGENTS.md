# project-standards

Conventions specific to this repository, which holds the standards every project
follows and the reusable GitHub Actions workflows that apply them. See
`docs/adr/` for the design.

## Reusable Workflows

This repo contains GitHub Actions workflows reused by other repos:

| Workflow                   | Purpose                                 |
| -------------------------- | --------------------------------------- |
| `deno-cloudrun.yaml`       | Build and deploy Deno apps to Cloud Run |
| `go-cloudrun.yaml`         | Build and deploy Go apps to Cloud Run   |
| `npm-packages.yaml`        | Publish npm packages to GitHub Packages |
| `terramate-apply-all.yaml` | Apply Terramate stacks                  |

### Workflow Conventions

- Use Workload Identity Federation for GCP auth (no service account keys)
- Hardcode GCP project/region in workflows (convention over configuration)
- App repos call workflows with
  `uses: dmikalova/project-standards/.github/workflows/<name>@main`
