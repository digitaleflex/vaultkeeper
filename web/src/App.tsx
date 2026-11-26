import { Download, LayoutDashboard, List, Lock, Moon, Radar, RefreshCw, Sun, Upload } from "lucide-react";
import { useCallback, useEffect, useState } from "react";
import { api, ApiError, initToken } from "./api";
import { Dashboard } from "./components/Dashboard";
import { FindingDrawer } from "./components/FindingDrawer";
import { Inventory, type Filters } from "./components/Inventory";
import { ScanPanel } from "./components/ScanPanel";
import { Button } from "./components/ui";
import type { Finding, Info, Report } from "./types";

type View = "dashboard" | "inventory" | "scan";

const NAV: { id: View; label: string; icon: typeof List }[] = [
  { id: "dashboard", label: "Tableau de bord", icon: LayoutDashboard },
  { id: "inventory", label: "Inventaire", icon: List },
  { id: "scan", label: "Analyse", icon: Radar },
];

function useTheme() {
  const [dark, setDark] = useState(() => {
    try {
      const v = localStorage.getItem("vk-theme");
      if (v) return v === "dark";
    } catch {
      /* ignore */
    }
    return window.matchMedia("(prefers-color-scheme: dark)").matches;
  });
  useEffect(() => {
    document.documentElement.classList.toggle("dark", dark);
    try {
      localStorage.setItem("vk-theme", dark ? "dark" : "light");
    } catch {
      /* ignore */
    }
  }, [dark]);
  return [dark, setDark] as const;
}

function loadSelection(): Set<string> {
  try {
    return new Set(JSON.parse(localStorage.getItem("vk-selection") ?? "[]"));
  } catch {
    return new Set();
  }
}

