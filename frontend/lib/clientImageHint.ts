import { photoTooLarge, photoTypeRejected } from "@/lib/copy";

const allowedTypes = new Set(["image/jpeg", "image/png", "image/webp"]);

const avatarMaxBytes = 5 * 1024 * 1024;
const coverMaxBytes = 10 * 1024 * 1024;

/** Client-side hint only. The API still sniffs magic bytes. */
export function clientImageHint(
  file: File,
  kind: "avatar" | "cover",
): string | null {
  if (file.type !== "" && !allowedTypes.has(file.type)) {
    return photoTypeRejected;
  }
  const max = kind === "avatar" ? avatarMaxBytes : coverMaxBytes;
  if (file.size > max) {
    return photoTooLarge;
  }
  return null;
}
