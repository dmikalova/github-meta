## Why

Sequential migration files scatter schema history across many files, making it hard to see "what is the schema now?" Atlas provides declarative schema-as-code where the schema definition IS the source of truth, and migrations are generated automatically by diffing against the live database.

## What Changes

- Add Atlas schema definition files (`db/schema.hcl`) to login and email-unsubscribe repos
- Add Deno tasks (`db:apply`, `db:diff`) for local Atlas operations
- Update the deno-cloudrun reusable workflow to run Atlas schema apply before Cloud Run deploy
- Remove the existing sequential migration system from login (scripts/migrate.ts, migrations/)
- Grant GitHub Actions service account access to database secrets for schema operations

## Capabilities

### New Capabilities

- `atlas-schema-workflow`: Declarative schema management using Atlas in CI/CD. Schema changes are defined in HCL, diffed against production, and applied automatically during deploy.

### Modified Capabilities

- `reusable-workflow`: Update deno-cloudrun.yaml to include Atlas schema apply step instead of deno task migrate

## Impact

- **github-meta**: deno-cloudrun.yaml workflow - replace migrate step with Atlas
- **login**: Add db/schema.hcl, remove migrations/ and scripts/migrate.ts
- **email-unsubscribe**: Add db/schema.hcl (when schema exists)
- **infrastructure**: Ensure GitHub Actions deploy SA has secretmanager.secretAccessor on database secrets
- **Dependencies**: Atlas CLI (installed via ariga/setup-atlas action)
