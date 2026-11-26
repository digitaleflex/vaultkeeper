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
| `vk` (Go) | CLI multi-plateforme : analyse, import, injection des `.env` | 🟢 `scan` et `ui` disponibles |
| [Vaultwarden](https://github.com/dani-garcia/vaultwarden) | Coffre mots de passe et SSH, compatible avec les applis Bitwarden | 🟢 `infra/` |
| [Infisical](https://infisical.com) | Gestion des variables d'environnement en équipe | 🟢 `infra/` |
| Portail (Go + React) | Inventaire serveurs ↔ clés ↔ personnes, rotations | ⚪ phase 4 |

**Principe de sécurité** : `vk` n'affiche ni n'écrit **jamais** une valeur secrète dans
ses rapports, uniquement l'emplacement, le type, les serveurs associés et les noms de variables.

## Interface graphique : `vk ui`

```bash
vk ui
```

La commande ouvre l'interface dans votre navigateur : lancement de l'analyse, tableau de bord par gravité,
inventaire filtrable, fiche détaillée de chaque secret (serveurs associés, empreinte, alertes),
sélection des éléments à importer, export JSON, thème clair ou sombre, affichage sur mobile.

Sécurité : serveur limité à `127.0.0.1`, jeton aléatoire à chaque lancement,
protection contre le DNS rebinding, CSP stricte, aucune valeur secrète transmise.

Développement de l'interface (Vite, React, Tailwind) :

```bash
vk ui --port 4321 --no-open     # terminal 1 : l'API
cd web && npm run dev           # terminal 2 : rechargement à chaud
make web                        # recompile web/dist, embarqué dans le binaire
```

## `vk scan` (ligne de commande)

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

Téléchargez le fichier de votre système dans les **[Releases](https://github.com/digitaleflex/vaultkeeper/releases/latest)** :

| Système | Fichier | Lancement |
|---|---|---|
| **Windows** (PC classique) | `vk_X.Y.Z_windows_amd64.zip` | Décompressez, puis **double-cliquez sur `vk.exe`**. SmartScreen : *Informations complémentaires → Exécuter quand même* |
| Windows ARM (Surface…) | `vk_X.Y.Z_windows_arm64.zip` | idem |
| **macOS** Apple Silicon (M1–M4) | `vk_X.Y.Z_macos_arm64.zip` | `xattr -d com.apple.quarantine ./vk && ./vk ui` |
| macOS Intel | `vk_X.Y.Z_macos_amd64.zip` | idem |
| **Ubuntu / Debian** | `vk_X.Y.Z_amd64.deb` | `sudo apt install ./vk_X.Y.Z_amd64.deb` puis `vk ui` |
| Autres Linux | `vk_X.Y.Z_linux_amd64.tar.gz` | `tar xzf …` puis `./vk ui` |

Les binaires ne sont pas encore signés numériquement, d'où les avertissements de Windows et de macOS.
Vérifiez l'intégrité du téléchargement avec `SHA256SUMS`.

Sans argument, `vk` ouvre directement l'interface graphique.

### Publier une nouvelle version

```bash
git tag v0.2.0 && git push origin v0.2.0
```

Vous pouvez aussi créer une release depuis l'onglet *Releases* de GitHub. Dans les deux cas, la CI construit
et attache automatiquement les 10 paquets et le fichier `SHA256SUMS`.
Pour construire localement : `make release VERSION=v0.2.0`, les fichiers arrivent dans `dist/`.

## Déployer les coffres sur un VPS

Voir **[infra/INSTALL.md](infra/INSTALL.md)**.

## Feuille de route

- [x] **Phase 0** : infrastructure (Vaultwarden, Infisical, Tailscale, sauvegardes restic)
- [x] **Phase 1** : `vk scan` multi-plateforme
- [x] **Phase 1b** : interface graphique `vk ui` (React embarqué)
- [ ] **Phase 2** : `vk import` (SSH et mots de passe vers Vaultwarden, `.env` vers Infisical) avec validation interactive
- [ ] **Phase 3** : `vk env pull` / `vk env run`, intégration des nouveaux membres
- [ ] **Phase 4** : portail web (inventaire, rotations, rapports de postes)
