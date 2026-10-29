import { HealthPage } from "@/components/HealthPage";
import { apiBaseUrl } from "@/lib/apiBaseUrl";
import { fetchReadyStatus } from "@/lib/readyStatus";

export const dynamic = "force-dynamic";

// Next.js App Router requires a default export from app/health/page.tsx.
export default async function Page() {
  const status = await fetchReadyStatus(apiBaseUrl());
  return <HealthPage status={status} />;
}
