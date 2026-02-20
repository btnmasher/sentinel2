package timers

const (
	defaultListLimit         = 200
	defaultSearchLimit       = 20
	systemNameCandidateLimit = 5

	severityRankCritical = 4
	severityRankHigh     = 3
	severityRankMedium   = 2
	severityRankLow      = 1

	timerStatusActive   = "active"
	timerStatusCanceled = "canceled"
)

var moonRequiredStructureTypes = map[string]struct{}{
	"metenox_moon_drill": {},
}

var planetRequiredStructureTypes = map[string]struct{}{
	"orbital_skyhook": {},
	"mercenary_den":   {},
}
