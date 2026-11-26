//go:build !windows

package scanner

// detectPlatform : rien de spécifique hors Windows pour l'instant
// (macOS Keychain / GNOME Keyring sont déjà chiffrés par le système).
func detectPlatform(ctx *context) []Finding { return nil }
