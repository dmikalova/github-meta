## 1. Config Schema

- [x] 1.1 Create `src/schema.ts` with MklvConfig interface
- [x] 1.2 Add deno.jsonc with package name for publishing
- [ ] 1.3 Test schema import works from external repo

## 2. Dagger Module Setup

- [x] 2.1 Initialize Dagger module at `dagger/deno/` with `dagger init`
- [x] 2.2 Create `dagger/deno/src/index.ts` with module skeleton
- [x] 2.3 Implement `lint` function (runs `deno lint`)
- [x] 2.4 Implement `check` function (runs `deno check`)
- [x] 2.5 Implement `test` function (runs `deno task test`)
- [x] 2.6 Implement `build` function (deno compile → distroless container)
- [x] 2.7 Implement `publish` function (push to GHCR)
- [x] 2.8 Implement `migrate` function (runs `deno task migrate` with DATABASE_URL)
- [x] 2.9 Implement `deploy` function (gcloud run deploy)
- [x] 2.10 Add config reading from `mklv.config.mts`

## 3. Reusable Workflow

- [x] 3.1 Create `.github/workflows/deno-cloudrun.yaml` as reusable workflow
- [x] 3.2 Add checkout step for app repo source
- [x] 3.3 Add GCP WIF authentication step with hardcoded provider/SA
- [x] 3.4 Add semantic-release step for version determination
- [x] 3.5 Add Dagger pipeline invocation with secrets
- [x] 3.6 Configure job to skip deploy if no release

## 4. Local Testing

- [x] 4.1 Test `dagger call lint --source=../email-unsubscribe`
- [x] 4.2 Test `dagger call check --source=../email-unsubscribe`
- [x] 4.3 Test `dagger call test --source=../email-unsubscribe`
- [x] 4.4 Test `dagger call build --source=../email-unsubscribe`

## 5. email-unsubscribe Migration

- [ ] 5.1 Create `mklv.config.mts` in email-unsubscribe repo
- [ ] 5.2 Replace `.github/workflows/ci.yaml` with reusable workflow call
- [ ] 5.3 Delete `.github/workflows/deploy.yml`
- [ ] 5.4 Delete `deploy.config.ts`
- [ ] 5.5 Push to main and verify pipeline runs
