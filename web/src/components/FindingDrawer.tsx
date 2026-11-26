import { AlertTriangle, ArrowRightLeft, Copy, X } from "lucide-react";
import { useEffect, useState } from "react";
import type { Finding } from "../types";
import { CATEGORY_DEST, CATEGORY_LABEL } from "../types";
import { CATEGORY_ICON, Mono, SeverityBadge } from "./ui";

export function FindingDrawer({ finding, onClose }: { finding: Finding | null; onClose: () => void }) {
  useEffect(() => {
    const h = (e: KeyboardEvent) => e.key === "Escape" && onClose();
    window.addEventListener("keydown", h);
    return () => window.removeEventListener("keydown", h);
  }, [onClose]);

  if (!finding) return null;
  const f = finding;
  const Icon = CATEGORY_ICON[f.category];

  return (
    <div className="fixed inset-0 z-40 flex justify-end" role="dialog" aria-modal="true" aria-label={f.type}>
      <div className="absolute inset-0 bg-zinc-950/40 backdrop-blur-[1px]" onClick={onClose} />
      <aside className="relative flex h-full w-full max-w-lg flex-col overflow-hidden bg-white shadow-2xl dark:bg-zinc-900">
        <header className="flex items-start gap-3 border-b border-zinc-200 px-5 py-4 dark:border-zinc-800">
          <div className="grid size-10 shrink-0 place-items-center rounded-lg bg-zinc-100 dark:bg-zinc-800">
            <Icon className="size-5 text-zinc-600 dark:text-zinc-300" />
          </div>
          <div className="min-w-0 flex-1">
            <h2 className="font-semibold">{f.type}</h2>
            <div className="mt-1 flex flex-wrap items-center gap-2 text-xs text-zinc-500">
              <SeverityBadge s={f.severity} />
              <span>{CATEGORY_LABEL[f.category]}</span>
            </div>
          </div>
          <button onClick={onClose} aria-label="Fermer" className="rounded-md p-1.5 text-zinc-400 hover:bg-zinc-100 hover:text-zinc-700 dark:hover:bg-zinc-800">
            <X className="size-5" />
          </button>
        </header>

        <div className="flex-1 space-y-6 overflow-y-auto px-5 py-5">
          {!!f.warnings?.length && (
            <section className="space-y-2">
              {f.warnings.map((w) => (
                <div key={w} className="flex gap-2.5 rounded-lg bg-amber-50 px-3 py-2.5 text-sm text-amber-900 dark:bg-amber-500/10 dark:text-amber-200">
                  <AlertTriangle className="mt-0.5 size-4 shrink-0" />
                  {w}
                </div>
              ))}
            </section>
          )}

          <Field label="Emplacement">
            <div className="flex items-start gap-2">
              <Mono>{f.path}</Mono>
              <CopyBtn text={f.path} />
            </div>
          </Field>

          {f.description && <Field label="Description">{f.description}</Field>}

          {f.fingerprint && (
            <Field label="Empreinte">
              <Mono>{f.fingerprint}</Mono>
            </Field>
          )}

          {f.modified && !f.modified.startsWith("0001") && (
            <Field label="Dernière modification">{new Date(f.modified).toLocaleString("fr-FR")}</Field>
          )}

          {!!f.targets?.length && (
            <Field label={f.category === "env" ? "Contexte" : `Cibles (${f.targets.length})`}>
              <ul className="space-y-1">
                {f.targets.map((t) => (
                  <li key={t} className="rounded-md bg-zinc-50 px-2.5 py-1.5 font-mono text-[13px] break-all dark:bg-zinc-800/60">
                    {t}
                  </li>
                ))}
              </ul>
            </Field>
          )}

          {!!f.keys?.length && (
            <Field label={f.category === "fuite" ? "Motifs détectés" : `Noms (${f.keys.length}) — valeurs jamais affichées`}>
              <div className="flex flex-wrap gap-1.5">
                {f.keys.map((k) => (
                  <span key={k} className="rounded-md border border-zinc-200 px-2 py-0.5 font-mono text-xs dark:border-zinc-700">
                    {k}
                  </span>
                ))}
              </div>
            </Field>
          )}

          <div className="flex items-center gap-3 rounded-lg border border-dashed border-zinc-300 px-4 py-3 text-sm dark:border-zinc-700">
            <ArrowRightLeft className="size-4 text-indigo-500" />
            <div>
              <div className="font-medium">Destination à l'import</div>
              <div className="text-zinc-500">{CATEGORY_DEST[f.category]}</div>
            </div>
          </div>
        </div>
      </aside>
    </div>
  );
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <section>
      <h3 className="mb-1.5 text-xs font-medium tracking-wide text-zinc-500 uppercase">{label}</h3>
      <div className="text-sm">{children}</div>
    </section>
  );
}

function CopyBtn({ text }: { text: string }) {
  const [ok, setOk] = useState(false);
  return (
    <button
      onClick={() => {
        navigator.clipboard?.writeText(text).then(() => {
          setOk(true);
          setTimeout(() => setOk(false), 1200);
        });
      }}
      className="shrink-0 rounded p-1 text-zinc-400 hover:bg-zinc-100 hover:text-zinc-700 dark:hover:bg-zinc-800"
      aria-label="Copier le chemin"
      title={ok ? "Copié !" : "Copier le chemin"}
    >
      <Copy className="size-3.5" />
    </button>
  );
}
