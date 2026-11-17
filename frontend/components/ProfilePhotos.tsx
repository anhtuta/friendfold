"use client";

import { useState } from "react";

import {
  deleteProfileImage,
  uploadProfileImage,
  type MeProfile,
} from "@/lib/api/client";
import { clientImageHint } from "@/lib/clientImageHint";
import {
  avatarLabel,
  chooseAvatarLabel,
  chooseCoverLabel,
  coverLabel,
  genericAuthError,
  photoTypeHint,
  removeAvatarLabel,
  removeCoverLabel,
} from "@/lib/copy";

import { Avatar } from "./Avatar";
import { CoverPhoto } from "./CoverPhoto";

type ProfilePhotosProps = {
  profile: MeProfile;
  refresh: () => Promise<void>;
};

export function ProfilePhotos({ profile, refresh }: ProfilePhotosProps) {
  const [error, setError] = useState<string | null>(null);
  const [pending, setPending] = useState<"avatar" | "cover" | "none">("none");

  async function onFile(kind: "avatar" | "cover", file: File | undefined) {
    if (!file) {
      return;
    }
    const hint = clientImageHint(file, kind);
    if (hint) {
      setError(hint);
      return;
    }
    setError(null);
    setPending(kind);
    try {
      await uploadProfileImage(kind, file);
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : genericAuthError);
    } finally {
      setPending("none");
    }
  }

  async function onRemove(kind: "avatar" | "cover") {
    setError(null);
    setPending(kind);
    try {
      await deleteProfileImage(kind);
      await refresh();
    } catch (err) {
      setError(err instanceof Error ? err.message : genericAuthError);
    } finally {
      setPending("none");
    }
  }

  return (
    <section className="flex flex-col gap-4">
      {error ? (
        <p role="alert" className="text-sm text-danger">
          {error}
        </p>
      ) : null}
      <div>
        <p className="text-sm text-muted">{avatarLabel}</p>
        <div className="mt-2 flex items-center gap-4">
          <Avatar displayName={profile.displayName} src={profile.avatarUrl} size="md" />
          <label className="text-sm text-accent underline underline-offset-2">
            {chooseAvatarLabel}
            <input
              className="sr-only"
              type="file"
              name="avatar"
              accept="image/jpeg,image/png,image/webp"
              disabled={pending !== "none"}
              onChange={(e) => {
                void onFile("avatar", e.target.files?.[0]);
                e.target.value = "";
              }}
            />
          </label>
          {profile.avatarUrl ? (
            <button
              type="button"
              disabled={pending !== "none"}
              onClick={() => {
                void onRemove("avatar");
              }}
              className="text-sm text-accent underline underline-offset-2 hover:text-foreground"
            >
              {removeAvatarLabel}
            </button>
          ) : null}
        </div>
      </div>
      <div>
        <p className="text-sm text-muted">{coverLabel}</p>
        <div className="mt-2 overflow-hidden rounded-md border border-border">
          <CoverPhoto displayName={profile.displayName} src={profile.coverUrl} />
        </div>
        <div className="mt-2 flex gap-4">
          <label className="text-sm text-accent underline underline-offset-2">
            {chooseCoverLabel}
            <input
              className="sr-only"
              type="file"
              name="cover"
              accept="image/jpeg,image/png,image/webp"
              disabled={pending !== "none"}
              onChange={(e) => {
                void onFile("cover", e.target.files?.[0]);
                e.target.value = "";
              }}
            />
          </label>
          {profile.coverUrl ? (
            <button
              type="button"
              disabled={pending !== "none"}
              onClick={() => {
                void onRemove("cover");
              }}
              className="text-sm text-accent underline underline-offset-2 hover:text-foreground"
            >
              {removeCoverLabel}
            </button>
          ) : null}
        </div>
      </div>
      <p className="text-sm text-subtle">{photoTypeHint}</p>
    </section>
  );
}
