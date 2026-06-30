package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// LocalConfig stores configuration specific to this local machine.
// It is stored in ~/.config/archkeeper/config.yaml
type LocalConfig struct {
	DotfilesDir string `yaml:"dotfiles_dir"`
}

// PackageConfig stores options for backing up package lists.
type PackageConfig struct {
	BackupEnabled bool   `yaml:"backup_enabled"`
	PacmanFile    string `yaml:"pacman_file"`
	AurFile       string `yaml:"aur_file"`
}

// TrackedFile maps a source path (relative to home) to a target path inside the repository.
type TrackedFile struct {
	Source string `yaml:"source"` // e.g. ".config/i3/config"
	Target string `yaml:"target"` // e.g. "config/i3/config"
}

// RepoManifest is versioned in the git repo at <dotfiles_dir>/archkeeper.yaml
type RepoManifest struct {
	Packages PackageConfig `yaml:"packages"`
	Files    []TrackedFile `yaml:"files"`
}

const (
	DefaultLocalConfigDir  = ".config/archkeeper"
	DefaultLocalConfigFile = "config.yaml"
	ManifestFileName       = "archkeeper.yaml"
)

// ResolvePath replaces "~" and environment variables with full paths.
func ResolvePath(path string) (string, error) {
	if path == "" {
		return "", nil
	}

	// Handle ~/ prefix
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("could not get home directory: %w", err)
		}
		path = filepath.Join(home, path[1:])
	}

	// Expand env vars like $HOME
	path = os.ExpandEnv(path)

	return filepath.Clean(path), nil
}

// DefaultRepoManifest returns a standard starting manifest.
func DefaultRepoManifest() *RepoManifest {
	return &RepoManifest{
		Packages: PackageConfig{
			BackupEnabled: true,
			PacmanFile:    "pkglist.txt",
			AurFile:       "pkglist-aur.txt",
		},
		Files: []TrackedFile{},
	}
}

// LoadLocalConfig loads the local configuration from ~/.config/archkeeper/config.yaml.
func LoadLocalConfig() (*LocalConfig, string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, "", fmt.Errorf("could not get home directory: %w", err)
	}

	configDir := filepath.Join(home, DefaultLocalConfigDir)
	configPath := filepath.Join(configDir, DefaultLocalConfigFile)

	// If file doesn't exist, return nil/error so CLI can guide the user to run init
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, configPath, fmt.Errorf("local configuration not found. Please run 'archkeeper init'")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, configPath, fmt.Errorf("failed to read local config: %w", err)
	}

	var cfg LocalConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, configPath, fmt.Errorf("failed to parse local config YAML: %w", err)
	}

	return &cfg, configPath, nil
}

// SaveLocalConfig writes the local configuration file.
func SaveLocalConfig(cfg *LocalConfig) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not get home directory: %w", err)
	}

	configDir := filepath.Join(home, DefaultLocalConfigDir)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create config directory %s: %w", configDir, err)
	}

	configPath := filepath.Join(configDir, DefaultLocalConfigFile)

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return "", fmt.Errorf("failed to marshal local config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write local config file: %w", err)
	}

	return configPath, nil
}

// LoadManifest loads the repo manifest (archkeeper.yaml) from the dotfiles directory.
func LoadManifest(dotfilesDir string) (*RepoManifest, string, error) {
	resolvedDir, err := ResolvePath(dotfilesDir)
	if err != nil {
		return nil, "", err
	}

	manifestPath := filepath.Join(resolvedDir, ManifestFileName)
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		return nil, manifestPath, fmt.Errorf("manifest file not found at: %s", manifestPath)
	}

	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, manifestPath, fmt.Errorf("failed to read manifest file: %w", err)
	}

	var manifest RepoManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, manifestPath, fmt.Errorf("failed to parse manifest YAML: %w", err)
	}

	return &manifest, manifestPath, nil
}

// SaveManifest writes the repo manifest (archkeeper.yaml) to the dotfiles directory.
func SaveManifest(dotfilesDir string, manifest *RepoManifest) (string, error) {
	resolvedDir, err := ResolvePath(dotfilesDir)
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(resolvedDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create dotfiles directory %s: %w", resolvedDir, err)
	}

	manifestPath := filepath.Join(resolvedDir, ManifestFileName)

	data, err := yaml.Marshal(manifest)
	if err != nil {
		return "", fmt.Errorf("failed to marshal manifest: %w", err)
	}

	if err := os.WriteFile(manifestPath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to write manifest file: %w", err)
	}

	return manifestPath, nil
}
