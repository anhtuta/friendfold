import { afterEach, describe, expect, it, vi } from "vitest";

import { fetchCurrentUser, readErrorMessage } from "./client";

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
