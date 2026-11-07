import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { signInLabel, signOutLabel, signUpLabel } from "@/lib/copy";

import { AuthContext, type AuthState } from "./AuthProvider";
import { SiteHeader } from "./SiteHeader";

function renderHeader(state: Partial<AuthState> = {}) {
  const logout = state.logout ?? (async () => {});
  const value: AuthState = {
    user: state.user ?? null,
    ready: state.ready ?? true,
    refresh: state.refresh ?? (async () => {}),
    logout,
  };
  return { ...render(
    <AuthContext.Provider value={value}>
      <SiteHeader />
    </AuthContext.Provider>,
  ), logout };
}

describe("SiteHeader", () => {
  it("links to sign-in and sign-up when anonymous", () => {
    renderHeader();
    expect(screen.getByRole("link", { name: signInLabel })).toHaveAttribute(
      "href",
      "/login",
    );
    expect(screen.getByRole("link", { name: signUpLabel })).toHaveAttribute(
      "href",
      "/signup",
    );
  });

  it("signs out the current user", async () => {
    const logout = vi.fn(async () => {});
    renderHeader({
      user: {
        id: "1",
        email: "minh@example.com",
        emailVerified: false,
        displayName: "Minh",
      },
      logout,
    });
    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: signOutLabel }));
    expect(logout).toHaveBeenCalledTimes(1);
  });
});
