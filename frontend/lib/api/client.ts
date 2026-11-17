import type { components } from "@/lib/api/schema";
import { genericAuthError } from "@/lib/copy";
import { publicApiBaseUrl } from "@/lib/publicApiBaseUrl";

export type User = components["schemas"]["User"];
export type MeProfile = components["schemas"]["MeProfile"];
export type PublicProfile = components["schemas"]["PublicProfile"];
export type PatchProfileRequest = components["schemas"]["PatchProfileRequest"];
export type RegisterRequest = components["schemas"]["RegisterRequest"];
export type LoginRequest = components["schemas"]["LoginRequest"];
export type VerifyEmailRequest = components["schemas"]["VerifyEmailRequest"];
export type ForgotPasswordRequest = components["schemas"]["ForgotPasswordRequest"];
export type ResetPasswordRequest = components["schemas"]["ResetPasswordRequest"];
export type ChangePasswordRequest = components["schemas"]["ChangePasswordRequest"];
export type ErrorEnvelope = components["schemas"]["ErrorEnvelope"];

export async function apiFetch(
  path: string,
  init: RequestInit = {},
): Promise<Response> {
  const headers = new Headers(init.headers);
  headers.set("Accept", "application/json");
  const isFormData =
    typeof FormData !== "undefined" && init.body instanceof FormData;
  if (
    init.body !== undefined &&
    !headers.has("Content-Type") &&
    !isFormData
  ) {
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

export async function fetchMyProfile(): Promise<MeProfile | null> {
  const res = await apiFetch("/v1/me/profile");
  if (res.status === 401) {
    return null;
  }
  if (!res.ok) {
    throw new Error(await readErrorMessage(res));
  }
  return (await res.json()) as MeProfile;
}

export async function fetchPublicProfile(
  username: string,
  init: RequestInit = {},
): Promise<PublicProfile | null> {
  const res = await apiFetch(`/v1/users/${encodeURIComponent(username)}`, init);
  if (res.status === 404) {
    return null;
  }
  if (!res.ok) {
    throw new Error(await readErrorMessage(res));
  }
  return (await res.json()) as PublicProfile;
}

export async function patchMyProfile(
  body: PatchProfileRequest,
): Promise<MeProfile> {
  const res = await apiFetch("/v1/me/profile", {
    method: "PATCH",
    body: JSON.stringify(body),
  });
  if (!res.ok) {
    throw new Error(await readErrorMessage(res));
  }
  return (await res.json()) as MeProfile;
}

export async function uploadProfileImage(
  kind: "avatar" | "cover",
  file: File,
): Promise<MeProfile> {
  const data = new FormData();
  data.append("file", file);
  const res = await apiFetch(`/v1/me/profile/${kind}`, {
    method: "POST",
    body: data,
  });
  if (!res.ok) {
    throw new Error(await readErrorMessage(res));
  }
  return (await res.json()) as MeProfile;
}

export async function deleteProfileImage(
  kind: "avatar" | "cover",
): Promise<MeProfile> {
  const res = await apiFetch(`/v1/me/profile/${kind}`, { method: "DELETE" });
  if (!res.ok) {
    throw new Error(await readErrorMessage(res));
  }
  return (await res.json()) as MeProfile;
}
