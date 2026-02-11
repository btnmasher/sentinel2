# Windows Development

Windows development is supported via **WSL2 (Ubuntu)**. The Taskfile and tooling are most reliable in a Linux shell.

## Recommended Approach
- Use WSL2 + Ubuntu for all project commands.
- Keep the repo in the Linux filesystem (for example `~/Code/sentinel2`) for better file-watch performance.

## 1) Install WSL2 + Ubuntu
Run in PowerShell as Administrator:

```powershell
wsl --install -d Ubuntu
```

Reboot if prompted, then open Ubuntu and complete first-time user setup.

## 2) Install Base Packages (inside WSL)

```bash
sudo apt-get update
sudo apt-get install -y build-essential zip unzip curl git
```

## 3) Install Go (inside WSL)

```bash
VERSION=1.25.1
cd /tmp
curl -LO "https://go.dev/dl/go${VERSION}.linux-amd64.tar.gz"
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf "go${VERSION}.linux-amd64.tar.gz"
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
go version
```

## 4) Install Task (inside WSL)

```bash
go install github.com/go-task/task/v3/cmd/task@latest
echo 'export PATH=$PATH:$HOME/go/bin' >> ~/.bashrc
source ~/.bashrc
task --version
```

## 5) Install Bun (inside WSL)

```bash
curl -fsSL https://bun.com/install | bash
echo 'export BUN_INSTALL="$HOME/.bun"' >> ~/.bashrc
echo 'export PATH="$BUN_INSTALL/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
bun --version
```

## 6) Project Setup

```bash
task setup
task dev
```

## Shell Completion (Optional)

```bash
task completion:install
```

## Optional Tools
- `lnav`:
  - `sudo apt-get install lnav`
- `golangci-lint`:
  - `curl -sSfL https://golangci-lint.run/install.sh | sh -s v2.9.0`
  - or `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest`

## Native Windows Notes
- Native Windows shells do not normally include an Info-ZIP compatible `zip` command.
- Release builds are simplest in WSL or CI.
