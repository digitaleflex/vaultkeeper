import type { Info, Report, UpdateInfo } from "./types";

const KEY = "vk-token";

// Le jeton arrive dans le fragment d'URL (#token=…), jamais envoyé au serveur
// dans l'URL ni conservé dans l'historique.
export function initToken(): string | null {
  const m = window.location.hash.match(/token=([a-f0-9]+)/);
  if (m) {
    try {
      sessionStorage.setItem(KEY, m[1]);
    } catch {
      /* stockage indisponible : on garde le jeton en mémoire */
    }
    history.replaceState(null, "", window.location.pathname);
    memToken = m[1];
    return m[1];
  }
  try {
    memToken = sessionStorage.getItem(KEY);
  } catch {
    /* ignore */
  }
  return memToken;
}

let memToken: string | null = null;

export class ApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

async function call<T>(method: string, path: string, body?: unknown): Promise<T | null> {
  const res = await fetch(path, {
    method,
    headers: {
      "X-VK-Token": memToken ?? "",
      ...(body ? { "Content-Type": "application/json" } : {}),
    },
    body: body ? JSON.stringify(body) : undefined,
  });
  if (res.status === 204) return null;
  const data = await res.json().catch(() => ({}));
  if (!res.ok) throw new ApiError(res.status, data.error ?? `Erreur ${res.status}`);
  return data as T;
}

export const api = {
  info: () => call<Info>("GET", "/api/info"),
  update: () => call<UpdateInfo>("GET", "/api/update"),
  report: () => call<Report>("GET", "/api/report"),
  scan: (paths: string[], depth: number, skipHistory: boolean) =>
    call<Report>("POST", "/api/scan", { paths, depth, skipHistory }),
};
