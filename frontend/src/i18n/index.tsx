import React, { createContext, useContext, useState, useEffect, useCallback } from "react";
import { en } from "./en";
import { zh } from "./zh";

type Lang = "zh" | "en";
type Dict = typeof en;

const STORAGE_KEY = "text2midi_lang";

// ─── Detect language ──────────────────────────────────────────────

function detectLanguage(): Lang {
  // Check localStorage first
  const stored = localStorage.getItem(STORAGE_KEY);
  if (stored === "zh" || stored === "en") return stored;

  // Browser language detection
  const navLang = navigator.language || (navigator as any).userLanguage || "";
  if (navLang.startsWith("zh")) return "zh";

  return "en";
}

const DICTIONARIES: Record<Lang, Dict> = { en, zh };

// ─── Context ──────────────────────────────────────────────────────

interface I18nContextType {
  lang: Lang;
  t: (key: keyof Dict) => string;
  setLang: (lang: Lang) => void;
}

const I18nContext = createContext<I18nContextType | null>(null);

// ─── Provider ─────────────────────────────────────────────────────

export function I18nProvider({ children }: { children: React.ReactNode }) {
  const [lang, setLangState] = useState<Lang>(detectLanguage);

  const setLang = useCallback((l: Lang) => {
    setLangState(l);
    localStorage.setItem(STORAGE_KEY, l);
  }, []);

  const dict = DICTIONARIES[lang];

  const t = useCallback(
    (key: keyof Dict): string => {
      return dict[key] ?? key;
    },
    [dict],
  );

  // Listen for browser language changes
  useEffect(() => {
    const stored = localStorage.getItem(STORAGE_KEY);
    if (!stored) {
      const detected = detectLanguage();
      if (detected !== lang) setLangState(detected);
    }
  }, []);

  return (
    <I18nContext.Provider value={{ lang, t, setLang }}>
      {children}
    </I18nContext.Provider>
  );
}

// ─── Hook ─────────────────────────────────────────────────────────

export function useT() {
  const ctx = useContext(I18nContext);
  if (!ctx) throw new Error("useT must be used within I18nProvider");
  return ctx.t;
}

export function useLang() {
  const ctx = useContext(I18nContext);
  if (!ctx) throw new Error("useLang must be used within I18nProvider");
  return { lang: ctx.lang, setLang: ctx.setLang };
}
