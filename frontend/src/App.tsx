import { useEffect, useRef, useState, type CSSProperties, type ReactNode } from "react";
import { LayoutDashboard, Search, FolderOpen, History as HistoryIcon, Settings as SettingsIcon, Info, Download, CheckCircle2, RefreshCw, TriangleAlert, PanelLeftClose, PanelLeftOpen } from "lucide-react";
import appIcon from "./assets/app-icon.png";
import { I18nProvider, useI18n } from "./i18n";
import { AppStoreProvider, useStore } from "./lib/store";
import { effectiveCollapsed, loadSidebarCollapsed, persistSidebarCollapsed, toggleUserCollapsed } from "./lib/sidebar";
import type * as services from "../bindings/github.com/kof-huskai/Junk-Fuck/services/models";
import { Dashboard } from "./pages/Dashboard";
import { Scanner } from "./pages/Scanner";
import { Results } from "./pages/Results";
import { History } from "./pages/History";
import { Settings } from "./pages/Settings";
import { About } from "./pages/About";

const PAGES = ["dashboard", "scanner", "results", "history", "settings", "about"] as const;
type Page = (typeof PAGES)[number];

/* Compact updater state pinned to the bottom of the Sidebar. It only
 * *consumes* the existing update service state (store.updateState); all
 * download/install UI stays in Settings → Updates. Clicking an actionable
 * state jumps to Settings, or re-runs the check on failure. */
function SidebarUpdateStatus({
  state,
  checking,
  collapsed = false,
  onOpenSettings,
  onRetry,
}: {
  state: services.UpdateState | null;
  checking: boolean;
  /** Icon-only mode (icon rail): full label moves to the native tooltip. */
  collapsed?: boolean;
  onOpenSettings: () => void;
  onRetry: () => void;
}) {
  const { t } = useI18n();

  let icon: ReactNode;
  let label: string;
  let tone: string;
  let action: () => void;
  let title: string;

  if (checking) {
    icon = <RefreshCw size={14} className="animate-spin" />;
    label = t("side.update.checking");
    tone = "text-muted";
    action = () => {}; // inert while a check is in flight
    title = t("side.update.checking");
  } else if (state?.state === "available") {
    icon = <Download size={14} className="text-accent" />;
    label = state.available
      ? t("side.update.available", { ver: state.available })
      : t("side.update.availableGeneric");
    tone = "text-accent";
    action = onOpenSettings;
    title = t("side.update.go");
  } else if (state?.state === "installed") {
    icon = <RefreshCw size={14} className="text-warn" />;
    label = t("side.update.ready");
    tone = "text-warn";
    action = onOpenSettings;
    title = t("side.update.go");
  } else if (state?.state === "up-to-date") {
    icon = <CheckCircle2 size={14} className="text-success" />;
    label = t("side.update.uptodate");
    tone = "text-muted";
    action = onOpenSettings;
    title = t("side.update.go");
  } else if (state?.state === "error") {
    icon = <TriangleAlert size={14} className="text-danger" />;
    label = t("side.update.failed");
    tone = "text-danger";
    action = onRetry;
    title = t("side.update.retry");
  } else {
    icon = <RefreshCw size={14} className="text-muted" />;
    label = t("side.update.notchecked");
    tone = "text-muted";
    action = onRetry;
    title = t("side.update.check");
  }

  return (
    <button
      type="button"
      onClick={action}
      // In collapsed (icon-only) mode the tooltip must explain the STATE,
      // not just the click action — e.g. "Update available — v4.2.0".
      title={collapsed ? label : title}
      disabled={checking}
      className={`flex w-full items-center overflow-hidden rounded-md py-1.5 text-xs transition-[gap,background-color] duration-150 hover:bg-panel-2 disabled:opacity-80 px-[18px] ${
        collapsed ? "gap-0" : "gap-2"
      }`}
    >
      {/* Fixed 20px icon slot: centres the 14px icon in the collapsed rail
          and keeps the icon column stable between states. */}
      <span className="flex w-5 shrink-0 justify-center">{icon}</span>
      {/* The label is always mounted; in collapsed mode it fades out while
          the rail narrows (grid column clips it). dir="auto" lets the
          browser pick the base direction from the first strong character:
          LTR for English labels, RTL for Persian ones. */}
      <span
        className={`min-w-0 flex-1 truncate text-start transition-[opacity,transform] duration-[140ms] ease-out ${tone} ${
          collapsed ? "-translate-x-1 opacity-0" : "translate-x-0 opacity-100"
        }`}
        dir="auto"
      >
        {label}
      </span>
    </button>
  );
}

