const defaultPublicAPIBaseURL = "http://localhost:8080";
const maxPublicAPIBaseURLLength = 200;

type EnvMap = Record<string, string | undefined>;

export function publicApiBaseUrl(env: EnvMap = process.env): string {
  const raw = env.NEXT_PUBLIC_API_BASE_URL?.trim();
  if (raw === undefined || raw === "") {
    return defaultPublicAPIBaseURL;
  }
  if (raw.length > maxPublicAPIBaseURLLength) {
    throw new Error("NEXT_PUBLIC_API_BASE_URL exceeds max length");
  }

  let parsed: URL;
  try {
    parsed = new URL(raw);
  } catch {
    throw new Error("NEXT_PUBLIC_API_BASE_URL must be an absolute http(s) URL");
  }
  if (parsed.protocol !== "http:" && parsed.protocol !== "https:") {
    throw new Error("NEXT_PUBLIC_API_BASE_URL must be an absolute http(s) URL");
  }
  return parsed.origin;
}
