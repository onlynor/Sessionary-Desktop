package main

import (
	"embed"
	"io/fs"
)

//go:embed resources/icon.png
var appIcon []byte

// The built frontend (frontend/dist), embedded in the app.
//
//go:embed all:frontend/dist
var dist embed.FS

func frontendFiles() fs.FS {
	sub, err := fs.Sub(dist, "frontend/dist")
	if err != nil {
		return nil
	}
	if _, err := fs.Stat(sub, "index.html"); err != nil {
		return nil
	}
	return sub
}
