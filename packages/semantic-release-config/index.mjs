/**
 * Shared semantic-release configuration.
 * Used by deno-cloudrun workflow via `extends: @dmikalova/semantic-release-config`
 *
 * Skips npm publish since consumer apps are Deno/Cloud Run, not npm packages.
 */
export default {
  branches: ["main"],
  plugins: [
    "@semantic-release/commit-analyzer",
    "@semantic-release/release-notes-generator",
    ["@semantic-release/github", { failComment: false }],
  ],
};
