import { useState } from "react";
import { Check } from "lucide-react";
import { useI18n, type Language } from "../i18n";
import { useStore } from "../lib/store";
import { Button, Card, Checkbox, Input, Select } from "../components/ui";

export function Settings() {
  const { t, lang, setLang } = useI18n();
  const { settings, setSettings } = useStore();
  const [targets, setTargets] = useState(settings.targets);
  const [saved, setSaved] = useState(false);

  const save = () => {
    setSettings({ targets });
    setSaved(true);
    setTimeout(() => setSaved(false), 1500);
  };

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
            {saved && <span className="inline-flex items-center text-success"><Check size={16} /></span>}
          </div>
        </div>
      </Card>
    </div>
  );
}
