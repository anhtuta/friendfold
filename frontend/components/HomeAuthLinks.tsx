"use client";

import Link from "next/link";

import {
  signInLabel,
  signUpLabel,
  signedInPrefix,
} from "@/lib/copy";

import { useAuth } from "./AuthProvider";

export function HomeAuthLinks() {
  const { user, ready } = useAuth();

  if (!ready) {
    return null;
  }

  if (user) {
    return (
      <p className="text-sm text-subtle">
        {signedInPrefix} {user.displayName}
      </p>
    );
  }

  return (
    <p className="flex gap-4 text-sm">
      <Link
        href="/login"
        className="text-accent underline underline-offset-2 hover:text-foreground focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent"
      >
        {signInLabel}
      </Link>
      <Link
        href="/signup"
        className="text-accent underline underline-offset-2 hover:text-foreground focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent"
      >
        {signUpLabel}
      </Link>
    </p>
  );
}
