#!/usr/bin/env bash
# Sauvegarde chiffrée hors site des coffres (Vaultwarden + Infisical) avec restic.
#
# Prérequis : apt install restic sqlite3
# Configuration : /etc/vaultkeeper/backup.env (chmod 600), par exemple :
#   RESTIC_REPOSITORY=s3:https://s3.eu-central-003.backblazeb2.com/mon-bucket/vaultkeeper
#   RESTIC_PASSWORD=<long mot de passe, AUSSI dans le kit de récupération>
#   AWS_ACCESS_KEY_ID=...        # clé applicative B2 / S3
#   AWS_SECRET_ACCESS_KEY=...
#
# Cron (chaque nuit à 3h17) :
#   17 3 * * * /opt/vaultkeeper/infra/backup.sh >> /var/log/vaultkeeper-backup.log 2>&1
set -euo pipefail
cd "$(dirname "$0")"

set -a; source /etc/vaultkeeper/backup.env; set +a

STAGE=$(mktemp -d /var/tmp/vk-backup.XXXXXX)
trap 'rm -rf "$STAGE"' EXIT
chmod 700 "$STAGE"

echo "[$(date -Is)] Démarrage de la sauvegarde"

# 1. Vaultwarden : copie cohérente de la base SQLite + pièces jointes / clés RSA
sqlite3 data/vaultwarden/db.sqlite3 ".backup '$STAGE/vaultwarden.sqlite3'"
tar -C data/vaultwarden --exclude='db.sqlite3*' -cf "$STAGE/vaultwarden-files.tar" .

# 2. Infisical : dump PostgreSQL
docker compose exec -T postgres pg_dump -U infisical -Fc infisical > "$STAGE/infisical.dump"

# 3. Configuration (contient des secrets : le dépôt restic est chiffré)
cp .env "$STAGE/infra.env"

# 4. Envoi chiffré + rétention
restic snapshots >/dev/null 2>&1 || restic init
restic backup --tag vaultkeeper "$STAGE"
restic forget --tag vaultkeeper --keep-daily 7 --keep-weekly 4 --keep-monthly 12 --prune
restic check --read-data-subset=5%

echo "[$(date -Is)] Sauvegarde terminée"
