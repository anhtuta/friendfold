import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { productName, signInLabel, tagline } from "@/lib/copy";

import { HomePage } from "./HomePage";

describe("HomePage", () => {
  it("renders the product name and tagline", () => {
    render(<HomePage />);

    expect(
      screen.getByRole("heading", { level: 1, name: productName }),
    ).toBeInTheDocument();
    expect(screen.getByText(tagline)).toBeInTheDocument();
    expect(screen.getByText(tagline)).toHaveClass("text-muted");
    expect(screen.getByText(signInLabel)).toHaveClass("text-subtle");
  });

  it("shows sign-in as plain text, not a link", () => {
    render(<HomePage />);

    expect(screen.getByText(signInLabel)).toBeInTheDocument();
    expect(screen.queryByRole("link", { name: signInLabel })).not.toBeInTheDocument();
  });
});
