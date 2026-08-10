import React, { createContext, useCallback, useContext, useEffect, useMemo, useState } from "react";
import type { model, platform, report } from "../../wailsjs/go/models";
import * as backend from "../../wailsjs/go/main/App";
import { EventsOff, EventsOn } from "../../wailsjs/runtime/runtime";

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
  setSettings: (s: Partial<Settings>) => void;
  startScan: (targets: string[]) => Promise<void>;
  cancelScan: () => void;
  refreshCandidates: () => Promise<void>;
  refreshMeta: () => Promise<void>;
  cleanup: (dryRun: boolean, selected: string[]) => Promise<report.Report | null>;
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
    setCandidates(await backend.GetCandidates(scanId));
  }, [scanId]);

  const refreshMeta = useCallback(async () => {
    const [info, app, paths] = await Promise.all([
      backend.GetSystemInfo().catch(() => null),
      backend.GetAppInfo().catch(() => null),
      backend.GetProtectedPaths().catch(() => [] as string[]),
    ]);
    if (info) setSystemInfo(info);
    if (app) setAppInfo(app);
    setProtectedCount(paths.length);
  }, []);

  const startScan = useCallback(async (targets: string[]) => {
    setScanErrors([]);
    setCandidates([]);
    setLastReport(null);
    setCancelled(false);
    setScanning(true);
    try {
      const id = await backend.StartScan(targets);
      setScanId(id);
      setProgress({ scanId: id, scannedFiles: 0, candidates: 0, errors: 0, currentPath: "", done: false });
    } catch (err) {
      setScanning(false);
      throw err;
    }
  }, []);

  const cancelScan = useCallback(() => {
    if (scanId) backend.CancelScan(scanId);
  }, [scanId]);

  const cleanup = useCallback(
    async (dryRun: boolean, selected: string[]) => {
      if (!scanId) return null;
      const rep = await backend.Cleanup(dryRun, scanId, selected);
      setLastReport(rep);
      return rep;
    },
    [scanId],
  );

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
      setSettings,
      startScan,
      cancelScan,
      refreshCandidates,
      refreshMeta,
      cleanup,
    }),
    [scanId, scanning, cancelled, progress, candidates, scanErrors, lastReport, systemInfo, appInfo, protectedCount, settings, setSettings, startScan, cancelScan, refreshCandidates, refreshMeta, cleanup],
  );

  // Live event wiring (runs once per provider mount; StrictMode-safe).
  useEffect(() => {
    EventsOn("scan:progress", (p: model.Progress) => {
      setProgress(p);
    });
    EventsOn("scan:done", (e: { scanId: string; cancelled: boolean; error?: string }) => {
      setScanning(false);
      setCancelled(!!e.cancelled);
      if (e.error) {
        setScanErrors((prev) => [...prev, { path: "", error: e.error ?? "unknown error" }]);
      }
      void (async () => {
        try {
          setCandidates(await backend.GetCandidates(e.scanId));
        } catch {
          // session expired; ignore
        }
        try {
          setScanErrors(await backend.GetScanErrors(e.scanId));
        } catch {
          // no errors yet
        }
      })();
    });
    return () => {
      EventsOff("scan:progress");
      EventsOff("scan:done");
    };
  }, []);

  return <Ctx.Provider value={value}>{children}</Ctx.Provider>;
}

export function useStore(): StoreValue {
  const c = useContext(Ctx);
  if (!c) throw new Error("useStore must be used inside AppStoreProvider");
  return c;
}
