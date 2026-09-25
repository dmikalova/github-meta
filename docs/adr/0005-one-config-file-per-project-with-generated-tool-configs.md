# 5. One config file per project, with generated tool configs

This decision records how a project customizes the standard, and why tool config files are generated rather than written by hand.

## Context

Every tool wanted its own config file (`.golangci.yaml`, `typos.toml`, `.markdownlint-cli2.yaml`, `commitlint.config.mjs`), and the copies drifted: vex, keyforge.cards and mklv.tech inline their commitlint rules, while infrastructure extends a shared preset. Most projects need nothing beyond the standard. golangci-lint can't extend or include another config file, and its maintainers have declined to add merging. Editors, however, read tool configs from their standard paths.

## Decision

**`mklv.config.json` is the only hand-edited configuration in a project, and it's optional. Base configs are embedded in project-standards. The merged result is written to each tool's standard path and committed.**

- **`mklv.config.json` holds:**
  - `kind`: cli, cloudrun, library or infra (ADR 0006)
  - deploy settings, as before
  - `tools.<tool>` overrides
  - ignore-file additions (ADR 0007)
- **Merge rules:**
  - Maps merge deeply.
  - Lists are appended to the base list.
  - Writing a list as `{"replace": [...]}` replaces the base list instead.
  - `null` removes a key.

  The object form keeps the file valid JSON and checkable against its schema, and makes replacing a list a visible choice.
- **Every `ci:fix` run, and every conformance-bot run, regenerates the files** at the tools' standard paths. Each file starts with a header saying it's generated from `mklv.config.json` and must not be edited. Tool-specific settings that aren't files, such as misspell's word lists, are passed as flags.
- **A hand edit to a generated file is drift.** The next fix overwrites it, so overrides must go in `mklv.config.json`. Editors and direct tool runs see exactly the configuration the gate uses.
- **Go-specific build steps** that aren't tool settings, such as vex's WebAssembly build and coverage-gated packages, are set in Go in the project's magefile.
