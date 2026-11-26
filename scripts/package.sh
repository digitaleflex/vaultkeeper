#!/usr/bin/env bash
# Construit les paquets de distribution de vk dans dist/ :
#   vk_<ver>_windows_{amd64,arm64}.zip
#   vk_<ver>_macos_{arm64,amd64}.zip
#   vk_<ver>_linux_{amd64,arm64}.zip + .tar.gz
#   vk_<ver>_{amd64,arm64}.deb   (Debian / Ubuntu)
#   SHA256SUMS
#
# Usage : scripts/package.sh v0.1.0
# Prérequis : go, zip, dpkg-deb (l'interface web/dist doit être compilée).
set -euo pipefail
cd "$(dirname "$0")/.."

TAG=${1:-$(git describe --tags --always 2>/dev/null || echo dev)}
VER=${TAG#v}
DIST=dist
LDFLAGS="-s -w -X main.version=${VER}"
rm -rf "$DIST" && mkdir -p "$DIST"

build() { # os arch sortie
  echo "→ build $1/$2"
  CGO_ENABLED=0 GOOS=$1 GOARCH=$2 go build -trimpath -ldflags "$LDFLAGS" -o "$3" ./cmd/vk
}

docs() { # dossier
  cp README.md "$1/"
  cat > "$1/LISEZMOI.txt" <<EOF
VaultKeeper ${VER}

Lancer l'interface graphique :
  - Windows : double-cliquez sur vk.exe (ou "vk.exe ui" dans un terminal)
  - macOS   : dans le Terminal, "./vk ui"
              (1re fois : xattr -d com.apple.quarantine ./vk  — binaire non signé)
  - Linux   : "./vk ui"

Analyse en ligne de commande : vk scan --html rapport.html
Aide : vk help
EOF
}

# --- archives zip / tar.gz -------------------------------------------------
for target in windows/amd64 windows/arm64 darwin/arm64 darwin/amd64 linux/amd64 linux/arm64; do
  os=${target%/*}; arch=${target#*/}
  label=$os; [[ $os == darwin ]] && label=macos
  name="vk_${VER}_${label}_${arch}"
  dir="$DIST/$name"; mkdir -p "$dir"
  exe=vk; [[ $os == windows ]] && exe=vk.exe
  build "$os" "$arch" "$dir/$exe"
  docs "$dir"
  (cd "$DIST" && zip -qr "$name.zip" "$name")
  [[ $os == linux ]] && tar -C "$DIST" -czf "$DIST/$name.tar.gz" "$name"
done

# --- paquets .deb ------------------------------------------------------------
for arch in amd64 arm64; do
  root="$DIST/deb-$arch"
  mkdir -p "$root/DEBIAN" "$root/usr/bin" "$root/usr/share/doc/vaultkeeper"
  cp "$DIST/vk_${VER}_linux_${arch}/vk" "$root/usr/bin/vk"
  chmod 0755 "$root/usr/bin/vk"
  cp README.md "$root/usr/share/doc/vaultkeeper/"
  debver=$VER; [[ $debver =~ ^[0-9] ]] || debver="0.0.0~$debver"
  cat > "$root/DEBIAN/control" <<EOF
Package: vaultkeeper
Version: ${debver}
Section: utils
Priority: optional
Architecture: ${arch}
Maintainer: VaultKeeper <noreply@github.com>
Homepage: https://github.com/digitaleflex/vaultkeeper
Description: Inventaire et protection des secrets DevOps
 vk analyse le poste (clés SSH, fichiers .env, identifiants cloud,
 mots de passe en clair) sans jamais exposer les valeurs, et propose
 une interface graphique locale (vk ui).
EOF
  dpkg-deb --root-owner-group --build "$root" "$DIST/vk_${VER}_${arch}.deb" >/dev/null
  rm -rf "$root"
done

# --- nettoyage + sommes de contrôle -----------------------------------------
find "$DIST" -mindepth 1 -maxdepth 1 -type d -exec rm -rf {} +
(cd "$DIST" && sha256sum * > SHA256SUMS)
echo
ls -lh "$DIST"
