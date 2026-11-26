import { Cloud, Database, KeyRound, ScrollText, Server, Wrench, type LucideIcon } from "lucide-react";
import type { ReactNode } from "react";
import type { Category, Severity } from "../types";

export const SEVERITY_STYLE: Record<Severity, { dot: string; badge: string; bar: string }> = {
  critique: {
    dot: "bg-sev-critique",
    badge: "bg-fuchsia-100 text-fuchsia-800 ring-fuchsia-600/20 dark:bg-fuchsia-500/15 dark:text-fuchsia-300",
    bar: "border-l-sev-critique",
  },
  élevé: {
    dot: "bg-sev-eleve",
    badge: "bg-red-100 text-red-800 ring-red-600/20 dark:bg-red-500/15 dark:text-red-300",
    bar: "border-l-sev-eleve",
  },
  moyen: {
    dot: "bg-sev-moyen",
    badge: "bg-amber-100 text-amber-800 ring-amber-600/20 dark:bg-amber-500/15 dark:text-amber-300",
    bar: "border-l-sev-moyen",
  },
  faible: {
    dot: "bg-sev-faible",
    badge: "bg-blue-100 text-blue-800 ring-blue-600/20 dark:bg-blue-500/15 dark:text-blue-300",
    bar: "border-l-sev-faible",
  },
  info: {
    dot: "bg-sev-info",
    badge: "bg-zinc-100 text-zinc-700 ring-zinc-500/20 dark:bg-zinc-500/15 dark:text-zinc-300",
    bar: "border-l-zinc-300 dark:border-l-zinc-700",
  },
};

export const CATEGORY_ICON: Record<Category, LucideIcon> = {
  ssh: KeyRound,
  env: Database,
  cloud: Cloud,
  devtool: Wrench,
  server: Server,
  fuite: ScrollText,
};

export function SeverityBadge({ s }: { s: Severity }) {
  return (
    <span
      className={`inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs font-medium ring-1 ring-inset ${SEVERITY_STYLE[s].badge}`}
    >
      <span className={`size-1.5 rounded-full ${SEVERITY_STYLE[s].dot}`} />
      {s}
    </span>
  );
}

export function Card({ children, className = "" }: { children: ReactNode; className?: string }) {
  return (
    <div
      className={`rounded-xl border border-zinc-200 bg-white shadow-sm dark:border-zinc-800 dark:bg-zinc-900 ${className}`}
    >
      {children}
    </div>
  );
}

export function Button({
  children,
  onClick,
  variant = "primary",
  disabled,
  title,
  type = "button",
}: {
  children: ReactNode;
  onClick?: () => void;
  variant?: "primary" | "secondary" | "ghost";
  disabled?: boolean;
  title?: string;
  type?: "button" | "submit";
}) {
  const styles = {
    primary: "bg-indigo-600 text-white hover:bg-indigo-500 shadow-sm",
    secondary:
      "bg-white text-zinc-800 ring-1 ring-inset ring-zinc-300 hover:bg-zinc-50 dark:bg-zinc-900 dark:text-zinc-100 dark:ring-zinc-700 dark:hover:bg-zinc-800",
    ghost: "text-zinc-600 hover:bg-zinc-100 dark:text-zinc-300 dark:hover:bg-zinc-800",
  }[variant];
  return (
    <button
      type={type}
      onClick={onClick}
      disabled={disabled}
      title={title}
      className={`inline-flex items-center justify-center gap-2 rounded-lg px-3.5 py-2 text-sm font-medium transition focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-indigo-500 disabled:cursor-not-allowed disabled:opacity-50 ${styles}`}
    >
      {children}
    </button>
  );
}

export function Mono({ children }: { children: ReactNode }) {
  return <span className="font-mono text-[13px] break-all text-zinc-500 dark:text-zinc-400">{children}</span>;
}
