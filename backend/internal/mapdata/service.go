package mapdata

import (
	"context"

	"github.com/pocketbase/pocketbase"

	"sentinel2/internal/logging"
)

type MapDataService struct {
	App    *pocketbase.PocketBase
	Logger *logging.Logger
}

func NewMapDataService(app *pocketbase.PocketBase, logger *logging.Logger) *MapDataService {
	if logger == nil {
		logger = logging.New(app)
	}
	return &MapDataService{
		App:    app,
		Logger: logger,
	}
}

func (s *MapDataService) RunStep(ctx context.Context, step string) error {
	switch step {
	case StepSDEImport:
		importer := NewSDEImporter(s.App)
		return importer.DownloadAndImport(ctx, "")
	case StepRealPositions:
		return CalculateRealPositions(ctx, s.App)
	case StepEve2DPositions:
		return UpdateRegionPositionsFromSystems(s.App)
	case StepDotlanImport:
		return DownloadDotlan(ctx, s.App)
	case StepMetroPositions:
		if err := CalculateSystemGraphs(ctx, s.App); err != nil {
			return err
		}
		return CalculateRegionLayouts(ctx, s.App)
	case StepBuildGraph:
		return CalculateSystemGraphs(ctx, s.App)
	case StepRegionLayout:
		return CalculateRegionLayouts(ctx, s.App)
	default:
		return ErrUnknownMapDataStep
	}
}
