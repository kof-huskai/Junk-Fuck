import { useEffect, useState } from "react";
import { ShieldCheck } from "lucide-react";
import { useI18n } from "../i18n";
import { useStore } from "../lib/store";
import { dashboardCounts, relativeTime, type RelativeTime } from "../lib/dashboard";
import { driveTypeKey, formatBytes } from "../lib/format";
import { Button, Card } from "../components/ui";

export function Dashboard({ onNavigate }: { onNavigate: (page: string) => void }) {
  const { t } = useI18n();
  const { scanning, candidates, lastScan, drives, drivesLoaded, selectedRoot, startScan, refreshCandidates } = useStore();

  // Canonical last-scan summary (persisted backend-side, survives restarts;
  // cancelled/failed scans never replace it). Counts fall back to the live
  // session while a scan is in progress or before the first scan.
  const counts = dashboardCounts(lastScan, scanning, candidates);

  // A lightweight minute-level tick keeps the relative "Last scan" label
  // fresh while the page stays open (e.g. "Just now" → "1 min ago"). It
  // only runs while a last scan actually exists.
  const [now, setNow] = useState(() => new Date());
  useEffect(() => {
    if (!lastScan) return;
    const id = setInterval(() => setNow(new Date()), 60_000);
    return () => clearInterval(id);
  }, [lastScan]);

  const lastTime: RelativeTime | null = lastScan ? relativeTime(lastScan.completedAt, now) : null;
  const renderRel = (rt: RelativeTime) =>
    rt.kind === "justNow"
      ? t("time.justNow")
      : rt.kind === "unit"
        ? t(rt.key, { n: rt.n })
        : t("time.date", { date: rt.date });

  // The single shared target state: prefer the backend-detected drive label.
  const selected = drives.find((d) => d.root.toLowerCase() === selectedRoot.toLowerCase());
  const hasTarget = drivesLoaded ? drives.some((d) => d.ready) : true;
  const targetLabel = selected
    ? `${selected.root} — ${selected.label || t(driveTypeKey(selected.type))}`
    : selectedRoot || t("dash.noTarget");

  const run = async () => {
    try {
      await startScan([selectedRoot]);
    } catch {
      // Backend rejected the target (e.g. drive disappeared) — stay put.
      return;
    }
    onNavigate("scanner");
    setTimeout(() => void refreshCandidates(), 500);
  };

  return (
    <div className="flex flex-col gap-6 p-8">
      <h1 className="text-2xl font-bold text-white">{t("dash.title")}</h1>

      <div className="grid grid-cols-2 gap-4">
        <Card>
          <p className="text-xs font-medium uppercase tracking-wide text-muted">{t("dash.status")}</p>
          <p className="mt-2 text-2xl font-bold text-white">
            {scanning ? t("dash.status.scanning") : t("dash.status.idle")}
          </p>
        </Card>

        <Card>
          <p className="text-xs font-medium uppercase tracking-wide text-muted">{t("dash.items")}</p>
          <p className="mt-2 text-2xl font-bold text-white">{counts.items}</p>
          <p className="mt-1 text-xs text-muted">{t("dash.lastScan")}: {lastTime ? renderRel(lastTime) : t("dash.never")}</p>
        </Card>

        <Card>
          <p className="text-xs font-medium uppercase tracking-wide text-muted">{t("dash.reclaimable")}</p>
          <p className="mt-2 text-2xl font-bold text-accent">{formatBytes(counts.reclaimable)}</p>
          {lastScan && <p className="mt-1 text-xs text-muted">{lastScan.target}</p>}
        </Card>

        <Card>
          <p className="text-xs font-medium uppercase tracking-wide text-muted">{t("dash.safety")}</p>
          <p className="mt-2 flex items-center gap-2 text-lg font-semibold text-white">
            <ShieldCheck size={18} className="shrink-0 text-success" />
            {t("dash.safety.ok")}
          </p>
        </Card>
      </div>

      <Card className="flex items-center justify-between gap-4">
        <div className="min-w-0">
          <h2 className="font-semibold text-white">
            {t("dash.target")}: <span className="font-mono text-sm" dir="ltr">{targetLabel}</span>
          </h2>
          <p className="mt-1 text-sm text-muted">{t("res.none")}</p>
        </div>
        <Button
          variant="primary"
          disabled={scanning || !hasTarget}
          onClick={() => void run()}
        >
          {t("dash.start")} →
        </Button>
      </Card>
    </div>
  );
}
