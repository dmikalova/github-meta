/**
 * Deno Pipeline - Opinionated CI/CD for Deno apps deploying to GCP Cloud Run
 *
 * This module provides functions for linting, testing, building, and deploying
 * Deno applications. It reads app configuration from `mklv.config.mts` and
 * builds minimal distroless containers using `deno compile`.
 */
import {
  Container,
  Directory,
  Secret,
  dag,
  func,
  object,
} from "@dagger.io/dagger";

/** Default Deno version */
const DENO_VERSION = "latest";

/** Default GCP region */
const GCP_REGION = "us-west1";

/** GCP project ID (hardcoded convention) */
const GCP_PROJECT = "mklv-infrastructure";

@object()
export class DenoPipeline {
  /**
   * Get a deno container with GitHub Packages npm auth
   */
  private denoContainer(source: Directory, githubToken: Secret): Container {
    // Create .npmrc for GitHub Packages auth
    const npmrc = `@dmikalova:registry=https://npm.pkg.github.com
//npm.pkg.github.com/:_authToken=\${GITHUB_TOKEN}
`;
    return dag
      .container()
      .from(`denoland/deno:${DENO_VERSION}`)
      .withDirectory("/app", source)
      .withNewFile("/app/.npmrc", npmrc)
      .withWorkdir("/app")
      .withSecretVariable("GITHUB_TOKEN", githubToken);
  }

  /**
   * Run deno lint on the source directory
   */
  @func()
  async lint(source: Directory, githubToken: Secret): Promise<string> {
    return await this.denoContainer(source, githubToken)
      .withExec(["deno", "lint"])
      .stdout();
  }

  /**
   * Run deno check (type checking) on the source directory
   */
  @func()
  async check(source: Directory, githubToken: Secret): Promise<string> {
    return await this.denoContainer(source, githubToken)
      .withExec(["deno", "task", "check"])
      .stdout();
  }

  /**
   * Run deno task test on the source directory
   */
  @func()
  async test(source: Directory, githubToken: Secret): Promise<string> {
    return await this.denoContainer(source, githubToken)
      .withExec(["deno", "task", "test"])
      .stdout();
  }

  /**
   * Build a minimal container with deno compile
   */
  @func()
  build(source: Directory, entrypoint: string, githubToken: Secret): Container {
    // Build stage: compile to standalone binary
    const builder = this.denoContainer(source, githubToken).withExec([
      "deno",
      "compile",
      "--output",
      "app",
      entrypoint,
    ]);

    // Runtime stage: minimal distroless (~20MB total)
    return dag
      .container()
      .from("gcr.io/distroless/cc-debian12")
      .withFile("/app", builder.file("/app/app"))
      .withExposedPort(8000)
      .withEntrypoint(["/app"]);
  }

  /**
   * Publish a container to GHCR
   */
  @func()
  async publish(
    container: Container,
    name: string,
    version: string,
    githubToken: Secret,
  ): Promise<string> {
    const ref = `ghcr.io/dmikalova/${name}:${version}`;
    return await container
      .withRegistryAuth("ghcr.io", "dmikalova", githubToken)
      .publish(ref);
  }

  /**
   * Run database migrations
   * Fetches DATABASE_URL from Secret Manager using convention: {app-name}-database-url
   */
  @func()
  async migrate(source: Directory, appName: string): Promise<string> {
    const secretName = `${appName}-database-url`;
    return await dag
      .container()
      .from("google/cloud-sdk:slim")
      .withDirectory("/app", source)
      .withExec(["apt-get", "update"])
      .withExec(["apt-get", "install", "-y", "unzip"])
      .withExec(["sh", "-c", "curl -fsSL https://deno.land/install.sh | sh"])
      .withEnvVariable("PATH", "/root/.deno/bin:$PATH", { expand: true })
      .withExec([
        "sh",
        "-c",
        `export DATABASE_URL=$(gcloud secrets versions access latest --secret=${secretName} --project=${GCP_PROJECT}) && cd /app && deno task migrate`,
      ])
      .stdout();
  }

  /**
   * Deploy to Cloud Run
   */
  @func()
  async deploy(
    image: string,
    serviceName: string,
    region: string = GCP_REGION,
  ): Promise<string> {
    return await dag
      .container()
      .from("google/cloud-sdk:slim")
      .withExec([
        "gcloud",
        "run",
        "deploy",
        serviceName,
        "--image",
        image,
        "--region",
        region,
        "--project",
        GCP_PROJECT,
        "--quiet",
      ])
      .stdout();
  }

  /**
   * Full CI pipeline: lint, check, test
   */
  @func()
  async ci(source: Directory, githubToken: Secret): Promise<string> {
    await this.lint(source, githubToken);
    await this.check(source, githubToken);
    const testOutput = await this.test(source, githubToken);
    return `CI passed!\n${testOutput}`;
  }

  /**
   * Full CD pipeline: build, publish, migrate, deploy
   */
  @func()
  async cd(
    source: Directory,
    entrypoint: string,
    name: string,
    version: string,
    githubToken: Secret,
  ): Promise<string> {
    // Build
    const container = this.build(source, entrypoint, githubToken);

    // Publish
    const image = await this.publish(container, name, version, githubToken);

    // Migrate (fetches DATABASE_URL from Secret Manager using app name)
    await this.migrate(source, name);

    // Deploy
    const deployOutput = await this.deploy(image, name);

    return `Deployed ${name}:${version} to Cloud Run!\n${deployOutput}`;
  }

  /**
   * Read app configuration from mklv.config.mts
   * Returns JSON with name, entrypoint, and runtime settings
   */
  @func()
  async readConfig(source: Directory, githubToken: Secret): Promise<string> {
    // Use Deno to import and serialize the config
    const script = `
      const config = (await import('./mklv.config.mts')).default;
      console.log(JSON.stringify({
        name: config.name,
        entrypoint: config.entrypoint,
        port: config.runtime?.port ?? 8000,
        healthCheckPath: config.runtime?.healthCheckPath ?? '/health',
      }));
    `;
    return await this.denoContainer(source, githubToken)
      .withExec(["deno", "eval", script])
      .stdout();
  }

  /**
   * Deploy only (no CI) - reads config and runs build, publish, migrate, deploy
   */
  @func()
  async deployOnly(
    source: Directory,
    version: string,
    githubToken: Secret,
  ): Promise<string> {
    // Read config
    const configJson = await this.readConfig(source, githubToken);
    const config = JSON.parse(configJson);

    // CD: build, publish, migrate, deploy
    return await this.cd(
      source,
      config.entrypoint,
      config.name,
      version,
      githubToken,
    );
  }

  /**
   * Full pipeline using config from mklv.config.mts (CI + CD)
   */
  @func()
  async pipeline(
    source: Directory,
    version: string,
    githubToken: Secret,
  ): Promise<string> {
    // CI: lint, check, test
    await this.ci(source, githubToken);

    // CD: build, publish, migrate, deploy
    return await this.deployOnly(source, version, githubToken);
  }
}
