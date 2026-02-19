# Design

**Alternatives considered:**

- Full Deno image (~350MB): Too large, slow pulls
- Alpine + Deno: Still needs runtime, ~150MB
- Cloud Run source deploy: Less control, slower builds

**Rationale:** Smallest possible image. Single static binary with no runtime
dependencies. Fast cold starts. Secure (no shell, no package manager).

**Pipeline derives GCP settings from conventions:**

- GCP project: `dmikalova-mklv` (hardcoded in pipeline)
- Cloud Run service: `{config.name}`
- Region: `us-west1` (default)
- WIF provider: known from infrastructure repo
- WIF service account:
  `github-actions-deploy@dmikalova-mklv.iam.gserviceaccount.com`

**Deployment steps:**

1. Build Deno binary using `deno compile`
