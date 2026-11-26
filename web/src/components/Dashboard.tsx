import { AlertTriangle, ArrowRight, ShieldCheck } from "lucide-react";
import { useMemo } from "react";
import type { Category, Finding, Report } from "../types";
import { CATEGORY_DEST, CATEGORY_LABEL, SEVERITIES, SEVERITY_RANK } from "../types";
import { CATEGORY_ICON, Card, SEVERITY_STYLE, SeverityBadge } from "./ui";

export function Dashboard({
  report,
  onOpen,
  onFilter,
}: {
  report: Report;
  onOpen: (f: Finding) => void;
  onFilter: (f: { category?: Category; severity?: string }) => void;
}) {
  const bySev = useMemo(() => {
    const m: Record<string, number> = {};
    report.findings.forEach((f) => (m[f.severity] = (m[f.severity] ?? 0) + 1));
    return m;
  }, [report]);

  const byCat = useMemo(() => {
    const m = new Map<Category, { count: number; worst: number }>();
    report.findings.forEach((f) => {
      const c = m.get(f.category) ?? { count: 0, worst: 0 };
      c.count++;
      c.worst = Math.max(c.worst, SEVERITY_RANK[f.severity]);
      m.set(f.category, c);
    });
    return [...m.entries()].sort((a, b) => b[1].worst - a[1].worst || b[1].count - a[1].count);
  }, [report]);

  const urgent = report.findings.filter((f) => SEVERITY_RANK[f.severity] >= 3).slice(0, 8);
  const sshKeys = report.findings.filter((f) => f.category === "ssh");
  const servers = new Set(report.findings.flatMap((f) => (f.category === "ssh" || f.category === "server" ? (f.targets ?? []) : [])));
  const envFiles = report.findings.filter((f) => f.category === "env");
  const envVars = envFiles.reduce((n, f) => n + (f.keys?.length ?? 0), 0);
  const total = report.findings.length;

  return (
    <div className="space-y-6">
      {/* chiffres clés */}
      <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
        <Stat label="Éléments détectés" value={total} />
        <Stat label="Clés SSH" value={sshKeys.length} hint={`${servers.size} serveur(s) référencé(s)`} />
        <Stat label="Fichiers .env" value={envFiles.length} hint={`${envVars} variable(s)`} />
        <Stat
          label="Actions urgentes"
          value={(bySev["critique"] ?? 0) + (bySev["élevé"] ?? 0)}
          tone={(bySev["critique"] ?? 0) + (bySev["élevé"] ?? 0) > 0 ? "danger" : "ok"}
        />
      </div>

      {/* répartition par gravité */}
      <Card className="p-5">
        <div className="mb-3 flex items-center justify-between">
          <h2 className="text-sm font-semibold">Répartition par gravité</h2>
          <span className="text-xs text-zinc-500">cliquez pour filtrer</span>
        </div>
        <div className="flex h-3 overflow-hidden rounded-full bg-zinc-100 dark:bg-zinc-800">
          {SEVERITIES.map((s) =>
            bySev[s] ? (
              <button
                key={s}
                onClick={() => onFilter({ severity: s })}
                title={`${bySev[s]} ${s}`}
                className={`${SEVERITY_STYLE[s].dot} h-full transition hover:opacity-80`}
                style={{ width: `${(bySev[s] / total) * 100}%` }}
              />
            ) : null,
          )}
        </div>
        <div className="mt-3 flex flex-wrap gap-x-5 gap-y-2 text-sm">
          {SEVERITIES.map((s) => (
            <button key={s} onClick={() => onFilter({ severity: s })} className="flex items-center gap-2 hover:underline">
              <span className={`size-2.5 rounded-full ${SEVERITY_STYLE[s].dot}`} />
              <span className="capitalize">{s}</span>
              <span className="font-semibold tabular-nums">{bySev[s] ?? 0}</span>
            </button>
          ))}
        </div>
      </Card>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-5">
        {/* à traiter en priorité */}
        <Card className="lg:col-span-3">
          <div className="flex items-center gap-2 border-b border-zinc-200 px-5 py-3.5 dark:border-zinc-800">
            <AlertTriangle className="size-4 text-red-500" />
            <h2 className="text-sm font-semibold">À traiter en priorité</h2>
          </div>
          {urgent.length === 0 ? (
            <div className="flex flex-col items-center gap-2 px-5 py-10 text-center text-sm text-zinc-500">
              <ShieldCheck className="size-8 text-emerald-500" />
              Aucun problème critique ou élevé. Beau travail !
            </div>
          ) : (
            <ul className="divide-y divide-zinc-100 dark:divide-zinc-800">
              {urgent.map((f) => {
                const Icon = CATEGORY_ICON[f.category];
                return (
                  <li key={f.id}>
                    <button
                      onClick={() => onOpen(f)}
                      className="flex w-full items-start gap-3 px-5 py-3 text-left transition hover:bg-zinc-50 dark:hover:bg-zinc-800/50"
                    >
                      <Icon className="mt-0.5 size-4 shrink-0 text-zinc-400" />
                      <div className="min-w-0 flex-1">
                        <div className="flex flex-wrap items-center gap-2">
                          <span className="text-sm font-medium">{f.type}</span>
                          <SeverityBadge s={f.severity} />
                        </div>
                        <p className="mt-0.5 truncate text-sm text-zinc-500">{f.warnings?.[0]}</p>
                      </div>
                      <ArrowRight className="mt-1 size-4 shrink-0 text-zinc-300" />
                    </button>
                  </li>
                );
              })}
            </ul>
          )}
        </Card>

        {/* par catégorie */}
        <Card className="lg:col-span-2">
          <div className="border-b border-zinc-200 px-5 py-3.5 dark:border-zinc-800">
            <h2 className="text-sm font-semibold">Par catégorie</h2>
          </div>
          <ul className="divide-y divide-zinc-100 dark:divide-zinc-800">
            {byCat.map(([cat, { count, worst }]) => {
              const Icon = CATEGORY_ICON[cat];
              const sev = SEVERITIES.find((s) => SEVERITY_RANK[s] === worst)!;
              return (
                <li key={cat}>
                  <button
                    onClick={() => onFilter({ category: cat })}
                    className="flex w-full items-center gap-3 px-5 py-3 text-left transition hover:bg-zinc-50 dark:hover:bg-zinc-800/50"
                  >
                    <div className="grid size-9 place-items-center rounded-lg bg-zinc-100 dark:bg-zinc-800">
                      <Icon className="size-4 text-zinc-600 dark:text-zinc-300" />
                    </div>
                    <div className="min-w-0 flex-1">
                      <div className="text-sm font-medium">{CATEGORY_LABEL[cat]}</div>
                      <div className="truncate text-xs text-zinc-500">→ {CATEGORY_DEST[cat]}</div>
                    </div>
                    <span className={`size-2 rounded-full ${SEVERITY_STYLE[sev].dot}`} title={`pire : ${sev}`} />
                    <span className="w-6 text-right text-sm font-semibold tabular-nums">{count}</span>
                  </button>
                </li>
              );
            })}
          </ul>
        </Card>
      </div>
    </div>
  );
}

function Stat({ label, value, hint, tone }: { label: string; value: number; hint?: string; tone?: "danger" | "ok" }) {
  const color = tone === "danger" ? "text-red-600 dark:text-red-400" : tone === "ok" ? "text-emerald-600 dark:text-emerald-400" : "";
  return (
    <Card className="p-4">
      <div className="text-xs font-medium text-zinc-500">{label}</div>
      <div className={`mt-1 text-3xl font-semibold tabular-nums ${color}`}>{value}</div>
      {hint && <div className="mt-0.5 text-xs text-zinc-500">{hint}</div>}
    </Card>
  );
}
