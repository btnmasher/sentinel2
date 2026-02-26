package timers

import "strings"

const (
	defaultListLimit         = 200
	defaultSearchLimit       = 20
	systemNameCandidateLimit = 5

	timerStatusActive   = "active"
	timerStatusCanceled = "canceled"
	systemCreatorName   = "System"

	TimerStandingOurs        = "ours"
	TimerStandingFriendly    = "friendly"
	TimerStandingNeutral     = "neutral"
	TimerStandingComplicated = "complicated"
	TimerStandingHostile     = "hostile"

	TimerSeverityCritical = "critical"
	TimerSeverityHigh     = "high"
	TimerSeverityMedium   = "medium"
	TimerSeverityLow      = "low"

	TimerKindReinforcement = "reinforcement"
	TimerKindAnchoring     = "anchoring"
	TimerKindUnanchoring   = "unanchoring"
	TimerKindExtraction    = "extraction"
	TimerKindCustom        = "custom"

	TimerStructureOrbitalSkyhook           = "orbital_skyhook"
	TimerStructureSovereigntyHub           = "sovereignty_hub"
	TimerStructureUpwellCitadelKeepstar    = "upwell_citadel_keepstar"
	TimerStructureUpwellCitadelFortizar    = "upwell_citadel_fortizar"
	TimerStructureUpwellCitadelAstrahus    = "upwell_citadel_astrahus"
	TimerStructureUpwellEngineeringSotiyo  = "upwell_engineering_sotiyo"
	TimerStructureUpwellEngineeringAzbel   = "upwell_engineering_azbel"
	TimerStructureUpwellEngineeringRaitaru = "upwell_engineering_raitaru"
	TimerStructureUpwellRefineryTatara     = "upwell_refinery_tatara"
	TimerStructureUpwellRefineryAthanor    = "upwell_refinery_athanor"
	TimerStructureAnsiblexJumpBridge       = "ansiblex_jump_bridge"
	TimerStructurePharoluxCynoBeacon       = "pharolux_cyno_beacon"
	TimerStructureTenebrexCynoJammer       = "tenebrex_cyno_jammer"
	TimerStructurePlayerOwnedStarbase      = "player_owned_starbase"
	TimerStructureMetenoxMoonDrill         = "metenox_moon_drill"
	TimerStructureMercenaryDen             = "mercenary_den"
	TimerStructureCustomsOfficePOCO        = "customs_office_poco"
	TimerStructureCustom                   = "custom"

	TimerStageArmor                = "armor"
	TimerStageReinforcement        = "reinforcement"
	TimerStageHull                 = "hull"
	TimerStageInitialVulnerability = "initial_vulnerability"
	TimerStageNotApplicable        = "not_applicable"
	TimerStageAnchoring            = "anchoring"
	TimerStageUnanchoring          = "unanchoring"
	TimerStageExtractionWindow     = "extraction_window"
	TimerStageCustom               = "custom"
)

const (
	severityRankLow = iota + 1
	severityRankMedium
	severityRankHigh
	severityRankCritical
)

var moonRequiredStructureTypes = map[string]struct{}{
	TimerStructureMetenoxMoonDrill: {},
}

var planetRequiredStructureTypes = map[string]struct{}{
	TimerStructureOrbitalSkyhook: {},
	TimerStructureMercenaryDen:   {},
}

var timerStandingSet = map[string]struct{}{
	TimerStandingOurs:        {},
	TimerStandingFriendly:    {},
	TimerStandingNeutral:     {},
	TimerStandingComplicated: {},
	TimerStandingHostile:     {},
}

var timerStageSet = map[string]struct{}{
	TimerStageArmor:                {},
	TimerStageReinforcement:        {},
	TimerStageHull:                 {},
	TimerStageInitialVulnerability: {},
	TimerStageNotApplicable:        {},
	TimerStageAnchoring:            {},
	TimerStageUnanchoring:          {},
	TimerStageExtractionWindow:     {},
	TimerStageCustom:               {},
}

var reinforcementDualStageStructures = map[string]struct{}{
	TimerStructureUpwellCitadelKeepstar:   {},
	TimerStructureUpwellCitadelFortizar:   {},
	TimerStructureUpwellEngineeringSotiyo: {},
	TimerStructureUpwellEngineeringAzbel:  {},
	TimerStructureUpwellRefineryTatara:    {},
}

var reinforcementSingleStageStructures = map[string]struct{}{
	TimerStructureUpwellCitadelAstrahus:    {},
	TimerStructureUpwellEngineeringRaitaru: {},
	TimerStructureUpwellRefineryAthanor:    {},
	TimerStructureAnsiblexJumpBridge:       {},
	TimerStructurePharoluxCynoBeacon:       {},
	TimerStructureTenebrexCynoJammer:       {},
	TimerStructureOrbitalSkyhook:           {},
	TimerStructureMetenoxMoonDrill:         {},
	TimerStructureMercenaryDen:             {},
	TimerStructureCustomsOfficePOCO:        {},
	TimerStructurePlayerOwnedStarbase:      {},
	TimerStructureSovereigntyHub:           {},
}

