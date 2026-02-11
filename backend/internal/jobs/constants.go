package jobs

const (
	StatusRunning  = "running"
	StatusSuccess  = "success"
	StatusFailed   = "failed"
	StatusPartial  = "partial"
	StatusSkipped  = "skipped"
	StatusCanceled = "canceled"
	StatusTimeout  = "timeout"

	MessageJobCompleted    = "job completed"
	MessageJobStarted      = "job started"
	MessageStepCompleted   = "job step completed"
	MessageStepStarted     = "job step started"
	MessageCanceled        = "canceled"
	MessageUpdateNotNeeded = "update skipped (not needed)"

	JobCleanup          = "cleanup"
	JobCharacterRefresh = "character_refresh"

	TriggerCronSchedule          = "cron.schedule"
	TriggerAdminManual           = "admin.manual"
	TriggerStaffJumpbridgeImport = "staff.jumpbridge_import"
)
