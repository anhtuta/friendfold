import { describe, expect, it } from "vitest";

import { fetchReadyStatus } from "./readyStatus";

describe("fetchReadyStatus", () => {
  it.each([
    {
      name: "ready 200",
      status: 200,
      body: { status: "ok" },
      want: "ok",
    },
    {
      name: "ready 503",
      status: 503,
      body: { status: "unavailable" },
      want: "unavailable",
    },
    {
      name: "unexpected 500",
      status: 500,
      body: { status: "ok" },
      want: "unreachable",
    },
    {
      name: "200 with the wrong JSON",
      status: 200,
      body: { status: "nope" },
      want: "unreachable",
    },
  ])("$name", async ({ status, body, want }) => {
    const fetchImpl: typeof fetch = async () =>
      new Response(JSON.stringify(body), {
        status,
        headers: { "Content-Type": "application/json" },
      });

    await expect(fetchReadyStatus("http://localhost:8080", fetchImpl)).resolves.toEqual(
      { kind: want },
    );
  });

  it("returns unreachable when fetch throws", async () => {
    const fetchImpl: typeof fetch = async () => {
      throw new TypeError("network down");
    };

    await expect(fetchReadyStatus("http://localhost:8080", fetchImpl)).resolves.toEqual(
      { kind: "unreachable" },
    );
  });
});
