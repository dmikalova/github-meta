# Design

## Context

New `dmikalova/github-meta` repo to host reusable CI/CD infrastructure for Deno apps deploying to
GCP Cloud Run. Currently email-unsubscribe has placeholder workflows calling non-existent
infrastructure. All Deno apps should use identical CI/CD patterns with minimal per-repo
configuration.

Dependencies:

- WIF infrastructure from `gcp-github-wif` in infrastructure repo (pool, service accounts)
- App repos follow conventional commits for semantic versioning

## Goals / Non-Goals

**Goals:**

- Opinionated Deno CI/CD: lint → test → build → migrate → deploy
- Minimal container images via `deno compile` + distroless
- Keyless GCP auth via WIF (no stored credentials)
- Automated versioning via semantic-release
- App repos need only `mklv.config.mts` + minimal workflow call

**Non-Goals:**

- Supporting non-Deno runtimes (this is Deno-specific by design)
- Multi-region deployments
- Private container registries
- App-specific build customization beyond deno tasks

## Decisions

### 1. Repository Structure

**Decision:** `dagger/deno/` module with reusable workflow at
`.github/workflows/deno-cloudrun.yaml`.

```text
github-meta/
├── .github/
│   └── workflows/
│       └── deno-cloudrun.yaml    # Reusable workflow
├── dagger/
│   └── deno/
│       ├── src/
│       │   └── index.ts          # Dagger pipeline
│       └── dagger.json
├── src/
│   └── schema.ts                 # MklvConfig type (publishable)
└── README.md
```

**Rationale:** Centralized CI/CD logic. App repos stay clean - changes to build process don't
require updating every app repo.

### 2. Container Build Strategy

**Decision:** `deno compile` → standalone binary → `gcr.io/distroless/cc-debian12`.

```typescript
// Build stage: compile to standalone binary
const builder = dag
  .container()
  .from('denoland/deno:2.1.0')
  .withDirectory('/app', source)
  .withWorkdir('/app')
  .withExec(['deno', 'compile', '--output', 'app', entrypoint]);

// Runtime stage: minimal distroless (~20MB total)
return dag
  .container()
  .from('gcr.io/distroless/cc-debian12')
  .withFile('/app', builder.file('/app/app'))
  .withExposedPort(8000)
  .withEntrypoint(['/app']);
```

**Alternatives considered:**

- Full Deno image (~350MB): Too large, slow pulls
- Alpine + Deno: Still needs runtime, ~150MB
- Cloud Run source deploy: Less control, slower builds

**Rationale:** Smallest possible image. Single static binary with no runtime dependencies. Fast cold
starts. Secure (no shell, no package manager).

### 3. Container Registry

**Decision:** GitHub Container Registry (ghcr.io).

```text
ghcr.io/dmikalova/email-unsubscribe:1.2.3
```

**Alternatives considered:**

- GCP Artifact Registry: Requires auth, costs money
- Docker Hub: Rate limits, less GitHub integration

**Rationale:** Free for public repos, integrated with GitHub auth, Cloud Run can pull public images
directly.

### 4. Database Migrations

**Decision:** Run `deno task db:migrate` as pipeline step before traffic shift.

```typescript
@func()
async migrate(source: Directory, databaseUrl: Secret): Promise<string> {
  return dag.container()
    .from("denoland/deno:2.1.0")
    .withDirectory("/app", source)
    .withWorkdir("/app")
    .withSecretVariable("DATABASE_URL", databaseUrl)
    .withExec(["deno", "task", "db:migrate"])
    .stdout();
}
```

**Pipeline order:**

1. test (fails fast on broken code)
2. build (create container)
3. publish (push to GHCR)
4. migrate (run schema changes)
5. deploy (update Cloud Run)

**Rationale:**

- Runs once per deploy, not per instance
- Failure blocks deploy (visible in CI)
- Uses same task convention as local dev
- Migrations complete before new code receives traffic

### 5. Versioning Strategy

**Decision:** semantic-release with conventional commits.

```text
feat: add new feature    → 1.x.0 (minor)
fix: bug fix             → 1.0.x (patch)
feat!: breaking change   → x.0.0 (major)
chore: maintenance       → no release
```

**Rationale:** Automatic versioning from commit messages. Creates GitHub releases with changelogs.
App repos already use conventional commits.