function Shell() {
  const { t } = useI18n();
  const { completedScan, refreshDrives, updateState, updateChecking, checkForUpdates } = useStore();
  const [page, setPage] = useState<Page>("dashboard");
  // Sidebar preference is read in the state initializer (before first paint)
  // so a persisted collapsed state never flashes expanded on startup.
  const [userCollapsed, setUserCollapsed] = useState(() => loadSidebarCollapsed(localStorage));
  // Transient hover state — only meaningful while userCollapsed is true
  // (temporary expansion). Never persisted.
  const [hovered, setHovered] = useState(false);
  const collapsed = effectiveCollapsed(userCollapsed, hovered);
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

  const toggleSidebar = () => {
    // Clear the transient hover so a collapse is visible immediately: the
    // pointer sits over the old (wide) sidebar, so without this the sidebar
    // would stay hover-expanded until the pointer happens to leave.
    setHovered(false);
    const next = toggleUserCollapsed(userCollapsed);
    setUserCollapsed(next);
    persistSidebarCollapsed(localStorage, next);
  };

  return (
    <div
      className="app-shell h-full"
      style={{ "--sidebar-width": collapsed ? "56px" : "224px" } as CSSProperties}
    >
      {/* Sidebar — stretches the full window height (no fixed px heights).
          Windows-11-inspired geometry: a modest right-corner curve with a
          low-opacity soft shadow for subtle separation from the content.
          The shadow needs `relative z-10` so it paints above main's
          background rather than being hidden underneath it.

          Collapse model: `w-56` expanded / `w-14` icon rail. The width is
          part of the flex layout (main shrinks as the sidebar grows), so
          temporary hover expansion never overlays the content. Hover
          handlers live on the aside itself — the left edge is fixed, so the
          pointer stays inside during the width transition (no flicker). */}
      <aside
        className="relative z-10 flex h-full min-w-0 flex-col rounded-r-[14px] border-e border-border bg-panel shadow-[4px_0_16px_-6px_rgba(0,0,0,0.55)]"
        onMouseEnter={() => setHovered(true)}
        onMouseLeave={() => setHovered(false)}
      >
        {/* Constant padding keeps the logo column optically stable between
            the expanded and collapsed states; the text block fades and is
            clipped by the shrinking rail. px-[19px] centers the 18px logo
            in the 56px rail (nav rows use px-[18px] for the 20px icons). */}
        <div
          className={`flex items-center overflow-hidden px-[19px] py-5 transition-[gap] duration-150 ${
            collapsed ? "gap-0" : "gap-2.5"
          }`}
        >
          <img
            src={appIcon}
            alt={t("app.name")}
            width={18}
            height={18}
            draggable={false}
            className="h-[18px] w-[18px] shrink-0 object-contain"
          />
          <div
            className={`min-w-0 transition-[opacity,transform] duration-[140ms] ease-out ${
              collapsed ? "-translate-x-1 opacity-0" : "translate-x-0 opacity-100"
            }`}
          >
            <p className="truncate text-sm font-bold text-white">{t("app.name")}</p>
            <p className="text-[10px] text-muted">v4 · Go + Wails</p>
          </div>
        </div>

        <nav className="flex flex-1 flex-col gap-1">
          {PAGES.map((p) => (
            <button
              key={p}
              onClick={() => setPage(p)}
              title={collapsed ? t(`nav.${p}`) : undefined}
              aria-label={collapsed ? t(`nav.${p}`) : undefined}
              className={`group flex items-center overflow-hidden rounded-md px-[18px] py-2 text-sm transition-all duration-150 active:scale-[0.98] focus:outline-none focus-visible:ring-2 focus-visible:ring-accent/60 ${
                page === p ? "bg-accent-soft/40 text-white" : "text-muted hover:bg-panel-2 hover:text-white"
              } ${collapsed ? "gap-0" : "gap-3"}`}
            >
              {/* Fixed 20px icon column: the rail's constant px-[18px]
                  keeps icons optically centred in the 56px rail AND
                  stationary between the collapsed/expanded states. */}
              <span
                className={`flex w-5 shrink-0 justify-center transition-transform duration-200 ease-out group-hover:scale-110 ${
                  page === p ? "text-accent" : ""
                }`}
              >
                {icons[p]}
              </span>
              {/* Label stays mounted: opacity fades while the shrinking
                  grid column clips it, so collapse/expand never snaps. */}
              <span
                className={`truncate text-sm transition-[opacity,transform] duration-[140ms] ease-out ${
                  collapsed ? "-translate-x-1 opacity-0" : "translate-x-0 opacity-100"
                }`}
              >
                {t(`nav.${p}`)}
              </span>
            </button>
          ))}
        </nav>

        {/* Bottom of the sidebar: real updater status from the existing
            update service + the collapse/expand pin. Compact by design —
            update details live in Settings. */}
        <div className="border-t border-border/70 pb-2.5 pt-2">
          <SidebarUpdateStatus
            state={updateState}
            checking={updateChecking}
            collapsed={collapsed}
            onOpenSettings={() => setPage("settings")}
            onRetry={() => void checkForUpdates()}
          />
          {/* The pin control reflects and toggles the *pinned* preference
              (userCollapsed), not the transient effective width, so the
              affordance always matches what the click will do. */}
          <button
            type="button"
            onClick={toggleSidebar}
            aria-label={userCollapsed ? t("side.expand") : t("side.collapse")}
            title={userCollapsed ? t("side.expand") : t("side.collapse")}
            className={`mt-1 flex w-full items-center overflow-hidden rounded-md px-[18px] py-1.5 text-muted transition-[gap,background-color] duration-150 hover:bg-panel-2 hover:text-white focus:outline-none focus-visible:ring-2 focus-visible:ring-accent/60 ${
              collapsed ? "gap-0" : "gap-2"
            }`}
          >
            <span className="flex w-5 shrink-0 justify-center">
              {userCollapsed ? <PanelLeftOpen size={15} /> : <PanelLeftClose size={15} />}
            </span>
            <span
              className={`truncate text-xs transition-[opacity,transform] duration-[140ms] ease-out ${
                collapsed ? "-translate-x-1 opacity-0" : "translate-x-0 opacity-100"
              }`}
            >
              {userCollapsed ? t("side.expand") : t("side.collapse")}
            </span>
          </button>
        </div>
      </aside>

      {/* Main — the shell itself never scrolls; scroll ownership belongs to
          the page container below (and to per-page scroll regions like the
          Results list). */}
      <main className="min-h-0 min-w-0 overflow-hidden">
        <div key={page} className="page-enter h-full min-h-0 overflow-y-auto">
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
