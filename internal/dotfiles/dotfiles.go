package dotfiles

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"archkeeper/internal/config"
)

func TrackFile(cfg *config.LocalConfig, manifest *config.RepoManifest, sourcePath string, targetRelPath string) error {
	srcAbs, err := config.ResolvePath(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to resolve source path %s: %w", sourcePath, err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get user home dir: %w", err)
	}
	relToHome, err := filepath.Rel(home, srcAbs)
	if err != nil || filepath.IsAbs(relToHome) {
		relToHome = srcAbs
	}

	dotfilesAbs, err := config.ResolvePath(cfg.DotfilesDir)
	if err != nil {
		return fmt.Errorf("failed to resolve dotfiles dir: %w", err)
	}
	tgtAbs := filepath.Join(dotfilesAbs, targetRelPath)

	srcInfo, err := os.Lstat(srcAbs)
	if os.IsNotExist(err) {
		return fmt.Errorf("source path does not exist: %s", srcAbs)
	}

	if srcInfo.Mode()&os.ModeSymlink != 0 {
		resolvedSym, err := os.Readlink(srcAbs)
		if err == nil && resolvedSym == tgtAbs {
			return fmt.Errorf("file is already tracked and symlinked correctly")
		}
		return fmt.Errorf("source is already a symlink pointing to: %s", resolvedSym)
	}

	if _, err := os.Stat(tgtAbs); !os.IsNotExist(err) {
		return fmt.Errorf("target already exists in dotfiles repo at: %s", tgtAbs)
	}

	if err := os.MkdirAll(filepath.Dir(tgtAbs), 0755); err != nil {
		return fmt.Errorf("failed to create target directories: %w", err)
	}

	if err := movePath(srcAbs, tgtAbs); err != nil {
		return fmt.Errorf("failed to move file/dir to repo: %w", err)
	}

	if err := os.Symlink(tgtAbs, srcAbs); err != nil {
		_ = movePath(tgtAbs, srcAbs)
		return fmt.Errorf("failed to create symlink: %w", err)
	}

	exists := false
	for _, f := range manifest.Files {
		if f.Source == relToHome {
			exists = true
			break
		}
	}

	if !exists {
		manifest.Files = append(manifest.Files, config.TrackedFile{
			Source: relToHome,
			Target: targetRelPath,
		})
		if _, err := config.SaveManifest(cfg.DotfilesDir, manifest); err != nil {
			return fmt.Errorf("failed to save updated manifest: %w", err)
		}
	}

	return nil
}

func RestoreLinks(cfg *config.LocalConfig, manifest *config.RepoManifest) ([]string, []string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get user home dir: %w", err)
	}

	dotfilesAbs, err := config.ResolvePath(cfg.DotfilesDir)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to resolve dotfiles dir: %w", err)
	}

	var restored []string
	var skipped []string

	for _, file := range manifest.Files {
		var srcAbs string
		if filepath.IsAbs(file.Source) {
			srcAbs = file.Source
		} else {
			srcAbs = filepath.Join(home, file.Source)
		}

		tgtAbs := filepath.Join(dotfilesAbs, file.Target)

		if _, err := os.Stat(tgtAbs); os.IsNotExist(err) {
			skipped = append(skipped, fmt.Sprintf("%s (target file not found in repository)", file.Source))
			continue
		}

		srcInfo, err := os.Lstat(srcAbs)
		if err == nil {
			if srcInfo.Mode()&os.ModeSymlink != 0 {
				resolvedSym, err := os.Readlink(srcAbs)
				if err == nil && resolvedSym == tgtAbs {
					skipped = append(skipped, fmt.Sprintf("%s (already correctly symlinked)", file.Source))
					continue
				}

				if err := os.Remove(srcAbs); err != nil {
					return nil, nil, fmt.Errorf("failed to remove invalid symlink at %s: %w", srcAbs, err)
				}
			} else {
				bakPath := srcAbs + ".bak"
				if err := os.Rename(srcAbs, bakPath); err != nil {
					return nil, nil, fmt.Errorf("failed to backup existing file at %s: %w", srcAbs, err)
				}
				skipped = append(skipped, fmt.Sprintf("%s (backed up existing to %s.bak)", file.Source, file.Source))
			}
		}

		if err := os.MkdirAll(filepath.Dir(srcAbs), 0755); err != nil {
			return nil, nil, fmt.Errorf("failed to create directory structure for %s: %w", srcAbs, err)
		}

		if err := os.Symlink(tgtAbs, srcAbs); err != nil {
			return nil, nil, fmt.Errorf("failed to create symlink for %s: %w", srcAbs, err)
		}
		restored = append(restored, file.Source)
	}

	return restored, skipped, nil
}

func movePath(src, dst string) error {
	err := os.Rename(src, dst)
	if err == nil {
		return nil
	}

	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	if info.IsDir() {
		if err := copyDir(src, dst); err != nil {
			return err
		}
		return os.RemoveAll(src)
	}

	if err := copyFile(src, dst); err != nil {
		return err
	}
	return os.Remove(src)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err = io.Copy(out, in); err != nil {
		return err
	}

	if err := out.Sync(); err != nil {
		return err
	}

	si, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, si.Mode())
}

func copyDir(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, info.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}
	return nil
}
