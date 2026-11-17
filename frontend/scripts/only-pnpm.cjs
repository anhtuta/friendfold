"use strict";

// Guard preinstall so npm/yarn cannot write a second lockfile.
// Uses npm_config_user_agent (set by npm, pnpm, and yarn). No extra packages.
const ua = process.env.npm_config_user_agent || "";
if (!ua.startsWith("pnpm/")) {
  console.error(
    "This package must be installed with pnpm (packageManager in package.json).\n" +
      "npm and yarn are not supported. Enable Corepack and retry:\n" +
      "  corepack enable && corepack prepare pnpm@12.18.0 --activate\n" +
      "  pnpm install",
  );
  process.exit(1);
}
