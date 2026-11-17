"use client";

import { usePathname, useRouter } from "next/navigation";
import { useEffect } from "react";

import { usernameGateAllowed } from "@/lib/usernameGate";

import { useAuth } from "./AuthProvider";

export function UsernameGate({ children }: { children: React.ReactNode }) {
  const { user, profile, ready } = useAuth();
  const pathname = usePathname();
  const router = useRouter();
  const needsUsername =
    ready &&
    user !== null &&
    profile !== null &&
    profile.username === null &&
    !usernameGateAllowed(pathname);

  useEffect(() => {
    if (needsUsername) {
      router.replace("/onboarding/username");
    }
  }, [needsUsername, router]);

  if (needsUsername) {
    return null;
  }
  return children;
}
