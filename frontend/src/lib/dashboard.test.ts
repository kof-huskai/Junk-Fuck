import { describe, expect, it } from "vitest";
import { dashboardCounts, relativeTime } from "./dashboard";

const NOW = new Date("2026-08-11T12:00:00Z");

describe("relativeTime", () => {
  it("recent timestamps read as just now", () => {
    expect(relativeTime("2026-08-11T11:59:30Z", NOW)).toEqual({ kind: "justNow" });
  });

  it("minutes", () => {
    expect(relativeTime("2026-08-11T11:58:00Z", NOW)).toEqual({ kind: "unit", key: "time.minAgo", n: 2 });
  });

  it("hours", () => {
    expect(relativeTime("2026-08-11T09:00:00Z", NOW)).toEqual({ kind: "unit", key: "time.hrAgo", n: 3 });
  });

  it("days", () => {
    expect(relativeTime("2026-08-09T12:00:00Z", NOW)).toEqual({ kind: "unit", key: "time.daysAgo", n: 2 });
  });

  it("older than 30 days renders a date", () => {
    expect(relativeTime("2026-06-01T12:00:00Z", NOW)?.kind).toBe("date");
  });

  it("a future timestamp clamps to just now", () => {
    expect(relativeTime("2026-08-11T12:30:00Z", NOW)).toEqual({ kind: "justNow" });
  });

  it("invalid timestamps return null (never crash the Dashboard)", () => {
    expect(relativeTime("", NOW)).toBeNull();
    expect(relativeTime("not-a-date", NOW)).toBeNull();
  });
});

describe("dashboardCounts", () => {
  const last = { junkItems: 231, reclaimableBytes: 1_234_567 };

  it("reads the canonical last scan when idle (after restart too)", () => {
    expect(dashboardCounts(last, false, [])).toEqual({ items: 231, reclaimable: 1_234_567 });
  });

  it("shows live session counts while a scan is running", () => {
    const live = [{ size: 5 }, { size: 10 }];
    expect(dashboardCounts(last, true, live)).toEqual({ items: 2, reclaimable: 15 });
  });

  it("falls back to live candidates before the first scan", () => {
    const live = [{ size: 5 }];
    expect(dashboardCounts(null, false, live)).toEqual({ items: 1, reclaimable: 5 });
  });

  it("a completed scan's summary agrees with the candidate-derived totals", () => {
    // After a successful scan the store holds the same candidates the
    // summary was built from: the values must agree (Dashboard == Results).
    const candidates = [{ size: 100 }, { size: 200 }];
    const summary = { junkItems: candidates.length, reclaimableBytes: 300 };
    expect(dashboardCounts(summary, false, candidates)).toEqual({ items: 2, reclaimable: 300 });
  });
});
