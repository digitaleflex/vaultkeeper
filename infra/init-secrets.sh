#!/usr/bin/env bash
# Génère les secrets du fichier infra/.env (idempotent : ne remplace pas une valeur existante).
set -euo pipefail
cd "$(dirname "$0")"

[[ -f .env ]] || { cp .env.example .env; echo "→ .env créé depuis .env.example"; }
chmod 600 .env

set_if_empty() {
  local key=$1 value=$2
  if grep -qE "^${key}=$" .env; then
    # | comme délimiteur : les valeurs base64 peuvent contenir /
    sed -i "s|^${key}=$|${key}=${value}|" .env
    echo "✔ ${key} généré"
  else
    echo "• ${key} déjà défini, conservé"
  fi
}

set_if_empty INFISICAL_ENCRYPTION_KEY "$(openssl rand -hex 16)"
set_if_empty INFISICAL_AUTH_SECRET "$(openssl rand -base64 32 | tr -d '\n')"
set_if_empty POSTGRES_PASSWORD "$(openssl rand -hex 24)"

if grep -qE "^VAULTWARDEN_ADMIN_TOKEN=$" .env; then
  admin_pw=$(openssl rand -base64 24 | tr -d '\n/+=')
  # hash argon2 fourni par l'image vaultwarden ; les $ sont doublés pour docker compose
  hash=$(echo -n "$admin_pw" | docker run --rm -i "vaultwarden/server:$(grep ^VAULTWARDEN_VERSION= .env | cut -d= -f2)" /vaultwarden hash --preset owasp 2>/dev/null \
         | grep -o "\$argon2id[^']*" || true)
  if [[ -z "$hash" ]]; then
    echo "✖ Impossible de générer le hash admin (VAULTWARDEN_VERSION défini ? docker lancé ?)" >&2
    exit 1
  fi
  sed -i "s|^VAULTWARDEN_ADMIN_TOKEN=$|VAULTWARDEN_ADMIN_TOKEN='${hash//\$/\$\$}'|" .env
  echo "✔ VAULTWARDEN_ADMIN_TOKEN généré"
  echo
  echo "  Mot de passe de la page /admin de Vaultwarden (affiché UNE seule fois) :"
  echo "      $admin_pw"
  echo "  → notez-le dans votre kit de récupération."
fi

echo
echo "⚠️  Copiez INFISICAL_ENCRYPTION_KEY et INFISICAL_AUTH_SECRET dans votre kit de récupération hors serveur."
