"use client";

import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";

import { patchMyProfile, type MeProfile } from "@/lib/api/client";
import {
  authLoadingLabel,
  bioLabel,
  displayNameLabel,
  genericAuthError,
  settingsProfileHeading,
  settingsProfileSaved,
  settingsProfileSubmit,
  usernameCooldownMessage,
  usernameHint,
  usernameLabel,
} from "@/lib/copy";
import { usernameCooldownEndsAt } from "@/lib/usernameCooldown";

import { useAuth } from "./AuthProvider";
import { ProfilePhotos } from "./ProfilePhotos";

const fieldClass =
  "mt-1 w-full rounded-md border border-border bg-surface px-3 py-2 text-foreground focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent";

export function SettingsProfileForm() {
  const router = useRouter();
  const { user, profile, ready, refresh } = useAuth();

  useEffect(() => {
    if (ready && !user) {
      router.replace("/login");
    }
  }, [ready, user, router]);

  if (!ready || !user || !profile) {
    return (
      <main className="mx-auto flex min-h-[calc(100vh-4rem)] max-w-xl flex-col justify-center gap-4 px-6 py-16">
        <p className="text-sm text-muted">{authLoadingLabel}</p>
      </main>
    );
  }

  return (
    <SettingsProfileFields profile={profile} refresh={refresh} />
  );
}

function SettingsProfileFields({
  profile,
  refresh,
}: {
  profile: MeProfile;
  refresh: () => Promise<void>;
}) {
  const [displayName, setDisplayName] = useState(profile.displayName);
  const [bio, setBio] = useState(profile.bio);
  const [username, setUsername] = useState(profile.username ?? "");
  const [error, setError] = useState<string | null>(null);
  const [saved, setSaved] = useState(false);
  const [pending, setPending] = useState(false);

  const cooldownEnds = usernameCooldownEndsAt(profile.usernameChangedAt);
  const usernameLocked = cooldownEnds !== null;

  async function onSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(null);
    setSaved(false);
    setPending(true);
    try {
      const body: {
        displayName: string;
        bio: string;
        username?: string;
      } = { displayName, bio };
      if (!usernameLocked) {
        body.username = username;
      }
      await patchMyProfile(body);
      await refresh();
      setSaved(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : genericAuthError);
    } finally {
      setPending(false);
    }
  }

  return (
    <main className="mx-auto flex min-h-[calc(100vh-4rem)] max-w-xl flex-col justify-center gap-6 px-6 py-16">
      <h1 className="text-3xl font-semibold tracking-tight text-foreground">
        {settingsProfileHeading}
      </h1>
      <form onSubmit={onSubmit} className="flex flex-col gap-4" noValidate>
        {error ? (
          <p role="alert" className="text-sm text-danger">
            {error}
          </p>
        ) : null}
        {saved ? (
          <p role="status" className="text-sm text-muted">
            {settingsProfileSaved}
          </p>
        ) : null}
        <ProfilePhotos profile={profile} refresh={refresh} />
        <label className="text-sm text-muted">
          {displayNameLabel}
          <input
            className={fieldClass}
            name="displayName"
            autoComplete="name"
            required
            maxLength={80}
            value={displayName}
            onChange={(e) => setDisplayName(e.target.value)}
          />
        </label>
        <label className="text-sm text-muted">
          {bioLabel}
          <textarea
            className={fieldClass}
            name="bio"
            maxLength={160}
            rows={3}
            value={bio}
            onChange={(e) => setBio(e.target.value)}
          />
        </label>
        <label className="text-sm text-muted">
          {usernameLabel}
          <input
            className={fieldClass}
            name="username"
            autoComplete="username"
            required
            minLength={3}
            maxLength={20}
            pattern="[A-Za-z0-9_]+"
            value={username}
            readOnly={usernameLocked}
            onChange={(e) => setUsername(e.target.value)}
          />
        </label>
        <p className="text-sm text-subtle">
          {usernameLocked ? usernameCooldownMessage : usernameHint}
        </p>
        <button
          type="submit"
          disabled={pending}
          className="mt-2 rounded-md bg-accent px-4 py-2 font-medium text-on-accent hover:opacity-90 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent disabled:opacity-60"
        >
          {settingsProfileSubmit}
        </button>
      </form>
    </main>
  );
}
