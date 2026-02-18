package mapdata

import "errors"

var (
	ErrETagRequestFailed  = errors.New("etag request failed")
	ErrMapJSONLNotFound   = errors.New("map jsonl files not found")
	ErrSDEDownloadFailed  = errors.New("failed to download sde")
	ErrSystemNotFound     = errors.New("system not found")
	ErrUnknownMapDataStep = errors.New("unknown map data step")
)
