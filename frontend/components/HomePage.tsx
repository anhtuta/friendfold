import { productName, tagline } from "@/lib/copy";

import { HomeAuthLinks } from "./HomeAuthLinks";

export function HomePage() {
  return (
    <main className="mx-auto flex min-h-[calc(100vh-4rem)] max-w-xl flex-col justify-center gap-4 px-6 py-16">
      <h1 className="text-4xl font-semibold tracking-tight text-foreground">
        {productName}
      </h1>
      <p className="text-lg text-muted">{tagline}</p>
      <HomeAuthLinks />
    </main>
  );
}
