import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import type { MeProfile } from "@/lib/api/client";

import { AuthContext, type AuthState } from "./AuthProvider";
import { UsernameGate } from "./UsernameGate";

const nav = {
  path: "/",
  replace: vi.fn(),
};

vi.mock("next/navigation", () => ({
  usePathname: () => nav.path,
  useRouter: () => ({ replace: nav.replace, push: vi.fn() }),
}));

const me: MeProfile = {
  id: "1",
  username: null,
  displayName: "Minh",
  bio: "",
  avatarUrl: null,
  coverUrl: null,
  email: "minh@example.com",
  emailVerified: true,
  usernameChangedAt: null,
};

function renderGate(state: Partial<AuthState> = {}) {
  const value: AuthState = {
    user: state.user ?? {
      id: "1",
      email: "minh@example.com",
      emailVerified: true,
      displayName: "Minh",
    },
    profile: state.profile === undefined ? me : state.profile,
    ready: state.ready ?? true,
    refresh: state.refresh ?? (async () => {}),
    logout: state.logout ?? (async () => {}),
  };
  return render(
    <AuthContext.Provider value={value}>
      <UsernameGate>
        <p>home content</p>
      </UsernameGate>
    </AuthContext.Provider>,
  );
}

describe("UsernameGate", () => {
  it("sends users without a username away from home", () => {
    nav.path = "/";
    nav.replace.mockClear();
    renderGate();
    expect(nav.replace).toHaveBeenCalledWith("/onboarding/username");
    expect(screen.queryByText("home content")).not.toBeInTheDocument();
  });

  it("allows onboarding and public profiles without a username", () => {
    nav.path = "/onboarding/username";
    nav.replace.mockClear();
    renderGate();
    expect(nav.replace).not.toHaveBeenCalled();
    expect(screen.getByText("home content")).toBeInTheDocument();

    nav.path = "/u/minh";
    nav.replace.mockClear();
    renderGate();
    expect(nav.replace).not.toHaveBeenCalled();
  });

  it("does not redirect when a username is set", () => {
    nav.path = "/";
    nav.replace.mockClear();
    renderGate({ profile: { ...me, username: "minh" } });
    expect(nav.replace).not.toHaveBeenCalled();
    expect(screen.getByText("home content")).toBeInTheDocument();
  });
});
