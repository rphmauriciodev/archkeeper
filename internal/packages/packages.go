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

func ExportPackages(cfg *config.LocalConfig, manifest *config.RepoManifest) (string, string, error) {
	if !manifest.Packages.BackupEnabled {
		return "", "", fmt.Errorf("packages backup is disabled in manifest")
	}

	dotfilesAbs, err := config.ResolvePath(cfg.DotfilesDir)
	if err != nil {
		return "", "", fmt.Errorf("failed to resolve dotfiles dir: %w", err)
	}

	nativePkgs, err := runCommand("pacman", "-Qqen")
	if err != nil {
		return "", "", fmt.Errorf("failed to list native packages: %w", err)
	}

	nativePath := filepath.Join(dotfilesAbs, manifest.Packages.PacmanFile)
	if err := os.WriteFile(nativePath, []byte(nativePkgs), 0644); err != nil {
		return "", "", fmt.Errorf("failed to write native package list: %w", err)
	}

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

func GetMissingPackages(cfg *config.LocalConfig, manifest *config.RepoManifest) ([]string, []string, error) {
	dotfilesAbs, err := config.ResolvePath(cfg.DotfilesDir)
	if err != nil {
		return nil, nil, err
	}

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
			if pkg != "" && !currentAur[pkg] && !currentNative[pkg] {
				missingAur = append(missingAur, pkg)
			}
		}
	}

	return missingNative, missingAur, nil
}

func InstallPackages(packages []string, isAur bool) error {
	if len(packages) == 0 {
		return nil
	}

	var cmdName string
	var args []string

	if isAur {
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

func detectAURHelper() string {
	helpers := []string{"yay", "paru"}
	for _, h := range helpers {
		if _, err := exec.LookPath(h); err == nil {
			return h
		}
	}
	return ""
}
