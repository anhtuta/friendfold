import { describe, expect, it } from "vitest";

import { displayInitials } from "./displayInitials";

describe("displayInitials", () => {
  it("uses two letters from a single word", () => {
    expect(displayInitials("Minh")).toBe("MI");
  });

  it("uses first and last words", () => {
    expect(displayInitials("Minh Tran")).toBe("MT");
  });

  it("returns a placeholder for an empty name", () => {
    expect(displayInitials("   ")).toBe("?");
  });
});
