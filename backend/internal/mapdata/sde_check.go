package mapdata

import (
	"time"

	"github.com/pocketbase/pocketbase"

	"sentinel2/internal/store"
)

func ShouldUpdateSDE(importer *SDEImporter, maxAge time.Duration) (bool, string, error) {
	lastUpdated, hasLastUpdate := sdeUpdatedAt(importer.App)
	stale := !hasLastUpdate || lastUpdated.Before(time.Now().Add(-maxAge))

	needs, etag, needsErr := importer.NeedsUpdate()
	if needsErr == nil {
		if needs {
			return true, etag, nil
		}
		if !hasLastUpdate {
			return true, etag, nil
		}
		if stale {
			return false, etag, nil
		}
		return false, etag, nil
	}

	if !stale {
		return false, "", needsErr
	}
	return true, "", nil
}

func sdeUpdatedAt(app *pocketbase.PocketBase) (time.Time, bool) {
	records, recordsErr := app.FindRecordsByFilter(store.CollectionSDEMeta, "key = {:key}", "", 1, 0, map[string]any{"key": "last_sde_update"})
	if recordsErr != nil || len(records) == 0 {
		return time.Time{}, false
	}
	updated := records[0].GetDateTime("updated_at")
	if updated.IsZero() {
		return time.Time{}, false
	}
	return updated.Time(), true
}
