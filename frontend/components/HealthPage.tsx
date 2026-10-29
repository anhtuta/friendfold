import Link from "next/link";

import {
  healthHeading,
  healthOk,
  healthUnavailable,
  healthUnreachable,
  homeNavLabel,
} from "@/lib/copy";
import type { ReadyStatus } from "@/lib/readyStatus";

const statusLabel: Record<ReadyStatus["kind"], string> = {
  ok: healthOk,
  unavailable: healthUnavailable,
  unreachable: healthUnreachable,
};

export function HealthPage({ status }: { status: ReadyStatus }) {
  return (
    <main className="mx-auto flex min-h-screen max-w-xl flex-col justify-center gap-4 px-6 py-16">
      <h1 className="text-4xl font-semibold tracking-tight text-foreground">
        {healthHeading}
      </h1>
      <p role="status" className="text-lg text-muted">
        {statusLabel[status.kind]}
      </p>
      <p>
        <Link
          href="/"
          className="text-sm text-accent underline underline-offset-2 hover:text-foreground focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent"
        >
          {homeNavLabel}
        </Link>
      </p>
    </main>
  );
}
