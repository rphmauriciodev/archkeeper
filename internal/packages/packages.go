package packages

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"archkeeper/internal/config"
)

// ExportPackages queries pacman for native and AUR packages and saves them to the configured files.
func ExportPackages(cfg *config.LocalConfig, manifest *config.RepoManifest) (string, string, error) {
	if !manifest.Packages.BackupEnabled {
		return "", "", fmt.Errorf("packages backup is disabled in manifest")
	}

	dotfilesAbs, err := config.ResolvePath(cfg.DotfilesDir)
	if err != nil {
		return "", "", fmt.Errorf("failed to resolve dotfiles dir: %w", err)
	}

	// 1. Export Native Packages: pacman -Qqen
	nativePkgs, err := runCommand("pacman", "-Qqen")
	if err != nil {
		return "", "", fmt.Errorf("failed to list native packages: %w", err)
	}

	nativePath := filepath.Join(dotfilesAbs, manifest.Packages.PacmanFile)
	if err := os.WriteFile(nativePath, []byte(nativePkgs), 0644); err != nil {
		return "", "", fmt.Errorf("failed to write native package list: %w", err)
	}

	// 2. Export AUR Packages: pacman -Qqem
	aurPkgs, err := runCommand("pacman", "-Qqem")
	if err != nil {
		return "", "", fmt.Errorf("failed to list AUR packages: %w", err)
	}

	aurPath := filepath.Join(dotfilesAbs, manifest.Packages.AurFile)
	if err := os.WriteFile(aurPath, []byte(aurPkgs), 0644); err != nil {
		return "", "", fmt.Errorf("failed to write AUR package list: %w", err)
	}

	return nativePath, aurPath, nil
}

// GetMissingPackages compares the backed-up package list files with currently installed packages
// and returns the lists of missing packages.
func GetMissingPackages(cfg *config.LocalConfig, manifest *config.RepoManifest) ([]string, []string, error) {
	dotfilesAbs, err := config.ResolvePath(cfg.DotfilesDir)
	if err != nil {
		return nil, nil, err
	}

	// 1. Get current packages
	currentNativeStr, err := runCommand("pacman", "-Qqen")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list current native packages: %w", err)
	}
	currentAurStr, err := runCommand("pacman", "-Qqem")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list current AUR packages: %w", err)
	}

	currentNative := makeSet(strings.Split(strings.TrimSpace(currentNativeStr), "\n"))
	currentAur := makeSet(strings.Split(strings.TrimSpace(currentAurStr), "\n"))

	// 2. Read backed-up packages
	nativePath := filepath.Join(dotfilesAbs, manifest.Packages.PacmanFile)
	aurPath := filepath.Join(dotfilesAbs, manifest.Packages.AurFile)

	var missingNative []string
	var missingAur []string

	if data, err := os.ReadFile(nativePath); err == nil {
		backedUpNative := strings.Split(strings.TrimSpace(string(data)), "\n")
		for _, pkg := range backedUpNative {
			pkg = strings.TrimSpace(pkg)
			if pkg != "" && !currentNative[pkg] {
				missingNative = append(missingNative, pkg)
			}
		}
	}

	if data, err := os.ReadFile(aurPath); err == nil {
		backedUpAur := strings.Split(strings.TrimSpace(string(data)), "\n")
		for _, pkg := range backedUpAur {
			pkg = strings.TrimSpace(pkg)
			if pkg != "" && !currentAur[pkg] && !currentNative[pkg] { // Check both to avoid double installing
				missingAur = append(missingAur, pkg)
			}
		}
	}

	return missingNative, missingAur, nil
}

// InstallPackages executes the package manager to install the list of missing packages.
func InstallPackages(packages []string, isAur bool) error {
	if len(packages) == 0 {
		return nil
	}

	var cmdName string
	var args []string

	if isAur {
		// Detect AUR Helper: yay or paru
		helper := detectAURHelper()
		if helper == "" {
			return fmt.Errorf("no AUR helper (yay or paru) detected. Please install packages manually:\n%s", strings.Join(packages, " "))
		}
		cmdName = helper
		args = append([]string{"-S", "--needed"}, packages...)
	} else {
		cmdName = "sudo"
		args = append([]string{"pacman", "-S", "--needed"}, packages...)
	}

	fmt.Printf("Running command: %s %s\n", cmdName, strings.Join(args, " "))

	cmd := exec.Command(cmdName, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("package installation command failed: %w", err)
	}

	return nil
}

// Helper to run a command and capture output
func runCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("command %s %s failed: %w (stderr: %s)", name, strings.Join(args, " "), err, stderr.String())
	}
	return stdout.String(), nil
}

// Helper to convert string slice to a boolean set
func makeSet(slice []string) map[string]bool {
	set := make(map[string]bool)
	for _, s := range slice {
		trimmed := strings.TrimSpace(s)
		if trimmed != "" {
			set[trimmed] = true
		}
	}
	return set
}

// Helper to check if a helper is in PATH
func detectAURHelper() string {
	helpers := []string{"yay", "paru"}
	for _, h := range helpers {
		if _, err := exec.LookPath(h); err == nil {
			return h
		}
	}
	return ""
}
