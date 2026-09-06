package web

import (
	"embed"
	"io/fs"
)

// DistFS embeds the production static frontend assets from the dist directory.
//
// NOTE: Until Issue #61 scaffolds the full React 19 / Vite application,
// this directory contains temporary placeholder assets to enable and verify
// Go BFF embed.FS packaging, caching headers, and SPA routing.
//
//go:embed all:dist
var DistFS embed.FS

// GetFS returns an fs.FS rooted directly at the embedded "dist" sub-tree.
func GetFS() (fs.FS, error) {
	return fs.Sub(DistFS, "dist")
}
