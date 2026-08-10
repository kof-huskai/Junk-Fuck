import { useState } from "react";
import { Check, Download, RefreshCw } from "lucide-react";
import { useI18n, type Language } from "../i18n";
import { useStore } from "../lib/store";
import { Button, Card, Checkbox, Input, Select } from "../components/ui";

export function Settings() {
  const { t, lang, setLang } = useI18n();
  const { settings, setSettings, appInfo, updateState, checkForUpdates, installUpdate } = useStore();
  const [targets, setTargets] = useState(settings.targets);
  const [saved, setSaved] = useState(false);
  const [busy, setBusy] = useState(false);
  const [updateError, setUpdateError] = useState<string | null>(null);

  const save = () => {
    setSettings({ targets });
    setSaved(true);
    setTimeout(() => setSaved(false), 1500);
  };

  const check = async () => {
    setBusy(true);
    setUpdateError(null);
    try {
      await checkForUpdates();
    } catch (err) {
      setUpdateError(String(err));
    } finally {
      setBusy(false);
    }
  };

  const install = async () => {
    setBusy(true);
    setUpdateError(null);
    try {
      await installUpdate();
    } catch (err) {
      setUpdateError(String(err));
    } finally {
      setBusy(false);
    }
  };

  const state = updateState?.state;

  return (
    <div className="flex flex-col gap-6 p-8">
      <div>
        <h1 className="text-2xl font-bold text-white">{t("set.title")}</h1>
      </div>

      <Card className="max-w-xl">
        <div className="flex flex-col gap-5">
          <div>
            <label className="mb-1.5 block text-sm font-medium text-slate-200">{t("set.language")}</label>
            <Select value={lang} onChange={(e) => setLang(e.target.value as Language)} className="w-44">
              <option value="en">English</option>
              <option value="fa">فارسی</option>
            </Select>
          </div>

          <div>
            <label className="mb-1.5 block text-sm font-medium text-slate-200">{t("set.target")}</label>
            <Input value={targets} onChange={(e) => setTargets(e.target.value)} spellCheck={false} dir="ltr" />
          </div>

          <div className="flex items-center gap-3">
            <Checkbox checked={settings.dryRun} onChange={(v) => setSettings({ dryRun: v })} />
            <div>
              <p className="text-sm font-medium text-slate-200">{t("set.dryRun")}</p>
              <p className="text-xs text-muted">{t("set.dryRunHint")}</p>
            </div>
          </div>

          <div className="flex items-center gap-3">
            <Button variant="primary" onClick={save}>
              {t("set.saved")}
            </Button>
            {saved && (
              <span className="inline-flex items-center text-success">
                <Check size={16} />
              </span>
            )}
          </div>
        </div>
      </Card>

      {/* Updates */}
      <Card className="max-w-xl">
        <h2 className="mb-4 flex items-center gap-2 text-sm font-semibold text-white">
          <RefreshCw size={15} className="text-accent" />
          {t("set.updates")}
        </h2>
        <div className="flex flex-col gap-4 text-sm">
          <div className="grid grid-cols-2 gap-4">
            <div>
              <p className="text-xs text-muted">{t("set.updates.current")}</p>
              <p className="mt-1 font-semibold text-white" dir="ltr">
                v{appInfo?.version ?? "—"}
              </p>
            </div>
            <div>
              <p className="text-xs text-muted">{t("set.updates.available")}</p>
              <p className="mt-1 font-semibold text-white" dir="ltr">
                {state === "available" && updateState?.available ? `v${updateState.available}` : state === "up-to-date" ? "—" : state === "installed" ? "—" : "—"}
              </p>
            </div>
          </div>

          {updateState?.lastChecked && (
            <p className="text-xs text-muted">
              {t("set.updates.lastChecked")}: {updateState.lastChecked}
            </p>
          )}

          {state === "up-to-date" && (
            <p className="rounded-lg bg-success/10 px-3 py-2 text-xs text-success">{t("set.updates.upToDate")}</p>
          )}
          {state === "available" && updateState?.releaseNotes && (
            <div className="rounded-lg bg-accent-soft/20 px-3 py-2">
              <p className="mb-1 text-xs font-medium text-accent">{t("set.updates.notes")}</p>
              <p className="whitespace-pre-wrap text-xs leading-relaxed text-slate-300">{updateState.releaseNotes}</p>
            </div>
          )}
          {state === "installed" && (
            <p className="rounded-lg bg-warn/10 px-3 py-2 text-xs text-warn">{t("set.updates.installed")}</p>
          )}
          {updateError && <p className="rounded-lg bg-danger-soft/40 px-3 py-2 text-xs text-danger">{updateError}</p>}

          <div className="flex items-center gap-2">
            <Button variant="secondary" size="sm" onClick={() => void check()} disabled={busy}>
              <RefreshCw size={13} className={busy ? "animate-spin" : ""} />
              {t("set.updates.check")}
            </Button>
            {state === "available" && (
              <Button variant="primary" size="sm" onClick={() => void install()} disabled={busy}>
                <Download size={13} />
                {t("set.updates.install")}
              </Button>
            )}
          </div>

          <p className="text-xs text-muted">{t("set.updates.hint")}</p>
        </div>
      </Card>
    </div>
  );
}
