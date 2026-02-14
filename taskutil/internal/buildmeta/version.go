package buildmeta

import (
	"os"
	"os/exec"
	"strings"
)

func DeriveBuildVersion() (string, error) {
	if explicit := strings.TrimSpace(os.Getenv("BUILD_VERSION")); explicit != "" {
		return explicit, nil
	}
	if _, err := runGit("rev-parse", "--is-inside-work-tree"); err != nil {
		return "", nil
	}
	branch := firstOrEmpty(runGit("rev-parse", "--abbrev-ref", "HEAD"))
	exactTag := firstOrEmpty(runGit("describe", "--tags", "--match", "v[0-9]*", "--exact-match"))
	latestTag := firstOrEmpty(runGit("describe", "--tags", "--match", "v[0-9]*", "--abbrev=0"))
	shortSHA := firstOrEmpty(runGit("rev-parse", "--short", "HEAD"))

	version := ""
	if exactTag != "" {
		version = exactTag
	} else {
		if latestTag != "" {
			version = latestTag
		} else {
			version = "v0.0.0"
		}
		if (branch == "main" || branch == "HEAD") && shortSHA != "" {
			version = version + "-" + shortSHA
		}
	}
	if version != "" {
		if dirty := firstOrEmpty(runGit("status", "--porcelain")); dirty != "" {
			version = version + "-dev"
		}
		if branch != "" && branch != "HEAD" && branch != "main" {
			version = version + "-branch-" + safeBranchName(branch)
		}
	}
	return version, nil
}

func runGit(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func firstOrEmpty(value string, err error) string {
	if err != nil {
		return ""
	}
	return strings.TrimSpace(value)
}

func safeBranchName(branch string) string {
	branch = strings.TrimSpace(branch)
	if branch == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range branch {
		switch {
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '.' || r == '_' || r == '-':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	return b.String()
}
