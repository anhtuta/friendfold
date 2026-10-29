import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { productName, signInLabel, signUpLabel, tagline } from "@/lib/copy";

import { AuthContext, type AuthState } from "./AuthProvider";
import { HomePage } from "./HomePage";

function renderHome(user: AuthState["user"] = null) {
  const value: AuthState = {
    user,
    ready: true,
    refresh: async () => {},
    logout: async () => {},
  };
  return render(
    <AuthContext.Provider value={value}>
      <HomePage />
    </AuthContext.Provider>,
  );
}

describe("HomePage", () => {
  it("renders the product name and tagline", () => {
    renderHome();

    expect(
      screen.getByRole("heading", { level: 1, name: productName }),
    ).toBeInTheDocument();
    expect(screen.getByText(tagline)).toBeInTheDocument();
    expect(screen.getByText(tagline)).toHaveClass("text-muted");
  });

  it("links sign-in and sign-up when anonymous", () => {
    renderHome();

    expect(screen.getByRole("link", { name: signInLabel })).toHaveAttribute(
      "href",
      "/login",
    );
    expect(screen.getByRole("link", { name: signUpLabel })).toHaveAttribute(
      "href",
      "/signup",
    );
  });

  it("greets a signed-in user", () => {
    renderHome({
      id: "1",
      email: "minh@example.com",
      emailVerified: false,
      displayName: "Minh",
    });

    expect(screen.getByText(/Signed in as Minh/)).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: signInLabel })).not.toBeInTheDocument();
  });
});
