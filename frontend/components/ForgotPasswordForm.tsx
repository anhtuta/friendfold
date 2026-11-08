"use client";

import Link from "next/link";
import { useState } from "react";

import { apiFetch, readErrorMessage } from "@/lib/api/client";
import {
  emailLabel,
  forgotPasswordHeading,
  forgotPasswordSent,
  forgotPasswordSubmit,
  genericAuthError,
  signInLabel,
} from "@/lib/copy";

const fieldClass =
  "mt-1 w-full rounded-md border border-border bg-surface px-3 py-2 text-foreground focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent";

export function ForgotPasswordForm() {
  const [email, setEmail] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [sent, setSent] = useState(false);
  const [pending, setPending] = useState(false);

  async function onSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(null);
    setPending(true);
    try {
      const res = await apiFetch("/v1/auth/password/forgot", {
        method: "POST",
        body: JSON.stringify({ email }),
      });
      if (res.status === 202) {
        setSent(true);
        return;
      }
      setError(await readErrorMessage(res));
    } catch {
      setError(genericAuthError);
    } finally {
      setPending(false);
    }
  }

  return (
    <main className="mx-auto flex min-h-[calc(100vh-4rem)] max-w-xl flex-col justify-center gap-6 px-6 py-16">
      <h1 className="text-3xl font-semibold tracking-tight text-foreground">
        {forgotPasswordHeading}
      </h1>
      {sent ? (
        <p role="status" className="text-sm text-muted">
          {forgotPasswordSent}
        </p>
      ) : (
        <form onSubmit={onSubmit} className="flex flex-col gap-4" noValidate>
          {error ? (
            <p role="alert" className="text-sm text-danger">
              {error}
            </p>
          ) : null}
          <label className="text-sm text-muted">
            {emailLabel}
            <input
              className={fieldClass}
              type="email"
              name="email"
              autoComplete="email"
              required
              maxLength={254}
              value={email}
              onChange={(e) => setEmail(e.target.value)}
            />
          </label>
          <button
            type="submit"
            disabled={pending}
            className="mt-2 rounded-md bg-accent px-4 py-2 font-medium text-on-accent hover:opacity-90 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent disabled:opacity-60"
          >
            {forgotPasswordSubmit}
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
