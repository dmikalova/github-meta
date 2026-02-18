## Context

Currently, database migrations use a sequential file approach (`migrations/001_*.sql`, `migrations/002_*.sql`) with a custom Deno runner script. This scatters schema history across many files, making it hard to understand "what is the schema now?"

Atlas provides declarative schema-as-code where a single HCL file defines the desired schema state. Atlas diffs this against the production database and generates/applies migrations automatically.

**Current state:**

- login app: Has `migrations/001_domain_logins.sql` and `scripts/migrate.ts`
- email-unsubscribe app: No migrations yet (good timing to start with Atlas)
- Workflow: deno-cloudrun.yaml has a "Database Migrate" step that runs `deno task migrate`

**Constraints:**

- Must work with Supabase PostgreSQL (each app has its own schema)
- GitHub Actions deploy service account needs secret access
- Atlas Community Edition (free) must suffice

## Goals / Non-Goals

**Goals:**

- Schema-as-code: Single `db/schema.hcl` shows current schema state
- Automatic diffing: Atlas computes required migrations
- CI/CD integration: Schema changes applied during deploy pipeline
- Local development: Developers can run Atlas locally for testing and manual applies
- Multi-app support: Works for login, email-unsubscribe, and future apps

**Non-Goals:**

- Data migrations (only schema DDL)
- Multi-environment drift detection (single prod environment)
- Atlas Cloud dashboard features (using CLI only)
- Backwards compatibility with existing migration files (clean break)

## Decisions

### Decision: Use Atlas CLI with setup-atlas action

**Choice:** Use `ariga/setup-atlas@v0` to install Atlas, then run manual `atlas schema` commands.

**Rationale:**

- Full control over CLI flags, especially `--schema` for Supabase schema isolation
- The `ariga/atlas-action` doesn't expose the `--schema` flag needed to avoid Supabase system table conflicts
- Three-step workflow (inspect, diff, apply) provides better visibility into what's happening
- Diff step can check for destructive changes via `--lint` flag

**Alternatives considered:**

- `ariga/atlas-action/schema/apply@v1` - Bundles everything but lacks `--schema` flag control
- Dagger integration - Adds complexity, GitHub Actions workflow is simpler

### Decision: Use HCL schema format (not SQL)

**Choice:** Define schemas in `db/schema.hcl` using Atlas HCL syntax.

**Rationale:**

- HCL is more expressive for constraints, indexes, relationships
- Consistent with infrastructure-as-code (Terraform/OpenTofu)
- Better tooling support (IDE completion, validation)

**Alternatives considered:**

- Plain SQL schema file - Works but less structured
- ORM models (e.g., Drizzle) - Requires runtime dependency

### Decision: Ephemeral dev database for planning

**Choice:** Use `docker://postgres/15` dev URL for Atlas planning.

**Rationale:**

- Atlas needs a database to load and introspect the schema
- Ephemeral container ensures clean state every run
- No persistent dev database to maintain

**Alternatives considered:**

- Supabase local container - Heavier, includes unnecessary services
- Schema registry - Atlas Cloud feature, not free

### Decision: Lint blocks destructive changes, auto-approve safe changes

**Choice:** Run `atlas schema lint` before apply. Lint fails on destructive changes, blocking CI. Safe changes auto-apply.

**Rationale:**

- Prevents accidental data loss (DROP TABLE, DROP COLUMN)
- Safe additive changes (CREATE TABLE, ADD COLUMN) apply automatically
- Destructive changes require manual local apply, then push again
- Human-in-the-loop for anything scary, automation for routine changes

**Workflow for destructive changes:**

1. Push schema change → CI lint fails
2. Review Atlas output locally: `deno task db:diff`
3. Apply manually if intentional: `DATABASE_URL=... deno task db:apply`
4. Push again → lint passes (no diff), deploy continues

**Alternatives considered:**

- Auto-approve everything with warnings - Risk of accidental drops
- Atlas Cloud approval workflow - Paid feature, overkill for personal projects

### Decision: App-scoped PostgreSQL schemas

**Choice:** Each app defines tables within its own PostgreSQL schema (`login`, `email_unsubscribe`).

