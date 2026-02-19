# Design

## Context

Currently, code quality checks run only in CI after commits are pushed. This means:

- Developers discover lint/format issues after pushing
- CI resources spent on obviously broken builds
- Fix-up commits pollute git history

Pre-commit hooks solve this by running checks locally before allowing commits. The challenge is
maintaining consistent configuration across multiple repos with different tech stacks (Deno,
Terraform, etc.).

## Goals / Non-Goals

**Goals:**

- Auto-detect repo type from file presence (deno.json, \*.tf, etc.)
- Run only relevant checks for detected stack
- Run same checks as CI/CD (parity between local and remote)
- Run checks on all files (not just staged) for consistency
- Block commits on failure (strict enforcement)
- Centralize base config in github-meta for consistency
- Zero-config experience - just extend base config
- Fast execution (parallel checks where possible)
- Enforce conventional commits via commit-msg hook

**Non-Goals:**

- Custom per-repo check configuration beyond extending
- IDE integration (rely on git hooks only)
- Ticket number enforcement in commit messages

## Decisions

### Decision: Use lefthook over Husky

**Choice:** lefthook (<https://github.com/evilmartians/lefthook>)

**Rationale:**

- Single Go binary - fast startup, no Node.js dependency at runtime
- Native `extends` support for remote configs (perfect for centralized config)
- Auto-installs via npm postinstall like Husky
- Supports `skip` based on file globs and custom conditions
- Parallel execution built-in

**Alternatives:**

- Husky: More popular but requires Node.js, no native remote extends
- pre-commit (Python): Requires Python runtime, different ecosystem
- simple-git-hooks: Lightweight but no auto-detection or extends

### Decision: Auto-detect via file presence

**Choice:** Detect repo type by checking for marker files in repo root.

Detection order (first match wins):

1. `deno.json` or `deno.jsonc` → Deno repo
2. `terramate.tm.hcl` or `*.tf` files → Terraform repo (use terramate for discovery)
3. `package.json` → Node.js repo (future)

**Rationale:** File presence is reliable, requires no configuration, and matches how developers
think about project types.

### Decision: Use Terramate for Terraform stack discovery

**Choice:** Use `terramate run` for discovering and running checks across Terraform stacks.

**Rationale:** Terramate already handles stack discovery and provides
`terramate run -- tofu fmt -check` and `terramate run -- tofu validate` which handles all stacks
automatically.

**Alternatives:**

- Manual find/glob patterns → Error-prone, misses stacks
- Only check root → Misses nested stacks

### Decision: Explore Biome for config file formatting

**Choice:** Evaluate Biome as a fast, opinionated formatter for JSON/config files.

**Rationale:** Biome is Rust-based, extremely fast, and provides opinionated defaults. While Deno
has its own formatter, Biome could handle non-Deno files (JSON, YAML) with better performance.

**Alternatives:**

- Prettier: Slower, more configurable (we want opinionated)
- dprint: Also fast, but Biome is gaining more traction

### Decision: Block prohibited file types

**Choice:** Block commits containing `.sh` shell scripts - all scripts should be TypeScript.

**Rationale:** Maintains consistency across repos. TypeScript scripts are type-safe, testable, and
work with Deno's permission model.

**Alternatives:**

- Explicit `type` field in config → Adds configuration burden
- Check all and skip failures → Slower, confusing output

### Decision: Parallel execution with fail-fast

**Choice:** Run checks in parallel within each category, fail-fast on first error.

**Rationale:** Maximizes speed while giving immediate feedback. Developer sees first failure quickly
rather than waiting for all checks.

### Decision: Centralized config with local extends

**Choice:** Base `lefthook.yml` in github-meta, repos use single-line extends.

```yaml
# In consuming repo - lefthook.yml
extends:
  - https://raw.githubusercontent.com/dmikalova/github-meta/main/lefthook-base.yml
```

**Rationale:**

- Single source of truth for check definitions
- Repos get updates automatically (main branch)
- Override possible via local additions

## Risks / Trade-offs

**\[Risk] Remote config fetch fails** → lefthook caches extended configs locally after first fetch.
Developers can work offline after initial setup.

**\[Risk] Breaking change in base config** → Could break commits in all repos. Mitigation: Test base
config changes carefully, consider versioned URLs for stability.

**\[Risk] Slow pre-commit on full repo** → Running on all files (not just staged) takes longer.
Mitigation: Parallel execution, caching where tools support it, tests are fast. Benefit is
consistency - same as CI.

**\[Trade-off] Strict blocking vs warnings** → Chose strict blocking for quality. Developers can
`--no-verify` for emergencies, but this is intentionally friction-full.

**\[Trade-off] All files vs staged only** → Chose all files for CI parity. Staged-only could miss
issues introduced by unstaged changes that interact with staged ones.

**\[Trade-off] Auto-detect vs explicit** → Auto-detect means less control but zero config. Chose
convention over configuration.
