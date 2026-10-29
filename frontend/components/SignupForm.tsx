"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";

import { apiFetch, readErrorMessage } from "@/lib/api/client";
import {
  dateOfBirthLabel,
  displayNameLabel,
  emailLabel,
  genericAuthError,
  haveAccountPrompt,
  passwordLabel,
  signInLabel,
  signupHeading,
  submitSignUp,
  termsLabel,
} from "@/lib/copy";

import { useAuth } from "./AuthProvider";

const fieldClass =
  "mt-1 w-full rounded-md border border-border bg-surface px-3 py-2 text-foreground focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent";

export function SignupForm() {
  const router = useRouter();
  const { refresh } = useAuth();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [dateOfBirth, setDateOfBirth] = useState("");
  const [termsAccepted, setTermsAccepted] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState(false);

  async function onSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(null);
    if (!termsAccepted) {
      setError("Terms of use and privacy policy must be accepted");
      return;
    }
    if (password.length < 8) {
      setError("password must be 8–72 bytes");
      return;
    }
    setPending(true);
    try {
      const res = await apiFetch("/v1/auth/register", {
        method: "POST",
        body: JSON.stringify({
          email,
          password,
          displayName,
          dateOfBirth,
          termsAccepted: true,
        }),
      });
      if (!res.ok) {
        setError(await readErrorMessage(res));
        return;
      }
      await refresh();
      router.push("/");
    } catch {
      setError(genericAuthError);
    } finally {
      setPending(false);
    }
  }

  return (
    <main className="mx-auto flex min-h-[calc(100vh-4rem)] max-w-xl flex-col justify-center gap-6 px-6 py-16">
      <h1 className="text-3xl font-semibold tracking-tight text-foreground">
        {signupHeading}
      </h1>
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
        <label className="text-sm text-muted">
          {passwordLabel}
          <input
            className={fieldClass}
            type="password"
            name="password"
            autoComplete="new-password"
            required
            minLength={8}
            maxLength={72}
            value={password}
            onChange={(e) => setPassword(e.target.value)}
          />
        </label>
        <label className="text-sm text-muted">
          {displayNameLabel}
          <input
            className={fieldClass}
            type="text"
            name="displayName"
            autoComplete="name"
            required
            minLength={1}
            maxLength={80}
            value={displayName}
            onChange={(e) => setDisplayName(e.target.value)}
          />
        </label>
        <label className="text-sm text-muted">
          {dateOfBirthLabel}
          <input
            className={fieldClass}
            type="date"
            name="dateOfBirth"
            required
            value={dateOfBirth}
            onChange={(e) => setDateOfBirth(e.target.value)}
          />
        </label>
        <label className="flex items-start gap-2 text-sm text-muted">
          <input
            className="mt-1"
            type="checkbox"
            name="termsAccepted"
            checked={termsAccepted}
            onChange={(e) => setTermsAccepted(e.target.checked)}
          />
          <span>{termsLabel}</span>
        </label>
        <button
          type="submit"
          disabled={pending}
          className="mt-2 rounded-md bg-accent px-4 py-2 font-medium text-on-accent hover:opacity-90 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent disabled:opacity-60"
        >
          {submitSignUp}
        </button>
      </form>
      <p className="text-sm text-subtle">
        {haveAccountPrompt}{" "}
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
