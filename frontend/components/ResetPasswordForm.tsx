"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { useState } from "react";

import { apiFetch, readErrorMessage } from "@/lib/api/client";
import {
  confirmPasswordLabel,
  genericAuthError,
  newPasswordLabel,
  passwordMismatch,
  resetPasswordHeading,
  resetPasswordInvalid,
  resetPasswordMissingToken,
  resetPasswordSubmit,
  resetPasswordSuccess,
  signInLabel,
} from "@/lib/copy";

const fieldClass =
  "mt-1 w-full rounded-md border border-border bg-surface px-3 py-2 text-foreground focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent";

export function ResetPasswordForm() {
  const router = useRouter();
  const params = useSearchParams();
  const token = params.get("token")?.trim() ?? "";
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [error, setError] = useState<string | null>(
    token ? null : resetPasswordMissingToken,
  );
  const [pending, setPending] = useState(false);
  const [done, setDone] = useState(false);

  async function onSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!token) {
      setError(resetPasswordMissingToken);
      return;
    }
    if (password !== confirm) {
      setError(passwordMismatch);
      return;
    }
    setError(null);
    setPending(true);
    try {
      const res = await apiFetch("/v1/auth/password/reset", {
        method: "POST",
        body: JSON.stringify({ token, newPassword: password }),
      });
      if (res.status === 204) {
        setDone(true);
        router.push("/login");
        return;
      }
      setError((await readErrorMessage(res)) || resetPasswordInvalid);
    } catch {
      setError(genericAuthError);
    } finally {
      setPending(false);
    }
  }

  return (
    <main className="mx-auto flex min-h-[calc(100vh-4rem)] max-w-xl flex-col justify-center gap-6 px-6 py-16">
      <h1 className="text-3xl font-semibold tracking-tight text-foreground">
        {resetPasswordHeading}
      </h1>
      {done ? (
        <p role="status" className="text-sm text-muted">
          {resetPasswordSuccess}
        </p>
      ) : !token ? (
        <p role="alert" className="text-sm text-danger">
          {resetPasswordMissingToken}
        </p>
      ) : (
        <form onSubmit={onSubmit} className="flex flex-col gap-4" noValidate>
          {error ? (
            <p role="alert" className="text-sm text-danger">
              {error}
            </p>
          ) : null}
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
            {resetPasswordSubmit}
          </button>
        </form>
      )}
      <p className="text-sm text-subtle">
        <Link
          href="/login"
          className="text-accent underline underline-offset-2 hover:text-foreground focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent"
        >
          {signInLabel}
        </Link>
      </p>
    </main>
  );
}
