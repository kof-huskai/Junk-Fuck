import { useEffect } from "react";
import { ShieldCheck, Sparkles } from "lucide-react";
import { useI18n } from "../i18n";
import { useStore } from "../lib/store";
import { Badge, Card } from "../components/ui";

export function About() {
  const { t } = useI18n();
  const { systemInfo, appInfo, protectedCount, refreshMeta } = useStore();

  useEffect(() => {
    void refreshMeta();
  }, [refreshMeta]);

  return (
    <div className="flex max-w-2xl flex-col gap-6 p-8">
      <div className="flex items-center gap-4">
        <div className="flex h-14 w-14 items-center justify-center rounded-[10px] bg-accent/15 text-accent">
          <Sparkles size={28} />
        </div>
        <div>
          <h1 className="text-2xl font-bold text-white">{t("app.name")}</h1>
          <p className="text-sm text-muted">{t("app.tagline")}</p>
        </div>
      </div>

      <Card>
        <p className="text-sm leading-relaxed text-slate-300">{t("about.desc")}</p>
      </Card>

      <Card>
        <div className="grid grid-cols-2 gap-4 text-sm">
          <div>
            <p className="text-xs text-muted">{t("about.version")}</p>
            <p className="mt-1 font-semibold text-white">{appInfo?.version ?? "—"}</p>
          </div>
          <div>
            <p className="text-xs text-muted">{t("about.os")}</p>
            <p className="mt-1 font-semibold text-white" dir="ltr">{systemInfo ? `${systemInfo.os} ${systemInfo.version}` : "—"}</p>
          </div>
          <div>
            <p className="text-xs text-muted">{t("about.arch")}</p>
            <p className="mt-1 font-semibold text-white" dir="ltr">{systemInfo?.arch ?? "—"}</p>
          </div>
          <div>
            <p className="text-xs text-muted">{t("about.admin")}</p>
            <p className="mt-1 font-semibold text-white">
              {systemInfo?.isAdmin ? <Badge color="green"><ShieldCheck size={12} className="me-1" /> Admin</Badge> : t("about.admin.no")}
            </p>
          </div>
        </div>
        <div className="mt-4 border-t border-border pt-4">
          <p className="text-xs text-muted">{t("about.protected")}</p>
          <p className="mt-1 text-sm font-semibold text-success">
            {t("about.protectedCount", { n: protectedCount })}
          </p>
        </div>
      </Card>

      <p className="text-xs text-muted">
        <a className="text-accent hover:underline" href="https://github.com/kof-huskai/Junk-Fuck" target="_blank" rel="noreferrer">
          {t("about.repo")} → github.com/kof-huskai/Junk-Fuck
        </a>
      </p>
    </div>
  );
}
