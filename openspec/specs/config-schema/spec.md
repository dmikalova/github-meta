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

### Requirement: Schema is importable

The schema SHALL be importable from a published package or direct URL.

#### Scenario: Import from package

- **WHEN** an app repo imports the schema
- **THEN** it can use `import type { MklvConfig } from "@dmikalova/mklv-config"`

### Requirement: Config file uses .mts extension

App repos SHALL define configuration in `mklv.config.mts` (TypeScript module).

#### Scenario: File is type-checked

- **WHEN** the config file is saved
- **THEN** TypeScript validates it against the MklvConfig type

#### Scenario: File is importable by Dagger

- **WHEN** the Dagger pipeline reads the config
- **THEN** it can import the `.mts` file directly
