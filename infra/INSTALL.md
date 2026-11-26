# Installation du VPS VaultKeeper

Résultat : Vaultwarden et Infisical accessibles **uniquement** via Tailscale,
en HTTPS valide, depuis vos PC (Windows / macOS / Linux) et téléphones.
Aucun port public hormis le minimum vital.

**VPS conseillé** : Debian 12 ou Ubuntu 24.04, 2 vCPU, 4 Go RAM, 40 Go disque.

---

## 1. Préparer le serveur

```bash
apt update && apt upgrade -y
apt install -y ca-certificates curl ufw restic sqlite3 openssl unattended-upgrades
curl -fsSL https://get.docker.com | sh
```

## 2. Tailscale (réseau privé)

1. Créez un compte sur <https://login.tailscale.com> (gratuit jusqu'à 3 utilisateurs).
2. Dans **DNS**, activez **MagicDNS** et **HTTPS Certificates**.
3. Sur le VPS :

```bash
curl -fsSL https://tailscale.com/install.sh | sh
tailscale up --ssh --hostname vps-secrets
tailscale status    # notez le nom complet : vps-secrets.<tailnet>.ts.net
```

4. Installez Tailscale sur vos PC et téléphones, puis connectez-les au même compte.
   Invitez vos 2 collègues depuis la console Tailscale.

## 3. Pare-feu : fermer Internet

> ⚠️ Vérifiez D'ABORD que `ssh root@vps-secrets` fonctionne via Tailscale.
> Gardez la console web de votre hébergeur ouverte en secours.

```bash
ufw default deny incoming
ufw default allow outgoing
ufw allow in on tailscale0
ufw allow 41641/udp          # connexions directes Tailscale
ufw enable
```

> Docker contourne parfois ufw pour les ports publiés : c'est pourquoi tous les
> services sont liés à `127.0.0.1` dans `docker-compose.yml`.

## 4. Déployer les coffres

```bash
git clone https://github.com/digitaleflex/vaultkeeper /opt/vaultkeeper
cd /opt/vaultkeeper/infra
cp .env.example .env && chmod 600 .env
nano .env        # VAULTWARDEN_VERSION, INFISICAL_VERSION, TS_HOSTNAME
./init-secrets.sh
docker compose up -d
docker compose logs -f infisical    # attendre « Server listening »
```

## 5. HTTPS via Tailscale

```bash
tailscale serve --bg --https=443  http://127.0.0.1:8081   # Vaultwarden
tailscale serve --bg --https=8443 http://127.0.0.1:8082   # Infisical
tailscale serve status
```

- Vaultwarden : `https://vps-secrets.<tailnet>.ts.net`
- Infisical : `https://vps-secrets.<tailnet>.ts.net:8443`

## 6. Créer les comptes

**Vaultwarden**
1. Ouvrez `https://…/admin` et entrez le mot de passe affiché par `init-secrets.sh`.
2. Dans **Users**, invitez les 3 membres. Sans SMTP, chacun crée ensuite son compte
   avec l'adresse invitée.
3. Chacun active la **2FA** (TOTP ou passkey) dans *Paramètres → Sécurité*.
4. Créez une **Organisation** « Équipe » avec des collections : `Serveurs`, `SSH`, `Cloud`, `Clients`.
5. Applis : Bitwarden desktop, mobile et extension navigateur → *Auto-hébergé* → URL du serveur.

**Infisical**
1. Ouvrez `https://…:8443` : le premier compte créé devient administrateur.
2. Dans *Server Admin*, **désactivez les inscriptions publiques**.
3. Créez un projet par application, avec les environnements `dev`, `staging` et `prod`.
4. Invitez vos 2 collègues, en limitant l'accès `prod` au strict nécessaire.

## 7. Sauvegardes

```bash
mkdir -p /etc/vaultkeeper && nano /etc/vaultkeeper/backup.env && chmod 600 /etc/vaultkeeper/backup.env
/opt/vaultkeeper/infra/backup.sh           # premier test manuel
crontab -e   # 17 3 * * * /opt/vaultkeeper/infra/backup.sh >> /var/log/vaultkeeper-backup.log 2>&1
```

**Testez une restauration** au moins une fois (voir `restic restore`).

## 8. Kit de récupération (papier, hors ligne)

À imprimer et ranger en lieu sûr, en 2 exemplaires :

- [ ] Mot de passe maître Vaultwarden de chaque admin, et les codes de récupération 2FA
- [ ] Mot de passe `/admin` Vaultwarden
- [ ] `INFISICAL_ENCRYPTION_KEY` et `INFISICAL_AUTH_SECRET`
- [ ] `RESTIC_PASSWORD` et les identifiants du stockage de sauvegarde
- [ ] Une clé SSH de secours (ou YubiKey) autorisée sur le VPS
- [ ] Les identifiants du compte de l'hébergeur VPS et du compte Tailscale

## Mises à jour

```bash
cd /opt/vaultkeeper/infra
nano .env                  # nouvelles versions
docker compose pull && docker compose up -d
```
