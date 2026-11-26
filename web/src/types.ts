export type Severity = "critique" | "élevé" | "moyen" | "faible" | "info";
export type Category = "ssh" | "env" | "cloud" | "devtool" | "server" | "fuite";

export interface Finding {
  id: string;
  category: Category;
  type: string;
  path: string;
  description: string;
  targets?: string[];
  keys?: string[];
  warnings?: string[];
  severity: Severity;
  fingerprint?: string;
  modified?: string;
}

export interface Report {
  host: string;
  os: string;
  user: string;
  scanned_at: string;
  findings: Finding[];
  errors?: string[];
}

export interface Info {
  version: string;
  host: string;
  os: string;
  home: string;
  suggestions: string[] | null;
}

export const SEVERITIES: Severity[] = ["critique", "élevé", "moyen", "faible", "info"];

export const SEVERITY_RANK: Record<Severity, number> = {
  critique: 4,
  élevé: 3,
  moyen: 2,
  faible: 1,
  info: 0,
};

export const CATEGORY_LABEL: Record<Category, string> = {
  ssh: "Clés SSH",
  env: "Variables d'env",
  cloud: "Cloud",
  devtool: "Outils dev",
  server: "Serveurs",
  fuite: "Fuites",
};

export const CATEGORY_DEST: Record<Category, string> = {
  ssh: "Vaultwarden · Clé SSH",
  env: "Infisical",
  cloud: "Vaultwarden · Note sécurisée",
  devtool: "Vaultwarden · Identifiant",
  server: "Vaultwarden · Identifiant",
  fuite: "Nettoyage + rotation",
};
