import { Search, X } from "lucide-react";
import { useMemo } from "react";
import type { Category, Finding, Report, Severity } from "../types";
import { CATEGORY_LABEL, SEVERITIES } from "../types";
import { CATEGORY_ICON, Card, Mono, SEVERITY_STYLE, SeverityBadge } from "./ui";

export interface Filters {
  q: string;
  category: Category | "";
  severity: Severity | "";
}

export function Inventory({
  report,
  filters,
  setFilters,
  selected,
  toggle,
  setSelected,
  onOpen,
}: {
  report: Report;
  filters: Filters;
  setFilters: (f: Filters) => void;
  selected: Set<string>;
  toggle: (id: string) => void;
  setSelected: (s: Set<string>) => void;
  onOpen: (f: Finding) => void;
}) {
  const categories = useMemo(() => [...new Set(report.findings.map((f) => f.category))], [report]);

  const visible = useMemo(() => {
    const q = filters.q.trim().toLowerCase();
    return report.findings.filter((f) => {
      if (filters.category && f.category !== filters.category) return false;
      if (filters.severity && f.severity !== filters.severity) return false;
      if (!q) return true;
      return [f.type, f.path, f.description, ...(f.targets ?? []), ...(f.keys ?? [])].join(" ").toLowerCase().includes(q);
    });
  }, [report, filters]);

  const allVisibleSelected = visible.length > 0 && visible.every((f) => selected.has(f.id));
  const active = filters.q || filters.category || filters.severity;

  return (
    <div className="space-y-4">
      {/* barre de filtres */}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
        <div className="relative flex-1">
          <Search className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-zinc-400" />
          <input
            value={filters.q}
            onChange={(e) => setFilters({ ...filters, q: e.target.value })}
            placeholder="Rechercher un serveur, un projet, une variable…"
            className="w-full rounded-lg border border-zinc-300 bg-white py-2 pr-3 pl-9 text-sm outline-none focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20 dark:border-zinc-700 dark:bg-zinc-900"
          />
        </div>
        <select
          value={filters.category}
          onChange={(e) => setFilters({ ...filters, category: e.target.value as Category | "" })}
          className="rounded-lg border border-zinc-300 bg-white px-3 py-2 text-sm dark:border-zinc-700 dark:bg-zinc-900"
        >
          <option value="">Toutes catégories</option>
          {categories.map((c) => (
            <option key={c} value={c}>
              {CATEGORY_LABEL[c]}
            </option>
          ))}
        </select>
        <select
          value={filters.severity}
          onChange={(e) => setFilters({ ...filters, severity: e.target.value as Severity | "" })}
          className="rounded-lg border border-zinc-300 bg-white px-3 py-2 text-sm dark:border-zinc-700 dark:bg-zinc-900"
        >
          <option value="">Toutes gravités</option>
          {SEVERITIES.map((s) => (
            <option key={s} value={s}>
              {s}
            </option>
          ))}
        </select>
        {active && (
          <button
            onClick={() => setFilters({ q: "", category: "", severity: "" })}
            className="inline-flex items-center gap-1 text-sm text-zinc-500 hover:text-zinc-800 dark:hover:text-zinc-200"
          >
            <X className="size-4" /> Réinitialiser
          </button>
        )}
      </div>

      <Card className="overflow-hidden">
        <div className="flex items-center gap-3 border-b border-zinc-200 bg-zinc-50 px-4 py-2.5 text-xs font-medium text-zinc-500 dark:border-zinc-800 dark:bg-zinc-900/60">
          <input
            type="checkbox"
            aria-label="Tout sélectionner"
            checked={allVisibleSelected}
            onChange={() => {
              const next = new Set(selected);
              visible.forEach((f) => (allVisibleSelected ? next.delete(f.id) : next.add(f.id)));
              setSelected(next);
            }}
            className="size-4 accent-indigo-600"
          />
          <span>
            {visible.length} élément(s){active ? ` sur ${report.findings.length}` : ""}
          </span>
        </div>

        {visible.length === 0 ? (
          <div className="px-4 py-12 text-center text-sm text-zinc-500">Aucun élément ne correspond aux filtres.</div>
        ) : (
          <ul className="divide-y divide-zinc-100 dark:divide-zinc-800">
            {visible.map((f) => {
              const Icon = CATEGORY_ICON[f.category];
              return (
                <li
                  key={f.id}
                  className={`flex items-start gap-3 border-l-[3px] px-4 py-3 transition hover:bg-zinc-50 dark:hover:bg-zinc-800/40 ${SEVERITY_STYLE[f.severity].bar}`}
                >
                  <input
                    type="checkbox"
                    aria-label={`Sélectionner ${f.type}`}
                    checked={selected.has(f.id)}
                    onChange={() => toggle(f.id)}
                    className="mt-1 size-4 accent-indigo-600"
                  />
                  <button onClick={() => onOpen(f)} className="flex min-w-0 flex-1 items-start gap-3 text-left">
                    <Icon className="mt-0.5 size-4 shrink-0 text-zinc-400" />
                    <div className="min-w-0 flex-1">
                      <div className="flex flex-wrap items-center gap-2">
                        <span className="text-sm font-medium">{f.type}</span>
                        <SeverityBadge s={f.severity} />
                      </div>
                      <div className="mt-0.5 truncate">
                        <Mono>{f.path}</Mono>
                      </div>
                      {f.description && <p className="mt-1 text-sm text-zinc-600 dark:text-zinc-400">{f.description}</p>}
                      {!!f.targets?.length && (
                        <div className="mt-1.5 flex flex-wrap gap-1.5">
                          {f.targets.slice(0, 4).map((t) => (
                            <span
                              key={t}
                              className="max-w-full truncate rounded-md bg-zinc-100 px-1.5 py-0.5 font-mono text-xs text-zinc-700 dark:bg-zinc-800 dark:text-zinc-300"
                            >
                              {t}
                            </span>
                          ))}
                          {f.targets.length > 4 && <span className="text-xs text-zinc-500">+{f.targets.length - 4}</span>}
                        </div>
                      )}
                    </div>
                    {!!f.warnings?.length && (
                      <span className="hidden shrink-0 rounded-md bg-amber-50 px-2 py-0.5 text-xs font-medium text-amber-700 sm:inline dark:bg-amber-500/10 dark:text-amber-300">
                        {f.warnings.length} alerte(s)
                      </span>
                    )}
                  </button>
                </li>
              );
            })}
          </ul>
        )}
      </Card>
    </div>
  );
}