var reinforcementInitialVulnerabilityStructures = map[string]struct{}{
	TimerStructureUpwellCitadelKeepstar:    {},
	TimerStructureUpwellCitadelFortizar:    {},
	TimerStructureUpwellCitadelAstrahus:    {},
	TimerStructureUpwellEngineeringSotiyo:  {},
	TimerStructureUpwellEngineeringAzbel:   {},
	TimerStructureUpwellEngineeringRaitaru: {},
	TimerStructureUpwellRefineryTatara:     {},
	TimerStructureUpwellRefineryAthanor:    {},
	TimerStructureAnsiblexJumpBridge:       {},
	TimerStructurePharoluxCynoBeacon:       {},
	TimerStructureTenebrexCynoJammer:       {},
	TimerStructureOrbitalSkyhook:           {},
	TimerStructureMetenoxMoonDrill:         {},
	TimerStructureMercenaryDen:             {},
	TimerStructureCustomsOfficePOCO:        {},
	TimerStructurePlayerOwnedStarbase:      {},
	TimerStructureSovereigntyHub:           {},
}

var anchoringTimerStructures = map[string]struct{}{
	TimerStructureUpwellCitadelKeepstar:    {},
	TimerStructureUpwellCitadelFortizar:    {},
	TimerStructureUpwellCitadelAstrahus:    {},
	TimerStructureUpwellEngineeringSotiyo:  {},
	TimerStructureUpwellEngineeringAzbel:   {},
	TimerStructureUpwellEngineeringRaitaru: {},
	TimerStructureUpwellRefineryTatara:     {},
	TimerStructureUpwellRefineryAthanor:    {},
	TimerStructureAnsiblexJumpBridge:       {},
	TimerStructurePharoluxCynoBeacon:       {},
	TimerStructureTenebrexCynoJammer:       {},
	TimerStructureOrbitalSkyhook:           {},
	TimerStructureMetenoxMoonDrill:         {},
	TimerStructureCustomsOfficePOCO:        {},
	TimerStructurePlayerOwnedStarbase:      {},
}

var unanchoringTimerStructures = map[string]struct{}{
	TimerStructureUpwellCitadelKeepstar:    {},
	TimerStructureUpwellCitadelFortizar:    {},
	TimerStructureUpwellCitadelAstrahus:    {},
	TimerStructureUpwellEngineeringSotiyo:  {},
	TimerStructureUpwellEngineeringAzbel:   {},
	TimerStructureUpwellEngineeringRaitaru: {},
	TimerStructureUpwellRefineryTatara:     {},
	TimerStructureUpwellRefineryAthanor:    {},
	TimerStructureAnsiblexJumpBridge:       {},
	TimerStructurePharoluxCynoBeacon:       {},
	TimerStructureTenebrexCynoJammer:       {},
	TimerStructureOrbitalSkyhook:           {},
	TimerStructureMetenoxMoonDrill:         {},
	TimerStructureCustomsOfficePOCO:        {},
	TimerStructurePlayerOwnedStarbase:      {},
}

var extractionTimerStructures = map[string]struct{}{
	TimerStructureUpwellRefineryAthanor: {},
	TimerStructureUpwellRefineryTatara:  {},
	TimerStructureOrbitalSkyhook:        {},
	TimerStructureMetenoxMoonDrill:      {},
	TimerStructureMercenaryDen:          {},
}

func IsStandingType(value string) bool {
	_, ok := timerStandingSet[value]
	return ok
}

func NormalizeStanding(value string) string {
	normalized := strings.TrimSpace(strings.ToLower(value))
	if IsStandingType(normalized) {
		return normalized
	}
	return TimerStandingHostile
}

func IsStageLabel(value string) bool {
	_, ok := timerStageSet[value]
	return ok
}

//nolint:gocognit // validation matrix mirrors timer business rules.
func IsAllowedTimerContext(timerKind, structureType, stageLabel string) bool {
	kind := strings.TrimSpace(timerKind)
	structure := strings.TrimSpace(structureType)
	stage := strings.TrimSpace(stageLabel)
	if kind == "" || stage == "" {
		return false
	}

	switch kind {
	case TimerKindReinforcement:
		if stage == TimerStageNotApplicable {
			return true
		}
		switch stage {
		case TimerStageArmor, TimerStageHull:
			_, ok := reinforcementDualStageStructures[structure]
			return ok
		case TimerStageReinforcement:
			_, ok := reinforcementSingleStageStructures[structure]
			return ok
		case TimerStageInitialVulnerability:
			_, ok := reinforcementInitialVulnerabilityStructures[structure]
			return ok
		default:
			return false
		}
	case TimerKindAnchoring:
		if stage != TimerStageAnchoring {
			return false
		}
		if structure == TimerStructureCustom {
			return true
		}
		_, ok := anchoringTimerStructures[structure]
		return ok
	case TimerKindUnanchoring:
		if stage != TimerStageUnanchoring {
			return false
		}
		if structure == TimerStructureCustom {
			return true
		}
		_, ok := unanchoringTimerStructures[structure]
		return ok
	case TimerKindExtraction:
		if stage != TimerStageExtractionWindow {
			return false
		}
		_, ok := extractionTimerStructures[structure]
		return ok
	case TimerKindCustom:
		return stage == TimerStageCustom
	default:
		return false
	}
}
