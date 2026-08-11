import { ShieldCheck } from "lucide-react";
import { useI18n } from "../i18n";
import { useStore } from "../lib/store";
import { formatBytes } from "../lib/format";
import { Button, Card } from "../components/ui";

export function Dashboard({ onNavigate }: { onNavigate: (page: string) => void }) {
  const { t } = useI18n();
  const { scanning, candidates, lastReport, drives, drivesLoaded, selectedRoot, startScan, refreshCandidates } = useStore();

  const total = candidates.reduce((s, c) => s + c.size, 0);
  const lastSize = lastReport ? lastReport.bytesFreed : 0;

  // The single shared target state: prefer the backend-detected drive label.
  const selected = drives.find((d) => d.root.toLowerCase() === selectedRoot.toLowerCase());
  const hasTarget = drivesLoaded ? drives.some((d) => d.ready) : true;
  const targetLabel = selected
    ? `${selected.root} — ${selected.label || selected.type}`
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
          <p className="mt-2 text-2xl font-bold text-white">{candidates.length}</p>
          <p className="mt-1 text-xs text-muted">{t("dash.lastScan")}: {lastReport ? "—" : t("dash.never")}</p>
        </Card>

        <Card>
          <p className="text-xs font-medium uppercase tracking-wide text-muted">{t("dash.reclaimable")}</p>
          <p className="mt-2 text-2xl font-bold text-accent">{formatBytes(total)}</p>
          <p className="mt-1 text-xs text-muted">{t("dash.lastScan")}: {lastReport ? formatBytes(lastSize) : t("dash.never")}</p>
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
