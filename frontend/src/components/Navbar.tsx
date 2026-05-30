import React, { useState } from "react";
import { Sparkles, Library as LibIcon, LogIn, LogOut, User as UserIcon, X, Eye, EyeOff, AlertCircle, Globe } from "lucide-react";
import { User } from "../types";
import * as api from "../utils/api";
import { useT, useLang } from "../i18n";

interface NavbarProps {
  activeTab: "generate" | "library";
  setActiveTab: (tab: "generate" | "library") => void;
  user: User | null;
  onLogin: (username: string, password: string) => Promise<api.AuthResponse>;
  onRegister: (username: string, password: string) => Promise<api.AuthResponse>;
  onLogout: () => void;
  showAuthModal?: boolean;
  setShowAuthModal?: (v: boolean) => void;
}

// ─── Auth Modal ────────────────────────────────────────────────────

interface AuthModalProps {
  onLogin: (username: string, password: string) => Promise<api.AuthResponse>;
  onRegister: (username: string, password: string) => Promise<api.AuthResponse>;
  onClose: () => void;
}

function AuthModal({ onLogin, onRegister, onClose }: AuthModalProps) {
  const t = useT();
  const [tab, setTab] = useState<"login" | "register">("login");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [showPw, setShowPw] = useState(false);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    if (!username.trim() || !password.trim()) {
      setError(t("auth.fillBoth"));
      return;
    }
    setLoading(true);
    try {
      if (tab === "login") {
        await onLogin(username, password);
      } else {
        await onRegister(username, password);
      }
    } catch (err: any) {
      setError(err.message || "Something went wrong");
    } finally {
      setLoading(false);
    }
  };

  const switchTab = (t: "login" | "register") => {
    setTab(t);
    setError("");
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 backdrop-blur-sm" onClick={onClose}>
      <div
        className="bg-surface-container-high border border-white/10 rounded-2xl p-xl w-[90vw] max-w-[580px] min-w-[400px] shadow-2xl neon-glow"
        onClick={e => e.stopPropagation()}
      >
        {/* Header */}
        <div className="flex items-center justify-between mb-6">
          <div className="flex flex-col gap-0.5">
            <h2 className="font-display font-bold text-2xl text-white">
              {tab === "login" ? t("auth.welcomeBack") : t("auth.join")}
            </h2>
            <p className="text-xs text-on-surface-variant">
              {tab === "login" ? t("auth.signInTo") : t("auth.createFree")}
            </p>
          </div>
          <button onClick={onClose} className="w-8 h-8 flex items-center justify-center rounded-lg text-on-surface-variant hover:text-white hover:bg-white/5 cursor-pointer transition-colors">
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Tab switcher */}
        <div className="flex mb-6 bg-surface-container-low rounded-xl p-1">
          <button
            onClick={() => switchTab("login")}
            className={`flex-1 py-2.5 rounded-lg text-sm font-semibold transition-all cursor-pointer ${
              tab === "login"
                ? "bg-gradient-to-r from-primary to-secondary text-black shadow-lg"
                : "text-on-surface-variant hover:text-white"
            }`}
          >
            {t("auth.signIn")}
          </button>
          <button
            onClick={() => switchTab("register")}
            className={`flex-1 py-2.5 rounded-lg text-sm font-semibold transition-all cursor-pointer ${
              tab === "register"
                ? "bg-gradient-to-r from-primary to-secondary text-black shadow-lg"
                : "text-on-surface-variant hover:text-white"
            }`}
          >
            {t("auth.register")}
          </button>
        </div>

        {/* Form */}
        <form onSubmit={handleSubmit} className="flex flex-col gap-5">
          <div>
            <label className="text-xs text-on-surface-variant font-semibold mb-1.5 block">
              {t("auth.username")}
            </label>
            <input
              type="text"
              value={username}
              onChange={e => setUsername(e.target.value)}
              placeholder={t("auth.enterUsername")}
              className="w-full bg-surface-container-low border border-white/10 rounded-xl px-4 py-3.5 text-sm text-white placeholder:text-on-surface-variant/30 focus:ring-2 focus:ring-primary/50 focus:border-primary/50 focus:outline-none transition-all"
              autoFocus
            />
          </div>

          <div>
            <label className="text-xs text-on-surface-variant font-semibold mb-1.5 block">
              {t("auth.password")}
            </label>
            <div className="relative">
              <input
                type={showPw ? "text" : "password"}
                value={password}
                onChange={e => setPassword(e.target.value)}
                placeholder={tab === "register" ? t("auth.atLeast6") : t("auth.enterPassword")}
                className="w-full bg-surface-container-low border border-white/10 rounded-xl px-4 py-3.5 pr-11 text-sm text-white placeholder:text-on-surface-variant/30 focus:ring-2 focus:ring-primary/50 focus:border-primary/50 focus:outline-none transition-all"
              />
              <button
                type="button"
                onClick={() => setShowPw(!showPw)}
                className="absolute right-3.5 top-1/2 -translate-y-1/2 text-on-surface-variant hover:text-white cursor-pointer"
              >
                {showPw ? <EyeOff className="w-4 h-4" /> : <Eye className="w-4 h-4" />}
              </button>
            </div>
          </div>

          {error && (
            <div className="flex items-center gap-2.5 text-red-400 text-sm bg-red-400/10 rounded-xl px-4 py-3 border border-red-400/20">
              <AlertCircle className="w-4 h-4 flex-shrink-0" />
              <span>{error}</span>
            </div>
          )}

          <button
            type="submit"
            disabled={loading}
            className="w-full bg-gradient-to-r from-primary to-secondary text-black font-bold py-3.5 rounded-xl hover:brightness-110 active:scale-[0.98] transition-all cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed text-sm"
          >
            {loading ? (
              <span className="flex items-center justify-center gap-2">
                <span className="w-4 h-4 border-2 border-black/30 border-t-black rounded-full animate-spin" />
                {t("auth.pleaseWait")}
              </span>
            ) : tab === "login" ? (
              t("auth.signIn")
            ) : (
              t("auth.createAccount")
            )}
          </button>
        </form>
      </div>
    </div>
  );
}

