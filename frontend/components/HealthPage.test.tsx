import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import {
  healthHeading,
  healthOk,
  healthUnavailable,
  healthUnreachable,
} from "@/lib/copy";
import type { ReadyKind } from "@/lib/readyStatus";

import { HealthPage } from "./HealthPage";

describe("HealthPage", () => {
  it.each([
    { kind: "ok" as ReadyKind, label: healthOk },
    { kind: "unavailable" as ReadyKind, label: healthUnavailable },
    { kind: "unreachable" as ReadyKind, label: healthUnreachable },
  ])("shows $kind as $label", ({ kind, label }) => {
    render(<HealthPage status={{ kind }} />);

    expect(
      screen.getByRole("heading", { level: 1, name: healthHeading }),
    ).toBeInTheDocument();
    expect(screen.getByRole("status")).toHaveTextContent(label);
  });
});
