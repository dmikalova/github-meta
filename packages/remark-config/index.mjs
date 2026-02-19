/**
 * Shared remark configuration preset
 * Usage in .remarkrc.mjs: export { default } from '@dmikalova/remark-config';
 */

import remarkFrontmatter from "remark-frontmatter";
import remarkGfm from "remark-gfm";
import remarkPresetLintConsistent from "remark-preset-lint-consistent";
import remarkPresetLintRecommended from "remark-preset-lint-recommended";
import remarkToc from "remark-toc";

const config = {
  settings: {
    bullet: "-",
    emphasis: "*",
    fences: true,
    listItemIndent: "one",
    strong: "*",
    tightDefinitions: true,
  },
  plugins: [
    // Syntax extensions
    remarkFrontmatter,
    remarkGfm,

    // Table of contents - updates ## Contents or ## Table of Contents
    [remarkToc, { heading: "(table[ -]of[ -])?contents?", tight: true }],

    // Linting presets
    remarkPresetLintRecommended,
    remarkPresetLintConsistent,
  ],
};

export default config;
