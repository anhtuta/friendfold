import { describe, expect, it } from "vitest";

import { publicApiBaseUrl } from "./publicApiBaseUrl";

describe("publicApiBaseUrl", () => {
  it("defaults to localhost:8080", () => {
    expect(publicApiBaseUrl({})).toBe("http://localhost:8080");
  });

  it("uses NEXT_PUBLIC_API_BASE_URL origin", () => {
    expect(
      publicApiBaseUrl({ NEXT_PUBLIC_API_BASE_URL: "http://127.0.0.1:8080/v1" }),
    ).toBe("http://127.0.0.1:8080");
  });

  it("rejects non-http URLs", () => {
    expect(() =>
      publicApiBaseUrl({ NEXT_PUBLIC_API_BASE_URL: "ftp://example.com" }),
    ).toThrow(/absolute http\(s\)/);
  });
});
