# project-standards

Conventions specific to this repository, which holds the standards every project
follows and the automation that applies them. See `docs/adr/` for the design and
`CONTEXT.md` for the vocabulary.

## Validating

Run `mage ci:fix && mage ci:check` before calling work done. This repo imports
its own `ci` package, so it checks itself with the same targets it gives every Go
project.

## Workflows and actions

| File                         | Purpose                                                    |
| ---------------------------- | ---------------------------------------------------------- |
| `workflows/cicd.yaml`        | The reusable CI/CD workflow every project calls            |
| `workflows/conform.yaml`     | Weekly conformance, and this repo's own action updates     |
| `workflows/self.yaml`        | This repo's own CI/CD, through `cicd.yaml`                 |
| `actions/detect`             | Detect a project's languages, gate and kind                |
| `actions/check`              | Run a project's checks, shared by cicd and conformance     |
| `actions/setup`              | Build the `project-standards` binary from the same ref     |
| `actions/annotations`        | Collect a run's warnings and notices for Discord           |

### Conventions

- Use Workload Identity Federation for GCP auth (no service account keys).
- Hardcode GCP project/region in workflows (convention over configuration).
- Reference this repo's actions by full path at `@main`: inside a reusable
  workflow, `./` resolves against the caller's repository.
- Automation that changes GitHub (branches, commits, repo listing) uses the `gh`
  CLI. Only this repo and infrastructure may use the full-permission GitHub
  token; every other project gets `PKG_READ_TOKEN`.
