import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import type { MeProfile } from "@/lib/api/client";
import {
  chooseAvatarLabel,
  photoTypeRejected,
  removeAvatarLabel,
} from "@/lib/copy";

import { ProfilePhotos } from "./ProfilePhotos";

const profile: MeProfile = {
  id: "1",
  username: "minh",
  displayName: "Minh",
  bio: "",
  avatarUrl: null,
  coverUrl: null,
  email: "minh@example.com",
  emailVerified: true,
  usernameChangedAt: null,
};

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("ProfilePhotos", () => {
  it("rejects a GIF on the client before uploading", async () => {
    const fetchMock = vi.fn();
    vi.stubGlobal("fetch", fetchMock);
    render(<ProfilePhotos profile={profile} refresh={async () => {}} />);
    const input = screen.getByLabelText(chooseAvatarLabel);
    const gif = new File([new Uint8Array([71, 73, 70])], "x.gif", {
      type: "image/gif",
    });
    fireEvent.change(input, { target: { files: [gif] } });
    expect(screen.getByRole("alert")).toHaveTextContent(photoTypeRejected);
    expect(fetchMock).not.toHaveBeenCalled();
  });

  it("uploads a JPEG and refreshes", async () => {
    const refresh = vi.fn(async () => {});
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            ...profile,
            avatarUrl:
              "http://localhost:8080/v1/media/profile/1/avatar/example-object.jpg",
          }),
          { status: 200, headers: { "Content-Type": "application/json" } },
        ),
      ),
    );
    render(<ProfilePhotos profile={profile} refresh={refresh} />);
    const user = userEvent.setup();
    const jpeg = new File([new Uint8Array([0xff, 0xd8, 0xff])], "a.jpg", {
      type: "image/jpeg",
    });
    await user.upload(screen.getByLabelText(chooseAvatarLabel), jpeg);
    await waitFor(() => expect(refresh).toHaveBeenCalled());
  });

  it("removes an existing avatar", async () => {
    const refresh = vi.fn(async () => {});
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify({ ...profile, avatarUrl: null }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      ),
    );
    render(
      <ProfilePhotos
        profile={{
          ...profile,
          avatarUrl:
            "http://localhost:8080/v1/media/profile/1/avatar/example-object.jpg",
        }}
        refresh={refresh}
      />,
    );
    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: removeAvatarLabel }));
    await waitFor(() => expect(refresh).toHaveBeenCalled());
  });
});
