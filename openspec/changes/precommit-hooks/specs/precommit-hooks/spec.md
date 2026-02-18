# Precommit Hooks Spec

## ADDED Requirements

### Requirement: Auto-detect repo type

The system SHALL auto-detect repository type based on file presence.

#### Scenario: Deno repo detection

- **WHEN** the repository contains `deno.json` or `deno.jsonc` in the root
- **THEN** the system SHALL classify the repo as a Deno project
- **AND** enable Deno-specific checks

#### Scenario: Terraform repo detection

- **WHEN** the repository contains `.tf` files in root or subdirectories
- **AND** `deno.json` is not present
- **THEN** the system SHALL classify the repo as a Terraform project
- **AND** enable Terraform-specific checks

#### Scenario: Unknown repo type

- **WHEN** no known marker files are detected
- **THEN** the system SHALL run only general checks (no stack-specific checks)

### Requirement: Execute Deno checks

The system SHALL execute standard Deno quality checks for Deno repositories.

#### Scenario: Deno format check

- **WHEN** committing in a Deno repository
- **THEN** the system SHALL run `deno fmt --check` on all source files
- **AND** block the commit if formatting issues are found

#### Scenario: Deno lint check

- **WHEN** committing in a Deno repository
- **THEN** the system SHALL run `deno lint` on all source files
- **AND** block the commit if lint errors are found

#### Scenario: Deno type check

- **WHEN** committing in a Deno repository
- **THEN** the system SHALL run `deno check` on all TypeScript files
- **AND** block the commit if type errors are found

#### Scenario: Deno test execution

- **WHEN** committing in a Deno repository
- **THEN** the system SHALL run `deno test` if test files exist
- **AND** block the commit if tests fail

### Requirement: Execute Terraform checks

The system SHALL execute standard Terraform quality checks via Terramate.

#### Scenario: Terraform format check

- **WHEN** committing in a Terraform repository
- **THEN** the system SHALL run `terramate run -- tofu fmt -check` on all stacks
- **AND** block the commit if formatting issues are found

#### Scenario: Terraform validation

- **WHEN** committing in a Terraform repository
- **THEN** the system SHALL run `terramate run -- tofu validate` in all stacks
- **AND** block the commit if validation fails

### Requirement: Execute general checks

The system SHALL execute general quality checks regardless of repo type.

#### Scenario: Large file prevention

- **WHEN** committing a file larger than 5MB
- **THEN** the system SHALL block the commit
- **AND** display a warning about the file size

#### Scenario: Secret detection

- **WHEN** committing files containing patterns matching known secret formats
- **THEN** the system SHALL block the commit via gitleaks
- **AND** identify the suspected secret location

#### Scenario: Private key detection

- **WHEN** committing files containing private key patterns
- **THEN** the system SHALL block the commit
- **AND** identify the private key file

#### Scenario: Merge conflict detection

- **WHEN** committing files containing unresolved merge conflict markers
- **THEN** the system SHALL block the commit
- **AND** identify the conflicting files

#### Scenario: Prohibited file types

- **WHEN** committing `.sh` shell script files
- **THEN** the system SHALL block the commit
- **AND** suggest using TypeScript scripts instead

#### Scenario: Spell checking

- **WHEN** committing source code files
- **THEN** the system SHALL run typos spell checker
- **AND** block the commit if typos are found

#### Scenario: Markdown linting

- **WHEN** committing Markdown files
- **THEN** the system SHALL run markdownlint
- **AND** block the commit if linting errors are found

### Requirement: Parallel execution

The system SHALL execute checks in parallel for performance.

#### Scenario: Concurrent check execution

- **WHEN** multiple checks are configured
- **THEN** the system SHALL execute them in parallel
- **AND** report all failures (not just the first)

#### Scenario: Fail-fast within category

- **WHEN** a check fails
- **THEN** the system SHALL continue other parallel checks
- **AND** block the commit after all parallel checks complete

### Requirement: Config extensibility

The system SHALL support centralized configuration with local extensions.

#### Scenario: Extend remote config

- **WHEN** a repository's `lefthook.yml` contains an `extends` directive
- **THEN** the system SHALL fetch and merge the remote configuration
- **AND** cache the config locally for offline use

#### Scenario: Local override

- **WHEN** a repository defines checks that conflict with extended config
- **THEN** the local definition SHALL take precedence

### Requirement: Bypass mechanism

The system SHALL provide an escape hatch for emergency commits.

#### Scenario: Skip with flag

- **WHEN** a developer commits with `--no-verify` flag
- **THEN** the system SHALL skip all pre-commit hooks
- **AND** allow the commit to proceed

#### Scenario: Skip logging

- **WHEN** hooks are bypassed via `--no-verify`
- **THEN** the system SHALL NOT log or record the bypass (git native behavior)

### Requirement: Validate commit messages

The system SHALL validate commit messages follow conventional commit format.

#### Scenario: Valid conventional commit

- **WHEN** a developer writes a commit message following conventional format
- **THEN** the system SHALL allow the commit to proceed

#### Scenario: Invalid commit message format

- **WHEN** a developer writes a commit message not following conventional format
- **THEN** the system SHALL block the commit via commitlint
- **AND** display guidance on the required format

#### Scenario: Allowed commit types

- **WHEN** validating commit type prefix
- **THEN** the system SHALL accept: feat, fix, docs, style, refactor, test, chore, ci, perf, build,
  revert
