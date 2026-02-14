package assets

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"sentinel2-taskutil/internal/project"
)

func PrepareEmbed(cfg project.Config) error {
	srcDir := cfg.EmbedSrc()
	destDir := cfg.EmbedDest()
	info, statErr := os.Stat(srcDir)
	if statErr != nil || !info.IsDir() {
		return fmt.Errorf("frontend/dist is missing. run 'task build:frontend' first")
	}
	if remErr := os.RemoveAll(destDir); remErr != nil {
		return remErr
	}
	if mkErr := os.MkdirAll(destDir, 0o755); mkErr != nil {
		return mkErr
	}
	return copyDirContents(srcDir, destDir)
}

func copyDirContents(srcDir, destDir string) error {
	return filepath.WalkDir(srcDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path == srcDir {
			return nil
		}
		rel, relErr := filepath.Rel(srcDir, path)
		if relErr != nil {
			return relErr
		}
		target := filepath.Join(destDir, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if mkErr := os.MkdirAll(filepath.Dir(target), 0o755); mkErr != nil {
			return mkErr
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		return err
	}
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	return nil
}
