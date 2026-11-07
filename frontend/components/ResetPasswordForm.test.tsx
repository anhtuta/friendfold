import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import {
  passwordMismatch,
  resetPasswordHeading,
  resetPasswordInvalid,
  resetPasswordMissingToken,
  resetPasswordSubmit,
} from "@/lib/copy";

import { ResetPasswordForm } from "./ResetPasswordForm";

const nav = {
  token: "",
  push: vi.fn(),
};

vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: nav.push }),
  useSearchParams: () =>
    new URLSearchParams(nav.token ? `token=${nav.token}` : ""),
}));

afterEach(() => {
  vi.unstubAllGlobals();
  nav.token = "";
  nav.push.mockReset();
});

describe("ResetPasswordForm", () => {
  it("shows an error when the token is missing", () => {
    render(<ResetPasswordForm />);
    expect(
      screen.getByRole("heading", { level: 1, name: resetPasswordHeading }),
    ).toBeInTheDocument();
    expect(screen.getByRole("alert")).toHaveTextContent(
      resetPasswordMissingToken,
    );
  });

  it("shows mismatch without calling the API", async () => {
    nav.token = "example-reset-token";
    const fetchMock = vi.fn();
    vi.stubGlobal("fetch", fetchMock);
    const user = userEvent.setup();
    render(<ResetPasswordForm />);
    await user.type(screen.getByLabelText("New password"), "correct-horse-1");
    await user.type(
      screen.getByLabelText("Confirm new password"),
      "correct-horse-2",
    );
    await user.click(screen.getByRole("button", { name: resetPasswordSubmit }));
    expect(await screen.findByRole("alert")).toHaveTextContent(passwordMismatch);
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("redirects to login on 204", async () => {
    nav.token = "example-reset-token";
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(new Response(null, { status: 204 })),
    );
    const user = userEvent.setup();
    render(<ResetPasswordForm />);
    await user.type(screen.getByLabelText("New password"), "correct-horse-battery");
    await user.type(
      screen.getByLabelText("Confirm new password"),
      "correct-horse-battery",
    );
    await user.click(screen.getByRole("button", { name: resetPasswordSubmit }));
    await waitFor(() => {
      expect(nav.push).toHaveBeenCalledWith("/login");
    });
  });

  it("shows invalid token errors", async () => {
    nav.token = "example-reset-token";
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            error: {
              code: "invalid_token",
              message: resetPasswordInvalid,
            },
          }),
          { status: 400, headers: { "Content-Type": "application/json" } },
        ),
      ),
    );
    const user = userEvent.setup();
    render(<ResetPasswordForm />);
    await user.type(screen.getByLabelText("New password"), "correct-horse-battery");
    await user.type(
      screen.getByLabelText("Confirm new password"),
      "correct-horse-battery",
    );
    await user.click(screen.getByRole("button", { name: resetPasswordSubmit }));
    expect(await screen.findByRole("alert")).toHaveTextContent(
      resetPasswordInvalid,
    );
  });
});
