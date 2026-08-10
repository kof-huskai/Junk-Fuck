import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import * as scannerSvc from "../../bindings/github.com/kof-huskai/Junk-Fuck/services/scannerservice";
import * as cleanupSvc from "../../bindings/github.com/kof-huskai/Junk-Fuck/services/cleanupservice";
import * as settingsSvc from "../../bindings/github.com/kof-huskai/Junk-Fuck/services/settingsservice";
import * as updateSvc from "../../bindings/github.com/kof-huskai/Junk-Fuck/services/updateservice";
import type * as model from "../../bindings/github.com/kof-huskai/Junk-Fuck/internal/model/models";
import type * as platform from "../../bindings/github.com/kof-huskai/Junk-Fuck/internal/platform/models";
import type * as report from "../../bindings/github.com/kof-huskai/Junk-Fuck/internal/report/models";
import type * as services from "../../bindings/github.com/kof-huskai/Junk-Fuck/services/models";
import { Events } from "@wailsio/runtime";

export interface Settings {
  targets: string;
  dryRun: boolean;
}

interface StoreValue {
  scanId: string | null;
  scanning: boolean;
  cancelled: boolean;
  progress: model.Progress | null;
  candidates: model.Candidate[];
  scanErrors: model.ScanError[];
  lastReport: report.Report | null;
  systemInfo: platform.Info | null;
  appInfo: Record<string, string> | null;
  protectedCount: number;
  settings: Settings;
  updateState: services.UpdateState | null;
  setSettings: (s: Partial<Settings>) => void;
  startScan: (targets: string[]) => Promise<void>;
  cancelScan: () => void;
  refreshCandidates: () => Promise<void>;
  refreshMeta: () => Promise<void>;
  cleanup: (dryRun: boolean, selected: string[]) => Promise<report.Report | null>;
  checkForUpdates: () => Promise<void>;
  installUpdate: () => Promise<void>;
  refreshUpdate: () => Promise<void>;
}

const Ctx = createContext<StoreValue | null>(null);

export function AppStoreProvider({ children }: { children: React.ReactNode }) {
  const [scanId, setScanId] = useState<string | null>(null);
  const [scanning, setScanning] = useState(false);
  const [cancelled, setCancelled] = useState(false);
  const [progress, setProgress] = useState<model.Progress | null>(null);
  const [candidates, setCandidates] = useState<model.Candidate[]>([]);
  const [scanErrors, setScanErrors] = useState<model.ScanError[]>([]);
  const [lastReport, setLastReport] = useState<report.Report | null>(null);
  const [systemInfo, setSystemInfo] = useState<platform.Info | null>(null);
  const [appInfo, setAppInfo] = useState<Record<string, string> | null>(null);
  const [protectedCount, setProtectedCount] = useState(0);
  const [updateState, setUpdateState] = useState<services.UpdateState | null>(null);
  const [settings, setSettingsState] = useState<Settings>(() => ({
    targets: localStorage.getItem("jf.targets") ?? "C:\\",
    dryRun: localStorage.getItem("jf.dryRun") !== "0",
  }));

  const setSettings = useCallback((s: Partial<Settings>) => {
    setSettingsState((prev) => {
      const next = { ...prev, ...s };
      localStorage.setItem("jf.targets", next.targets);
      localStorage.setItem("jf.dryRun", next.dryRun ? "1" : "0");
      return next;
    });
  }, []);

  const refreshCandidates = useCallback(async () => {
    if (!scanId) return;
    const list = await scannerSvc.GetCandidates(scanId);
    if (list) setCandidates(list);
  }, [scanId]);

  const refreshMeta = useCallback(async () => {
    const [info, app, paths] = await Promise.all([
      scannerSvc.GetSystemInfo().catch(() => null),
      settingsSvc.GetAppInfo().catch(() => null),
      scannerSvc.GetProtectedPaths().catch(() => null),
    ]);
    if (info) setSystemInfo(info);
    if (app) setAppInfo(app as Record<string, string>);
    setProtectedCount((paths ?? []).length);
  }, []);

  const refreshUpdate = useCallback(async () => {
    setUpdateState(await updateSvc.GetUpdateState().catch(() => null));
  }, []);

  const startScan = useCallback(async (targets: string[]) => {
    setScanErrors([]);
    setCandidates([]);
    setLastReport(null);
    setCancelled(false);
    setScanning(true);
    try {
      const id = await scannerSvc.StartScan(targets);
      setScanId(id);
      setProgress({ scanId: id, scannedFiles: 0, candidates: 0, errors: 0, currentPath: "", done: false });
    } catch (err) {
      setScanning(false);
      throw err;
    }
  }, []);

  const cancelScan = useCallback(() => {
    if (scanId) void scannerSvc.CancelScan(scanId);
  }, [scanId]);

  const cleanup = useCallback(
    async (dryRun: boolean, selected: string[]) => {
      if (!scanId) return null;
      const rep = await cleanupSvc.Cleanup(dryRun, scanId, selected);
      setLastReport(rep);
      return rep;
    },
    [scanId],
  );

  const checkForUpdates = useCallback(async () => {
    setUpdateState(await updateSvc.CheckForUpdates());
  }, []);

  const installUpdate = useCallback(async () => {
    setUpdateState(await updateSvc.InstallUpdate());
  }, []);

  const value = useMemo<StoreValue>(
    () => ({
      scanId,
      scanning,
      cancelled,
      progress,
      candidates,
      scanErrors,
      lastReport,
      systemInfo,
      appInfo,
      protectedCount,
      settings,
      updateState,
      setSettings,
      startScan,
      cancelScan,
      refreshCandidates,
      refreshMeta,
      cleanup,
      checkForUpdates,
      installUpdate,
      refreshUpdate,
    }),
    [scanId, scanning, cancelled, progress, candidates, scanErrors, lastReport, systemInfo, appInfo, protectedCount, settings, updateState, setSettings, startScan, cancelScan, refreshCandidates, refreshMeta, cleanup, checkForUpdates, installUpdate, refreshUpdate],
  );

  // Live event wiring (runs once per provider mount; StrictMode-safe).
  useEffect(() => {
    const offProgress = Events.On("scan:progress", (ev) => {
      setProgress(ev.data as model.Progress);
    });
    const offDone = Events.On("scan:done", (ev) => {
      const e = ev.data as { scanId: string; cancelled: boolean; error?: string };
      setScanning(false);
      setCancelled(!!e.cancelled);
      if (e.error) {
        setScanErrors((prev) => [...prev, { path: "", error: e.error ?? "unknown error" }]);
      }
      void (async () => {
        try {
          const list = await scannerSvc.GetCandidates(e.scanId);
          if (list) setCandidates(list);
        } catch {
          // session expired; ignore
        }
        try {
          const errs = await scannerSvc.GetScanErrors(e.scanId);
          if (errs) setScanErrors(errs);
        } catch {
          // no errors yet
        }
      })();
    });
    return () => {
      offProgress();
      offDone();
      Events.Off("scan:progress", "scan:done");
    };
  }, []);

  // Keep the updater state fresh after install/check operations.
  useEffect(() => {
    void refreshUpdate();
  }, [refreshUpdate]);

  return <Ctx.Provider value={value}>{children}</Ctx.Provider>;
}

export function useStore(): StoreValue {
  const c = useContext(Ctx);
  if (!c) throw new Error("useStore must be used inside AppStoreProvider");
  return c;
}
