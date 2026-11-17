import { describe, expect, it } from "vitest";

import { photoTooLarge, photoTypeRejected } from "@/lib/copy";

import { clientImageHint } from "./clientImageHint";

describe("clientImageHint", () => {
  it("rejects GIF by MIME hint", () => {
    const file = new File([new Uint8Array([71, 73, 70])], "x.gif", {
      type: "image/gif",
    });
    expect(clientImageHint(file, "avatar")).toBe(photoTypeRejected);
  });

  it("rejects oversized avatars", () => {
    const file = new File([new Uint8Array(5 * 1024 * 1024 + 1)], "x.jpg", {
      type: "image/jpeg",
    });
    expect(clientImageHint(file, "avatar")).toBe(photoTooLarge);
  });

  it("allows a small JPEG", () => {
    const file = new File([new Uint8Array([0xff, 0xd8])], "x.jpg", {
      type: "image/jpeg",
    });
    expect(clientImageHint(file, "avatar")).toBeNull();
  });
});
