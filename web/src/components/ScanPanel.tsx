import { FolderPlus, Loader2, Play, Plus, ShieldCheck, Trash2 } from "lucide-react";
import { useEffect, useState } from "react";
import type { Info } from "../types";
import { Button, Card } from "./ui";

const KEY = "vk-scan-settings";

interface Settings {
  paths: string[];
  depth: number;
  skipHistory: boolean;
}

function load(): Settings | null {
  try {
    return JSON.parse(localStorage.getItem(KEY) ?? "null");
  } catch {
    return null;
  }
}

export function ScanPanel({
  info,
  scanning,
  error,
  onScan,
  firstRun,
}: {
  info: Info | null;
  scanning: boolean;
  error: string | null;
  onScan: (paths: string[], depth: number, skipHistory: boolean) => void;
  firstRun: boolean;
}) {
  const [s, setS] = useState<Settings>(() => load() ?? { paths: [], depth: 6, skipHistory: false });
  const [draft, setDraft] = useState("");

  useEffect(() => {
    try {
      localStorage.setItem(KEY, JSON.stringify(s));
    } catch {
      /* ignore */
    }
  }, [s]);

  const add = (p: string) => {
    p = p.trim();
    if (p && !s.paths.includes(p)) setS({ ...s, paths: [...s.paths, p] });
    setDraft("");
  };
  const suggestions = (info?.suggestions ?? []).filter((p) => !s.paths.includes(p));

  return (
    <div className="mx-auto max-w-2xl space-y-6">
      {firstRun && (
        <div className="text-center">
          <div className="mx-auto mb-4 grid size-14 place-items-center rounded-2xl bg-indigo-100 dark:bg-indigo-500/15">
            <ShieldCheck className="size-7 text-indigo-600 dark:text-indigo-400" />
          </div>
          <h1 className="text-2xl font-semibold">Faisons l'inventaire de vos secrets</h1>
          <p className="mt-2 text-zinc-600 dark:text-zinc-400">
            VaultKeeper analyse {info ? <b>{info.host}</b> : "ce poste"} à la recherche de clés SSH, fichiers .env, accès cloud
            et mots de passe stockés en clair. <br className="hidden sm:inline" />
            L'analyse est 100 % locale : aucune valeur secrète n'est lue à l'écran ni envoyée.
          </p>
        </div>
      )}

      <Card className="divide-y divide-zinc-100 dark:divide-zinc-800">
        <div className="p-5">
          <h2 className="text-sm font-semibold">Dossiers de projets</h2>
          <p className="mt-1 text-sm text-zinc-500">
            Où chercher les fichiers .env. Vide = tout le dossier utilisateur{info ? ` (${info.home})` : ""}.
          </p>

          {s.paths.length > 0 && (
            <ul className="mt-3 space-y-1.5">
              {s.paths.map((p) => (
                <li key={p} className="flex items-center gap-2 rounded-lg bg-zinc-50 px-3 py-2 font-mono text-[13px] dark:bg-zinc-800/60">
                  <span className="min-w-0 flex-1 break-all">{p}</span>
                  <button
                    onClick={() => setS({ ...s, paths: s.paths.filter((x) => x !== p) })}
                    aria-label={`Retirer ${p}`}
                    className="rounded p-1 text-zinc-400 hover:text-red-600"
                  >
                    <Trash2 className="size-4" />
                  </button>
                </li>
              ))}
            </ul>
          )}

          <form
            className="mt-3 flex gap-2"
            onSubmit={(e) => {
              e.preventDefault();
              add(draft);
            }}
          >
            <input
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              placeholder={info?.os === "Windows" ? "D:\\Projets" : "~/code"}
              className="min-w-0 flex-1 rounded-lg border border-zinc-300 bg-white px-3 py-2 font-mono text-sm outline-none focus:border-indigo-500 focus:ring-2 focus:ring-indigo-500/20 dark:border-zinc-700 dark:bg-zinc-900"
            />
            <Button type="submit" variant="secondary" disabled={!draft.trim()}>
              <Plus className="size-4" /> Ajouter
            </Button>
          </form>

          {suggestions.length > 0 && (
            <div className="mt-3 flex flex-wrap items-center gap-2 text-xs text-zinc-500">
              Suggestions :
              {suggestions.map((p) => (
                <button
                  key={p}
                  onClick={() => add(p)}
                  className="inline-flex items-center gap-1 rounded-full border border-zinc-300 px-2.5 py-1 font-mono hover:border-indigo-400 hover:text-indigo-600 dark:border-zinc-700"
                >
                  <FolderPlus className="size-3" /> {p}
                </button>
              ))}
            </div>
          )}
        </div>

        <div className="grid gap-5 p-5 sm:grid-cols-2">
          <label className="block">
            <span className="text-sm font-semibold">Profondeur de recherche</span>
            <div className="mt-2 flex items-center gap-3">
              <input
                type="range"
                min={2}
                max={12}
                value={s.depth}
                onChange={(e) => setS({ ...s, depth: +e.target.value })}
                className="flex-1 accent-indigo-600"
              />
              <span className="w-6 text-right text-sm tabular-nums">{s.depth}</span>
            </div>
          </label>
          <label className="flex items-start gap-3">
            <input
              type="checkbox"
              checked={!s.skipHistory}
              onChange={(e) => setS({ ...s, skipHistory: !e.target.checked })}
              className="mt-1 size-4 accent-indigo-600"
            />
            <span>
              <span className="text-sm font-semibold">Analyser les historiques</span>
              <span className="block text-sm text-zinc-500">bash, zsh, PowerShell… (mots de passe tapés en clair)</span>
            </span>
          </label>
        </div>

        <div className="flex items-center justify-between gap-4 p-5">
          <div className="min-h-5 text-sm text-red-600 dark:text-red-400">{error}</div>
          <Button onClick={() => onScan(s.paths, s.depth, s.skipHistory)} disabled={scanning}>
            {scanning ? <Loader2 className="size-4 animate-spin" /> : <Play className="size-4" />}
            {scanning ? "Analyse en cours…" : "Lancer l'analyse"}
          </Button>
        </div>
      </Card>
    </div>
  );
}
