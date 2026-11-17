import type { PublicProfile } from "@/lib/api/client";
import {
  profileHandlePrefix,
  profileNotFoundBody,
  profileNotFoundHeading,
  profilePostsEmpty,
} from "@/lib/copy";

import { Avatar } from "./Avatar";
import { CoverPhoto } from "./CoverPhoto";

type ProfilePageProps = {
  profile: PublicProfile | null;
};

export function ProfilePage({ profile }: ProfilePageProps) {
  if (profile === null) {
    return (
      <main className="mx-auto max-w-xl px-6 py-16">
        <h1 className="text-2xl font-semibold text-foreground">
          {profileNotFoundHeading}
        </h1>
        <p className="mt-2 text-muted">{profileNotFoundBody}</p>
      </main>
    );
  }

  return (
    <main>
      <CoverPhoto displayName={profile.displayName} src={profile.coverUrl} />
      <div className="mx-auto max-w-xl px-6 pb-16">
        <div className="-mt-12">
          <Avatar displayName={profile.displayName} src={profile.avatarUrl} />
        </div>
        <h1 className="mt-4 text-2xl font-semibold text-foreground">
          {profile.displayName}
        </h1>
        <p className="text-sm text-subtle">
          {profileHandlePrefix}
          {profile.username}
        </p>
        {profile.bio !== "" ? (
          <p className="mt-4 text-muted">{profile.bio}</p>
        ) : null}
        <section className="mt-10 border-t border-border pt-6">
          <p className="text-sm text-subtle">{profilePostsEmpty}</p>
        </section>
      </div>
    </main>
  );
}
