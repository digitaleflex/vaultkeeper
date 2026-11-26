// Package scanner détecte les secrets présents sur un poste de travail
// (Windows, macOS, Linux) et produit un inventaire SANS jamais exposer
// les valeurs secrètes elles-mêmes.
package scanner

import (
	"crypto/sha256"
	"encoding/hex"
	"time"
)

// Severity indique l'urgence d'une action sur un secret.
type Severity string

const (
	SevInfo     Severity = "info"
	SevLow      Severity = "faible"
	SevMedium   Severity = "moyen"
	SevHigh     Severity = "élevé"
	SevCritical Severity = "critique"
)

func (s Severity) Rank() int {
	switch s {
	case SevCritical:
		return 4
	case SevHigh:
		return 3
	case SevMedium:
		return 2
	case SevLow:
		return 1
	}
	return 0
}

// Category regroupe les secrets par destination d'import.
type Category string

const (
	CatSSH     Category = "ssh"     // -> Vaultwarden (élément Clé SSH)
	CatEnv     Category = "env"     // -> Infisical
	CatCloud   Category = "cloud"   // -> Vaultwarden (note sécurisée)
	CatDevTool Category = "devtool" // -> Vaultwarden
	CatServer  Category = "server"  // -> Vaultwarden (identifiant)
	CatLeak    Category = "fuite"   // -> à nettoyer + secret à renouveler
)

// Finding décrit un secret trouvé. Aucun champ ne contient de valeur secrète.
type Finding struct {
	ID          string    `json:"id"`
	Category    Category  `json:"category"`
	Type        string    `json:"type"`
	Path        string    `json:"path"`
	Description string    `json:"description"`
	Targets     []string  `json:"targets,omitempty"` // serveurs, registres, clusters…
	Keys        []string  `json:"keys,omitempty"`    // NOMS de variables / profils, jamais les valeurs
	Warnings    []string  `json:"warnings,omitempty"`
	Severity    Severity  `json:"severity"`
	Fingerprint string    `json:"fingerprint,omitempty"`
	ModTime     time.Time `json:"modified,omitempty"`
}

func (f *Finding) warn(sev Severity, msg string) {
	f.Warnings = append(f.Warnings, msg)
	if sev.Rank() > f.Severity.Rank() {
		f.Severity = sev
	}
}

func makeID(parts ...string) string {
	h := sha256.New()
	for _, p := range parts {
		h.Write([]byte(p))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))[:12]
}
