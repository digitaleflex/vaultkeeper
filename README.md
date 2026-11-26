# 🔐 VaultKeeper

Centraliser et sécuriser **tous les secrets DevOps** d'une équipe : clés SSH,
accès serveurs, mots de passe, variables d'environnement, identifiants cloud.

Fini les clés SSH qui ne vivent que sur un seul PC, et les `.env` envoyés par mail ou WhatsApp.

## Architecture (approche hybride)

```
  PC Windows / macOS / Linux                     VPS (accessible uniquement via Tailscale)
 ┌──────────────────────────┐   HTTPS + VPN   ┌───────────────────────────────────────┐
 │ vk scan    → inventaire  │───────────────▶ │ Vaultwarden → mots de passe, SSH       │
 │ vk import  → coffres     │   (bientôt)     │ Infisical   → .env par projet / env    │
 │ vk env run → .env        │                 │ restic      → sauvegarde chiffrée      │
 └──────────────────────────┘                 └───────────────────────────────────────┘
        📱 appli Bitwarden et navigateur ─────────────▶
```

| Composant | Rôle | Statut |
|---|---|---|
| `vk` (Go) | CLI multi-plateforme : analyse, import, injection des `.env` | 🟢 `scan` disponible |
| [Vaultwarden](https://github.com/dani-garcia/vaultwarden) | Coffre mots de passe et SSH, compatible avec les applis Bitwarden | 🟢 `infra/` |
| [Infisical](https://infisical.com) | Gestion des variables d'environnement en équipe | 🟢 `infra/` |
| Portail (Go + React) | Inventaire serveurs ↔ clés ↔ personnes, rotations | ⚪ phase 4 |

**Principe de sécurité** : `vk` n'affiche ni n'écrit **jamais** une valeur secrète dans
ses rapports, uniquement l'emplacement, le type, les serveurs associés et les noms de variables.

## `vk scan`

```bash
vk scan                              # analyse le dossier utilisateur
vk scan -p D:\Projets -p C:\dev      # dossiers de projets pour les .env
vk scan --html rapport.html          # rapport HTML filtrable
vk scan --json rapport.json -v       # export JSON, tous les détails
```

Éléments détectés :

| Catégorie | Éléments |
|---|---|
| SSH | Clés privées OpenSSH, PEM et PuTTY `.ppk` : algorithme, taille, passphrase, empreinte, âge, et **serveurs associés** via `~/.ssh/config` |
| Serveurs | `known_hosts`, sessions PuTTY et WinSCP (registre Windows), FileZilla |
| Env | `.env`, `.env.local`, `.env.production`… (les `*.example` sont ignorés), avec alerte si le fichier n'est pas dans le `.gitignore` |
| Cloud | AWS, Kubernetes, GCP (y compris les JSON de comptes de service égarés), Azure |
| Outils de développement | Docker, npm, PyPI, `.git-credentials`, GitHub CLI, `.netrc`, Terraform |
| Fuites | Secrets tapés dans les historiques bash, zsh, fish et PowerShell, psql, mysql |

Niveaux de gravité : `critique` › `élevé` › `moyen` › `faible` › `info`.

## Installation

Téléchargez le binaire de votre système dans les [Releases](https://github.com/digitaleflex/vaultkeeper/releases), ou :

```bash
go install github.com/digitaleflex/vaultkeeper/cmd/vk@latest
```

Pour compiler les 6 binaires (Windows, macOS et Linux, en amd64 et arm64) :

```bash
make release
```

## Déployer les coffres sur un VPS

Voir **[infra/INSTALL.md](infra/INSTALL.md)**.

## Feuille de route

- [x] **Phase 0** : infrastructure (Vaultwarden, Infisical, Tailscale, sauvegardes restic)
- [x] **Phase 1** : `vk scan` multi-plateforme
- [ ] **Phase 2** : `vk import` (SSH et mots de passe vers Vaultwarden, `.env` vers Infisical) avec validation interactive
- [ ] **Phase 3** : `vk env pull` / `vk env run`, intégration des nouveaux membres
- [ ] **Phase 4** : portail web (inventaire, rotations, rapports de postes)
