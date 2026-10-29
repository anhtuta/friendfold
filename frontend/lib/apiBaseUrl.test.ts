import { afterEach, describe, expect, it } from "vitest";

import { apiBaseUrl } from "./apiBaseUrl";

describe("apiBaseUrl", () => {
  afterEach(() => {
    delete process.env.API_BASE_URL;
  });

  it("defaults to the local API origin", () => {
    expect(apiBaseUrl({})).toBe("http://localhost:8080");
  });

  it("uses API_BASE_URL origin and strips a path", () => {
    expect(apiBaseUrl({ API_BASE_URL: "http://127.0.0.1:8080/v1/" })).toBe(
      "http://127.0.0.1:8080",
    );
  });

  it("rejects a non-http URL", () => {
    expect(() => apiBaseUrl({ API_BASE_URL: "javascript:alert(1)" })).toThrow(
      /http\(s\)/,
    );
  });

  it("rejects a value that is too long", () => {
    expect(() =>
      apiBaseUrl({ API_BASE_URL: `http://localhost:8080/${"a".repeat(200)}` }),
    ).toThrow(/max length/);
  });
});
