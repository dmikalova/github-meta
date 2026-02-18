# Config Schema

## ADDED Requirements

### Requirement: Schema defines app name

The MklvConfig type SHALL require a `name` field for the application identifier.

#### Scenario: Name used for image tagging

- **WHEN** the pipeline builds a container
- **THEN** it uses `config.name` in the image tag

#### Scenario: Name used for service name

- **WHEN** the pipeline deploys to Cloud Run
- **THEN** it uses `config.name` as the service name

### Requirement: Schema defines entrypoint

The MklvConfig type SHALL require an `entrypoint` field specifying the Deno entry file.

#### Scenario: Entrypoint used for compile

- **WHEN** the pipeline runs `deno compile`
- **THEN** it compiles `config.entrypoint`

### Requirement: Schema defines runtime configuration

The MklvConfig type SHALL include a `runtime` object for port and health check settings.

#### Scenario: Port has default

- **WHEN** `runtime.port` is not specified
- **THEN** the pipeline uses 8000 as default

#### Scenario: Health check path has default

- **WHEN** `runtime.healthCheckPath` is not specified
- **THEN** the pipeline uses `/health` as default

### Requirement: Schema is TypeScript interface

The schema SHALL be defined as a TypeScript interface that provides compile-time validation.

#### Scenario: IDE shows type errors

- **WHEN** a developer edits `mklv.config.mts` with invalid fields
- **THEN** the IDE shows type errors immediately

#### Scenario: Config uses satisfies keyword

- **WHEN** defining the config export
- **THEN** it uses `satisfies MklvConfig` for type checking

### Requirement: Schema is plain JSON

App repos SHALL define configuration in `mklv.config.json` (plain JSON, no TypeScript dependency).

#### Scenario: Config is readable by jq

- **WHEN** the CI pipeline reads the config
- **THEN** it uses `jq` to extract values without a runtime dependency

#### Scenario: No npm dependency needed

- **WHEN** an app repo defines its config
- **THEN** it does not require importing any npm package
