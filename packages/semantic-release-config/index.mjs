/**
 * Shared semantic-release configuration.
 * Used by deno-cloudrun workflow via `extends: @dmikalova/semantic-release-config`
 *
 * Skips npm publish since consumer apps are Deno/Cloud Run, not npm packages.
 * Releases on ALL commit types (including refactor, chore, style, etc.).
 */
export default {
  branches: ["main"],
  plugins: [
    [
      "@semantic-release/commit-analyzer",
      {
        releaseRules: [
          { breaking: true, release: "major" },
          { type: "build", release: "patch" },
          { type: "chore", release: "patch" },
          { type: "ci", release: "patch" },
          { type: "docs", release: "patch" },
          { type: "feat", release: "minor" },
          { type: "fix", release: "patch" },
          { type: "perf", release: "patch" },
          { type: "refactor", release: "patch" },
          { type: "revert", release: "patch" },
          { type: "style", release: "patch" },
          { type: "test", release: "patch" },
        ],
      },
    ],
    "@semantic-release/release-notes-generator",
    ["@semantic-release/github", { failComment: false }],
  ],
};
