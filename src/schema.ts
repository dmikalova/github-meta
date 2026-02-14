/**
 * MklvConfig - App deployment configuration schema
 *
 * Apps define their runtime requirements in `mklv.config.mts`.
 * The CI/CD pipeline reads this config and derives GCP deployment settings.
 *
 * Usage in app repo:
 * ```typescript
 * import type { MklvConfig } from "@dmikalova/mklv-config";
 *
 * export default {
 *   name: "my-app",
 *   entrypoint: "src/main.ts",
 *   runtime: {
 *     port: 8000,
 *     healthCheckPath: "/health",
 *   },
 * } satisfies MklvConfig;
 * ```
 */

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
