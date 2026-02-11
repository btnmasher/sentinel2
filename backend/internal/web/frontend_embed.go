//go:build embed_frontend
// +build embed_frontend

package web

import (
	"embed"
	"io/fs"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

//go:embed dist/* dist/assets/*
var frontendFS embed.FS

func MountFrontend(r *router.Router[*core.RequestEvent]) {
	dist, distErr := fs.Sub(frontendFS, "dist")
	if distErr != nil {
		return
	}

	r.GET("/{path...}", apis.Static(dist, true))
}