export default function App() {
  const [token] = useState(initToken);
  const [dark, setDark] = useTheme();
  const [view, setView] = useState<View>("scan");
  const [info, setInfo] = useState<Info | null>(null);
  const [report, setReport] = useState<Report | null>(null);
  const [scanning, setScanning] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [fatal, setFatal] = useState<string | null>(null);
  const [open, setOpen] = useState<Finding | null>(null);
  const [filters, setFilters] = useState<Filters>({ q: "", category: "", severity: "" });
  const [selected, setSelected] = useState<Set<string>>(loadSelection);

  useEffect(() => {
    try {
      localStorage.setItem("vk-selection", JSON.stringify([...selected]));
    } catch {
      /* ignore */
    }
  }, [selected]);

  useEffect(() => {
    if (!token) {
      setFatal("Aucun jeton d'accès. Lancez `vk ui` : il ouvre automatiquement la bonne adresse.");
      return;
    }
    Promise.all([api.info(), api.report()])
      .then(([i, r]) => {
        setInfo(i);
        if (r) {
          setReport(r);
          setView("dashboard");
        }
      })
      .catch((e) =>
        setFatal(e instanceof ApiError && e.status === 401 ? "Jeton expiré : relancez `vk ui` dans votre terminal." : "Impossible de joindre vk. Est-il toujours lancé ?"),
      );
  }, [token]);

  const runScan = useCallback(async (paths: string[], depth: number, skipHistory: boolean) => {
    setScanning(true);
    setError(null);
    try {
      const r = await api.scan(paths, depth, skipHistory);
      if (r) {
        setReport(r);
        setView("dashboard");
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setScanning(false);
    }
  }, []);

  const toggle = (id: string) => {
    const next = new Set(selected);
    if (next.has(id)) next.delete(id);
    else next.add(id);
    setSelected(next);
  };

  const exportJSON = () => {
    if (!report) return;
    const blob = new Blob([JSON.stringify(report, null, 2)], { type: "application/json" });
    const a = document.createElement("a");
    a.href = URL.createObjectURL(blob);
    a.download = `vk-report-${report.host}-${report.scanned_at.slice(0, 10)}.json`;
    a.click();
    URL.revokeObjectURL(a.href);
  };

  if (fatal) {
    return (
      <div className="grid h-full place-items-center p-6">
        <div className="max-w-md text-center">
          <Lock className="mx-auto mb-4 size-10 text-zinc-400" />
          <h1 className="text-lg font-semibold">Session VaultKeeper fermée</h1>
          <p className="mt-2 text-sm text-zinc-500">{fatal}</p>
        </div>
      </div>
    );
  }

  const selectedInReport = report ? report.findings.filter((f) => selected.has(f.id)).length : 0;

  return (
    <div className="flex h-full flex-col md:flex-row">
      {/* navigation */}
      <aside className="flex shrink-0 items-center gap-1 border-b border-zinc-200 bg-white px-3 py-2 md:w-60 md:flex-col md:items-stretch md:gap-1 md:border-r md:border-b-0 md:px-3 md:py-5 dark:border-zinc-800 dark:bg-zinc-900">
        <div className="mr-2 flex items-center gap-2.5 px-2 md:mb-6 md:mr-0">
          <div className="grid size-8 place-items-center rounded-lg bg-indigo-600 text-white">
            <Lock className="size-4" />
          </div>
          <div className="hidden sm:block">
            <div className="text-sm font-semibold leading-tight">VaultKeeper</div>
            <div className="text-[11px] text-zinc-500">{info ? `v${info.version} · ${info.os}` : "…"}</div>
          </div>
        </div>
        <nav className="flex flex-1 gap-1 md:flex-none md:flex-col">
          {NAV.map(({ id, label, icon: Icon }) => {
            const disabled = id !== "scan" && !report;
            return (
              <button
                key={id}
                onClick={() => setView(id)}
                disabled={disabled}
                className={`flex items-center gap-2.5 rounded-lg px-2.5 py-2 text-sm font-medium transition disabled:cursor-not-allowed disabled:opacity-40 ${
                  view === id
                    ? "bg-indigo-50 text-indigo-700 dark:bg-indigo-500/15 dark:text-indigo-300"
                    : "text-zinc-600 hover:bg-zinc-100 dark:text-zinc-400 dark:hover:bg-zinc-800"
                }`}
              >
                <Icon className="size-4" />
                <span className="hidden sm:inline">{label}</span>
              </button>
            );
          })}
        </nav>
        <div className="md:mt-auto md:space-y-3">
          {report && (
            <div className="hidden rounded-lg bg-zinc-50 p-3 text-xs text-zinc-500 md:block dark:bg-zinc-800/50">
              <div className="font-medium text-zinc-700 dark:text-zinc-300">{report.host}</div>
              Analysé le {new Date(report.scanned_at).toLocaleString("fr-FR", { dateStyle: "short", timeStyle: "short" })}
            </div>
          )}
          <button
            onClick={() => setDark(!dark)}
            aria-label="Changer de thème"
            className="flex items-center gap-2.5 rounded-lg px-2.5 py-2 text-sm text-zinc-600 hover:bg-zinc-100 md:w-full dark:text-zinc-400 dark:hover:bg-zinc-800"
          >
            {dark ? <Sun className="size-4" /> : <Moon className="size-4" />}
            <span className="hidden md:inline">{dark ? "Thème clair" : "Thème sombre"}</span>
          </button>
        </div>
      </aside>

      {/* contenu */}
      <main className="min-w-0 flex-1 overflow-y-auto">
        {view !== "scan" && report && (
          <header className="sticky top-0 z-10 flex flex-wrap items-center gap-3 border-b border-zinc-200 bg-zinc-50/85 px-4 py-3 backdrop-blur md:px-8 dark:border-zinc-800 dark:bg-zinc-950/85">
            <h1 className="mr-auto text-lg font-semibold">{view === "dashboard" ? "Tableau de bord" : "Inventaire"}</h1>
            {selectedInReport > 0 && (
              <span className="text-sm text-zinc-500">{selectedInReport} sélectionné(s)</span>
            )}
            <Button variant="secondary" disabled title="Disponible avec la phase 2 (vk import)">
              <Upload className="size-4" /> Importer dans les coffres
            </Button>
            <Button variant="ghost" onClick={exportJSON} title="Exporter le rapport (sans valeurs secrètes)">
              <Download className="size-4" />
              <span className="hidden sm:inline">Exporter</span>
            </Button>
            <Button variant="ghost" onClick={() => setView("scan")} title="Relancer une analyse">
              <RefreshCw className="size-4" />
            </Button>
          </header>
        )}

        <div className="px-4 py-6 md:px-8">
          {view === "scan" && (
            <ScanPanel info={info} scanning={scanning} error={error} onScan={runScan} firstRun={!report} />
          )}
          {view === "dashboard" && report && (
            <Dashboard
              report={report}
              onOpen={setOpen}
              onFilter={(f) => {
                setFilters({ q: "", category: f.category ?? "", severity: (f.severity as Filters["severity"]) ?? "" });
                setView("inventory");
              }}
            />
          )}
          {view === "inventory" && report && (
            <Inventory
              report={report}
              filters={filters}
              setFilters={setFilters}
              selected={selected}
              toggle={toggle}
              setSelected={setSelected}
              onOpen={setOpen}
            />
          )}
        </div>
      </main>

      <FindingDrawer finding={open} onClose={() => setOpen(null)} />
    </div>
  );
}
