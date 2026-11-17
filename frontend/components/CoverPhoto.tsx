type CoverPhotoProps = {
  displayName: string;
  src: string | null;
};

export function CoverPhoto({ displayName, src }: CoverPhotoProps) {
  if (src) {
    return (
      // User-controlled URL after server transcode; next/image remote allowlist is later.
      // eslint-disable-next-line @next/next/no-img-element
      <img
        src={src}
        alt={`${displayName}'s cover photo`}
        className="h-48 w-full object-cover"
      />
    );
  }
  return (
    <div
      className="h-48 w-full bg-surface"
      aria-hidden="true"
    />
  );
}
