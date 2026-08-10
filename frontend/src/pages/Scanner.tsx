import { useState } from "react";
import { useI18n } from "../i18n";
import { useStore } from "../lib/store";
import { Button, Card, EmptyState, Input, ProgressBar } from "../components/ui";

export function Scanner() {
  const { t } = useI18n();
  const { scanning, progress, scanErrors, scanId, cancelled, settings, startScan, cancelScan } = useStore();
  const [targets, setTargets] = useState(settings.targets || "C:\\");
  const [error, setError] = useState<string | null>(null);

  const run = async () => {
    setError(null);
    const list = targets.split(",").map((s) => s.trim()).filter(Boolean);
    if (list.length === 0) {
      setError(t("scan.pick"));
      return;
    }
    try {
      await startScan(list);
    } catch (err) {
      setError(String(err));
    }
  };

  return (
    <div className="flex flex-col gap-6 p-8">
      <div>
        <h1 className="text-2xl font-bold text-white">{t("scan.title")}</h1>
        <p className="mt-1 text-sm text-muted">{t("scan.targetsHint")}</p>
      </div>

      <Card>
        <label className="mb-2 block text-xs font-medium uppercase tracking-wide text-muted">{t("scan.target")}</label>
        <div className="flex gap-2">
          <Input value={targets} onChange={(e) => setTargets(e.target.value)} disabled={scanning} spellCheck={false} />
          {scanning ? (
            <Button variant="danger" onClick={cancelScan}>
              {t("scan.cancel")}
            </Button>
          ) : (
            <Button variant="primary" onClick={() => void run()}>
              {t("scan.start")}
            </Button>
          )}
        </div>
        {error && <p className="mt-2 text-sm text-danger">{error}</p>}
      </Card>

      {!scanning && !scanId && <EmptyState title={t("scan.pick")} />}

      {scanning && progress && (
        <Card>
          <div className="mb-2 flex items-center justify-between text-sm">
            <span className="font-medium text-white">{t("scan.running")}</span>
            <span className="text-muted">{progress.candidates} {t("scan.candidates")}</span>
          </div>
          <ProgressBar value={progress.scannedFiles} max={Math.max(progress.scannedFiles + 1, 10)} />
          <div className="mt-4 grid grid-cols-3 gap-4 text-sm">
            <div>
              <p className="text-xs text-muted">{t("scan.files")}</p>
              <p className="mt-1 font-semibold text-white">{progress.scannedFiles.toLocaleString()}</p>
            </div>
            <div>
              <p className="text-xs text-muted">{t("scan.candidates")}</p>
              <p className="mt-1 font-semibold text-accent">{progress.candidates}</p>
            </div>
            <div>
              <p className="text-xs text-muted">{t("scan.errors")}</p>
              <p className="mt-1 font-semibold text-warn">{progress.errors}</p>
            </div>
          </div>
          <p className="mt-4 truncate rounded-lg bg-panel-2 px-3 py-2 font-mono text-xs text-muted" dir="ltr">
            {t("scan.current")}: {progress.currentPath || "—"}
          </p>
        </Card>
      )}

      {!scanning && scanId && (
        <Card className="text-sm">
          <p className="font-medium text-white">
            {cancelled ? t("scan.cancelled") : t("scan.done", { secs: "—" })}
          </p>
          <p className="mt-1 text-muted">
            {t("res.found", { n: progress?.candidates ?? 0, size: "—" })}
          </p>
        </Card>
      )}

      {scanErrors.length > 0 && (
        <Card>
          <h3 className="mb-3 text-sm font-semibold text-warn">{t("scan.errors")}: {scanErrors.length}</h3>
          <div className="max-h-40 overflow-y-auto">
            {scanErrors.slice(0, 100).map((e, i) => (
              <p key={i} className="truncate py-0.5 font-mono text-xs text-muted" dir="ltr">
                {e.path || "(root)"} — {e.error}
              </p>
            ))}
          </div>
        </Card>
      )}
    </div>
  );
}
