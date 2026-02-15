/**
 * Shared semantic-release configuration.
 * Used by deno-cloudrun workflow via `extends: dmikalova/github-meta`
 */
export default {
  branches: ["main"],
  plugins: [
    "@semantic-release/commit-analyzer",
    "@semantic-release/release-notes-generator",
    ["@semantic-release/github", { failComment: false }],
  ],
};
