package intel

import (
	"github.com/pocketbase/pocketbase"

	"sentinel2/internal/config"
	"sentinel2/internal/intel"
)

func NewIntelHandler(app *pocketbase.PocketBase, cfg *config.Config, service *intel.IntelService) *IntelHandler {
	return &IntelHandler{
		App:     app,
		Config:  cfg,
		Service: service,
	}
}
