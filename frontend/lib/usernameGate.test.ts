import { describe, expect, it } from "vitest";

import { usernameGateAllowed } from "./usernameGate";

describe("usernameGateAllowed", () => {
  it("allows auth, health, onboarding, and public profiles", () => {
    expect(usernameGateAllowed("/login")).toBe(true);
    expect(usernameGateAllowed("/onboarding/username")).toBe(true);
    expect(usernameGateAllowed("/u/minh")).toBe(true);
    expect(usernameGateAllowed("/health")).toBe(true);
  });

  it("blocks home, settings, and change-password", () => {
    expect(usernameGateAllowed("/")).toBe(false);
    expect(usernameGateAllowed("/settings/profile")).toBe(false);
    expect(usernameGateAllowed("/change-password")).toBe(false);
  });
});
