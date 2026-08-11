import { useEffect, useRef, useState } from "react";
import { RefreshCw } from "lucide-react";
import { useI18n } from "../i18n";
import { useStore } from "../lib/store";
import { driveTypeKey, formatBytes } from "../lib/format";
import { Button, Card, ProgressBar, Select } from "../components/ui";
import * as settingsSvc from "../../bindings/github.com/kof-huskai/Junk-Fuck/services/settingsservice";

export function Scanner() {
  const { t } = useI18n();
  const { scanning, progress, scanErrors, scanId, cancelled, drives, drivesLoaded, drivesError, driveSwitchedFrom, selectedRoot, setSelectedRoot, refreshDrives, startScan, cancelScan, systemInfo } = useStore();
  const [error, setError] = useState<string | null>(null);
  const [refreshing, setRefreshing] = useState(false);
  // One quiet admin hint per scan, and only for permission-related errors:
  // the first access-denied error sets it; extra permission errors in the
  // same scan never repeat it, and a new scan resets it.
  const [showAdminHint, setShowAdminHint] = useState(false);
  const [relaunchMsg, setRelaunchMsg] = useState<string | null>(null);
  const adminHintShown = useRef(false);

  // Refresh the drive list whenever the Scanner page is opened, so newly
  // attached/removed drives are reflected without continuous polling.
  useEffect(() => {
    void refreshDrives();
  }, [refreshDrives]);

  // A new scan = a fresh hint budget.
  useEffect(() => {
    adminHintShown.current = false;
    setShowAdminHint(false);
    setRelaunchMsg(null);
  }, [scanId]);

  // Show the hint once, only when a permission-classified error arrives and
  // the app is not already elevated (no elevation loop).
  useEffect(() => {
    if (adminHintShown.current || !systemInfo || systemInfo.isAdmin) return;
    if (scanErrors.some((e) => e.permission)) {
      adminHintShown.current = true;
      setShowAdminHint(true);
    }
  }, [scanErrors, systemInfo]);

  const relaunchAdmin = async () => {
    setRelaunchMsg(null);
    try {
      const ok = await settingsSvc.RelaunchElevated();
      // ok=true: the elevated instance is starting and this one will quit.
      // ok=false: UAC was cancelled — the current app stays open.
      setRelaunchMsg(ok ? t("scan.adminRelaunching") : t("scan.adminCancelled"));
    } catch (err) {
      setRelaunchMsg(String(err));
    }
  };

  const readyDrives = drives.filter((d) => d.ready);
  const selectedIsDrive = drives.some((d) => d.root.toLowerCase() === selectedRoot.toLowerCase());
  const selectedStillValid = readyDrives.some((d) => d.root.toLowerCase() === selectedRoot.toLowerCase());

  const run = async () => {
    setError(null);
    if (!selectedRoot || !selectedStillValid) {
      setError(t("scan.pick"));
      return;
    }
    try {
      await startScan([selectedRoot]);
    } catch (err) {
      // Backend validation failures (stale drive, removed drive, malformed
      // path) surface here and stay visible on the page.
      setError(String(err));
      void refreshDrives();
    }
  };

  const doRefresh = async () => {
    setRefreshing(true);
    await refreshDrives();
    setRefreshing(false);
  };

  return (
    <div className="flex flex-col gap-6 p-8">
      <div>
        <h1 className="text-2xl font-bold text-white">{t("scan.title")}</h1>
        <p className="mt-1 text-sm text-muted">{t("scan.targetsHint")}</p>
      </div>

      <Card>
        <label className="mb-2 block text-xs font-medium uppercase tracking-wide text-muted">{t("scan.target")}</label>
        <div className="flex items-center gap-2">
          <Select
            value={selectedRoot}
            onChange={(e) => setSelectedRoot(e.target.value)}
            disabled={scanning || !drivesLoaded || drives.length === 0}
            className="min-w-0 flex-1"
          >
            {drives.length === 0 && (
              <option value={selectedRoot}>{drivesLoaded ? t("dash.noTarget") : "…"}</option>
            )}
            {/* A non-drive root (set via Settings' raw path editor) has no
                matching option — show it explicitly so the dropdown never
                silently displays a different target than what is scanned. */}
            {drives.length > 0 && !selectedIsDrive && (
              <option value={selectedRoot}>{selectedRoot} ({t("scan.customTarget")})</option>
            )}
            {drives.map((d) => (
              <option key={d.root} value={d.root} disabled={!d.ready}>
                {d.root} — {d.label || t(driveTypeKey(d.type))}
                {d.ready ? ` · ${formatBytes(d.freeBytes)} free` : ` (${t("scan.notReady")})`}
              </option>
            ))}
          </Select>
          <Button size="sm" variant="ghost" onClick={() => void doRefresh()} disabled={scanning || refreshing} title={t("scan.refresh")}>
            <RefreshCw size={15} className={refreshing ? "animate-spin" : ""} />
          </Button>
          {scanning ? (
            <Button variant="danger" onClick={cancelScan}>
              {t("scan.cancel")}
            </Button>
          ) : (
            <Button variant="primary" onClick={() => void run()} disabled={!selectedStillValid}>
              {t("scan.start")}
            </Button>
          )}
        </div>
        {(error || (drivesLoaded && !selectedStillValid)) && (
          <p className="mt-2 text-sm text-danger">
            {error ?? t("scan.pick")}
          </p>
        )}
        {drivesError && !error && <p className="mt-2 text-sm text-danger">{drivesError}</p>}
        {!error && driveSwitchedFrom && (
          <p className="mt-2 text-sm text-warn">
            {t("scan.driveSwitched", { old: driveSwitchedFrom, new: selectedRoot })}
          </p>
        )}
      </Card>

      {/* Status + progress */}
      <Card>
        <div className="flex items-baseline justify-between">
          <p className="text-xs font-medium uppercase tracking-wide text-muted">{t("dash.status")}</p>
          <p className="text-sm font-semibold text-white">
            {scanning ? t("scan.running") : cancelled ? t("scan.cancelled") : t("dash.status.idle")}
          </p>
        </div>

        {scanning && progress && (
          <>
            <div className="mt-4">
              <ProgressBar value={progress.scannedFiles} max={Math.max(progress.scannedFiles + 1, 10)} />
            </div>
            <p className="mt-3 truncate font-mono text-xs text-muted" dir="ltr">
              {t("scan.current")}: {progress.currentPath || "—"}
            </p>
          </>
        )}

        <div className="mt-4 grid grid-cols-3 gap-4 text-sm">
          <div>
            <p className="text-xs text-muted">{t("scan.files")}</p>
            <p className="mt-1 font-semibold text-white">{progress?.scannedFiles.toLocaleString() ?? 0}</p>
          </div>
          <div>
            <p className="text-xs text-muted">{t("scan.candidates")}</p>
            <p className="mt-1 font-semibold text-accent">{progress?.candidates ?? 0}</p>
          </div>
          <div>
            <p className="text-xs text-muted">{t("scan.errors")}</p>
            <p className="mt-1 font-semibold text-warn">{progress?.errors ?? 0}</p>
          </div>
        </div>

        {/* Quiet one-time hint after the first permission-related scan error.
            Small and muted on purpose; the error count above stays accurate. */}
        {showAdminHint && (
          <p className="mt-3 rounded-md border border-warn/25 bg-warn/10 px-3 py-2 text-xs leading-relaxed text-warn">
            {t("scan.adminHint")}{" "}
            <button
              type="button"
              onClick={() => void relaunchAdmin()}
              className="rounded-[4px] font-medium underline decoration-warn/50 underline-offset-2 hover:text-white focus:outline-none focus-visible:ring-2 focus-visible:ring-warn/60"
            >
              {t("scan.adminRelaunch")}
            </button>
            {relaunchMsg && <span className="mt-1 block opacity-90">{relaunchMsg}</span>}
          </p>
        )}
        {relaunchMsg && !showAdminHint && <p className="mt-2 text-xs text-muted">{relaunchMsg}</p>}
      </Card>

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

      {!scanning && scanId && !cancelled && (
        <p className="text-sm text-muted">{t("scan.done", { secs: "—" })}</p>
      )}
    </div>
  );
}
