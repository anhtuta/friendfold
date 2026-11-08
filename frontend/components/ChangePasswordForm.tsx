"use client";

import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";

import { apiFetch, readErrorMessage } from "@/lib/api/client";
import {
  authLoadingLabel,
  changePasswordHeading,
  changePasswordSubmit,
  changePasswordSuccess,
  confirmPasswordLabel,
  currentPasswordLabel,
  genericAuthError,
  newPasswordLabel,
  passwordMismatch,
} from "@/lib/copy";

import { useAuth } from "./AuthProvider";

const fieldClass =
  "mt-1 w-full rounded-md border border-border bg-surface px-3 py-2 text-foreground focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent";

export function ChangePasswordForm() {
  const router = useRouter();
  const { user, ready, refresh } = useAuth();
  const [currentPassword, setCurrentPassword] = useState("");
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);
  const [done, setDone] = useState(false);

  useEffect(() => {
    if (ready && !user) {
      router.replace("/login");
    }
  }, [ready, user, router]);

  async function onSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (password !== confirm) {
      setError(passwordMismatch);
      return;
    }
    setError(null);
    setPending(true);
    try {
      const res = await apiFetch("/v1/auth/password/change", {
        method: "POST",
        body: JSON.stringify({
          currentPassword,
          newPassword: password,
        }),
      });
      if (res.status === 204) {
        setDone(true);
        await refresh();
        return;
      }
      setError(await readErrorMessage(res));
    } catch {
      setError(genericAuthError);
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
        {changePasswordHeading}
      </h1>
      {done ? (
        <p role="status" className="text-sm text-muted">
          {changePasswordSuccess}
        </p>
      ) : (
        <form onSubmit={onSubmit} className="flex flex-col gap-4" noValidate>
          {error ? (
            <p role="alert" className="text-sm text-danger">
              {error}
            </p>
          ) : null}
          <label className="text-sm text-muted">
            {currentPasswordLabel}
            <input
              className={fieldClass}
              type="password"
              name="currentPassword"
              autoComplete="current-password"
              required
              minLength={1}
              maxLength={72}
              value={currentPassword}
              onChange={(e) => setCurrentPassword(e.target.value)}
            />
          </label>
          <label className="text-sm text-muted">
            {newPasswordLabel}
            <input
              className={fieldClass}
              type="password"
              name="newPassword"
              autoComplete="new-password"
              required
              minLength={8}
              maxLength={72}
              value={password}
              onChange={(e) => setPassword(e.target.value)}
            />
          </label>
          <label className="text-sm text-muted">
            {confirmPasswordLabel}
            <input
              className={fieldClass}
              type="password"
              name="confirmPassword"
              autoComplete="new-password"
              required
              minLength={8}
              maxLength={72}
              value={confirm}
              onChange={(e) => setConfirm(e.target.value)}
            />
          </label>
          <button
            type="submit"
            disabled={pending}
            className="mt-2 rounded-md bg-accent px-4 py-2 font-medium text-on-accent hover:opacity-90 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent disabled:opacity-60"
          >
            {changePasswordSubmit}
          </button>
        </form>
      )}
    </main>
  );
}
