import { useMemo, useState } from "react";
import { ArrowDown, ArrowUp, CheckCircle2 } from "lucide-react";
import { useI18n } from "../i18n";
import { useStore } from "../lib/store";
import { formatBytes, CATEGORY_KEYS } from "../lib/format";
import type * as model from "../../bindings/github.com/kof-huskai/Junk-Fuck/internal/model/models";
import { Badge, Button, Card, Checkbox, Dialog, EmptyState, Input, Select } from "../components/ui";

const SORT_KEYS = ["size", "name", "category"] as const;
type SortKey = (typeof SORT_KEYS)[number];

export function Results() {
  const { t } = useI18n();
  const { scanId, candidates, settings, setSettings, cleanup, refreshCandidates, lastReport } = useStore();

  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [search, setSearch] = useState("");
  const [category, setCategory] = useState<string>("all");
  const [sortKey, setSortKey] = useState<SortKey>("size");
  const [sortDesc, setSortDesc] = useState(true);
  const [confirmOpen, setConfirmOpen] = useState(false);

  const selectable = candidates.filter((c) => !c.protected);

  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase();
    return candidates
      .filter((c) => (category === "all" || c.category === category) && (!q || c.name.toLowerCase().includes(q) || c.path.toLowerCase().includes(q)))
      .sort((a, b) => {
        let cmp = 0;
        if (sortKey === "size") cmp = a.size - b.size;
        else if (sortKey === "name") cmp = a.name.localeCompare(b.name);
        else cmp = a.category.localeCompare(b.category);
        return sortDesc ? -cmp : cmp;
      });
  }, [candidates, category, search, sortKey, sortDesc]);

  const toggle = (path: string, protectedFlag: boolean) => {
    if (protectedFlag) return;
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(path)) next.delete(path);
      else next.add(path);
      return next;
    });
  };

  const allSelected = selectable.length > 0 && selectable.every((c) => selected.has(c.path));
  const selectedSize = candidates.filter((c) => selected.has(c.path)).reduce((s, c) => s + c.size, 0);

  const categories: string[] = [...new Set(candidates.map((c) => c.category))];

  const confirmClean = async (dryRun: boolean) => {
    setConfirmOpen(false);
    const rep = await cleanup(dryRun, [...selected]);
    setSelected(new Set());
    await refreshCandidates();
    return rep;
  };

  return (
    <div className="flex h-full flex-col gap-4 p-8">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-white">{t("res.title")}</h1>
          {scanId && (
            <p className="mt-1 text-sm text-muted">
              {t("res.found", { n: candidates.length, size: formatBytes(candidates.reduce((s, c) => s + c.size, 0)) })}
            </p>
          )}
        </div>
        <div className="flex items-center gap-2">
          <label className="flex cursor-pointer items-center gap-2 text-sm text-muted" title={t("res.dryRunHint")}>
            <Checkbox checked={settings.dryRun} onChange={(v) => setSettings({ dryRun: v })} />
            {t("res.dryRun")}
          </label>
          <Button
            variant="danger"
            disabled={selected.size === 0}
            onClick={() => setConfirmOpen(true)}
          >
            {t("res.clean")} ({selected.size})
          </Button>
        </div>
      </div>

      {lastReport && (
        <div className={`slide-in flex flex-wrap items-center gap-x-5 gap-y-1 rounded-xl border px-4 py-3 text-sm ${lastReport.dryRun ? "border-accent/40 bg-accent-soft/30" : "border-success/30 bg-[#0f2b23]"}`}>
          <span className="font-semibold text-white">{t(lastReport.dryRun ? "hist.dryRun" : "hist.title")}</span>
          <span className="inline-flex items-center gap-1 text-success">
            <CheckCircle2 size={14} /> {t("hist.deleted")}: {lastReport.deletedCount}
          </span>
          <span className="text-warn">{t("hist.skipped")}: {lastReport.skippedCount}</span>
          <span className="text-danger">{t("hist.failed")}: {lastReport.failedCount}</span>
          <span className="text-accent">{t("hist.freed")}: {formatBytes(lastReport.bytesFreed)}</span>
        </div>
      )}

      {candidates.length === 0 ? (
        <EmptyState title={t("res.none")} />
      ) : (
        <Card className="flex min-h-0 flex-1 flex-col p-0">
          {/* toolbar */}
          <div className="flex flex-wrap items-center gap-2 border-b border-border p-3">
            <Button size="sm" variant="ghost" onClick={() => setSelected(allSelected ? new Set() : new Set(selectable.map((c) => c.path)))}>
              {allSelected ? t("res.selectNone") : t("res.selectAll")}
            </Button>
            <span className="text-xs text-muted">{t("res.selected", { n: selected.size })} · {formatBytes(selectedSize)}</span>
            <span className="mx-1 h-4 w-px bg-border" />
            <Select value={category} onChange={(e) => setCategory(e.target.value)} className="!py-1.5 text-xs">
              <option value="all">{t("res.filterAll")}</option>
              {categories.map((cat) => (
                <option key={cat} value={cat}>
                  {t(CATEGORY_KEYS[cat])}
                </option>
              ))}
            </Select>
            <Select value={sortKey} onChange={(e) => setSortKey(e.target.value as SortKey)} className="!py-1.5 text-xs">
              {SORT_KEYS.map((k) => (
                <option key={k} value={k}>
                  {t(`res.col.${k === "size" ? "size" : k}`)}
                </option>
              ))}
            </Select>
            <Button size="sm" variant="ghost" onClick={() => setSortDesc(!sortDesc)} title={sortDesc ? t("res.sortAsc") : t("res.sortDesc")}>
              {sortDesc ? <ArrowDown size={14} /> : <ArrowUp size={14} />}
            </Button>
            <Input value={search} onChange={(e) => setSearch(e.target.value)} placeholder={t("res.search")} className="ms-auto max-w-56 !py-1.5 text-xs" />
          </div>

          {/* table */}
          <div className="min-h-0 flex-1 overflow-y-auto">
            <table className="w-full border-collapse text-sm">
              <thead className="sticky top-0 bg-panel">
                <tr className="text-left text-xs uppercase tracking-wide text-muted [&>th]:px-4 [&>th]:py-2.5">
                  <th className="w-10"></th>
                  <th>{t("res.col.name")}</th>
                  <th className="w-40">{t("res.col.category")}</th>
                  <th className="w-28 text-end">{t("res.col.size")}</th>
                  <th className="w-28">{t("res.col.type")}</th>
                  <th>{t("res.col.reason")}</th>
                </tr>
              </thead>
              <tbody>
                {filtered.map((c: model.Candidate, idx) => {
                  const isSel = selected.has(c.path);
                  return (
                    <tr
                      key={c.path}
                      style={{ animationDelay: `${Math.min(idx * 18, 300)}ms` }}
                      className={`row-in cursor-pointer border-t border-border/60 transition-colors hover:bg-panel-2 ${c.protected ? "opacity-50" : ""} ${isSel ? "bg-accent-soft/20" : ""}`}
                      onClick={() => toggle(c.path, c.protected)}
                    >
                      <td className="px-4 py-2">
                        <Checkbox checked={isSel} onChange={() => toggle(c.path, c.protected)} disabled={c.protected} title={c.protected ? t("res.protected") : undefined} />
                      </td>
                      <td className="max-w-56 truncate px-4 py-2">
                        <span className="font-medium text-slate-200" dir="ltr">{c.name}</span>
                        {c.protected && <Badge color="red" className="ms-2">{t("res.protected")}</Badge>}
                      </td>
                      <td className="px-4 py-2">
                        <Badge color={c.protected ? "red" : "accent"}>{t(CATEGORY_KEYS[c.category])}</Badge>
                      </td>
                      <td className="px-4 py-2 text-end font-mono text-xs text-slate-300">{formatBytes(c.size)}</td>
                      <td className="px-4 py-2 text-xs text-muted">{c.isDir ? t("res.type.dir") : t("res.type.file")}</td>
                      <td className="max-w-72 truncate px-4 py-2 text-xs text-muted" dir="ltr">{c.reason}</td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </Card>
      )}

      <Dialog open={confirmOpen} onClose={() => setConfirmOpen(false)} title={t("confirm.title")}>
        {settings.dryRun && <p className="mb-3 rounded-lg bg-accent-soft/30 px-3 py-2 text-sm text-accent">{t("confirm.dryRun")}</p>}
        <p className="mb-6 text-sm leading-relaxed text-slate-300">
          {t("confirm.permanent", { n: selected.size, size: formatBytes(selectedSize) })}
        </p>
        <div className="flex justify-end gap-2">
          <Button variant="ghost" onClick={() => setConfirmOpen(false)}>
            {t("confirm.cancel")}
          </Button>
          <Button variant="danger" onClick={() => void confirmClean(settings.dryRun)}>
            {settings.dryRun ? t("res.dryRun") : t("confirm.continue")}
          </Button>
        </div>
      </Dialog>
    </div>
  );
}