### 6. Deno Task Conventions

**Decision:** App repos must provide standard deno tasks.

```json
{
  "tasks": {
    "check": "deno check src/main.ts",
    "lint": "deno lint",
    "test": "deno test --allow-env --allow-read",
    "db:migrate": "deno run -A src/db/migrate.ts"
  }
}
```

**Required tasks:**

- `check`: Type checking
- `lint`: Linting
- `test`: Unit/integration tests

**Optional tasks:**

- `db:migrate`: Database migrations (skipped if not present)

**Rationale:** Standardized interface. Pipeline calls these tasks, doesn't need app-specific
knowledge.

### 7. App Configuration Schema

**Decision:** App repos define `mklv.config.mts` with only app runtime concerns. The pipeline
derives GCP settings from conventions.

```typescript
// github-meta/src/schema.ts - published as @dmikalova/mklv-config
export interface MklvConfig {
  /** App name - used for image tagging and Cloud Run service name */
  name: string;
  /** Deno entrypoint file */
  entrypoint: string;
  /** Runtime configuration */
  runtime: {
    /** Port the app listens on (default: 8000) */
    port?: number;
    /** Health check endpoint path (default: /health) */
    healthCheckPath?: string;
  };
}
```

```typescript
// email-unsubscribe/mklv.config.mts
import type { MklvConfig } from '@dmikalova/mklv-config';

export default {
  name: 'email-unsubscribe',
  entrypoint: 'src/main.ts',
  runtime: {
    port: 8000,
    healthCheckPath: '/health',
  },
} satisfies MklvConfig;
```

**Pipeline derives GCP settings from conventions:**

- GCP project: `dmikalova-mklv` (hardcoded in pipeline)
- Cloud Run service: `{config.name}`
- Region: `us-west1` (default)
- WIF provider: known from infrastructure repo
- WIF service account: `github-actions-deploy@dmikalova-mklv.iam.gserviceaccount.com`

**Rationale:**

- App declares what it is, not where it deploys
- Type safety at authorship (IDE catches errors)
- GCP infrastructure is a pipeline concern, not app concern
- Changing deploy target doesn't require app changes

### 8. Reusable Workflow Interface

**Decision:** Minimal workflow - just calls Dagger which reads config.

```yaml
# App repo: .github/workflows/ci.yaml
name: CI/CD

on:
  push:
    branches: [main]

jobs:
  pipeline:
    uses: dmikalova/github-meta/.github/workflows/deno-cloudrun.yaml@main
    secrets: inherit
```

**No inputs required** - Dagger reads `mklv.config.mts` for app settings, uses hardcoded GCP
conventions.

The reusable workflow:

1. Checks out app repo
2. Authenticates to GCP using WIF (hardcoded provider/SA)
3. Runs Dagger pipeline which reads app config for name, entrypoint, runtime

**Rationale:** ~5 lines in app repo. App config is purely about the app, not infrastructure.

## Risks / Trade-offs

| Risk                            | Mitigation                                    |
| ------------------------------- | --------------------------------------------- |
| Dagger cold start (~30s)        | Built-in caching, acceptable for CI           |
| Migration failures block deploy | Intentional - fail fast is correct behavior   |
| Reusable workflow versioning    | Pin to `@main`, create tags for stability     |
| GHCR rate limits                | Generous for public repos, cache in Dagger    |
| Config schema changes           | Semantic versioning on @dmikalova/mklv-config |

## Migration Plan

### Phase 1: github-meta Setup

1. Create `dmikalova/github-meta` repo (public)
2. Create `src/schema.ts` with `MklvConfig` type
3. Initialize Dagger module at `dagger/deno/`
4. Create reusable workflow at `.github/workflows/deno-cloudrun.yaml`
5. Test locally: `dagger call build --source=../email-unsubscribe`

### Phase 2: email-unsubscribe Migration

1. Create `mklv.config.mts` with GCP/WIF settings
2. Replace `.github/workflows/ci.yaml` with reusable workflow call
3. Delete `.github/workflows/deploy.yml` (Northflank)
4. Delete `deploy.config.ts` (superseded by mklv.config.mts)
5. Push to trigger first automated pipeline

## Open Questions

None - design is complete.
