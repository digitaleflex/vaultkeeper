// Package web embarque l'application React compilée (web/dist) dans le binaire vk.
// Régénérer avec : make web
package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var dist embed.FS

// Dist renvoie le contenu de web/dist.
func Dist() fs.FS {
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err)
	}
	return sub
}
