import { productName, signInLabel, tagline } from "@/lib/copy";

export function HomePage() {
  return (
    <main className="mx-auto flex min-h-screen max-w-xl flex-col justify-center gap-4 px-6 py-16">
      <h1 className="text-4xl font-semibold tracking-tight text-foreground">
        {productName}
      </h1>
      <p className="text-lg text-muted">{tagline}</p>
      <p className="text-sm text-subtle">{signInLabel}</p>
    </main>
  );
}
