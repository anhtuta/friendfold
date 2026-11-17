import { afterEach, describe, expect, it, vi } from "vitest";

import {
  fetchCurrentUser,
  fetchPublicProfile,
  readErrorMessage,
  uploadProfileImage,
} from "./client";

afterEach(() => {
  vi.unstubAllGlobals();
});

describe("readErrorMessage", () => {
  it("reads the envelope message", async () => {
    const res = new Response(
      JSON.stringify({
        error: { code: "invalid_credentials", message: "Email or password is incorrect" },
      }),
      { status: 401, headers: { "Content-Type": "application/json" } },
    );
    await expect(readErrorMessage(res)).resolves.toBe(
      "Email or password is incorrect",
    );
  });
});

describe("fetchCurrentUser", () => {
  it("returns null on 401", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(new Response("{}", { status: 401 })),
    );
    await expect(fetchCurrentUser()).resolves.toBeNull();
  });

  it("returns the user on 200", async () => {
    const user = {
      id: "1",
      email: "minh@example.com",
      emailVerified: false,
      displayName: "Minh",
    };
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify(user), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      ),
    );
    await expect(fetchCurrentUser()).resolves.toEqual(user);
  });
});

describe("fetchPublicProfile", () => {
  it("returns null on 404", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(new Response("{}", { status: 404 })),
    );
    await expect(fetchPublicProfile("nobodyhere")).resolves.toBeNull();
  });

  it("returns the public profile on 200", async () => {
    const profile = {
      id: "1",
      username: "minh",
      displayName: "Minh",
      bio: "Hi",
      avatarUrl: null,
      coverUrl: null,
    };
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        new Response(JSON.stringify(profile), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      ),
    );
    await expect(fetchPublicProfile("minh")).resolves.toEqual(profile);
  });
});

describe("uploadProfileImage", () => {
  it("sends FormData without a JSON Content-Type", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          id: "1",
          username: "minh",
          displayName: "Minh",
          bio: "",
          avatarUrl: "http://localhost:8080/v1/media/profile/1/avatar/example.jpg",
          coverUrl: null,
          email: "minh@example.com",
          emailVerified: true,
          usernameChangedAt: null,
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      ),
    );
    vi.stubGlobal("fetch", fetchMock);
    const file = new File([new Uint8Array([0xff, 0xd8])], "a.jpg", {
      type: "image/jpeg",
    });
    await uploadProfileImage("avatar", file);
    const callArgs = fetchMock.mock.calls[0];
    expect(callArgs).toBeDefined();
    if (callArgs === undefined) {
      throw new Error("expected fetch to be called");
    }
    const init = callArgs[1] as RequestInit;
    expect(init.body).toBeInstanceOf(FormData);
    const headers = new Headers(init.headers);
    expect(headers.get("Content-Type")).toBeNull();
  });
});
