# Dagger Deno Pipeline

## ADDED Requirements

### Requirement: Pipeline provides lint function

The Dagger module SHALL provide a `lint` function that runs `deno lint` on the
source directory.

#### Scenario: Lint passes

- **WHEN** the pipeline runs lint on valid code
- **THEN** the function completes successfully with exit code 0

#### Scenario: Lint fails on violations

- **WHEN** the pipeline runs lint on code with linting errors
- **THEN** the function fails and outputs the lint errors

### Requirement: Pipeline provides check function

The Dagger module SHALL provide a `check` function that runs `deno check` for
TypeScript type checking.

#### Scenario: Type check passes

- **WHEN** the pipeline runs check on code with no type errors
- **THEN** the function completes successfully

#### Scenario: Type check fails on errors

- **WHEN** the pipeline runs check on code with type errors
- **THEN** the function fails and outputs the type errors

### Requirement: Pipeline provides test function

The Dagger module SHALL provide a `test` function that runs `deno task test`.

#### Scenario: Tests pass

- **WHEN** the pipeline runs test on code where all tests pass
- **THEN** the function completes successfully and outputs test results

#### Scenario: Tests fail

- **WHEN** the pipeline runs test on code where tests fail
- **THEN** the function fails and outputs the failing test details

### Requirement: Pipeline provides build function

The Dagger module SHALL provide a `build` function that compiles the Deno app to
a standalone binary and packages it in a minimal container.

#### Scenario: Build produces container

- **WHEN** the pipeline runs build with source directory and entrypoint
- **THEN** it returns a Container with the compiled binary

#### Scenario: Container uses distroless base

- **WHEN** the pipeline builds a container
- **THEN** the base image is `gcr.io/distroless/cc-debian12`

#### Scenario: Binary is standalone

- **WHEN** the container runs
- **THEN** it executes without requiring a Deno runtime

### Requirement: Pipeline provides publish function

The Dagger module SHALL provide a `publish` function that pushes a container to
GHCR.

#### Scenario: Image pushed to GHCR

- **WHEN** the pipeline publishes a container with name and version
- **THEN** it pushes to `ghcr.io/dmikalova/{name}:{version}`

#### Scenario: Authentication uses GitHub token

- **WHEN** the pipeline authenticates to GHCR
- **THEN** it uses the provided GitHub token secret

### Requirement: Pipeline provides deploy function

The Dagger module SHALL provide a `deploy` function that deploys a container to
Cloud Run.

#### Scenario: Deploy to Cloud Run

- **WHEN** the pipeline deploys with image, service name, and region
- **THEN** it runs `gcloud run deploy` with the specified parameters

#### Scenario: Deploy uses gcloud CLI

- **WHEN** the pipeline deploys
- **THEN** it uses the `google/cloud-sdk` container image

### Requirement: Pipeline reads config from source

The Dagger module SHALL read `mklv.config.mts` from the source directory to get
app configuration.

#### Scenario: Config parsed at runtime

- **WHEN** the pipeline starts
- **THEN** it imports and parses `mklv.config.mts` from the source directory

#### Scenario: Build uses config entrypoint

- **WHEN** the pipeline builds
- **THEN** it uses `config.entrypoint` as the compile target

#### Scenario: Deploy uses config name

- **WHEN** the pipeline deploys
- **THEN** it uses `config.name` as the Cloud Run service name
