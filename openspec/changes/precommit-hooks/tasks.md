# Tasks

## 1. Base Configuration

- [ ] 1.1 Create `lefthook-base.yml` in github-meta root with general checks
- [ ] 1.2 Add gitleaks secret detection
- [ ] 1.3 Add private key detection
- [ ] 1.4 Add merge conflict marker detection
- [ ] 1.5 Add large file check (5MB limit)
- [ ] 1.6 Add prohibited file type check (.sh files)
- [ ] 1.7 Add typos spell checker
- [ ] 1.8 Add markdownlint for documentation
- [ ] 1.9 Set up parallel execution within check categories

## 2. Commit Message Validation

- [ ] 2.1 Add commit-msg hook with commitlint
- [ ] 2.2 Create commitlint config for conventional commits
- [ ] 2.3 Configure allowed commit types (feat, fix, docs, style, refactor, test, chore, ci, perf,
      build, revert)
- [ ] 2.4 Test commit-msg hook blocks invalid messages

## 3. Deno Checks

- [ ] 3.1 Add Deno format check (`deno fmt --check`) on all files
- [ ] 3.2 Add Deno lint check (`deno lint`) on all files
- [ ] 3.3 Add Deno type check (`deno check`) on all files
- [ ] 3.4 Add Deno test execution (`deno test`)
- [ ] 3.5 Add skip conditions based on deno.json presence

## 4. Terraform Checks

- [ ] 4.1 Add Terraform format via `terramate run -- tofu fmt -check`
- [ ] 4.2 Add Terraform validation via `terramate run -- tofu validate`
- [ ] 4.3 Add skip conditions based on terramate.tm.hcl presence

## 5. github-meta Setup

- [ ] 5.1 Add lefthook as devDependency in package.json
- [ ] 5.2 Add commitlint and config as devDependencies
- [ ] 5.3 Add postinstall script to run `lefthook install`
- [ ] 5.4 Create local `lefthook.yml` that extends base config
- [ ] 5.5 Test pre-commit and commit-msg hooks work locally

## 6. Documentation

- [ ] 6.1 Add setup instructions to README (how repos consume base config)
- [ ] 6.2 Document bypass mechanism (`--no-verify`)
- [ ] 6.3 Document CI parity (same checks run locally and in CI)

## 7. Consumer Repos

- [ ] 7.1 Add lefthook to email-unsubscribe repo
- [ ] 7.2 Add lefthook to login repo
- [ ] 7.3 Add lefthook to infrastructure repo
- [ ] 7.4 Verify hooks work in each repo type (Deno, Terraform)

## 8. Verification

- [ ] 8.1 Test Deno checks block on lint errors
- [ ] 8.2 Test Deno checks block on format errors
- [ ] 8.3 Test secret detection blocks on mock secret
- [ ] 8.4 Test commit-msg blocks non-conventional message
- [ ] 8.5 Test prohibited file type blocks .sh files
- [ ] 8.6 Test terramate integration finds all stacks
