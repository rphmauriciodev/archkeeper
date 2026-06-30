package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"archkeeper/internal/config"
	"archkeeper/internal/dotfiles"
	"archkeeper/internal/git"
	"archkeeper/internal/packages"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var (
	archBlue = lipgloss.Color("#1793D1")
	successG = lipgloss.Color("#A6E22E")
	warnY    = lipgloss.Color("#F4BF75")
	errorR   = lipgloss.Color("#F92672")
	gray     = lipgloss.Color("#7F7F7F")

	styleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(archBlue).
			Padding(0, 1).
			MarginBottom(1)

	styleHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(archBlue)

	styleSuccess = lipgloss.NewStyle().
			Bold(true).
			Foreground(successG)

	styleWarn = lipgloss.NewStyle().
			Bold(true).
			Foreground(warnY)

	styleError = lipgloss.NewStyle().
			Bold(true).
			Foreground(errorR)

	styleMuted = lipgloss.NewStyle().
			Foreground(gray)

	styleCard = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(archBlue).
			Padding(1, 2).
			MarginBottom(1)
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "archkeeper",
		Short: "archkeeper is a dotfiles & pacman/AUR package list manager for Arch Linux",
		Long:  styleTitle.Render(" ARCHKEEPER ") + "\nUma ferramenta moderna e elegante em Go para gerenciar seus dotfiles e pacotes no Arch Linux.",
	}

	rootCmd.AddCommand(initCmd())
	rootCmd.AddCommand(addCmd())
	rootCmd.AddCommand(backupCmd())
	rootCmd.AddCommand(restoreCmd())
	rootCmd.AddCommand(statusCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(styleError.Render("Erro:"), err)
		os.Exit(1)
	}
}

func initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Inicializa o repositório de dotfiles e a configuração local",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(styleHeader.Render("Inicializando o Archkeeper..."))

			reader := bufio.NewReader(os.Stdin)
			home, err := os.UserHomeDir()
			if err != nil {
				fmt.Println(styleError.Render("❌ Erro ao detectar a pasta home:"), err)
				return
			}
			defaultDotDir := filepath.Join(home, "dotfiles")

			fmt.Printf("Digite a pasta para armazenar seus dotfiles [%s]: ", defaultDotDir)
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)
			if input == "" {
				input = defaultDotDir
			}

			resolvedDir, err := config.ResolvePath(input)
			if err != nil {
				fmt.Println(styleError.Render("❌ Caminho inválido:"), err)
				return
			}

			localCfg := &config.LocalConfig{DotfilesDir: input}
			localConfigPath, err := config.SaveLocalConfig(localCfg)
			if err != nil {
				fmt.Println(styleError.Render("❌ Erro ao salvar configuração local:"), err)
				return
			}

			manifestPath := filepath.Join(resolvedDir, config.ManifestFileName)
			manifestCreated := false
			if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
				defManifest := config.DefaultRepoManifest()
				_, err = config.SaveManifest(input, defManifest)
				if err != nil {
					fmt.Println(styleError.Render("❌ Erro ao inicializar arquivo de manifesto:"), err)
					return
				}
				manifestCreated = true
			}

			gitInitialized := false
			if !git.IsGitRepo(resolvedDir) {
				if err := git.InitGitRepo(resolvedDir); err != nil {
					fmt.Printf(styleWarn.Render("⚠️  Configurado localmente, mas falhou ao inicializar o Git: %v\n"), err)
				} else {
					gitInitialized = true
				}
			}

			cardContent := fmt.Sprintf(
				"Configuração Local: %s\nRepositório Dotfiles: %s\n\n",
				styleSuccess.Render(localConfigPath),
				styleSuccess.Render(resolvedDir),
			)
			if manifestCreated {
				cardContent += styleSuccess.Render("✓") + " Manifesto criado em: " + manifestPath + "\n"
			} else {
				cardContent += styleWarn.Render("i") + " Manifesto já existente encontrado no repositório.\n"
			}

			if gitInitialized {
				cardContent += styleSuccess.Render("✓") + " Repositório Git inicializado em: " + resolvedDir + "\n"
			} else if git.IsGitRepo(resolvedDir) {
				cardContent += styleMuted.Render("•") + " Repositório Git já existente detectado.\n"
			}

			cardContent += "\n" + styleSuccess.Render("Pronto! Agora você pode começar a rastrear arquivos usando 'archkeeper add <caminho>'")

			fmt.Println(styleCard.Render(cardContent))
		},
	}
}

func addCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add [caminho_do_arquivo]",
		Short: "Adiciona e rastreia um arquivo ou diretório de configuração",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			srcPath := args[0]

			localCfg, _, err := config.LoadLocalConfig()
			if err != nil {
				fmt.Println(styleError.Render("❌ Erro:"), err)
				return
			}

			manifest, _, err := config.LoadManifest(localCfg.DotfilesDir)
			if err != nil {
				manifest = config.DefaultRepoManifest()
			}

			srcAbs, err := config.ResolvePath(srcPath)
			if err != nil {
				fmt.Println(styleError.Render("❌ Erro ao resolver caminho do arquivo:"), err)
				return
			}

			home, err := os.UserHomeDir()
			if err != nil {
				fmt.Println(styleError.Render("❌ Erro ao obter pasta home:"), err)
				return
			}

			var targetRelPath string
			if strings.HasPrefix(srcAbs, home) {
				rel, err := filepath.Rel(home, srcAbs)
				if err != nil {
					targetRelPath = filepath.Base(srcAbs)
				} else {
					targetRelPath = rel
				}
			} else {
				targetRelPath = filepath.Join("root", srcAbs)
			}

			fmt.Printf("Rastreando %s -> %s\n", styleHeader.Render(srcAbs), styleMuted.Render(filepath.Join(localCfg.DotfilesDir, targetRelPath)))

			err = dotfiles.TrackFile(localCfg, manifest, srcPath, targetRelPath)
			if err != nil {
				fmt.Println(styleError.Render("❌ Falha ao rastrear arquivo:"), err)
				return
			}

			fmt.Println(styleSuccess.Render("✓ Arquivo adicionado, link simbólico criado e manifesto atualizado com sucesso!"))
		},
	}
}

func backupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "backup",
		Short: "Exporta pacotes instalados, salva alterações e envia para o Git",
		Run: func(cmd *cobra.Command, args []string) {
			localCfg, _, err := config.LoadLocalConfig()
			if err != nil {
				fmt.Println(styleError.Render("❌ Erro:"), err)
				return
			}

			manifest, _, err := config.LoadManifest(localCfg.DotfilesDir)
			if err != nil {
				fmt.Println(styleError.Render("❌ Erro ao carregar manifesto:"), err)
				return
			}

			fmt.Println(styleHeader.Render("Salvando pacotes do Arch Linux..."))
			pacmanPath, aurPath, err := packages.ExportPackages(localCfg, manifest)
			if err != nil {
				fmt.Printf(styleWarn.Render("⚠️ Falha ao exportar listas de pacotes: %v\n"), err)
			} else {
				fmt.Printf("✓ Pacman list: %s\n", styleSuccess.Render(pacmanPath))
				fmt.Printf("✓ AUR list:    %s\n", styleSuccess.Render(aurPath))
			}

			fmt.Println(styleHeader.Render("\nSincronizando com o Git..."))
			commitMsg := fmt.Sprintf("archkeeper: auto-backup %s", time.Now().Format("2006-01-02 15:04:05"))

			pushed, err := git.CommitAndPush(localCfg.DotfilesDir, commitMsg)
			if err != nil {
				fmt.Println(styleError.Render("❌ Erro na sincronização do Git:"), err)
				return
			}

			if pushed {
				fmt.Println(styleSuccess.Render("✓ Alterações salvas e enviadas com sucesso para o Git remoto!"))
			} else {
				fmt.Println(styleSuccess.Render("✓ Alterações salvas no repositório local com sucesso! (Nenhum git remote configurado)"))
			}
		},
	}
}

func restoreCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "restore",
		Short: "Restaura todos os symlinks e instala pacotes ausentes",
		Run: func(cmd *cobra.Command, args []string) {
			localCfg, _, err := config.LoadLocalConfig()
			if err != nil {
				fmt.Println(styleError.Render("❌ Erro:"), err)
				return
			}

			manifest, _, err := config.LoadManifest(localCfg.DotfilesDir)
			if err != nil {
				fmt.Println(styleError.Render("❌ Erro ao carregar manifesto:"), err)
				return
			}

			fmt.Println(styleHeader.Render("Restaurando links simbólicos..."))
			restored, skipped, err := dotfiles.RestoreLinks(localCfg, manifest)
			if err != nil {
				fmt.Println(styleError.Render("❌ Erro ao restaurar links simbólicos:"), err)
				return
			}

			for _, file := range restored {
				fmt.Printf("✓ Link criado: %s\n", styleSuccess.Render(file))
			}
			for _, file := range skipped {
				fmt.Printf("• Ignorado/Backup: %s\n", styleMuted.Render(file))
			}

			fmt.Println(styleHeader.Render("\nVerificando pacotes ausentes..."))
			missingNative, missingAur, err := packages.GetMissingPackages(localCfg, manifest)
			if err != nil {
				fmt.Printf(styleWarn.Render("Falha ao verificar pacotes ausentes: %v\n"), err)
				return
			}

			reader := bufio.NewReader(os.Stdin)

			if len(missingNative) > 0 {
				fmt.Println(styleWarn.Render(fmt.Sprintf("Detectados %d pacotes nativos do Pacman ausentes:", len(missingNative))))
				fmt.Printf("%s\n\n", strings.Join(missingNative, ", "))

				fmt.Print("Deseja instalar esses pacotes agora? (s/N): ")
				ans, _ := reader.ReadString('\n')
				ans = strings.ToLower(strings.TrimSpace(ans))
				if ans == "s" || ans == "sim" || ans == "y" || ans == "yes" {
					if err := packages.InstallPackages(missingNative, false); err != nil {
						fmt.Println(styleError.Render("❌ Falha na instalação de pacotes nativos:"), err)
					} else {
						fmt.Println(styleSuccess.Render("✓ Pacotes nativos instalados com sucesso!"))
					}
				}
			} else {
				fmt.Println(styleSuccess.Render("✓ Todos os pacotes nativos já estão instalados."))
			}

			if len(missingAur) > 0 {
				fmt.Println(styleWarn.Render(fmt.Sprintf("\nDetectados %d pacotes do AUR ausentes:", len(missingAur))))
				fmt.Printf("%s\n\n", strings.Join(missingAur, ", "))

				fmt.Print("Deseja instalar esses pacotes do AUR agora? (s/N): ")
				ans, _ := reader.ReadString('\n')
				ans = strings.ToLower(strings.TrimSpace(ans))
				if ans == "s" || ans == "sim" || ans == "y" || ans == "yes" {
					if err := packages.InstallPackages(missingAur, true); err != nil {
						fmt.Println(styleError.Render("❌ Falha na instalação de pacotes do AUR:"), err)
					} else {
						fmt.Println(styleSuccess.Render("✓ Pacotes do AUR instalados com sucesso!"))
					}
				}
			} else {
				fmt.Println(styleSuccess.Render("✓ Todos os pacotes do AUR já estão instalados."))
			}

			fmt.Println(styleSuccess.Render("\n✓ Processo de restauração concluído!"))
		},
	}
}

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Exibe o status do repositório de dotfiles e pacotes",
		Run: func(cmd *cobra.Command, args []string) {
			localCfg, localCfgPath, err := config.LoadLocalConfig()
			if err != nil {
				fmt.Println(styleError.Render("❌ Erro de configuração local:"), err)
				return
			}

			manifest, manifestPath, err := config.LoadManifest(localCfg.DotfilesDir)
			if err != nil {
				fmt.Println(styleError.Render("❌ Erro de manifesto:"), err)
				return
			}

			resolvedRepoDir, _ := config.ResolvePath(localCfg.DotfilesDir)

			cardContent := fmt.Sprintf(
				"%s\nLocal Config:  %s\nManifest File: %s\nRepository:    %s\n\n",
				styleTitle.Render(" ARCHKEEPER STATUS "),
				styleSuccess.Render(localCfgPath),
				styleSuccess.Render(manifestPath),
				styleSuccess.Render(resolvedRepoDir),
			)

			cardContent += fmt.Sprintf(
				"%s %d arquivos\n",
				styleHeader.Render("Arquivos Rastreados:"),
				len(manifest.Files),
			)
			for _, f := range manifest.Files {
				cardContent += fmt.Sprintf("  • ~/%s -> %s\n", f.Source, f.Target)
			}

			cardContent += fmt.Sprintf(
				"\n%s %v\n",
				styleHeader.Render("Backup de Pacotes:"),
				map[bool]string{true: styleSuccess.Render("Ativo"), false: styleWarn.Render("Inativo")}[manifest.Packages.BackupEnabled],
			)
			if manifest.Packages.BackupEnabled {
				cardContent += fmt.Sprintf("  • Pacman File: %s\n", manifest.Packages.PacmanFile)
				cardContent += fmt.Sprintf("  • AUR File:    %s\n", manifest.Packages.AurFile)

				missingNative, missingAur, err := packages.GetMissingPackages(localCfg, manifest)
				if err == nil {
					cardContent += fmt.Sprintf(
						"  • Status Local: %s nativos ausentes, %s AUR ausentes\n",
						styleColorCount(len(missingNative)),
						styleColorCount(len(missingAur)),
					)
				}
			}

			gitRepo := git.IsGitRepo(resolvedRepoDir)
			cardContent += fmt.Sprintf(
				"\n%s %v\n",
				styleHeader.Render("Git Repository:"),
				map[bool]string{true: styleSuccess.Render("Sim"), false: styleWarn.Render("Não")}[gitRepo],
			)

			fmt.Println(styleCard.Render(cardContent))
		},
	}
}

func styleColorCount(count int) string {
	if count == 0 {
		return styleSuccess.Render("0")
	}
	return styleWarn.Render(fmt.Sprintf("%d", count))
}
