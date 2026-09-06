package web

import (
	"embed"
	"io/fs"
)

// DistFS embeds the production static frontend assets from the dist directory.
//
// Scaffolding provided in Issue #61 packages the React 19 / Vite application,
// Tailwind CSS v4 styling, vendor-split chunks, and PWA assets for zero-overhead
// delivery by the Go BFF service.
//
//go:embed all:dist
var DistFS embed.FS

// GetFS returns an fs.FS rooted directly at the embedded "dist" sub-tree.
func GetFS() (fs.FS, error) {
	return fs.Sub(DistFS, "dist")
}
