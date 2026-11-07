"use client";

import Link from "next/link";

import {
  authLoadingLabel,
  productName,
  signInLabel,
  signOutLabel,
  signUpLabel,
  signedInPrefix,
} from "@/lib/copy";

import { useAuth } from "./AuthProvider";

export function SiteHeader() {
  const { user, ready, logout } = useAuth();

  return (
    <header className="border-b border-border">
      <div className="mx-auto flex max-w-xl items-center justify-between gap-4 px-6 py-4">
        <Link
          href="/"
          className="text-sm font-semibold text-foreground focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent"
        >
          {productName}
        </Link>
        <nav aria-label="Account" className="flex items-center gap-4 text-sm">
          {!ready ? (
            <span className="text-subtle">{authLoadingLabel}</span>
          ) : user ? (
            <>
              <span className="text-muted">
                {signedInPrefix} {user.displayName}
              </span>
              <button
                type="button"
                onClick={() => {
                  void logout();
                }}
                className="text-accent underline underline-offset-2 hover:text-foreground focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent"
              >
                {signOutLabel}
              </button>
            </>
          ) : (
            <>
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
            </>
          )}
        </nav>
      </div>
    </header>
  );
}
