import React, { useEffect } from "react";

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
    primary:
      "bg-accent text-white hover:bg-[#6b9dff] active:bg-[#3f76e0] shadow-[0_0_20px_rgba(79,140,255,0.25)]",
    secondary: "bg-panel-2 text-slate-200 border border-border hover:border-accent/60 hover:text-white",
    danger: "bg-danger/90 text-white hover:bg-danger",
    ghost: "text-muted hover:text-white hover:bg-panel-2",
  };
  const sizes: Record<ButtonSize, string> = {
    sm: "text-xs px-3 py-1.5 rounded-lg gap-1.5",
    md: "text-sm px-4 py-2 rounded-lg gap-2",
  };
  return (
    <button
      {...props}
      className={`inline-flex items-center justify-center font-medium transition-all duration-150 disabled:opacity-40 disabled:pointer-events-none focus:outline-none focus-visible:ring-2 focus-visible:ring-accent/60 ${variants[variant]} ${sizes[size]} ${className}`}
    />
  );
}

/* ---------- Card ---------- */
export function Card({ className = "", children }: { className?: string; children: React.ReactNode }) {
  return (
    <div className={`rounded-xl border border-border bg-panel p-5 ${className}`}>{children}</div>
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
    <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-[11px] font-medium leading-4 ${badgeColors[color] ?? badgeColors.slate} ${className}`}>
      {children}
    </span>
  );
}

/* ---------- Input ---------- */
export function Input({ className = "", ...props }: React.InputHTMLAttributes<HTMLInputElement>) {
  return (
    <input
      {...props}
      className={`w-full rounded-lg border border-border bg-panel-2 px-3 py-2 text-sm text-slate-100 placeholder:text-muted/60 focus:outline-none focus:border-accent/60 focus:ring-1 focus:ring-accent/40 transition-colors ${className}`}
    />
  );
}

/* ---------- Select ---------- */
export function Select({ className = "", children, ...props }: React.SelectHTMLAttributes<HTMLSelectElement>) {
  return (
    <select
      {...props}
      className={`rounded-lg border border-border bg-panel-2 px-3 py-2 text-sm text-slate-100 focus:outline-none focus:border-accent/60 ${className}`}
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
      className={`inline-flex h-[18px] w-[18px] shrink-0 items-center justify-center rounded-[5px] border transition-colors disabled:opacity-30 disabled:pointer-events-none ${
        checked ? "border-accent bg-accent" : "border-border bg-panel-2 hover:border-accent/60"
      }`}
    >
      {checked && (
        <svg width="11" height="11" viewBox="0 0 12 12" fill="none">
          <path d="M2 6.2 4.8 9 10 3.4" stroke="white" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" />
        </svg>
      )}
    </button>
  );
}

/* ---------- ProgressBar ---------- */
export function ProgressBar({ value, max = 100 }: { value: number; max?: number }) {
  const pct = max > 0 ? Math.min(100, Math.round((value / max) * 100)) : 0;
  return (
    <div className="h-2 w-full overflow-hidden rounded-full bg-panel-2">
      <div
        className="h-full rounded-full bg-gradient-to-r from-accent to-[#7fb0ff] transition-[width] duration-200"
        style={{ width: `${pct}%` }}
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
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    if (open) window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [open, onClose]);

  if (!open) return null;
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm" onClick={onClose}>
      <div
        className="w-[520px] max-w-[90vw] rounded-2xl border border-border bg-panel p-6 shadow-2xl"
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
      <div className="text-3xl">🧹</div>
      <p className="text-sm font-medium text-slate-300">{title}</p>
      {hint && <p className="text-xs text-muted">{hint}</p>}
    </div>
  );
}
