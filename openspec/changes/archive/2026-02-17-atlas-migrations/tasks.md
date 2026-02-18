# Tasks

## 1. Infrastructure

- [x] 1.1 Verify GitHub Actions deploy SA has `secretmanager.secretAccessor` on `login-database-url`
      secret
- [x] 1.2 Run `tofu plan` in `gcp/apps/login` to confirm no changes needed

## 2. Reusable Workflow

- [x] 2.1 Update `deno-cloudrun.yaml` to add Atlas schema apply step after image publish
- [x] 2.2 Add conditional check to skip Atlas if `db/schema.hcl` doesn't exist
- [x] 2.3 Fetch DATABASE_URL from Secret Manager using `gcloud secrets versions access`
- [x] 2.4 Add `ariga/atlas-action/schema/apply@v1` action with lint for destructive changes

## 3. Login App

- [x] 3.1 Create `db/schema.hcl` with `schema "login"` block and `domain_logins` table
- [x] 3.2 Add `db:apply` and `db:diff` tasks to `deno.jsonc`
- [x] 3.3 Add Atlas CLI installation instructions to README
- [x] 3.4 Remove `migrations/` directory
- [x] 3.5 Remove `scripts/migrate.ts`
- [x] 3.6 Remove `migrate` task from `deno.jsonc`

## 4. Testing

- [x] 4.1 Test workflow in a branch with login app deploy
- [x] 4.2 Verify Atlas logs planned changes before applying
- [x] 4.3 Verify lint blocks destructive changes (test with DROP COLUMN)
