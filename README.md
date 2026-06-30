# 🛡️ archkeeper

**archkeeper** é um utilitário CLI moderno, rápido e elegante escrito em Go para gerenciar dotfiles e pacotes instalados (`pacman` e `AUR`) no Arch Linux. 

O `archkeeper` automatiza o rastreamento dos seus arquivos de configuração, o backup dos seus pacotes instalados e a sincronização com o Git, permitindo replicar todo o seu ecossistema pessoal em qualquer outra máquina com facilidade.

---

## ✨ Recursos

- 🔗 **Symlink Facilitado**: Rastreie qualquer configuração (arquivo ou pasta) com um comando. O `archkeeper` move o arquivo para o repositório e cria um link simbólico (symlink) no lugar.
- 📦 **Backup do Pacman e AUR**: Exporta automaticamente a lista de pacotes nativos instalados de forma explícita (`pacman -Qqen`) e pacotes instalados pelo AUR (`pacman -Qqem`).
- 🤖 **Restauração Inteligente**: Recria links simbólicos que estejam faltando em uma nova máquina (com backup automático `.bak` se encontrar arquivos preexistentes conflitantes) e compara/instala os pacotes ausentes via `pacman` ou `yay`/`paru`.
- 🐙 **Sincronização Git**: Realiza `add`, `commit` automático com marcação de data/hora e executa `push` direto para o repositório remoto.
- 🎨 **Visual Moderno**: Interface construída usando Cobra e Lipgloss com estilo inspirado nas cores do Arch Linux.

---

## 🚀 Instalação

Como o projeto está escrito em Go, você pode compilá-lo localmente e colocá-lo no seu PATH:

```bash
# Compilar o binário
go build -o archkeeper ./cmd/archkeeper/main.go

# Mover para o PATH do sistema (exemplo)
sudo mv archkeeper /usr/local/bin/
```

Ou instalar usando o próprio Go:

```bash
go install ./cmd/archkeeper
```

---

## 🛠️ Como Funciona a Sincronização?

O `archkeeper` divide suas configurações em duas partes para viabilizar o uso em múltiplos computadores:

1. **Configuração Local (`~/.config/archkeeper/config.yaml`)**:
   Salva apenas onde o repositório de dotfiles está clonado no seu computador atual (ex: `~/dotfiles`). Cada máquina pode clonar o repositório em diretórios diferentes.
2. **Manifesto Compartilhado (`<dotfiles_dir>/archkeeper.yaml`)**:
   Fica salvo dentro da própria pasta de dotfiles e é enviado para o Git. Ele guarda a lista de todos os arquivos que devem ser criados links simbólicos e configurações de pacotes.

---

## 📂 Guia Rápido de Uso

### 1. Inicializando
Crie o repositório local e gere as configurações iniciais:
```bash
archkeeper init
```
*Ele perguntará onde você quer colocar a pasta de dotfiles (padrão: `~/dotfiles`) e inicializará o Git automaticamente se a pasta ainda não for um repositório Git.*

### 2. Adicionando Arquivos para Rastreamento
Adicione os arquivos ou diretórios de configurações que você deseja rastrear:
```bash
archkeeper add ~/.zshrc
archkeeper add ~/.config/i3
```
*O `archkeeper` moverá esses arquivos para dentro de seu diretório de dotfiles e criará links simbólicos apontando para lá de forma transparente.*

### 3. Visualizando o Status
Veja quais arquivos estão sendo rastreados, o estado dos pacotes locais e as informações do repositório Git:
```bash
archkeeper status
```

### 4. Fazendo Backup (Salvar e Subir para o Git)
Gere a lista atualizada de pacotes instalados, realize o commit das alterações locais de dotfiles e envie para o Git remoto:
```bash
archkeeper backup
```
*Se você configurar um git remote na sua pasta de dotfiles (`git remote add origin ...`), o `archkeeper backup` enviará automaticamente as alterações para a nuvem.*

### 5. Restaurando em uma nova máquina
Ao configurar um novo sistema Arch Linux:
1. Instale o `archkeeper` na nova máquina.
2. Clone seu repositório de dotfiles em qualquer pasta (ex: `~/dotfiles`).
3. Execute `archkeeper init` e aponte para essa pasta.
4. Execute o comando de restauração:
   ```bash
   archkeeper restore
   ```
*O `archkeeper` lerá o manifesto `archkeeper.yaml`, recriará todos os links simbólicos e perguntará de forma interativa se você deseja instalar os pacotes ausentes via `pacman` e o seu AUR helper instalado (`yay`/`paru`).*
