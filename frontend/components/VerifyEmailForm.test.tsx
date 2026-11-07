import { render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import {
  verifyEmailHeading,
  verifyEmailInvalid,
  verifyEmailMissingToken,
  verifyEmailSuccess,
  verifyEmailWorking,
} from "@/lib/copy";

import { AuthContext, type AuthState } from "./AuthProvider";
import {
  resetVerifyEmailInflightForTests,
  VerifyEmailForm,
} from "./VerifyEmailForm";

const nav = {
  token: "",
  push: vi.fn(),
};

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: nav.push }),
  useSearchParams: () => new URLSearchParams(nav.token ? `token=${nav.token}` : ""),
}));

function renderForm() {
  const refresh = vi.fn(async () => {});
  const value: AuthState = {
    user: null,
    ready: true,
    refresh,
    logout: async () => {},
  };
  return {
    refresh,
    ...render(
      <AuthContext.Provider value={value}>
        <VerifyEmailForm />
      </AuthContext.Provider>,
    ),
  };
}

afterEach(() => {
  vi.unstubAllGlobals();
  nav.token = "";
  nav.push.mockReset();
  resetVerifyEmailInflightForTests();
});

describe("VerifyEmailForm", () => {
  it("shows an error when the token is missing", () => {
    nav.token = "";
    renderForm();
    expect(
      screen.getByRole("heading", { level: 1, name: verifyEmailHeading }),
    ).toBeInTheDocument();
    expect(screen.getByRole("alert")).toHaveTextContent(verifyEmailMissingToken);
  });

  it("verifies a token and redirects home", async () => {
    nav.token = "valid-token";
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(new Response(null, { status: 204 })),
    );
    const { refresh } = renderForm();
    expect(screen.getByText(verifyEmailWorking)).toBeInTheDocument();
    await waitFor(() => {
      expect(screen.getByText(verifyEmailSuccess)).toBeInTheDocument();
    });
    expect(refresh).toHaveBeenCalled();
    expect(nav.push).toHaveBeenCalledWith("/");
  });

  it("shows an error when the token is invalid", async () => {
    nav.token = "used-token";
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            error: {
              code: "invalid_token",
              message: verifyEmailInvalid,
            },
          }),
          { status: 400, headers: { "Content-Type": "application/json" } },
        ),
      ),
    );
    renderForm();
    expect(await screen.findByRole("alert")).toHaveTextContent(
      verifyEmailInvalid,
    );
    expect(nav.push).not.toHaveBeenCalled();
  });
});
