const cooldownMs = 14 * 24 * 60 * 60 * 1000;

/** When the owner may change username again, or null if they can change now. */
export function usernameCooldownEndsAt(
  usernameChangedAt: string | null,
  now: Date = new Date(),
): Date | null {
  if (usernameChangedAt === null || usernameChangedAt === "") {
    return null;
  }
  const start = Date.parse(usernameChangedAt);
  if (Number.isNaN(start)) {
    return null;
  }
  const ends = new Date(start + cooldownMs);
  if (ends.getTime() <= now.getTime()) {
    return null;
  }
  return ends;
}