// ─── Navbar ────────────────────────────────────────────────────────

export default function Navbar({ activeTab, setActiveTab, user, onLogin, onRegister, onLogout, showAuthModal, setShowAuthModal }: NavbarProps) {
  const t = useT();
  const { lang, setLang } = useLang();
  // Use external modal control if provided (for opening from App), otherwise internal
  const [localShow, setLocalShow] = useState(false);
  const showModal = showAuthModal !== undefined ? showAuthModal : localShow;
  const setShowModal = setShowAuthModal || setLocalShow;

  return (
    <>
      <nav className="w-full bg-surface/60 backdrop-blur-xl border-b border-white/10 shadow-sm" id="top-nav">
        <div className="flex justify-between items-center px-lg py-md w-full">
          {/* Brand Logo */}
          <div 
            onClick={() => setActiveTab("generate")}
            className="text-headline-lg font-display font-extrabold bg-gradient-to-r from-primary to-secondary bg-clip-text text-transparent cursor-pointer select-none active:scale-95 transition-transform"
          >
            MidiMind
          </div>

          {/* Middle Navigation Routes */}
          <div className="hidden md:flex items-center gap-xl">
            <button
              onClick={() => setActiveTab("generate")}
              className={`flex items-center gap-2 pb-1 text-sm font-medium tracking-wide border-b-2 hover:text-white transition-all cursor-pointer ${
                activeTab === "generate"
                  ? "border-primary text-primary"
                  : "border-transparent text-on-surface-variant hover:border-white/20"
              }`}
            >
              <Sparkles className="w-4 h-4" />
              {t("nav.generate")}
            </button>
            <button
              onClick={() => setActiveTab("library")}
              className={`flex items-center gap-2 pb-1 text-sm font-medium tracking-wide border-b-2 hover:text-white transition-all cursor-pointer ${
                activeTab === "library"
                  ? "border-primary text-primary"
                  : "border-transparent text-on-surface-variant hover:border-white/20"
              }`}
            >
              <LibIcon className="w-4 h-4" />
              {t("nav.library")}
            </button>
          </div>

          {/* Right Controls — Auth + Lang */}
          <div className="flex items-center gap-md">
            {/* Language toggle */}
            <button
              onClick={() => setLang(lang === "zh" ? "en" : "zh")}
              className="w-8 h-8 rounded-lg bg-surface-container-high border border-white/10 flex items-center justify-center text-xs font-bold text-on-surface-variant hover:text-white hover:border-white/30 transition-all cursor-pointer"
              title={lang === "zh" ? "English" : "中文"}
            >
              <Globe className="w-3.5 h-3.5" />
            </button>

            {user ? (
              <div className="flex items-center gap-md">
                <span className="hidden md:flex items-center gap-1.5 text-xs text-on-surface-variant font-medium">
                  <UserIcon className="w-3.5 h-3.5 text-secondary" />
                  {user.username}
                </span>
                <button
                  onClick={onLogout}
                  className="flex items-center gap-1.5 px-md py-sm rounded-lg bg-surface-container-high border border-white/10 text-xs font-semibold text-on-surface-variant hover:text-white hover:border-red-500/40 transition-all cursor-pointer"
                >
                  <LogOut className="w-3.5 h-3.5" />
                  <span className="hidden md:inline">{t("nav.logout")}</span>
                </button>
              </div>
            ) : (
              <button
                onClick={() => setShowModal(true)}
                className="flex items-center gap-1.5 px-md py-sm rounded-lg bg-gradient-to-r from-primary to-secondary text-black text-xs font-bold hover:brightness-110 active:scale-95 transition-all cursor-pointer"
              >
                <LogIn className="w-3.5 h-3.5" />
                <span className="hidden md:inline">{t("nav.signIn")}</span>
              </button>
            )}
          </div>
        </div>
      </nav>

      {/* Auth Modal */}
      {showModal && (
        <AuthModal
          onLogin={async (u, p) => { const res = await onLogin(u, p); setShowModal(false); return res; }}
          onRegister={async (u, p) => { const res = await onRegister(u, p); setShowModal(false); return res; }}
          onClose={() => setShowModal(false)}
        />
      )}
    </>
  );
}
