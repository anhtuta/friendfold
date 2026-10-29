import { productName, signInLabel, tagline } from "@/lib/copy";

export function HomePage() {
  return (
    <main className="mx-auto flex min-h-screen max-w-xl flex-col justify-center gap-4 px-6 py-16 text-zinc-900">
      <h1 className="text-4xl font-semibold tracking-tight">{productName}</h1>
      <p className="text-lg text-zinc-600">{tagline}</p>
      <p className="text-sm text-zinc-500">{signInLabel}</p>
    </main>
  );
}
