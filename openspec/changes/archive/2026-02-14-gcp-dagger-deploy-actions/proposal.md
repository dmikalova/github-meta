# Proposal

## Why

Deno apps need automated CI/CD to GCP Cloud Run. Currently email-unsubscribe has
placeholder workflows calling non-existent infrastructure. Need a reusable,
opinionated pipeline that:

- Standardizes CI/CD across all Deno apps
- Deploys to Cloud Run with keyless WIF authentication
- Runs database migrations safely before traffic shifts
- Eliminates per-repo pipeline maintenance

## What Changes

- Create Dagger pipeline (TypeScript SDK) for Deno apps: lint → test → build →
  migrate → deploy
- Build minimal containers: `deno compile` to standalone binary → distroless
  base image
- Create reusable GitHub Actions workflow that orchestrates the Dagger pipeline
- Use GHCR for container images (free for public repos)
- Run migrations as dedicated pipeline step before deploy
- Integrate semantic-release for automated versioning from conventional commits
- App repos call reusable workflow with \~5 lines of config
- App repos define `mklv.config.mts` with typed schema for all deployment
  settings

## Capabilities

### New Capabilities

- `dagger-deno-pipeline`: Dagger module with Deno-specific build/test/deploy
  functions
- `reusable-workflow`: GitHub Actions workflow that calls Dagger and handles WIF
  auth
- `config-schema`: TypeScript types for `mklv.config.mts` app configuration
- `migration-step`: Database migration execution in pipeline before traffic
  shift

### Modified Capabilities

None - github-meta is a new repo.

## Impact

- **github-meta repo**: New Dagger module at `dagger/deno/`, reusable workflow
  at `.github/workflows/deno-cloudrun.yaml`
- **App repos**: Replace existing workflows with single reusable workflow call
- **Cloud Run**: Receives deployments via Dagger + gcloud
- **GHCR**: Public images at `ghcr.io/dmikalova/<app-name>`
- **WIF**: Assumes `gcp-github-wif` infrastructure already exists (pool, service
  accounts)
- **Database**: Migrations run via `deno task db:migrate` before deploy
