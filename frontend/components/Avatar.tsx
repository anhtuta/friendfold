import { displayInitials } from "@/lib/displayInitials";

type AvatarProps = {
  displayName: string;
  src: string | null;
  size?: "md" | "lg";
};

const sizeClass = {
  md: "h-10 w-10 text-sm",
  lg: "h-24 w-24 text-2xl",
} as const;

export function Avatar({ displayName, src, size = "lg" }: AvatarProps) {
  const box = `overflow-hidden rounded-full border border-border bg-surface ${sizeClass[size]}`;
  if (src) {
    return (
      // User-controlled URL after server transcode; next/image remote allowlist is later.
      // eslint-disable-next-line @next/next/no-img-element
      <img
        src={src}
        alt={displayName}
        className={`${box} object-cover`}
      />
    );
  }
  return (
    <div
      role="img"
      aria-label={displayName}
      className={`${box} flex items-center justify-center font-semibold text-foreground`}
    >
      {displayInitials(displayName)}
    </div>
  );
}