**Rationale:**

- Matches existing Supabase app-database module setup
- Prevents Atlas from touching Supabase system tables (`auth.*`, `storage.*`)
- Clear ownership boundaries

**Schema naming convention:** The PostgreSQL schema name is derived from the app name by replacing dashes with underscores. This matches the Terraform `supabase/app-database` module convention:

- `login` → `login`
- `email-unsubscribe` → `email_unsubscribe`

**Important:** Atlas commands must use `--schema <schema_name>` (e.g., `--schema login`) to limit introspection to the app's schema. Without this, Atlas introspects all schemas and may encounter conflicts with Supabase system constraints. The workflow derives the schema name automatically from the app name.

**Schema bootstrap:** The HCL file must explicitly declare the schema block so Atlas creates the PostgreSQL schema namespace if it doesn't exist:

```hcl
schema "login" {
}

table "domain_logins" {
  schema = schema.login
  // columns...
}
```

### Decision: Deno tasks for local Atlas operations

**Choice:** Provide `deno task db:apply` and `deno task db:diff` for local schema operations.

**Rationale:**

- Developers can test schema changes locally before pushing
- Consistent interface across apps
- Can apply migrations manually when needed (e.g., initial setup, debugging)

**Local workflow:**

```bash
# View what would change
DATABASE_URL=... deno task db:diff

# Apply schema changes
DATABASE_URL=... deno task db:apply
```

**Prerequisites:** Atlas CLI must be installed locally. Add installation instructions to app READMEs:

```bash
# macOS
brew install ariga/tap/atlas

# Or via curl
curl -sSf https://atlasgo.sh | sh
```

**Alternatives considered:**

- Direct `atlas` commands - Works but requires remembering flags
- Makefile - Not idiomatic for Deno projects

### Decision: Do not commit generated migrations

**Choice:** Schema.hcl is the only versioned artifact. Generated SQL is ephemeral.

**Rationale:**

- schema.hcl IS the source of truth - no need for redundant migration files
- Avoids "which is correct?" confusion between schema and migrations
- Git history of schema.hcl provides full audit trail of schema evolution
- Keeps repo clean - no accumulating migration files

**Alternatives considered:**

- Commit generated SQL for audit - Redundant with schema.hcl history, adds noise

## Risks / Trade-offs

**[Risk] Destructive changes applied accidentally** → Mitigated by atlas lint blocking destructive changes in CI. Manual local apply required for intentional drops.

**[Risk] Dev container adds ~10-20s to CI** → Acceptable overhead for schema-as-code benefits. Can optimize with Docker layer caching if needed.

**[Risk] HCL learning curve** → Minor friction. HCL syntax is straightforward and well-documented.

**[Risk] Atlas version drift** → `setup-atlas@v0` may pull different versions over time. Mitigation: Pin to specific version if issues arise.

## Migration Plan

1. **Infrastructure** (first)
   - Verify GitHub Actions deploy SA has `secretmanager.secretAccessor` on `login-database-url` secret
   - Run `tofu apply` in `gcp/apps/login` if needed

2. **github-meta** (second)
   - Update `deno-cloudrun.yaml`: Replace "Database Migrate" step with "Schema Apply" using Atlas
   - Add setup-atlas action step

3. **login** (third)
   - Create `db/schema.hcl` matching current `domain_logins` table
   - Remove `migrations/` directory
   - Remove `scripts/migrate.ts`
   - Remove `migrate` task from `deno.jsonc`

4. **email-unsubscribe** (when ready)
   - Create `db/schema.hcl` when adding database tables

**Rollback strategy:**

- Revert workflow to use `deno task migrate`
- Restore migration files from git history
- No data loss since schema changes are additive

**Rollback procedure for failed schema changes:**

1. If schema apply fails mid-deploy, Cloud Run deploy is skipped (app keeps running old version)
2. Fix the schema.hcl error locally
3. Test with `deno task db:diff` to verify the fix
4. Push again to retry the deploy
5. If database is in inconsistent state, connect manually and fix with SQL, then update schema.hcl to match
