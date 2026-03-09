package cleanup

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"sentinel2-taskutil/internal/project"
)

func CleanRoot(cfg project.Config) error {
	plan, err := buildCleanPlan(cfg)
	if err != nil {
		return err
	}

	if len(plan.entries) == 0 {
		fmt.Println("No clean targets matched.")
		return nil
	}

	printCleanPlan(os.Stdout, plan)
	if !cfg.CleanYes {
		ok, promptErr := promptYesNo(os.Stdin, os.Stdout, "Proceed with deletion? [y/N]: ")
		if promptErr != nil {
			return promptErr
		}
		if !ok {
			fmt.Println("Clean aborted.")
			return nil
		}
	}

	for _, e := range plan.entries {
		if remErr := os.RemoveAll(e.abs); remErr != nil && !os.IsNotExist(remErr) {
			return remErr
		}
	}
	return nil
}

func buildCleanPlan(cfg project.Config) (cleanPlan, error) {
	rootDir := filepath.Clean(cfg.RootDir)
	rules, err := parseCleanRules(cfg.CleanRules)
	if err != nil {
		return cleanPlan{}, err
	}

	if len(rules) == 0 {
		return cleanPlan{}, nil
	}

	entries, err := scanRoot(rootDir)
	if err != nil {
		return cleanPlan{}, err
	}

	ignored := map[string]struct{}{}
	candidates := make([]cleanEntry, 0, len(entries))
	for _, e := range entries {
		if matchesAnyRule(e.rel, e.isDir, rules, false) {
			ignored[e.rel] = struct{}{}
			continue
		}
		if matchesAnyRule(e.rel, e.isDir, rules, true) {
			candidates = append(candidates, e)
		}
	}

	exePath, _ := os.Executable()
	exePath = filepath.Clean(exePath)

	sort.Slice(candidates, func(i, j int) bool {
		di := depth(candidates[i].rel)
		dj := depth(candidates[j].rel)
		if di != dj {
			return di > dj
		}
		return candidates[i].rel > candidates[j].rel
	})

	filtered := make([]cleanEntry, 0, len(candidates))
	for _, e := range candidates {
		if samePath(e.abs, exePath) {
			continue
		}
		if e.isDir && hasIgnoredDescendant(e.rel, ignored) {
			continue
		}
		filtered = append(filtered, e)
	}
	display, files, dirs, bytes := summarizePlan(filtered)
	return cleanPlan{
		entries: filtered,
		display: display,
		files:   files,
		dirs:    dirs,
		bytes:   bytes,
	}, nil
}
