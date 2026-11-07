import type { components } from "@/lib/api/schema";
import { genericAuthError } from "@/lib/copy";
import { publicApiBaseUrl } from "@/lib/publicApiBaseUrl";

export type User = components["schemas"]["User"];
export type RegisterRequest = components["schemas"]["RegisterRequest"];
export type LoginRequest = components["schemas"]["LoginRequest"];
export type ErrorEnvelope = components["schemas"]["ErrorEnvelope"];

export async function apiFetch(
  path: string,
  init: RequestInit = {},
): Promise<Response> {
  const headers = new Headers(init.headers);
  headers.set("Accept", "application/json");
  if (init.body !== undefined && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }
  return fetch(`${publicApiBaseUrl()}${path}`, {
    ...init,
    headers,
    credentials: "include",
  });
}

export async function readErrorMessage(res: Response): Promise<string> {
  try {
    const body = (await res.json()) as ErrorEnvelope;
    if (body.error?.message) {
      return body.error.message;
    }
  } catch {
    // Fall through to the generic copy.
  }
  return genericAuthError;
}

export async function fetchCurrentUser(): Promise<User | null> {
  const res = await apiFetch("/v1/auth/me");
  if (res.status === 401) {
    return null;
  }
  if (!res.ok) {
    throw new Error(await readErrorMessage(res));
  }
  return (await res.json()) as User;
}
