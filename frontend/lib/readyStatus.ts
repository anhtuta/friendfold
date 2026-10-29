export type ReadyKind = "ok" | "unavailable" | "unreachable";

export type ReadyStatus = {
  kind: ReadyKind;
};

const readyTimeoutMs = 3000;

export async function fetchReadyStatus(
  baseUrl: string,
  fetchImpl: typeof fetch = fetch,
): Promise<ReadyStatus> {
  const url = `${baseUrl.replace(/\/+$/, "")}/v1/ready`;

  let response: Response;
  try {
    response = await fetchImpl(url, {
      method: "GET",
      headers: { Accept: "application/json" },
      cache: "no-store",
      signal: AbortSignal.timeout(readyTimeoutMs),
    });
  } catch {
    return { kind: "unreachable" };
  }

  if (response.status === 503) {
    return { kind: "unavailable" };
  }
  if (response.status !== 200) {
    return { kind: "unreachable" };
  }

  try {
    const body: unknown = await response.json();
    if (isOkBody(body)) {
      return { kind: "ok" };
    }
  } catch {
    return { kind: "unreachable" };
  }
  return { kind: "unreachable" };
}

function isOkBody(body: unknown): body is { status: "ok" } {
  if (typeof body !== "object" || body === null) {
    return false;
  }
  return Reflect.get(body, "status") === "ok";
}
