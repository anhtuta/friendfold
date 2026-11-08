import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import {
  changePasswordHeading,
  changePasswordSubmit,
  changePasswordSuccess,
  passwordMismatch,
} from "@/lib/copy";

import { AuthContext, type AuthState } from "./AuthProvider";
import { ChangePasswordForm } from "./ChangePasswordForm";

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
  ready: true,
  refresh: async () => {},
  logout: async () => {},
};

function renderForm(state: AuthState = signedIn) {
  return render(
    <AuthContext.Provider value={state}>
      <ChangePasswordForm />
    </AuthContext.Provider>,
  );
}

afterEach(() => {
  vi.unstubAllGlobals();
  nav.replace.mockReset();
});

describe("ChangePasswordForm", () => {
  it("renders the signed-in form", () => {
    renderForm();
    expect(
      screen.getByRole("heading", { level: 1, name: changePasswordHeading }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: changePasswordSubmit }),
    ).toBeEnabled();
  });

  it("sends anonymous visitors to login", async () => {
    renderForm({
      user: null,
      ready: true,
      refresh: async () => {},
      logout: async () => {},
    });
    await waitFor(() => {
      expect(nav.replace).toHaveBeenCalledWith("/login");
    });
  });

  it("shows mismatch without calling the API", async () => {
    const fetchMock = vi.fn();
    vi.stubGlobal("fetch", fetchMock);
    const user = userEvent.setup();
    renderForm();
    await user.type(screen.getByLabelText("Current password"), "old-password-1");
    await user.type(screen.getByLabelText("New password"), "correct-horse-1");
    await user.type(
      screen.getByLabelText("Confirm new password"),
      "correct-horse-2",
    );
    await user.click(screen.getByRole("button", { name: changePasswordSubmit }));
    expect(await screen.findByRole("alert")).toHaveTextContent(passwordMismatch);
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("shows the API error when the current password is wrong", async () => {
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
    renderForm();
    await user.type(screen.getByLabelText("Current password"), "wrong-password-1");
    await user.type(screen.getByLabelText("New password"), "correct-horse-battery");
    await user.type(
      screen.getByLabelText("Confirm new password"),
      "correct-horse-battery",
    );
    await user.click(screen.getByRole("button", { name: changePasswordSubmit }));
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "Email or password is incorrect",
    );
  });

  it("shows success on 204", async () => {
    const refresh = vi.fn(async () => {});
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(new Response(null, { status: 204 })),
    );
    const user = userEvent.setup();
    renderForm({ ...signedIn, refresh });
    await user.type(screen.getByLabelText("Current password"), "old-password-1");
    await user.type(screen.getByLabelText("New password"), "correct-horse-battery");
    await user.type(
      screen.getByLabelText("Confirm new password"),
      "correct-horse-battery",
    );
    await user.click(screen.getByRole("button", { name: changePasswordSubmit }));
    expect(await screen.findByRole("status")).toHaveTextContent(
      changePasswordSuccess,
    );
    expect(refresh).toHaveBeenCalled();
  });
});
