package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"archkeeper/internal/config"
)

// IsGitRepo checks if the dotfiles directory contains a .git folder.
func IsGitRepo(dotfilesDir string) bool {
	resolved, err := config.ResolvePath(dotfilesDir)
	if err != nil {
		return false
	}
	gitPath := filepath.Join(resolved, ".git")
	info, err := os.Stat(gitPath)
	return err == nil && info.IsDir()
}

// InitGitRepo initializes a new git repository in the dotfiles directory.
func InitGitRepo(dotfilesDir string) error {
	resolved, err := config.ResolvePath(dotfilesDir)
	if err != nil {
		return err
	}

	cmd := exec.Command("git", "init")
	cmd.Dir = resolved
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to init git repo: %w (stderr: %s)", err, stderr.String())
	}
	return nil
}

// CommitAndPush stages all files, commits with the given message, and pushes if a remote is configured.
func CommitAndPush(dotfilesDir string, commitMessage string) (bool, error) {
	resolved, err := config.ResolvePath(dotfilesDir)
	if err != nil {
		return false, err
	}

	if !IsGitRepo(resolved) {
		return false, fmt.Errorf("directory is not a git repository. Run 'git init' inside %s first", resolved)
	}

	// 1. Git add
	if err := runGitCmd(resolved, "add", "-A"); err != nil {
		return false, fmt.Errorf("failed to stage files: %w", err)
	}

	// Check if there are changes to commit (git diff-index --quiet HEAD --)
	// Note: diff-index returns exit status 1 if there are changes, 0 if clean, and 128 if no commits yet (empty repo).
	hasChanges := true
	if err := runGitCmd(resolved, "diff-index", "--quiet", "HEAD", "--"); err == nil {
		hasChanges = false
	}

	if !hasChanges {
		// No changes to commit, return gracefully
		return false, nil
	}

	// 2. Git commit
	if err := runGitCmd(resolved, "commit", "-m", commitMessage); err != nil {
		return false, fmt.Errorf("failed to commit changes: %w", err)
	}

	// 3. Check for remote and push
	hasRemote, err := hasGitRemote(resolved)
	if err != nil {
		return true, fmt.Errorf("committed changes locally, but failed checking remote: %w", err)
	}

	if hasRemote {
		// Run git push
		// We'll try to push current branch. First find current branch name.
		branch, err := getCurrentBranch(resolved)
		if err != nil {
			branch = "main" // fallback
		}
		if err := runGitCmd(resolved, "push", "origin", branch); err != nil {
			return true, fmt.Errorf("committed changes locally, but failed to push to origin: %w", err)
		}
		return true, nil
	}

	return false, nil
}

func runGitCmd(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git %s failed: %w (stderr: %s)", strings.Join(args, " "), err, stderr.String())
	}
	return nil
}

func hasGitRemote(dir string) (bool, error) {
	cmd := exec.Command("git", "remote")
	cmd.Dir = dir
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return false, err
	}
	// If output is not empty, there is at least one remote
	remotes := strings.TrimSpace(stdout.String())
	if remotes == "" {
		return false, nil
	}
	// Check if "origin" is one of them
	for _, r := range strings.Split(remotes, "\n") {
		if strings.TrimSpace(r) == "origin" {
			return true, nil
		}
	}
	return false, nil
}

func getCurrentBranch(dir string) (string, error) {
	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = dir
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return strings.TrimSpace(stdout.String()), nil
}
