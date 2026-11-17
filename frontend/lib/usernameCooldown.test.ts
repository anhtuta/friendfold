import { describe, expect, it } from "vitest";

import { usernameCooldownEndsAt } from "./usernameCooldown";

describe("usernameCooldownEndsAt", () => {
  it("is null when never changed", () => {
    expect(usernameCooldownEndsAt(null)).toBeNull();
  });

  it("is null after 14 days", () => {
    const now = new Date("2026-12-02T12:00:00Z");
    expect(
      usernameCooldownEndsAt("2026-11-17T12:00:00Z", now),
    ).toBeNull();
  });

  it("returns the end date during the cooldown", () => {
    const now = new Date("2026-11-18T12:00:00Z");
    const ends = usernameCooldownEndsAt("2026-11-17T12:00:00Z", now);
    expect(ends?.toISOString()).toBe("2026-12-01T12:00:00.000Z");
  });
});
