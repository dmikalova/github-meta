# Reusable Workflow

## MODIFIED Requirements

### Requirement: Workflow passes secrets to Dagger

The workflow SHALL pass required secrets (GITHUB_TOKEN, DATABASE_URL) to the
Dagger pipeline.

#### Scenario: GitHub token available

- **WHEN** Dagger needs to push to GHCR
- **THEN** GITHUB_TOKEN is available as a Dagger secret

#### Scenario: Database URL fetched for Atlas

- **WHEN** Atlas needs to apply schema changes
- **THEN** DATABASE_URL is fetched from GCP Secret Manager using
  `gcloud secrets versions access`

## REMOVED Requirements

### Requirement: Workflow uses deno task migrate

**Reason**: Replaced by Atlas declarative schema management. Sequential
migration files are replaced by schema-as-code with automatic diffing.

**Migration**: Apps should:

1. Remove `migrations/` directory and `scripts/migrate.ts`
2. Remove `migrate` task from `deno.jsonc`
3. Add `db/schema.hcl` defining the declarative schema
