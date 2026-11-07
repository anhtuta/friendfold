import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import {
  resendVerificationLabel,
  resendVerificationSent,
  unverifiedBannerMessage,
} from "@/lib/copy";

import { AuthContext, type AuthState } from "./AuthProvider";
import { UnverifiedEmailBanner } from "./UnverifiedEmailBanner";

const nav = { path: "/" };

vi.mock("next/navigation", () => ({
  usePathname: () => nav.path,
}));

function renderBanner(state: Partial<AuthState> = {}) {
  const value: AuthState = {
    user: state.user ?? {
      id: "1",
      email: "minh@example.com",
      emailVerified: false,
      displayName: "Minh",
    },
    ready: state.ready ?? true,
    refresh: state.refresh ?? (async () => {}),
    logout: state.logout ?? (async () => {}),
  };
  return render(
    <AuthContext.Provider value={value}>
      <UnverifiedEmailBanner />
    </AuthContext.Provider>,
  );
}

afterEach(() => {
  vi.unstubAllGlobals();
  nav.path = "/";
});

describe("UnverifiedEmailBanner", () => {
  it("is hidden when the user is verified", () => {
    renderBanner({
      user: {
        id: "1",
        email: "minh@example.com",
        emailVerified: true,
        displayName: "Minh",
      },
    });
    expect(screen.queryByText(unverifiedBannerMessage)).not.toBeInTheDocument();
  });

  it("is hidden on the verify-email page", () => {
    nav.path = "/verify-email";
    renderBanner();
    expect(screen.queryByText(unverifiedBannerMessage)).not.toBeInTheDocument();
  });

  it("resends a verification email", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(new Response(null, { status: 204 })),
    );
    renderBanner();
    expect(screen.getByText(unverifiedBannerMessage)).toBeInTheDocument();
    const user = userEvent.setup();
    await user.click(
      screen.getByRole("button", { name: resendVerificationLabel }),
    );
    expect(await screen.findByRole("status")).toHaveTextContent(
      resendVerificationSent,
    );
  });

  it("shows an error when resend fails", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            error: { code: "rate_limited", message: "Too many attempts. Try again later" },
          }),
          { status: 429, headers: { "Content-Type": "application/json" } },
        ),
      ),
    );
    renderBanner();
    const user = userEvent.setup();
    await user.click(
      screen.getByRole("button", { name: resendVerificationLabel }),
    );
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Too many attempts. Try again later",
    );
  });
});
