import { cookies } from "next/headers";

import { ProfilePage } from "@/components/ProfilePage";
import { apiFetch, type PublicProfile } from "@/lib/api/client";
import { productName } from "@/lib/copy";

const usernamePattern = /^[A-Za-z0-9_]{3,20}$/;

type PageProps = {
  params: Promise<{ username: string }>;
};

export async function generateMetadata({ params }: PageProps) {
  const { username } = await params;
  return { title: `${username} · ${productName}` };
}

export default async function Page({ params }: PageProps) {
  const { username } = await params;
  const profile = await loadPublicProfile(username);
  return <ProfilePage profile={profile} />;
}

async function loadPublicProfile(raw: string): Promise<PublicProfile | null> {
  if (!usernamePattern.test(raw)) {
    return null;
  }
  const cookieStore = await cookies();
  const session = cookieStore.get("friendfold_session");
  const headers = new Headers({ Accept: "application/json" });
  if (session?.value) {
    headers.set("Cookie", `friendfold_session=${session.value}`);
  }
  const res = await apiFetch(`/v1/users/${encodeURIComponent(raw)}`, {
    headers,
    cache: "no-store",
  });
  if (res.status === 404 || res.status === 400) {
    return null;
  }
  if (!res.ok) {
    throw new Error("profile lookup failed");
  }
  return (await res.json()) as PublicProfile;
}
