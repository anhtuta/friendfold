import { readFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

import { describe, expect, it } from "vitest";

const globalsPath = path.join(
  path.dirname(fileURLToPath(import.meta.url)),
  "../app/globals.css",
);

function srgbChannelLuminance(hex: string, start: number): number {
  const channel = parseInt(hex.slice(start, start + 2), 16) / 255;
  // WCAG 2.2 relative luminance (sRGB).
  return channel <= 0.04045
    ? channel / 12.92
    : ((channel + 0.055) / 1.055) ** 2.4;
}

function relativeLuminance(hex: string): number {
  const r = srgbChannelLuminance(hex, 1);
  const g = srgbChannelLuminance(hex, 3);
  const b = srgbChannelLuminance(hex, 5);
  return 0.2126 * r + 0.7152 * g + 0.0722 * b;
}

function contrastRatio(a: string, b: string): number {
  const first = relativeLuminance(a);
  const second = relativeLuminance(b);
  const [higher, lower] = first > second ? [first, second] : [second, first];
  return (higher + 0.05) / (lower + 0.05);
}

function themeColor(css: string, name: string): string {
  const match = css.match(new RegExp(`--color-${name}:\\s*(#[0-9a-fA-F]{6})`));
  const value = match?.[1];
  if (!value) {
    throw new Error(`missing --color-${name} hex in globals.css`);
  }
  return value;
}

describe("dark theme tokens", () => {
  const css = readFileSync(globalsPath, "utf8");

  it("meets WCAG 2.2 AA contrast for text on the page background", () => {
    const background = themeColor(css, "background");
    for (const name of ["foreground", "muted", "subtle", "accent"] as const) {
      expect(
        contrastRatio(background, themeColor(css, name)),
        `${name} on background`,
      ).toBeGreaterThanOrEqual(4.5);
    }
  });
});
