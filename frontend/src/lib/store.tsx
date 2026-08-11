import React, { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import * as scannerSvc from "../../bindings/github.com/kof-huskai/Junk-Fuck/services/scannerservice";
import * as cleanupSvc from "../../bindings/github.com/kof-huskai/Junk-Fuck/services/cleanupservice";
import * as settingsSvc from "../../bindings/github.com/kof-huskai/Junk-Fuck/services/settingsservice";
import * as updateSvc from "../../bindings/github.com/kof-huskai/Junk-Fuck/services/updateservice";
import * as rulesSvc from "../../bindings/github.com/kof-huskai/Junk-Fuck/services/rulesservice";
import type * as model from "../../bindings/github.com/kof-huskai/Junk-Fuck/internal/model/models";
import type * as platform from "../../bindings/github.com/kof-huskai/Junk-Fuck/internal/platform/models";
import type * as report from "../../bindings/github.com/kof-huskai/Junk-Fuck/internal/report/models";
import type * as services from "../../bindings/github.com/kof-huskai/Junk-Fuck/services/models";
import { Events } from "@wailsio/runtime";

export interface Settings {
  dryRun: boolean;
}

/** A scan that reached its terminal state, used to auto-navigate to Results
 *  only on real success (not cancellation, not failure). */
export interface CompletedScan {
  scanId: string;
  ok: boolean;
}

interface StoreValue {
  scanId: string | null;
  scanning: boolean;
  cancelled: boolean;
  progress: model.Progress | null;
  candidates: model.Candidate[];
  scanErrors: model.ScanError[];
  lastReport: report.Report | null;
  /** Canonical record of the most recent successful scan (Dashboard). */
  lastScan: model.ScanSummary | null;
  refreshLastScan: () => Promise<void>;
  systemInfo: platform.Info | null;
  appInfo: Record<string, string> | null;
  protectedCount: number;
  updateState: services.UpdateState | null;
  /** True while an update check is in flight (startup or manual). */
  updateChecking: boolean;
  /** Protection-whitelist rules snapshot (Settings → Protection rules). */
  rulesStatus: services.RulesStatus | null;
  /** Forces a fresh whitelist check (network) and updates rulesStatus. */
  refreshRules: () => Promise<void>;
  settings: Settings;
  setSettings: (s: Partial<Settings>) => void;
  // Drive state — single source of truth shared by Dashboard and Scanner.
  drives: model.DriveInfo[];
  drivesLoaded: boolean;
  drivesError: string | null;
  /** Set when the previously selected drive disappeared and we auto-switched. */
  driveSwitchedFrom: string | null;
  selectedRoot: string;
  setSelectedRoot: (root: string) => void;
  refreshDrives: () => Promise<void>;
  completedScan: CompletedScan | null;
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

/** Pick the first safe scan root from the backend drive list: the backend
 *  already orders fixed/system drives first, so index 0 is the best default. */
function defaultRoot(drives: model.DriveInfo[], fallback: string): string {
  const ready = drives.filter((d) => d.ready);
  if (ready.length > 0) return ready[0].root;
  return fallback;
}

export function AppStoreProvider({ children }: { children: React.ReactNode }) {
  const [scanId, setScanId] = useState<string | null>(null);
  const [scanning, setScanning] = useState(false);
  const [cancelled, setCancelled] = useState(false);
  const [progress, setProgress] = useState<model.Progress | null>(null);
  const [candidates, setCandidates] = useState<model.Candidate[]>([]);
  const [scanErrors, setScanErrors] = useState<model.ScanError[]>([]);
  const [lastReport, setLastReport] = useState<report.Report | null>(null);
  const [lastScan, setLastScan] = useState<model.ScanSummary | null>(null);
  const [systemInfo, setSystemInfo] = useState<platform.Info | null>(null);
  const [appInfo, setAppInfo] = useState<Record<string, string> | null>(null);
  const [protectedCount, setProtectedCount] = useState(0);
  const [updateState, setUpdateState] = useState<services.UpdateState | null>(null);
  const [rulesStatus, setRulesStatus] = useState<services.RulesStatus | null>(null);
  // Starts true because a startup check always runs: the Sidebar shows
  // "Checking…" from the very first frame instead of flashing "not checked".
  const [updateChecking, setUpdateChecking] = useState(true);
  // Exactly one background update check per app launch. A ref guard means
  // React StrictMode's double effect invocation cannot fire two checks.
  const startupCheckDone = useRef(false);
  const [settings, setSettingsState] = useState<Settings>(() => ({
    dryRun: localStorage.getItem("jf.dryRun") !== "0",
  }));
  const [drives, setDrives] = useState<model.DriveInfo[]>([]);
  const [drivesLoaded, setDrivesLoaded] = useState(false);
  const [drivesError, setDrivesError] = useState<string | null>(null);
  const [driveSwitchedFrom, setDriveSwitchedFrom] = useState<string | null>(null);
  const [selectedRoot, setSelectedRootState] = useState<string>(() => localStorage.getItem("jf.targets") ?? "C:\\");
  const [completedScan, setCompletedScan] = useState<CompletedScan | null>(null);
  // Mirrors for use inside async callbacks (avoids stale closures and keeps
  // state updaters pure).
  const selectedRootRef = useRef<string>(selectedRoot);
  const activeScanRef = useRef<string | null>(null);

  const setSettings = useCallback((s: Partial<Settings>) => {
    setSettingsState((prev) => {
      const next = { ...prev, ...s };
      localStorage.setItem("jf.dryRun", next.dryRun ? "1" : "0");
      return next;
    });
  }, []);

  // Single source of truth for the selected scan target. Both Dashboard and
  // Scanner read/write this; Settings' raw path editor writes it too.
  const setSelectedRoot = useCallback((root: string) => {
    const trimmed = root.trim();
    setSelectedRootState(trimmed);
    selectedRootRef.current = trimmed;
    setDriveSwitchedFrom(null);
    localStorage.setItem("jf.targets", trimmed);
  }, []);

  const refreshDrives = useCallback(async () => {
    setDrivesError(null);
    try {
      const list = (await scannerSvc.ListDrives()) ?? [];
      setDrives(list);
      setDrivesLoaded(true);
      // If the stored target is no longer an available/ready drive, fall
      // back to the backend's preferred (system/fixed) drive and surface a
      // short notice so the user knows the selection changed.
      const prev = selectedRootRef.current;
      const stillValid = list.some((d) => d.ready && d.root.toLowerCase() === prev.toLowerCase());
      if (!stillValid) {
        const next = defaultRoot(list, prev);
        setSelectedRootState(next);
        selectedRootRef.current = next;
        setDriveSwitchedFrom(prev);
        localStorage.setItem("jf.targets", next);
      }
    } catch (err) {
      setDrivesError(String(err));
      setDrivesLoaded(true);
    }
  }, []);

  const refreshCandidates = useCallback(async () => {
    if (!scanId) return;
    const list = await scannerSvc.GetCandidates(scanId);
    if (list) setCandidates(list);
  }, [scanId]);

  // Reads the backend's canonical last-scan summary (persisted, survives
  // restarts). Called after every successful completion and at startup; the
  // backend only records REAL successful terminal scans, so cancelled or
  // failed scans never show up here.
  const refreshLastScan = useCallback(async () => {
    const s = await scannerSvc.GetLastScanSummary().catch(() => null);
    setLastScan(s);
  }, []);

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
    setCompletedScan(null);
    setScanning(true);
    try {
      const id = await scannerSvc.StartScan(targets);
      activeScanRef.current = id;
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

  const refreshRules = useCallback(async () => {
    try {
      setRulesStatus(await rulesSvc.RefreshRules());
    } catch {
      // A failed check must not blank the card: re-read the backend's
      // snapshot (which now reports the error state) and keep showing the
      // last known version/count.
      setRulesStatus(await rulesSvc.GetRulesStatus().catch(() => null));
    }
  }, []);

  const checkForUpdates = useCallback(async () => {
    setUpdateChecking(true);
    try {
      setUpdateState(await updateSvc.CheckForUpdates());
    } catch {
      // A failed network/binding check must never block the app: pull
      // whatever the backend recorded (e.g. "error") so the Sidebar can
      // show a quiet "Update check failed". The user can retry from
      // Settings at any time.
      await refreshUpdate();
    } finally {
      setUpdateChecking(false);
    }
  }, [refreshUpdate]);

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
      lastScan,
      refreshLastScan,
      systemInfo,
      appInfo,
      protectedCount,
      updateState,
      updateChecking,
      rulesStatus,
      refreshRules,
      settings,
      setSettings,
      drives,
      drivesLoaded,
      drivesError,
      driveSwitchedFrom,
      selectedRoot,
      setSelectedRoot,
      refreshDrives,
      completedScan,
      startScan,
      cancelScan,
      refreshCandidates,
      refreshMeta,
      cleanup,
      checkForUpdates,
      installUpdate,
      refreshUpdate,
    }),
    [scanId, scanning, cancelled, progress, candidates, scanErrors, lastReport, lastScan, refreshLastScan, systemInfo, appInfo, protectedCount, updateState, updateChecking, rulesStatus, refreshRules, settings, drives, drivesLoaded, drivesError, driveSwitchedFrom, selectedRoot, setSelectedRoot, refreshDrives, completedScan, startScan, cancelScan, refreshCandidates, refreshMeta, cleanup, checkForUpdates, installUpdate, refreshUpdate],
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
        setScanErrors((prev) => [...prev, { path: "", error: e.error ?? "unknown error", permission: false }]);
      }
      // Persist the final results BEFORE signalling completion so the
      // Results page never navigates into an empty store.
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
        }		// The backend persisted the canonical summary BEFORE emitting
		// scan:done, so this read is already authoritative. Cancelled/failed
		// scans never record a summary, so this only ever shows the latest
		// REAL successful scan — and survives restarts via the backend file.
		void refreshLastScan();
		// One completed scan => exactly one completion signal. ok means
		// "finished with results" — cancelled and failed scans stay put.
		// The active-scan guard drops stale completions from a scan that was
		// superseded before its final fetch finished.
		if (activeScanRef.current === e.scanId) {
			setCompletedScan({ scanId: e.scanId, ok: !e.cancelled && !e.error });
		}
      })();
    });
    return () => {
      offProgress();
      offDone();
      Events.Off("scan:progress", "scan:done");
    };
  }, []);

  // Startup: load the initial system/app metadata (so the Scanner admin
  // hint can rely on systemInfo without the user ever visiting About), read
  // the updater snapshot, and kick off exactly ONE background update check.
  // Everything here is async and non-blocking — the UI is interactive
  // immediately.
  useEffect(() => {
    void refreshMeta();
    void refreshUpdate();
    // Restore the last successful scan across restarts (Dashboard + Results
    // both read from this single canonical state).
    void refreshLastScan();
    if (startupCheckDone.current) return;
    startupCheckDone.current = true;
    void checkForUpdates();
    // Read the backend's whitelist-rules snapshot (bundled rules are always
    // active; the backend runs its own background TTL check).
    void rulesSvc.GetRulesStatus().then(setRulesStatus).catch(() => null);
  }, [refreshUpdate, checkForUpdates, refreshMeta, refreshLastScan]);

  return <Ctx.Provider value={value}>{children}</Ctx.Provider>;
}

export function useStore(): StoreValue {
  const c = useContext(Ctx);
  if (!c) throw new Error("useStore must be used inside AppStoreProvider");
  return c;
}
