"use client";

import { useRouter, useSearchParams } from "next/navigation";
import { useEffect, useState } from "react";

import { apiFetch, readErrorMessage } from "@/lib/api/client";
import {
  genericAuthError,
  verifyEmailHeading,
  verifyEmailInvalid,
  verifyEmailMissingToken,
  verifyEmailSuccess,
  verifyEmailWorking,
} from "@/lib/copy";

import { useAuth } from "./AuthProvider";

type Status = "working" | "ok" | "invalid";

const inflight = new Map<
  string,
  Promise<{ status: number; message: string }>
>();

export function resetVerifyEmailInflightForTests(): void {
  inflight.clear();
}

function verifyToken(
  token: string,
): Promise<{ status: number; message: string }> {
  const hit = inflight.get(token);
  if (hit) {
    return hit;
  }
  const pending = (async () => {
    const res = await apiFetch("/v1/auth/verify-email", {
      method: "POST",
      body: JSON.stringify({ token }),
    });
    if (res.status === 204) {
      return { status: 204, message: "" };
    }
    return { status: res.status, message: await readErrorMessage(res) };
  })();
  inflight.set(token, pending);
  return pending;
}

export function VerifyEmailForm() {
  const router = useRouter();
  const params = useSearchParams();
  const token = params.get("token")?.trim() ?? "";
  const { refresh } = useAuth();
  const [status, setStatus] = useState<Status>(token ? "working" : "invalid");
  const [message, setMessage] = useState(
    token ? verifyEmailWorking : verifyEmailMissingToken,
  );

  useEffect(() => {
    if (!token) {
      return;
    }
    let cancelled = false;
    void verifyToken(token)
      .then(async (result) => {
        if (cancelled) {
          return;
        }
        if (result.status === 204) {
          setStatus("ok");
          setMessage(verifyEmailSuccess);
          await refresh();
          router.push("/");
          return;
        }
        setStatus("invalid");
        setMessage(result.message || verifyEmailInvalid);
      })
      .catch(() => {
        if (!cancelled) {
          setStatus("invalid");
          setMessage(genericAuthError);
        }
      });
    return () => {
      cancelled = true;
    };
  }, [token, refresh, router]);

  return (
    <main className="mx-auto flex min-h-[calc(100vh-4rem)] max-w-xl flex-col justify-center gap-4 px-6 py-16">
      <h1 className="text-3xl font-semibold tracking-tight text-foreground">
        {verifyEmailHeading}
      </h1>
      <p
        role={status === "invalid" ? "alert" : "status"}
        className={
          status === "invalid" ? "text-sm text-danger" : "text-sm text-muted"
        }
      >
        {message}
      </p>
    </main>
  );
}
