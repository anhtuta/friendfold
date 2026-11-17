import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import {
  settingsProfileHeading,
  settingsProfileSaved,
  settingsProfileSubmit,
  usernameCooldownMessage,
} from "@/lib/copy";

import { AuthContext, type AuthState } from "./AuthProvider";
import { SettingsProfileForm } from "./SettingsProfileForm";

vi.mock("next/navigation", () => ({
  useRouter: () => ({ replace: vi.fn(), push: vi.fn() }),
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
    username: "minh",
    displayName: "Minh",
    bio: "",
    avatarUrl: null,
    coverUrl: null,
    email: "minh@example.com",
    emailVerified: true,
    usernameChangedAt: "2020-01-01T00:00:00Z",
  },
  ready: true,
  refresh: async () => {},
  logout: async () => {},
};

function renderForm(state: AuthState = signedIn) {
  return render(
    <AuthContext.Provider value={state}>
      <SettingsProfileForm />
    </AuthContext.Provider>,
  );
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("SettingsProfileForm", () => {
  it("renders edit fields", () => {
    renderForm();
    expect(
      screen.getByRole("heading", { level: 1, name: settingsProfileHeading }),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: settingsProfileSubmit })).toBeEnabled();
  });

  it("saves a bio", async () => {
    const refresh = vi.fn(async () => {});
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            ...signedIn.profile,
            bio: "Into the fold of friends.",
          }),
          { status: 200, headers: { "Content-Type": "application/json" } },
        ),
      ),
    );
    renderForm({ ...signedIn, refresh });
    const user = userEvent.setup();
    await user.clear(screen.getByLabelText(/^Bio/));
    await user.type(screen.getByLabelText(/^Bio/), "Into the fold of friends.");
    await user.click(screen.getByRole("button", { name: settingsProfileSubmit }));
    await waitFor(() =>
      expect(screen.getByRole("status")).toHaveTextContent(settingsProfileSaved),
    );
    expect(refresh).toHaveBeenCalled();
  });

  it("shows a validation error", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            error: { code: "validation_error", message: "display name must be 1–80 characters" },
          }),
          { status: 400, headers: { "Content-Type": "application/json" } },
        ),
      ),
    );
    renderForm();
    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: settingsProfileSubmit }));
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "display name must be 1–80 characters",
    );
  });

  it("shows cooldown copy when a username was just set", () => {
    const profile = signedIn.profile;
    if (profile === null) {
      throw new Error("fixture");
    }
    renderForm({
      ...signedIn,
      profile: {
        ...profile,
        usernameChangedAt: new Date().toISOString(),
      },
    });
    expect(screen.getByText(usernameCooldownMessage)).toBeInTheDocument();
    expect(screen.getByLabelText(/^Username/)).toHaveAttribute("readOnly");
  });
});
