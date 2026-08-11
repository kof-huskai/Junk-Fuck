import { useEffect } from "react";
import { ShieldCheck } from "lucide-react";
import { useI18n } from "../i18n";
import { useStore } from "../lib/store";
import { Badge, Card } from "../components/ui";
import appIcon from "../assets/app-icon.png";

export function About() {
  const { t } = useI18n();
  const { systemInfo, appInfo, protectedCount, refreshMeta } = useStore();

  useEffect(() => {
    void refreshMeta();
  }, [refreshMeta]);

  return (
    <div className="flex max-w-2xl flex-col gap-6 p-8">
      <div className="flex items-center gap-4">
        <img
          src={appIcon}
          alt={t("app.name")}
          width={72}
          height={72}
          draggable={false}
          className="h-[72px] w-[72px] shrink-0 object-contain"
        />
        <div className="min-w-0">
          <h1 className="text-2xl font-bold text-white">{t("app.name")}</h1>
          <p className="mt-1 text-sm text-muted">
            {t("about.version")} {appInfo?.version ?? "—"} · Go + Wails v3
          </p>
        </div>
      </div>

      <Card>
        <p className="text-sm leading-relaxed text-slate-300">{t("about.desc")}</p>
      </Card>

      <Card>
        <div className="grid grid-cols-3 gap-4 text-sm">
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
