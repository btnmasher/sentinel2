package jobs

const (
	StatusRunning  = "running"
	StatusSuccess  = "success"
	StatusFailed   = "failed"
	StatusPartial  = "partial"
	StatusSkipped  = "skipped"
	StatusCanceled = "canceled"
	StatusTimeout  = "timeout"

	MessageJobCompleted            = "job completed"
	MessageJobCompletedWithErrors  = "job completed with errors"
	MessageJobFailed               = "job failed"
	MessageJobTimedOut             = "job timed out"
	MessageJobStarted              = "job started"
	MessageStepCompleted           = "job step completed"
	MessageStepCompletedWithErrors = "job step completed with errors"
	MessageStepFailed              = "job step failed"
	MessageStepTimedOut            = "job step timed out"
	MessageStepStarted             = "job step started"
	MessageCanceled                = "canceled"
	MessageUpdateNotNeeded         = "update skipped (not needed)"

	JobCleanup            = "cleanup"
	JobCharacterRefresh   = "character_refresh"
	JobUploaderReleases   = "uploader_releases"
	JobSovCampaignSync    = "sov_campaign_sync"
	JobSkyhookSync        = "skyhook_sync"
	JobJumpbridgeValidate = "jumpbridge_validate"
	JobUpdateJumpbridges  = "update_jumpbridges"

	TriggerCronSchedule          = "cron.schedule"
	TriggerServerStartup         = "server.startup"
	TriggerAdminManual           = "admin.manual"
	TriggerStaffJumpbridgeImport = "staff.jumpbridge_import"
)
