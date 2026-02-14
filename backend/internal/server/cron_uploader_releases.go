package server

import (
	"context"
	"time"

	"github.com/pocketbase/pocketbase"

	"sentinel2/internal/jobs"
	"sentinel2/internal/logging"
)

const uploaderReleasesTimeout = 30 * time.Second

func runUploaderReleaseRefresh(app *pocketbase.PocketBase, deps dependencies, trigger string) {
	runner := jobs.NewRunner(app, jobs.RunOptions{
		JobName: jobs.JobUploaderReleases,
		JobOptions: jobs.JobOptions{
			Kind:    jobs.JobUploaderReleases,
			Trigger: trigger,
		},
		Timeout: uploaderReleasesTimeout,
	})
	_ = runner.Run(func(ctx context.Context, stepper jobs.Stepper) error {
		needed := false
		if err := stepper.Run("check_release", true, func(context.Context) error {
			needed = deps.uploaderReleases.RefreshNeeded(time.Now())
			return nil
		}); err != nil {
			return err
		}
		if !needed {
			stepper.WithMessage(jobs.MessageUpdateNotNeeded)
			return stepper.SkipParent(jobs.MessageUpdateNotNeeded)
		}
		return stepper.Run("get_latest_release", true, func(ctx context.Context) error {
			changed, err := deps.uploaderReleases.Refresh(ctx)
			if err != nil {
				return err
			}
			if !changed {
				stepper.WithMessage(jobs.MessageUpdateNotNeeded)
			}
			links := deps.uploaderReleases.Snapshot()
			runner.WithFields(logging.Fields{
				"updated":           changed,
				"linux_available":   links.LinuxURL != "",
				"windows_available": links.WindowsURL != "",
				"macos_available":   links.MacOSURL != "",
			})
			return nil
		})
	})
}
