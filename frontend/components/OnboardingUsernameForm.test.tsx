import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import {
  onboardingUsernameHeading,
  onboardingUsernameSubmit,
} from "@/lib/copy";

import { AuthContext, type AuthState } from "./AuthProvider";
import { OnboardingUsernameForm } from "./OnboardingUsernameForm";

const nav = {
  replace: vi.fn(),
};

vi.mock("next/navigation", () => ({
  useRouter: () => ({ replace: nav.replace, push: vi.fn() }),
}));

const signedIn: AuthState = {
  user: {
    id: "1",
    email: "minh@example.com",
    emailVerified: true,
    displayName: "Minh",
  },
  profile: {
    id: "1",
    username: null,
    displayName: "Minh",
    bio: "",
    avatarUrl: null,
    coverUrl: null,
    email: "minh@example.com",
    emailVerified: true,
    usernameChangedAt: null,
  },
  ready: true,
  refresh: async () => {},
  logout: async () => {},
};

function renderForm(state: AuthState = signedIn) {
  return render(
    <AuthContext.Provider value={state}>
      <OnboardingUsernameForm />
    </AuthContext.Provider>,
  );
}

afterEach(() => {
  vi.unstubAllGlobals();
  nav.replace.mockClear();
});

describe("OnboardingUsernameForm", () => {
  it("renders the username field", () => {
    renderForm();
    expect(
      screen.getByRole("heading", { level: 1, name: onboardingUsernameHeading }),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: onboardingUsernameSubmit })).toBeEnabled();
  });

  it("saves a username and goes home", async () => {
    const refresh = vi.fn(async () => {});
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            id: "1",
            username: "minh",
            displayName: "Minh",
            bio: "",
            avatarUrl: null,
            coverUrl: null,
            email: "minh@example.com",
            emailVerified: true,
            usernameChangedAt: "2026-11-17T12:00:00Z",
          }),
          { status: 200, headers: { "Content-Type": "application/json" } },
        ),
      ),
    );
    renderForm({ ...signedIn, refresh });
    const user = userEvent.setup();
    await user.type(screen.getByLabelText(/^Username/), "minh");
    await user.click(screen.getByRole("button", { name: onboardingUsernameSubmit }));
    await waitFor(() => expect(refresh).toHaveBeenCalled());
    expect(nav.replace).toHaveBeenCalledWith("/");
  });

  it("shows a server validation error", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            error: { code: "username_reserved", message: "This username is not available" },
          }),
          { status: 400, headers: { "Content-Type": "application/json" } },
        ),
      ),
    );
    renderForm();
    const user = userEvent.setup();
    await user.type(screen.getByLabelText(/^Username/), "admin");
    await user.click(screen.getByRole("button", { name: onboardingUsernameSubmit }));
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "This username is not available",
    );
  });
});
