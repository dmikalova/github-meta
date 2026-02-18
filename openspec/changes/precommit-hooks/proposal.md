## Why

Commits with lint errors, formatting issues, or failing tests waste CI time and create noisy git
history with fix-up commits. A pre-commit hook that auto-detects repo type and runs the appropriate
checks catches issues before they're pushed, improving code quality and developer experience.

## What Changes

- Add lefthook as the pre-commit framework (modern, fast, supports config sharing via `extends`)
- Create base lefthook configuration in github-meta with auto-detection logic
- Define standard check suites for each repo type:
  - **Deno repos**: `deno fmt --check`, `deno lint`, `deno check`, `deno test`
  - **Terraform/OpenTofu repos**: `tofu fmt -check`, `tofu validate`, `tflint`
  - **General**: file size limits, no secrets committed, trailing whitespace
- Repos extend the base config - auto-detection runs only applicable checks
- Hooks block commits on failure (strict mode)

## Capabilities

### New Capabilities

- `precommit-hooks`: Shared pre-commit hook configuration with auto-detection of repo type and
  convention-based check execution

### Modified Capabilities

<!-- None - this is new infrastructure -->

## Impact

- **github-meta**: New `lefthook.yml` base config, documentation
- **All repos**: Add lefthook devDependency and `extends` config
- **Developer workflow**: Commits blocked until checks pass
- **CI**: Reduced failed builds from preventable issues
