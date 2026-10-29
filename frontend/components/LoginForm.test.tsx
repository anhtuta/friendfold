import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import { loginHeading, needAccountPrompt, submitSignIn } from "@/lib/copy";

import { AuthContext, type AuthState } from "./AuthProvider";
import { LoginForm } from "./LoginForm";

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: vi.fn() }),
}));

function renderLogin() {
  const value: AuthState = {
    user: null,
    ready: true,
    refresh: async () => {},
    logout: async () => {},
  };
  return render(
    <AuthContext.Provider value={value}>
      <LoginForm />
    </AuthContext.Provider>,
  );
}

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("LoginForm", () => {
  it("renders the sign-in form", () => {
    renderLogin();
    expect(
      screen.getByRole("heading", { level: 1, name: loginHeading }),
    ).toBeInTheDocument();
    expect(screen.getByRole("button", { name: submitSignIn })).toBeEnabled();
    expect(screen.getByText(needAccountPrompt)).toBeInTheDocument();
  });

  it("shows the API error on failed login", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            error: {
              code: "invalid_credentials",
              message: "Email or password is incorrect",
            },
          }),
          { status: 401, headers: { "Content-Type": "application/json" } },
        ),
      ),
    );
    const user = userEvent.setup();
    renderLogin();
    await user.type(screen.getByLabelText("Email"), "minh@example.com");
    await user.type(screen.getByLabelText("Password"), "wrong-password-1");
    await user.click(screen.getByRole("button", { name: submitSignIn }));
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Email or password is incorrect",
    );
  });
});
