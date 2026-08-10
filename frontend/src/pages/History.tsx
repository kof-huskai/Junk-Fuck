import { useEffect, useState } from "react";
import { useI18n } from "../i18n";
import type * as cleanup from "../../bindings/github.com/kof-huskai/Junk-Fuck/services/models";
import type * as report from "../../bindings/github.com/kof-huskai/Junk-Fuck/internal/report/models";
import * as cleanupSvc from "../../bindings/github.com/kof-huskai/Junk-Fuck/services/cleanupservice";
import { formatBytes } from "../lib/format";
import { Badge, Card, EmptyState } from "../components/ui";

export function History() {
  const { t } = useI18n();
  const [entry, setEntry] = useState<cleanup.lastReportEntry | null>(null);
  const [loaded, setLoaded] = useState(false);

  useEffect(() => {
    cleanupSvc
      .GetLastReport()
      .then(setEntry)
      .catch(() => setEntry(null))
      .finally(() => setLoaded(true));
  }, []);

  if (!loaded) return <div className="p-8 text-sm text-muted">…</div>;

  if (!entry) return <div className="p-8"><EmptyState title={t("hist.none")} /></div>;

  const report: report.Report = entry.report;

  return (
    <div className="flex flex-col gap-6 p-8">
      <div>
        <h1 className="text-2xl font-bold text-white">{t("hist.title")}</h1>
        {entry.at && <p className="mt-1 text-sm text-muted">{t("hist.at")}: {entry.at}</p>}
      </div>

      <div className="grid grid-cols-2 gap-4 xl:grid-cols-4">
        <Card>
          <p className="text-xs text-muted">{t("hist.deleted")}</p>
          <p className="mt-1 text-2xl font-bold text-success">{report.deletedCount}</p>
        </Card>
        <Card>
          <p className="text-xs text-muted">{t("hist.skipped")}</p>
          <p className="mt-1 text-2xl font-bold text-warn">{report.skippedCount}</p>
        </Card>
        <Card>
          <p className="text-xs text-muted">{t("hist.failed")}</p>
          <p className="mt-1 text-2xl font-bold text-danger">{report.failedCount}</p>
        </Card>
        <Card>
          <p className="text-xs text-muted">{t("hist.freed")}</p>
          <p className="mt-1 text-2xl font-bold text-accent">{formatBytes(report.bytesFreed)}</p>
        </Card>
      </div>

      {report.dryRun && <Badge color="accent" className="self-start">{t("hist.dryRun")}</Badge>}

      <Card className="p-0">
        <div className="max-h-[420px] overflow-y-auto">
          <table className="w-full border-collapse text-sm">
            <thead className="sticky top-0 bg-panel">
              <tr className="text-left text-xs uppercase tracking-wide text-muted [&>th]:px-4 [&>th]:py-2.5">
                <th>{t("res.col.name")}</th>
                <th>{t("res.col.status")}</th>
                <th>{t("res.col.reason")}</th>
              </tr>
            </thead>
            <tbody>
              {(report.items ?? []).map((it, i) => (
                <tr key={i} className="border-t border-border/60">
                  <td className="max-w-72 truncate px-4 py-2 font-medium text-slate-200" dir="ltr">{it.name}</td>
                  <td className="px-4 py-2">
                    <Badge color={it.status === "deleted" ? "green" : it.status === "skipped" ? "amber" : "red"}>
                      {t(`status.${it.status}`)}
                    </Badge>
                  </td>
                  <td className="max-w-80 truncate px-4 py-2 text-xs text-muted" dir="ltr">{it.reason || "—"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Card>
    </div>
  );
}
