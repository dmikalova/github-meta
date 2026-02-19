# Atlas Schema Workflow

## Requirements

### Requirement: Schema defined in HCL

The app SHALL define its database schema in a declarative HCL file.

#### Scenario: Schema file location

- **WHEN** an app has database tables
- **THEN** the schema is defined in `db/schema.hcl` at the repo root

#### Scenario: Schema scoped to app

- **WHEN** defining the schema
- **THEN** the schema SHALL only define tables in the app's dedicated PostgreSQL
  schema (e.g., `login`, `email_unsubscribe`)

### Requirement: Atlas runnable locally

Developers SHALL be able to run Atlas commands locally for schema operations.

#### Scenario: Local schema apply

- **WHEN** a developer runs `atlas schema apply` locally with DATABASE_URL set
- **THEN** Atlas diffs and applies schema changes to the target database

#### Scenario: Local schema diff

- **WHEN** a developer runs `atlas schema diff` locally
- **THEN** Atlas outputs the planned SQL without applying

#### Scenario: Deno task for local apply

- **WHEN** an app has a database schema
- **THEN** it SHALL provide a `deno task db:apply` that runs Atlas with the
  local DATABASE_URL

### Requirement: CI applies schema changes via Atlas

The workflow SHALL use Atlas to diff and apply schema changes during deploy.

#### Scenario: Atlas CLI setup

- **WHEN** the workflow runs
- **THEN** it uses `ariga/setup-atlas@v0` to install the Atlas CLI

#### Scenario: Schema diff against production

- **WHEN** Atlas runs
- **THEN** it compares `db/schema.hcl` against the production database schema

#### Scenario: Schema isolation with --schema flag

- **WHEN** Atlas commands run
- **THEN** they use `--schema <app_schema>` to limit introspection to the app's
  schema

### Requirement: Schema changes applied automatically

The workflow SHALL apply safe schema changes without manual intervention.

#### Scenario: Additive changes auto-applied

- **WHEN** schema changes only add tables, columns, or indexes
- **THEN** Atlas applies them automatically

#### Scenario: Destructive changes blocked by lint

- **WHEN** schema changes would drop tables or columns
- **THEN** the lint step SHALL fail and block the apply

#### Scenario: Manual apply for destructive changes

- **WHEN** lint fails due to destructive changes
- **THEN** the developer SHALL apply manually via `deno task db:apply` locally
- **AND** push again with no schema diff to continue deploy

### Requirement: Schema apply runs before deploy

The workflow SHALL execute schema changes after image push but before Cloud Run
deploy.

#### Scenario: Pipeline order

- **WHEN** the full pipeline runs
- **THEN** the order is: lint → test → build → publish → schema lint → schema
  apply → deploy

#### Scenario: Schema lint failure blocks deploy

- **WHEN** Atlas schema lint detects destructive changes
- **THEN** the deploy step does not run

#### Scenario: Schema apply failure blocks deploy

- **WHEN** Atlas schema apply fails
- **THEN** the deploy step does not run

### Requirement: Schema apply is optional

The workflow SHALL skip Atlas if the app has no schema file.

#### Scenario: No schema file

- **WHEN** `db/schema.hcl` does not exist
- **THEN** the workflow skips the Atlas step

#### Scenario: Schema file exists

- **WHEN** `db/schema.hcl` exists
- **THEN** the workflow runs Atlas schema apply

### Requirement: Database URL fetched from Secret Manager

The workflow SHALL fetch the database connection string from GCP Secret Manager.

#### Scenario: Secret naming convention

- **WHEN** fetching the database URL
- **THEN** the workflow uses secret name `{app-name}-database-url`

#### Scenario: Service account has access

- **WHEN** the workflow fetches the secret
- **THEN** the GitHub Actions deploy service account has
  `secretmanager.secretAccessor` role on the secret
