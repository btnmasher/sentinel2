package cleanup

import "runtime"

var caseInsensitivePatterns = runtime.GOOS == "windows"

type cleanEntry struct {
	abs   string
	rel   string
	isDir bool
}

type cleanRule struct {
	include  bool
	anchored bool
	pattern  string
}

type cleanPlan struct {
	entries []cleanEntry
	display []string
	files   int
	dirs    int
	bytes   int64
}
