//go:build !embed_frontend
// +build !embed_frontend

package web

import (
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/router"
)

func MountFrontend(r *router.Router[*core.RequestEvent]) {
	_ = r
}
