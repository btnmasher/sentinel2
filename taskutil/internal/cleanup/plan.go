package cleanup

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

func printCleanPlan(w io.Writer, plan cleanPlan) {
	fmt.Fprintf(w, "Clean plan: %d files, %d dirs, %s\n", plan.files, plan.dirs, formatBytes(plan.bytes))
	fmt.Fprintln(w, "Targets:")
	for _, item := range plan.display {
		fmt.Fprintf(w, "  - %s\n", item)
	}
}

func promptYesNo(in io.Reader, out io.Writer, prompt string) (bool, error) {
	fmt.Fprint(out, prompt)
	reader := bufio.NewReader(in)
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return false, err
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes", nil
}

func summarizePlan(entries []cleanEntry) ([]string, int, int, int64) {
	if len(entries) == 0 {
		return nil, 0, 0, 0
	}
	files := 0
	dirs := 0
	var bytes int64
	for _, e := range entries {
		if e.isDir {
			dirs++
			continue
		}
		files++
		if info, err := os.Lstat(e.abs); err == nil {
			bytes += info.Size()
		}
	}

	sorted := append([]cleanEntry(nil), entries...)
	sort.Slice(sorted, func(i, j int) bool {
		di := depth(sorted[i].rel)
		dj := depth(sorted[j].rel)
		if di != dj {
			return di < dj
		}
		return sorted[i].rel < sorted[j].rel
	})

	display := make([]string, 0, len(sorted))
	representedDirs := make([]string, 0, 32)
	for _, e := range sorted {
		if hasParentRepresentation(e.rel, representedDirs) {
			continue
		}
		if e.isDir {
			display = append(display, e.rel+"/*")
			representedDirs = append(representedDirs, e.rel)
			continue
		}
		display = append(display, e.rel)
	}
	return display, files, dirs, bytes
}

func hasParentRepresentation(rel string, represented []string) bool {
	for _, dir := range represented {
		if rel == dir || strings.HasPrefix(rel, dir+"/") {
			return true
		}
	}
	return false
}

func formatBytes(v int64) string {
	const unit = 1024
	if v < unit {
		return fmt.Sprintf("%d B", v)
	}
	div, exp := int64(unit), 0
	for n := v / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	suffix := "KMGTPE"[exp]
	return fmt.Sprintf("%.1f %ciB", float64(v)/float64(div), suffix)
}
