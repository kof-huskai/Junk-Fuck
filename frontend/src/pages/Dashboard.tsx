import { useI18n } from "../i18n";
import { useStore } from "../lib/store";
import { formatBytes } from "../lib/format";
import { Badge, Button, Card } from "../components/ui";

export function Dashboard({ onNavigate }: { onNavigate: (page: string) => void }) {
  const { t } = useI18n();
  const { scanning, candidates, lastReport, settings, startScan, refreshCandidates } = useStore();

  const total = candidates.reduce((s, c) => s + c.size, 0);
  const lastSize = lastReport ? lastReport.bytesFreed : 0;

  return (
    <div className="flex flex-col gap-6 p-8">
      <div>
        <h1 className="text-2xl font-bold text-white">{t("dash.title")}</h1>
        <p className="mt-1 text-sm text-muted">{t("app.tagline")}</p>
      </div>

      <div className="grid grid-cols-2 gap-4 xl:grid-cols-4">
        <Card>
          <p className="text-xs font-medium uppercase tracking-wide text-muted">{t("dash.status")}</p>
          <div className="mt-2 flex items-center gap-2">
            <span className={`h-2.5 w-2.5 rounded-full ${scanning ? "animate-pulse bg-warn" : "bg-success"}`} />
            <span className="text-lg font-semibold text-white">
              {scanning ? t("dash.status.scanning") : t("dash.status.idle")}
            </span>
          </div>
        </Card>

        <Card>
          <p className="text-xs font-medium uppercase tracking-wide text-muted">{t("dash.items")}</p>
          <p className="mt-2 text-2xl font-bold text-white">{candidates.length}</p>
          <p className="mt-0.5 text-xs text-muted">{t("dash.lastScan")}: {lastReport ? "—" : t("dash.never")}</p>
        </Card>

        <Card>
          <p className="text-xs font-medium uppercase tracking-wide text-muted">{t("dash.reclaimable")}</p>
          <p className="mt-2 text-2xl font-bold text-accent">{formatBytes(total)}</p>
          <p className="mt-0.5 text-xs text-muted">{t("dash.lastScan")}: {lastReport ? formatBytes(lastSize) : t("dash.never")}</p>
        </Card>

        <Card>
          <p className="text-xs font-medium uppercase tracking-wide text-muted">{t("dash.safety")}</p>
          <div className="mt-2">
            <Badge color="green">{t("dash.safety.ok")}</Badge>
          </div>
        </Card>
      </div>

      <Card className="flex items-center justify-between gap-4">
        <div>
          <h2 className="font-semibold text-white">{t("dash.target")}: {settings.targets || "C:\\"}</h2>
          <p className="mt-1 text-sm text-muted">{t("res.none")}</p>
        </div>
        <Button
          variant="primary"
          disabled={scanning}
          onClick={async () => {
            await startScan(settings.targets.split(",").map((s) => s.trim()).filter(Boolean));
            onNavigate("scanner");
            setTimeout(() => void refreshCandidates(), 500);
          }}
        >
          {t("dash.start")} →
        </Button>
      </Card>
    </div>
  );
}
