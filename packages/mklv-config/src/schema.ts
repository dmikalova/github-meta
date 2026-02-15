/**
 * MklvConfig - App deployment configuration schema
 *
 * Apps define their runtime requirements in `mklv.config.mts`.
 * The CI/CD pipeline reads this config and derives GCP deployment settings.
 *
 * Usage in app repo:
 * ```typescript
 * import { defineConfig } from "@dmikalova/mklv-config";
 *
 * export default defineConfig({
 *   name: "my-app",
 *   entrypoint: "src/main.ts",
 *   runtime: {
 *     port: 8000,
 *     healthCheckPath: "/health",
 *   },
 * }, import.meta);
 * ```
 *
 * Run directly to get JSON: `deno run mklv.config.mts | jq .name`
 */

// Augment ImportMeta for Deno's `main` property
declare global {
  interface ImportMeta {
    main?: boolean;
  }
}

/**
 * Runtime configuration for the application.
 */
export interface RuntimeConfig {
  /** Port the app listens on (default: 8000) */
  port?: number;
  /** Health check endpoint path (default: /health) */
  healthCheckPath?: string;
}

/**
 * App deployment configuration.
 * Defines what the app is, not where it deploys.
 */
export interface MklvConfig {
  /** App name - used for image tagging and Cloud Run service name */
  name: string;
  /** Deno entrypoint file (e.g., "src/main.ts") */
  entrypoint: string;
  /** Runtime configuration */
  runtime?: RuntimeConfig;
}

/**
 * Default values used by the pipeline when not specified in config.
 */
export const defaults = {
  port: 8000,
  healthCheckPath: "/health",
} as const;

/**
 * Define and register app configuration.
 * Outputs JSON when the config file is run directly (import.meta.main).
 *
 * Usage:
 * ```typescript
 * import { defineConfig } from "@dmikalova/mklv-config";
 *
 * export default defineConfig({
 *   name: "my-app",
 *   entrypoint: "src/main.ts",
 * }, import.meta);
 * ```
 */
export function defineConfig(config: MklvConfig, meta: ImportMeta): MklvConfig {
  if (meta.main) {
    console.log(JSON.stringify(config));
  }
  return config;
}
