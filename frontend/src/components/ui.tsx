import React, { useEffect, useState } from "react";
import { Sparkles } from "lucide-react";

/* ---------- Button ---------- */
type ButtonVariant = "primary" | "secondary" | "danger" | "ghost";
type ButtonSize = "sm" | "md";

export function Button({
  variant = "secondary",
  size = "md",
  className = "",
  ...props
}: React.ButtonHTMLAttributes<HTMLButtonElement> & { variant?: ButtonVariant; size?: ButtonSize }) {
  const variants: Record<ButtonVariant, string> = {
    // Light #66c0f4 accent with dark text keeps AA contrast on the button.
    primary: "bg-accent text-on-accent hover:bg-accent-hover active:bg-accent-active shadow-sm",
    secondary: "bg-panel-2 text-slate-200 border border-border hover:border-accent/60 hover:text-white",
    danger: "bg-danger/90 text-white hover:bg-danger",
    ghost: "text-muted hover:text-white hover:bg-panel-2",
  };
  const sizes: Record<ButtonSize, string> = {
    sm: "text-xs px-3 py-1.5 rounded-md gap-1.5",
    md: "text-sm px-4 py-2 rounded-md gap-2",
  };
  return (
    <button
      {...props}
      className={`inline-flex items-center justify-center font-medium transition-all duration-150 active:scale-[0.97] disabled:opacity-40 disabled:pointer-events-none focus:outline-none focus-visible:ring-2 focus-visible:ring-accent/60 ${variants[variant]} ${sizes[size]} ${className}`}
    />
  );
}

/* ---------- Card ---------- */
export function Card({ className = "", children }: { className?: string; children: React.ReactNode }) {
  return (
    <div
      className={`rounded-[10px] border border-border bg-panel p-5 transition-colors duration-200 hover:border-accent/40 ${className}`}
    >
      {children}
    </div>
  );
}

/* ---------- Badge ---------- */
const badgeColors: Record<string, string> = {
  slate: "bg-panel-2 text-muted border border-border",
  accent: "bg-accent-soft text-accent",
  green: "bg-[#123f31] text-success",
  red: "bg-danger-soft text-danger",
  amber: "bg-[#3d3318] text-warn",
};

export function Badge({ color = "slate", className = "", children }: { color?: string; className?: string; children: React.ReactNode }) {
  return (
    <span className={`inline-flex items-center rounded-[4px] border px-2 py-0.5 text-[11px] font-medium leading-4 ${badgeColors[color] ?? badgeColors.slate} ${className}`}>
      {children}
    </span>
  );
}

/* ---------- Input ---------- */
export function Input({ className = "", ...props }: React.InputHTMLAttributes<HTMLInputElement>) {
  return (
    <input
      {...props}
      className={`w-full rounded-md border border-border bg-panel-2 px-3 py-2 text-sm text-slate-100 placeholder:text-muted/60 focus:outline-none focus:border-accent/60 focus:ring-1 focus:ring-accent/40 transition-colors ${className}`}
    />
  );
}

/* ---------- Select ---------- */
export function Select({ className = "", children, ...props }: React.SelectHTMLAttributes<HTMLSelectElement>) {
  return (
    <select
      {...props}
      className={`rounded-md border border-border bg-panel-2 px-3 py-2 text-sm text-slate-100 focus:outline-none focus:border-accent/60 ${className}`}
    >
      {children}
    </select>
  );
}

/* ---------- Checkbox ---------- */
export function Checkbox({
  checked,
  onChange,
  disabled,
  title,
}: {
  checked: boolean;
  onChange: (v: boolean) => void;
  disabled?: boolean;
  title?: string;
}) {
  return (
    <button
      type="button"
      title={title}
      disabled={disabled}
      onClick={(e) => {
        e.stopPropagation();
        if (!disabled) onChange(!checked);
      }}
      className={`inline-flex h-[18px] w-[18px] shrink-0 items-center justify-center rounded-[4px] border transition-all duration-150 disabled:opacity-30 disabled:pointer-events-none active:scale-90 ${
        checked ? "border-accent bg-accent" : "border-border bg-panel-2 hover:border-accent/60"
      }`}
    >
      {checked && (
        <svg className="animate-check" width="11" height="11" viewBox="0 0 12 12" fill="none">
          <path d="M2 6.2 4.8 9 10 3.4" stroke="var(--color-on-accent)" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" />
        </svg>
      )}
    </button>
  );
}

/* ---------- ProgressBar ---------- */
export function ProgressBar({ value, max = 100 }: { value: number; max?: number }) {
  const pct = max > 0 ? Math.min(100, Math.round((value / max) * 100)) : 0;
  const running = pct < 100;
  return (
    <div className="h-2 w-full overflow-hidden rounded-full bg-panel-2">
      <div
        className={`relative h-full overflow-hidden rounded-full bg-gradient-to-r from-accent to-accent-hover transition-[width] duration-300 ease-out ${running ? "progress-shimmer" : ""}`}
        style={{ width: `${Math.max(pct, 2)}%` }}
      />
    </div>
  );
}

/* ---------- Dialog ---------- */
export function Dialog({
  open,
  onClose,
  title,
  children,
}: {
  open: boolean;
  onClose: () => void;
  title: string;
  children: React.ReactNode;
}) {
  // Keep the dialog mounted through the exit animation instead of
  // unmounting instantly, so close feels as polished as open.
  const [closing, setClosing] = useState(false);

  const requestClose = () => {
    if (closing) return;
    setClosing(true);
    setTimeout(() => {
      setClosing(false);
      onClose();
    }, 160);
  };

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") requestClose();
    };
    if (open) window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open, requestClose]);

  useEffect(() => {
    if (open) setClosing(false);
  }, [open]);

  if (!open) return null;
  return (
    <div
      className={`${closing ? "dialog-exit" : "dialog-backdrop"} fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm`}
      onClick={requestClose}
    >
      <div
        className={`${closing ? "dialog-exit-panel" : "dialog-panel"} w-[520px] max-w-[90vw] rounded-[10px] border border-border bg-panel p-6 shadow-xl`}
        onClick={(e) => e.stopPropagation()}
      >
        <h2 className="mb-4 text-lg font-semibold text-white">{title}</h2>
        {children}
      </div>
    </div>
  );
}

/* ---------- Empty state ---------- */
export function EmptyState({ title, hint }: { title: string; hint?: string }) {
  return (
    <div className="flex flex-col items-center justify-center gap-2 py-16 text-center">
      <div className="flex h-14 w-14 items-center justify-center rounded-[10px] bg-panel-2 text-accent">
        <Sparkles size={26} />
      </div>
      <p className="text-sm font-medium text-slate-300">{title}</p>
      {hint && <p className="text-xs text-muted">{hint}</p>}
    </div>
  );
}
