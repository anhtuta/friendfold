"use client";

import { usePathname } from "next/navigation";
import { useState } from "react";

import { apiFetch, readErrorMessage } from "@/lib/api/client";
import {
  genericAuthError,
  resendVerificationLabel,
  resendVerificationSent,
  unverifiedBannerMessage,
} from "@/lib/copy";

import { useAuth } from "./AuthProvider";

export function UnverifiedEmailBanner() {
  const { user, ready } = useAuth();
  const pathname = usePathname();
  const [pending, setPending] = useState(false);
  const [message, setMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  if (!ready || !user || user.emailVerified || pathname === "/verify-email") {
    return null;
  }

  async function onResend() {
    setPending(true);
    setError(null);
    setMessage(null);
    try {
      const res = await apiFetch("/v1/auth/resend-verification", {
        method: "POST",
      });
      if (!res.ok) {
        setError(await readErrorMessage(res));
        return;
      }
      setMessage(resendVerificationSent);
    } catch {
      setError(genericAuthError);
    } finally {
      setPending(false);
    }
  }

  return (
    <div className="border-b border-border bg-surface">
      <div className="mx-auto flex max-w-xl flex-col gap-2 px-6 py-3 text-sm sm:flex-row sm:items-center sm:justify-between">
        <p className="text-muted">{unverifiedBannerMessage}</p>
        <div className="flex flex-col items-start gap-1 sm:items-end">
          <button
            type="button"
            onClick={() => {
              void onResend();
            }}
            disabled={pending}
            className="text-accent underline underline-offset-2 hover:text-foreground focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent disabled:opacity-60"
          >
            {resendVerificationLabel}
          </button>
          {message ? (
            <p role="status" className="text-xs text-subtle">
              {message}
            </p>
          ) : null}
          {error ? (
            <p role="alert" className="text-xs text-danger">
              {error}
            </p>
          ) : null}
        </div>
      </div>
    </div>
  );
}
