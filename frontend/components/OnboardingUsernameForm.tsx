"use client";

import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";

import { patchMyProfile } from "@/lib/api/client";
import {
  authLoadingLabel,
  genericAuthError,
  onboardingUsernameBody,
  onboardingUsernameHeading,
  onboardingUsernameSubmit,
  usernameHint,
  usernameLabel,
} from "@/lib/copy";

import { useAuth } from "./AuthProvider";

const fieldClass =
  "mt-1 w-full rounded-md border border-border bg-surface px-3 py-2 text-foreground focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent";

export function OnboardingUsernameForm() {
  const router = useRouter();
  const { user, profile, ready, refresh } = useAuth();
  const [username, setUsername] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);

  useEffect(() => {
    if (ready && !user) {
      router.replace("/login");
    }
  }, [ready, user, router]);

  useEffect(() => {
    if (ready && user && profile?.username) {
      router.replace("/");
    }
  }, [ready, user, profile, router]);

  async function onSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(null);
    setPending(true);
    try {
      await patchMyProfile({ username });
      await refresh();
      router.replace("/");
    } catch (err) {
      setError(err instanceof Error ? err.message : genericAuthError);
    } finally {
      setPending(false);
    }
  }

  if (!ready || !user) {
    return (
      <main className="mx-auto flex min-h-[calc(100vh-4rem)] max-w-xl flex-col justify-center gap-4 px-6 py-16">
        <p className="text-sm text-muted">{authLoadingLabel}</p>
      </main>
    );
  }

  return (
    <main className="mx-auto flex min-h-[calc(100vh-4rem)] max-w-xl flex-col justify-center gap-6 px-6 py-16">
      <h1 className="text-3xl font-semibold tracking-tight text-foreground">
        {onboardingUsernameHeading}
      </h1>
      <p className="text-muted">{onboardingUsernameBody}</p>
      <form onSubmit={onSubmit} className="flex flex-col gap-4" noValidate>
        {error ? (
          <p role="alert" className="text-sm text-danger">
            {error}
          </p>
        ) : null}
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
            onChange={(e) => setUsername(e.target.value)}
          />
        </label>
        <p className="text-sm text-subtle">{usernameHint}</p>
        <button
          type="submit"
          disabled={pending}
          className="mt-2 rounded-md bg-accent px-4 py-2 font-medium text-on-accent hover:opacity-90 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent disabled:opacity-60"
        >
          {onboardingUsernameSubmit}
        </button>
      </form>
    </main>
  );
}
