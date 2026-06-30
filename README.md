# archkeeper

archkeeper is a modern, fast, and elegant CLI tool written in Go to manage dotfiles and installed packages (both native pacman and AUR) on Arch Linux.

It automates tracking configuration files, backing up package lists, and synchronizing changes with Git, making it simple to replicate your personal environment on any other machine.

---

## Features

- Easy Symlinking: Track any file or directory with a single command. archkeeper moves the file to the repository and creates a symlink in its original place.
- Pacman & AUR Backup: Automatically exports list of explicitly installed native packages (via pacman -Qqen) and foreign/AUR packages (via pacman -Qqem).
- Safe Restoration: Recreates missing symbolic links on a new machine (creating automatic .bak backups if conflicting files exist) and compares and installs missing packages via pacman or an AUR helper (yay/paru).
- Git Integration: Stages changes, commits them with automatic timestamped messages, and pushes to a remote repository.
- Sleek Terminal UI: Clean, styled terminal output inspired by the Arch Linux theme, built using Cobra and Lipgloss.

---

## Installation

Since the project is written in Go, you can compile it locally and place it in your path:

```bash
# Compile the binary
go build -o archkeeper ./cmd/archkeeper/main.go

# Move to system path (example)
sudo mv archkeeper /usr/local/bin/
```

Or install it directly via Go:

```bash
go install ./cmd/archkeeper
```

---

## How Synchronization Works

archkeeper splits your configuration into two parts for cross-machine portability:

1. Local Configuration (~/.config/archkeeper/config.yaml):
   Stores only the path to where the dotfiles repository is cloned on the current machine (e.g., ~/dotfiles). This allows different computers to clone the repo to different paths.
2. Shared Manifest (<dotfiles_dir>/archkeeper.yaml):
   Stored inside the dotfiles folder itself and versioned in Git. It tracks which files should be symlinked and package backup configurations.

---

## Command Reference

### 1. Initialize
Set up the local config and dotfiles repository path:
```bash
archkeeper init
```
This prompts for a destination path (defaults to ~/dotfiles) and initializes a Git repository if it does not already exist.

### 2. Track Files
Add a configuration file or directory to track:
```bash
archkeeper add ~/.zshrc
archkeeper add ~/.config/i3
```
archkeeper will move these targets into your dotfiles directory and create relative symbolic links pointing to them.

### 3. Check Status
View details of tracked files, package status, and git repository details:
```bash
archkeeper status
```

### 4. Backup & Sync
Export installed package lists, commit changes, and push to a remote repository:
```bash
archkeeper backup
```
If you set up a Git remote in your dotfiles repository (git remote add origin <url>), this command will automatically push the changes online.

### 5. Restore on a New Machine
To set up a new Arch Linux installation:
1. Install archkeeper on the new machine.
2. Clone your dotfiles repository (e.g., to ~/dotfiles).
3. Run archkeeper init and point it to the cloned directory.
4. Run the restore command:
   ```bash
   archkeeper restore
   ```
archkeeper will read the archkeeper.yaml manifest, recreate all symlinks, and prompt you to install missing native and AUR packages using pacman and your installed helper (yay/paru).
