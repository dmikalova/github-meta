# Review

## Summary

The design is solid - lefthook is the right choice, auto-detection approach is
pragmatic, and the spec coverage is thorough. A few considerations around secret
detection tooling, test execution scope, and Node.js repo support are noted
below.

## Security

- [x] **Remote config fetch over HTTPS** → The `extends` URL uses HTTPS from
      GitHub, which is secure. No action needed.
- [x] **Secret detection needs tooling** → Spec mentions secret detection but
      doesn't specify how. Will use `gitleaks` (standard tool). **Address in
      tasks.**

## Patterns

- [x] **Consistent with existing workflows** → Pre-commit checks mirror what CI
      does (lint, fmt, test), maintaining consistency. No issues.

## Alternatives

- [ ] **Consider `gitleaks` for secret detection** → Rather than custom regex
      patterns, use `gitleaks` which has comprehensive patterns and is actively
      maintained. **Address in tasks** - integrate gitleaks as the secret
      detection mechanism.

## Simplifications

- [x] **Test execution scope** → Tests are fast enough to stay in pre-commit. No
      change needed.
- [x] **Terraform stack discovery** → Use Terramate for easy discovery:
      `terramate run -- tofu fmt -check` and `terramate run -- tofu validate`.
      **Address in tasks.**

## Missing Considerations

- [x] **No Node.js/npm repo support yet** → Design mentions it as future but
      proposal doesn't scope it out. **Defer** - explicitly out of scope for v1.
- [x] **lefthook installation mechanism** → Need to document how repos install
      lefthook (npm postinstall vs manual). **Address in tasks** - include setup
      documentation.
- [x] **CI parity** → Pre-commit runs same checks as CI. Running on all files
      (not staged) ensures identical behavior. **Address in design.**
- [x] **All files not staged** → Run checks on entire repo for CI parity.
      **Address in tasks.**
- [x] **Prohibited file types** → Block .sh scripts, require TypeScript.
      **Address in tasks.**

## Valuable Additions

- [x] **Commit message linting** → Add conventional commit format checking via
      commitlint. **Address in tasks.**
- [x] **Spell checking** → Add `typos` for fast code spell checking. **Address
      in tasks.**
- [x] **Markdown linting** → Add `markdownlint` for documentation quality.
      **Address in tasks.**
- [x] **Insensitive writing check** → Add `alex` to catch insensitive/
      inconsiderate language. **Address in tasks.**
- [x] **Code spell checking** → Add `cspell` for code-aware dictionaries.
      **Address in tasks.**
- [x] **Merge conflict detection** → Check for unresolved merge markers.
      **Address in tasks.**
- [x] **Private key detection** → Detect accidental private key commits.
      **Address in tasks.**
- [x] **Better formatters** → Explore Biome as opinionated, fast alternative to
      Prettier. **Address in design.**

## Action Items

Items to address in implementation:

1. Integrate `gitleaks` for secret detection
2. Run checks on all files (not staged-only) for CI parity
3. Document installation process for consuming repos
4. Use Terramate for Terraform stack discovery and validation
5. Add commit-msg hook with commitlint for conventional commits
6. Add additional quality checks: typos, markdownlint, alex, cspell
7. Add safety checks: merge conflict detection, private key detection
8. Block prohibited file types (.sh scripts)
9. Explore Biome for config file formatting

## Deferred Items

Items acknowledged but not in v1:

- Node.js/npm repo support
- Ticket number enforcement in commit messages

## Updates Required

Design updated with:

- Commit-msg hook in goals
- All files (not staged) execution mode
- Terramate for Terraform discovery
- Biome exploration
- Block prohibited file types
