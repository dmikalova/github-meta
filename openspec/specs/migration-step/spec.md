## ADDED Requirements

### Requirement: Pipeline provides migrate function

The Dagger module SHALL provide a `migrate` function that runs database migrations.

#### Scenario: Migrations run via deno task

- **WHEN** the pipeline runs migrate
- **THEN** it executes `deno task migrate` in a Deno container

#### Scenario: DATABASE_URL injected

- **WHEN** migrations run
- **THEN** the DATABASE_URL secret is available as an environment variable

### Requirement: Migrations run before deploy

The pipeline SHALL execute migrations after publishing the image but before deploying to Cloud Run.

#### Scenario: Pipeline order

- **WHEN** the full pipeline runs
- **THEN** the order is: lint → test → build → publish → migrate → deploy

#### Scenario: Migration failure blocks deploy

- **WHEN** migrations fail
- **THEN** the deploy step does not run

### Requirement: Migrations are optional

The pipeline SHALL skip migrations if the app does not have a `migrate` task.

#### Scenario: No migrate task

- **WHEN** `deno.jsonc` does not define a `migrate` task
- **THEN** the pipeline skips the migration step

#### Scenario: Migrate task exists

- **WHEN** `deno.jsonc` defines a `migrate` task
- **THEN** the pipeline runs migrations

### Requirement: Migration output is visible

The pipeline SHALL output migration logs for debugging.

#### Scenario: Success logs shown

- **WHEN** migrations complete successfully
- **THEN** the migration output is visible in CI logs

#### Scenario: Failure logs shown

- **WHEN** migrations fail
- **THEN** the error output is visible in CI logs

### Requirement: Migrations run once per deploy

The pipeline SHALL run migrations exactly once, not per Cloud Run instance.

#### Scenario: Single execution

- **WHEN** a deploy triggers
- **THEN** migrations run once in CI before the deploy command

#### Scenario: Not at container startup

- **WHEN** a Cloud Run instance starts
- **THEN** it does not run migrations (already done in CI)
