import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import {
  forgotPasswordHeading,
  forgotPasswordSent,
  forgotPasswordSubmit,
} from "@/lib/copy";

import { ForgotPasswordForm } from "./ForgotPasswordForm";

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("ForgotPasswordForm", () => {
  it("renders the request form", () => {
    render(<ForgotPasswordForm />);
    expect(
      screen.getByRole("heading", { level: 1, name: forgotPasswordHeading }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole("button", { name: forgotPasswordSubmit }),
    ).toBeEnabled();
  });

  it("shows the same success copy on 202", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(new Response(null, { status: 202 })),
    );
    const user = userEvent.setup();
    render(<ForgotPasswordForm />);
    await user.type(screen.getByLabelText("Email"), "minh@example.com");
    await user.click(screen.getByRole("button", { name: forgotPasswordSubmit }));
    expect(await screen.findByRole("status")).toHaveTextContent(
      forgotPasswordSent,
    );
  });

  it("shows the API error when forgot is rejected", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            error: { code: "validation_error", message: "email is invalid" },
          }),
          { status: 400, headers: { "Content-Type": "application/json" } },
        ),
      ),
    );
    const user = userEvent.setup();
    render(<ForgotPasswordForm />);
    await user.type(screen.getByLabelText("Email"), "not-an-email");
    await user.click(screen.getByRole("button", { name: forgotPasswordSubmit }));
    expect(await screen.findByRole("alert")).toHaveTextContent(
      "email is invalid",
    );
  });
});
