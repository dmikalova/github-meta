/**
 * Shared commitlint configuration.
 * Used via `extends: ['@dmikalova/commitlint-config']` in commitlint.config.mjs
 *
 * Enforces conventional commit format with standard type prefixes.
 */
export default {
  extends: ["@commitlint/config-conventional"],
  rules: {
    "type-enum": [
      2,
      "always",
      [
        "build",
        "chore",
        "ci",
        "docs",
        "feat",
        "fix",
        "perf",
        "refactor",
        "revert",
        "style",
        "test",
      ],
    ],
  },
};
