# github-meta

Shared configurations and reusable workflows for all repositories.

## Lefthook Pre-commit Hooks

This repo provides a base lefthook configuration that can be extended by other
repos.

### Setup for npm repos

```bash
npm install --save-dev lefthook
```

Create `lefthook.jsonc`:

```jsonc
{
  "remotes": [
    {
      "git_url": "https://github.com/dmikalova/github-meta",
      "ref": "main",
      "refetch": true,
      "configs": ["lefthook.jsonc"]
    }
  ]
}
```

Add postinstall script to `package.json`:

```json
{
  "scripts": {
    "postinstall": "lefthook install"
  }
}
```

### Setup for Deno repos

Install lefthook:

```bash
brew install lefthook
```

Create `lefthook.jsonc`:

```jsonc
{
  "remotes": [
    {
      "git_url": "https://github.com/dmikalova/github-meta",
      "ref": "main",
      "refetch": true,
      "configs": ["lefthook.jsonc"]
    }
  ]
}
```

Add setup task to `deno.jsonc`:

```jsonc
{
  "tasks": {
    "setup": "lefthook install"
  }
}
```

Run `deno task setup` after cloning.

### Setup for Terraform repos

Install lefthook:

```bash
brew install lefthook
```

Create `lefthook.jsonc` in the repo root:

```jsonc
{
  "remotes": [
    {
      "git_url": "https://github.com/dmikalova/github-meta",
      "ref": "main",
      "refetch": true,
      "configs": ["lefthook.jsonc"]
    }
  ]
}
```

Run `lefthook install` after cloning.

### Included Checks

**General (all repos):**

- `gitleaks` - Secret detection
- `check-merge-conflict` - Unresolved merge markers
- `detect-private-key` - Block .pem/.key files
- `check-large-files` - Block files > 5MB
- `no-shell-scripts` - Block .sh files (use TypeScript)
- `typos` - Auto-fix typos and stage
- `markdownlint` - Auto-fix markdown and stage

**Deno repos (when deno.json exists):**

- `deno fmt` - Auto-fix and stage
- `deno lint --fix` - Auto-fix and stage
- `deno check`
- `deno test`

**Terraform repos (when terramate.tm.hcl exists):**

- `terramate run -- tofu fmt` - Auto-fix and stage
- `terramate run -- tofu validate`

**Commit message:**

- `commitlint` - Conventional commit format

### Bypass

For emergency commits, use `--no-verify`:

```bash
git commit --no-verify -m "emergency fix"
```

### Manual Checks

Run all checks manually on all files:

```bash
lefthook run pre-commit --force
```

### CI Parity

Pre-commit hooks run the same checks as CI. Running on all files ensures
identical behavior between local development and CI pipelines.
