/** Initials for the default avatar. Two letters from the display name. */
export function displayInitials(name: string): string {
  const parts = name.trim().split(/\s+/).filter((p) => p.length > 0);
  const first = parts[0];
  if (first === undefined) {
    return "?";
  }
  const last = parts[parts.length - 1];
  if (parts.length === 1 || last === undefined || last === first) {
    return Array.from(first).slice(0, 2).join("").toUpperCase();
  }
  const a = Array.from(first)[0];
  const b = Array.from(last)[0];
  if (a === undefined || b === undefined) {
    return "?";
  }
  return (a + b).toUpperCase();
}
