package timers

import (
	"github.com/pocketbase/pocketbase"

	esipkg "sentinel2/internal/esi"
)

type Service struct {
	App       *pocketbase.PocketBase
	PublicESI *esipkg.ESIPublicClient
	ESI       esipkg.ESIClient
}

func NewService(app *pocketbase.PocketBase, publicESI *esipkg.ESIPublicClient, esiClient esipkg.ESIClient) *Service {
	return &Service{
		App:       app,
		PublicESI: publicESI,
		ESI:       esiClient,
	}
}

func (s *Service) ParseText(raw string) (ParseResult, error) {
	return parseText(raw)
}
