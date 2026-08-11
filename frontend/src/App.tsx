import { useEffect, useRef, useState, type ReactNode } from "react";
import { LayoutDashboard, Search, FolderOpen, History as HistoryIcon, Settings as SettingsIcon, Info, Sparkles } from "lucide-react";
import { I18nProvider, useI18n } from "./i18n";
import { AppStoreProvider, useStore } from "./lib/store";
import { Dashboard } from "./pages/Dashboard";
import { Scanner } from "./pages/Scanner";
import { Results } from "./pages/Results";
import { History } from "./pages/History";
import { Settings } from "./pages/Settings";
import { About } from "./pages/About";

const PAGES = ["dashboard", "scanner", "results", "history", "settings", "about"] as const;
type Page = (typeof PAGES)[number];

function Shell() {
  const { t } = useI18n();
  const { completedScan, refreshDrives } = useStore();
  const [page, setPage] = useState<Page>("dashboard");
  // Guards against double navigation if a completion signal ever re-fires.
  const lastNavigatedScan = useRef<string | null>(null);

  useEffect(() => {
    void refreshDrives();
  }, [refreshDrives]);

  // Auto-navigate Scanner → Results only on a real successful completion
  // (cancelled/failed scans never produce an `ok` signal). Results reads the
  // already-persisted candidates, so the store is populated before we move.
  useEffect(() => {
    if (completedScan && completedScan.ok && lastNavigatedScan.current !== completedScan.scanId) {
      lastNavigatedScan.current = completedScan.scanId;
      setPage("results");
    }
  }, [completedScan]);

  const icons: Record<Page, ReactNode> = {
    dashboard: <LayoutDashboard size={18} />,
    scanner: <Search size={18} />,
    results: <FolderOpen size={18} />,
    history: <HistoryIcon size={18} />,
    settings: <SettingsIcon size={18} />,
    about: <Info size={18} />,
  };

  return (
    <div className="app-shell flex h-screen">
      {/* Sidebar */}
      <aside className="flex w-56 shrink-0 flex-col border-e border-border bg-panel">
        <div className="flex items-center gap-2.5 px-5 py-5">
          <span className="flex h-9 w-9 items-center justify-center rounded-lg bg-accent/15 text-accent">
            <Sparkles size={18} />
          </span>
          <div>
            <p className="text-sm font-bold text-white">{t("app.name")}</p>
            <p className="text-[10px] text-muted">v4 · Go + Wails</p>
          </div>
        </div>

        <nav className="flex flex-1 flex-col gap-1 px-3">
          {PAGES.map((p) => (
            <button
              key={p}
              onClick={() => setPage(p)}
              className={`group flex items-center gap-3 rounded-md px-3 py-2 text-sm transition-all duration-150 active:scale-[0.98] ${
                page === p ? "bg-accent-soft/40 text-white" : "text-muted hover:bg-panel-2 hover:text-white"
              }`}
            >
              <span
                className={`flex w-5 justify-center transition-transform duration-200 ease-out group-hover:scale-110 ${
                  page === p ? "text-accent" : ""
                }`}
              >
                {icons[p]}
              </span>
              {t(`nav.${p}`)}
            </button>
          ))}
        </nav>
      </aside>

      {/* Main */}
      <main className="min-w-0 flex-1 overflow-y-auto">
        <div key={page} className="page-enter h-full">
          {page === "dashboard" && <Dashboard onNavigate={(p) => setPage(p as Page)} />}
          {page === "scanner" && <Scanner />}
          {page === "results" && <Results />}
          {page === "history" && <History />}
          {page === "settings" && <Settings />}
          {page === "about" && <About />}
        </div>
      </main>
    </div>
  );
}

export default function App() {
  return (
    <I18nProvider>
      <AppStoreProvider>
        <Shell />
      </AppStoreProvider>
    </I18nProvider>
  );
}
