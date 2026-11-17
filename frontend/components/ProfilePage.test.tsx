import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import type { PublicProfile } from "@/lib/api/client";
import {
  profileNotFoundBody,
  profileNotFoundHeading,
  profilePostsEmpty,
} from "@/lib/copy";

import { ProfilePage } from "./ProfilePage";

const minh: PublicProfile = {
  id: "193848201234567890",
  username: "minh",
  displayName: "Minh",
  bio: "Into the fold of friends.",
  avatarUrl: null,
  coverUrl: null,
};

describe("ProfilePage", () => {
  it("shows name, handle, bio, and empty posts", () => {
    render(<ProfilePage profile={minh} />);
    expect(
      screen.getByRole("heading", { level: 1, name: "Minh" }),
    ).toBeInTheDocument();
    expect(screen.getByText("@minh")).toBeInTheDocument();
    expect(screen.getByText("Into the fold of friends.")).toBeInTheDocument();
    expect(screen.getByText(profilePostsEmpty)).toBeInTheDocument();
    expect(screen.getByRole("img", { name: "Minh" })).toBeInTheDocument();
    expect(screen.queryByText(/minh@example.com/)).not.toBeInTheDocument();
    expect(screen.queryByText(/email/i)).not.toBeInTheDocument();
  });

  it("renders a 404 state for a missing user", () => {
    render(<ProfilePage profile={null} />);
    expect(
      screen.getByRole("heading", { level: 1, name: profileNotFoundHeading }),
    ).toBeInTheDocument();
    expect(screen.getByText(profileNotFoundBody)).toBeInTheDocument();
    expect(screen.queryByText(profilePostsEmpty)).not.toBeInTheDocument();
  });

  it("uses the photo alt text when an avatar is present", () => {
    render(
      <ProfilePage
        profile={{
          ...minh,
          avatarUrl: "http://localhost:8080/v1/media/profile/1/avatar/example-object.jpg",
          coverUrl: "http://localhost:8080/v1/media/profile/1/cover/example-object.jpg",
        }}
      />,
    );
    expect(screen.getByRole("img", { name: "Minh" })).toHaveAttribute(
      "src",
      "http://localhost:8080/v1/media/profile/1/avatar/example-object.jpg",
    );
    expect(
      screen.getByRole("img", { name: "Minh's cover photo" }),
    ).toBeInTheDocument();
  });
});
