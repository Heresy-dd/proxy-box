package proxybox

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

// GetFrontendFS returns the embedded frontend static file system
func GetFrontendFS() (fs.FS, error) {
	return fs.Sub(distFS, "dist")
}
