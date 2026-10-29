import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import {
  haveAccountPrompt,
  signupHeading,
  submitSignUp,
  termsLabel,
} from "@/lib/copy";

import { AuthContext, type AuthState } from "./AuthProvider";
import { SignupForm } from "./SignupForm";

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: vi.fn() }),
}));

function renderSignup() {
  const value: AuthState = {
    user: null,
    ready: true,
    refresh: async () => {},
    logout: async () => {},
  };
  return render(
    <AuthContext.Provider value={value}>
      <SignupForm />
    </AuthContext.Provider>,
  );
}

describe("SignupForm", () => {
  it("renders labeled fields", () => {
    renderSignup();
    expect(
      screen.getByRole("heading", { level: 1, name: signupHeading }),
    ).toBeInTheDocument();
    expect(screen.getByLabelText("Email")).toBeInTheDocument();
    expect(screen.getByLabelText("Password")).toBeInTheDocument();
    expect(screen.getByLabelText("Display name")).toBeInTheDocument();
    expect(screen.getByLabelText("Date of birth")).toBeInTheDocument();
    expect(screen.getByLabelText(termsLabel)).toBeInTheDocument();
    expect(screen.getByText(haveAccountPrompt)).toBeInTheDocument();
  });

  it("shows an error when terms are not accepted", async () => {
    const user = userEvent.setup();
    renderSignup();
    await user.type(screen.getByLabelText("Email"), "minh@example.com");
    await user.type(screen.getByLabelText("Password"), "correct-horse");
    await user.type(screen.getByLabelText("Display name"), "Minh");
    await user.type(screen.getByLabelText("Date of birth"), "1998-04-12");
    await user.click(screen.getByRole("button", { name: submitSignUp }));
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Terms of use and privacy policy must be accepted",
    );
  });
});
