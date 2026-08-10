import React, { createContext, useContext, useEffect, useMemo, useState } from "react";
import { en } from "./en";
import { fa } from "./fa";

export type Language = "en" | "fa";

const dicts: Record<Language, Record<string, string>> = { en, fa };

interface I18nValue {
  lang: Language;
  setLang: (l: Language) => void;
  t: (key: string, vars?: Record<string, string | number>) => string;
}

const Ctx = createContext<I18nValue | null>(null);

function applyLanguage(l: Language) {
  // Applied synchronously (not in an effect) so the first paint already
  // uses the right dir and font — no LTR/Segoe flash for Persian users.
  document.documentElement.lang = l;
  document.documentElement.dir = l === "fa" ? "rtl" : "ltr";
}

export function I18nProvider({ children }: { children: React.ReactNode }) {
  const [lang, setLang] = useState<Language>(() => {
    const saved = localStorage.getItem("jf.lang");
    const l: Language = saved === "fa" ? "fa" : "en";
    applyLanguage(l);
    return l;
  });

  useEffect(() => {
    applyLanguage(lang);
    localStorage.setItem("jf.lang", lang);
  }, [lang]);

  const value = useMemo<I18nValue>(
    () => ({
      lang,
      setLang,
      t: (key, vars) => {
        let s = dicts[lang][key] ?? dicts.en[key] ?? key;
        if (vars) {
          for (const [k, v] of Object.entries(vars)) {
            s = s.replace(`{${k}}`, String(v));
          }
        }
        return s;
      },
    }),
    [lang],
  );

  return <Ctx.Provider value={value}>{children}</Ctx.Provider>;
}

export function useI18n(): I18nValue {
  const c = useContext(Ctx);
  if (!c) throw new Error("useI18n must be used inside I18nProvider");
  return c;
}
