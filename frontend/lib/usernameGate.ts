const allowedExact = new Set([
  "/login",
  "/signup",
  "/verify-email",
  "/forgot-password",
  "/reset-password",
  "/health",
  "/onboarding/username",
]);

/** Paths a signed-in user may visit before claiming a username (F-03-04). */
export function usernameGateAllowed(pathname: string): boolean {
  if (allowedExact.has(pathname)) {
    return true;
  }
  return pathname === "/u" || pathname.startsWith("/u/");
}
