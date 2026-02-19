# Agent Guidelines

Conventions for AI coding agents working with this repository and Deno apps.

## Deno Conventions

### Permission Sets

Always use `--permission-set` instead of individual permission flags when
running Deno scripts:

```bash
# Good
deno run --permission-set src/main.ts
deno task dev  # tasks should use --permission-set internally

# Bad
deno run --allow-env --allow-net --allow-read src/main.ts
```

Permission sets are defined in `deno.jsonc` under the `permissions` key and
provide:

- Consistent permissions across development and production
- Self-documenting security requirements
- Easier auditing of what the app needs

### Task Definitions

When adding tasks to `deno.jsonc`:

```jsonc
{
  "tasks": {
    // Use --permission-set for runtime tasks
    "dev": "deno run --permission-set --watch src/main.ts",
    "start": "deno run --permission-set src/main.ts",

    // Scripts that need specific permissions can declare them
    "migrate": "deno run --allow-env --allow-net --allow-read scripts/migrate.ts"
  }
}
```

## Reusable Workflows

This repo contains GitHub Actions workflows reused by other repos:

| Workflow                   | Purpose                                 |
| -------------------------- | --------------------------------------- |
| `deno-cloudrun.yaml`       | Build and deploy Deno apps to Cloud Run |
| `npm-packages.yaml`        | Publish npm packages to GitHub Packages |
| `terramate-apply-all.yaml` | Apply Terramate stacks                  |

### Workflow Conventions

- Use Workload Identity Federation for GCP auth (no service account keys)
- Hardcode GCP project/region in workflows (convention over configuration)
- App repos call workflows with
  `uses: dmikalova/github-meta/.github/workflows/<name>@main`
