// Dashboard domain logic kept pure and testable. The backend's canonical
// ScanSummary (RFC3339 UTC timestamp) is the single source of truth for
// "last scan"; the UI only ever renders a timestamp, it never stores one.

export type RelativeTime =
  | { kind: "justNow" }
  | { kind: "unit"; key: "time.minAgo" | "time.hrAgo" | "time.daysAgo"; n: number }
  | { kind: "date"; date: string };

// relativeTime renders a stored RFC3339 timestamp as a relative label. The
// UI maps the result to localized strings; nothing here formats text itself.
export function relativeTime(iso: string, now: Date = new Date()): RelativeTime | null {
  const at = new Date(iso).getTime();
  if (Number.isNaN(at)) return null;
  const secs = Math.max(0, Math.floor((now.getTime() - at) / 1000));
  if (secs < 60) return { kind: "justNow" };
  const mins = Math.floor(secs / 60);
  if (mins < 60) return { kind: "unit", key: "time.minAgo", n: mins };
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return { kind: "unit", key: "time.hrAgo", n: hrs };
  const days = Math.floor(hrs / 24);
  if (days < 30) return { kind: "unit", key: "time.daysAgo", n: days };
  // Older than ~30 days: a compact absolute date is more useful than a
  // vague "47 days ago".
  return { kind: "date", date: new Date(at).toLocaleDateString() };
}

export interface ScanCounts {
  items: number;
  reclaimable: number;
}

// dashboardCounts derives the Dashboard summary values from the canonical
// last-completed scan. While a scan is in progress the live session counts
// are shown instead (they converge on the summary at completion); before
// the first scan the live (empty) session is used. After a successful scan
// these values agree with the Results page by construction.
export function dashboardCounts(
  lastScan: { junkItems: number; reclaimableBytes: number } | null,
  scanning: boolean,
  candidates: { size: number }[],
): ScanCounts {
  const live = candidates.reduce((s, c) => s + c.size, 0);
  if (lastScan && !scanning) {
    return { items: lastScan.junkItems, reclaimable: lastScan.reclaimableBytes };
  }
  return { items: candidates.length, reclaimable: live };
}
