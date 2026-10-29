const defaultAPIBaseURL = "http://localhost:8080";
const maxAPIBaseURLLength = 200;

type EnvMap = Record<string, string | undefined>;

export function apiBaseUrl(env: EnvMap = process.env): string {
  const raw = env.API_BASE_URL?.trim();
  if (raw === undefined || raw === "") {
    return defaultAPIBaseURL;
  }
  if (raw.length > maxAPIBaseURLLength) {
    throw new Error("API_BASE_URL exceeds max length");
  }

  let parsed: URL;
  try {
    parsed = new URL(raw);
  } catch {
    throw new Error("API_BASE_URL must be an absolute http(s) URL");
  }
  if (parsed.protocol !== "http:" && parsed.protocol !== "https:") {
    throw new Error("API_BASE_URL must be an absolute http(s) URL");
  }
  return parsed.origin;
}
