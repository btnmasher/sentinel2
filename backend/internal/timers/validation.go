package timers

import (
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

func validateCreateInput(input *CreateInput) error {
	if input == nil {
		return ErrMissingInput
	}
	if input.ExpiresAt.IsZero() {
		return ErrMissingExpiresAt
	}
	if err := validateCreateTimerContext(input); err != nil {
		return err
	}
	if requiresMoon(input.StructureType) && input.MoonID <= 0 {
		return ErrMoonRequired
	}
	if requiresPlanet(input.StructureType) && input.PlanetID <= 0 {
		return ErrPlanetRequired
	}
	if err := validateCreateSkyhookExtraction(input); err != nil {
		return err
	}
	return nil
}

func validateCreateTimerContext(input *CreateInput) error {
	if input == nil {
		return ErrMissingInput
	}
	stageLabel := strings.TrimSpace(input.StageLabel)
	if stageLabel != "" && !IsStageLabel(stageLabel) {
		return ErrInvalidStageLabel
	}
	timerKind := strings.TrimSpace(input.TimerKind)
	if timerKind == "" {
		timerKind = TimerKindReinforcement
	}
	structureType := strings.TrimSpace(input.StructureType)
	if structureType == "" {
		structureType = TimerStructureCustom
	}
	if !IsAllowedTimerContext(timerKind, structureType, stageLabel) {
		return ErrInvalidTimerContext
	}
	return nil
}

func validateCreateSkyhookExtraction(input *CreateInput) error {
	if input == nil {
		return ErrMissingInput
	}
	if input.TimerKind != TimerKindExtraction || input.StructureType != TimerStructureOrbitalSkyhook {
		return nil
	}
	if input.SkyhookFullnessPct == nil || *input.SkyhookFullnessPct < 0 || *input.SkyhookFullnessPct > 100 {
		return ErrInvalidSkyhookFullnessPercentage
	}
	return nil
}

func validateUpdateInput(record *core.Record, input *UpdateInput) error {
	if input == nil {
		return ErrMissingInput
	}
	stageLabel := strings.TrimSpace(record.GetString("stage_label"))
	if input.StageLabel != nil {
		stageLabel = strings.TrimSpace(*input.StageLabel)
	}
	if stageLabel != "" && !IsStageLabel(stageLabel) {
		return ErrInvalidStageLabel
	}
	structureType, moonID, planetID := resolveUpdateContext(record, input)
	timerKind := resolveUpdateTimerKind(record, input)
	if !IsAllowedTimerContext(timerKind, structureType, stageLabel) {
		return ErrInvalidTimerContext
	}
	if requiresMoon(structureType) && moonID <= 0 {
		return ErrMoonRequired
	}
	if requiresPlanet(structureType) && planetID <= 0 {
		return ErrPlanetRequired
	}

	if timerKind == TimerKindExtraction && structureType == TimerStructureOrbitalSkyhook {
		fullness := resolveUpdateSkyhookFullness(record, input)
		if fullness < 0 || fullness > 100 {
			return ErrInvalidSkyhookFullnessPercentage
		}
	}
	return nil
}

func resolveUpdateContext(record *core.Record, input *UpdateInput) (structureType string, moonID, planetID int) {
	structureType = record.GetString("structure_type")
	if input.StructureType != nil && strings.TrimSpace(*input.StructureType) != "" {
		structureType = strings.TrimSpace(*input.StructureType)
	}

	moonID = record.GetInt("moon_id")
	if input.MoonID != nil {
		moonID = *input.MoonID
	}

	planetID = record.GetInt("planet_id")
	if input.PlanetID != nil {
		planetID = *input.PlanetID
	}
	return structureType, moonID, planetID
}

func resolveUpdateTimerKind(record *core.Record, input *UpdateInput) string {
	timerKind := record.GetString("timer_kind")
	if input.TimerKind != nil && strings.TrimSpace(*input.TimerKind) != "" {
		timerKind = strings.TrimSpace(*input.TimerKind)
	}
	return timerKind
}

func resolveUpdateSkyhookFullness(record *core.Record, input *UpdateInput) int {
	fullness := record.GetInt("skyhook_fullness_pct")
	if input.SkyhookFullnessPct != nil {
		fullness = *input.SkyhookFullnessPct
	}
	return fullness
}

func requiresMoon(structureType string) bool {
	_, ok := moonRequiredStructureTypes[strings.TrimSpace(structureType)]
	return ok
}

func requiresPlanet(structureType string) bool {
	_, ok := planetRequiredStructureTypes[strings.TrimSpace(structureType)]
	return ok
}
