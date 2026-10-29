# Developing CopyTyGo with Windows Security enabled (WSL2)

CopyTyGo v5 supports an optional **full Native Runtime inside WSL2** while keeping Windows Smart App Control enabled on the host. This is not an exception to Smart App Control: the Go compiler and application execute as Linux processes within WSL, not as Windows executables.

## Why this mode exists

If CodeIntegrity event 3077 reports that `C:\Program Files\Go\bin\go.exe` is blocked under policy `VerifiedAndReputableDesktop`, neither `go run`, `go build` nor a different temporary build location can fix that host-level denial. CopyTyGo's earlier Lite Runtime can run some routes inside an existing trusted CLI but does **not** support the full TypeScript browser authentication / Go application ecosystem.

The documented WSL mode preserves the complete Go backend, Vite frontend, sessions, workers and Studio as long as the required Linux toolchain is installed and the project is configured correctly.

## 1. Check WSL from PowerShell

```powershell
wsl --status
wsl --list --verbose
```

If WSL and a Linux distribution are not installed, follow [Microsoft's WSL installation guide](https://learn.microsoft.com/en-us/windows/wsl/install). A common Windows 11 installation command (administrator PowerShell) is:

```powershell
wsl --install -d Ubuntu
```

Complete any required restart and first-run account setup. Installation must be permitted by your machine's administrator.

## 2. Install tools INSIDE Ubuntu

Open an Ubuntu terminal using `wsl -d Ubuntu`, or select the WSL profile in VS Code.

You need:
- Git
- **Linux Go 1.27.1 or newer** to match CopyTyGo's go.mod
- **Linux Node.js 22 or newer** and npm for Vite/TypeScript

Check, without invoking Windows go.exe:

```bash
command -v git
command -v go
go version
command -v node
node --version
npm --version
```

Install Go from the official [Go downloads](https://go.dev/dl/) and [installation guide](https://go.dev/doc/install) for **linux-amd64** (or your Linux architecture). Verify the release's published checksum before installing. Install Node.js using [official Node downloads](https://nodejs.org/en/download) for Linux. Your Windows Go and Node installations are not substitutes for Linux toolchains.

## 3. Build the v5 candidate CLI entirely in WSL

The v5 branch is a release candidate and has no final public tag. Clone and build from source inside your Linux home directory:

```bash
mkdir -p ~/projects ~/go/bin
cd ~/projects
git clone --branch feature/v5-release-candidate https://github.com/arfajhf/copytygo.git
cd copytygo
go build -o ~/go/bin/ctg ./cmd/ctg
export PATH="$HOME/go/bin:$PATH"
ctg version
```

For a persistent PATH, configure your Linux shell startup file. This Linux `ctg` binary is distinct from `ctg.exe` on Windows.

## 4. Work on an application

For the stable v4 application, open or clone your application into `~/projects`. Then run:

```bash
cd ~/projects/your-app
ctg doctor
ctg dev
```

Replace `your-app` with your actual application directory. The complete **Native Runtime** starts inside Linux. With default WSL localhost forwarding, browse to the application's URL from Windows, typically `http://127.0.0.1:8080`. The Studio route is `/__copytygo` in local development.

The v5 candidate's project generator currently references an unreleased release-candidate version. For a **throwaway development project only**, add a local module replacement pointing to your checked-out framework source, then run `go mod tidy`. Never commit that local replacement or use it in a published release:

```bash
go mod edit -replace=github.com/arfajhf/copytygo/v5="$HOME/projects/copytygo"
go mod tidy
```

For best filesystem performance, keep Linux-built projects in `~/projects` rather than `/mnt/c`. VS Code can open WSL projects directly using its WSL extension and `code .`.

## 5. Optional convenience flag from Windows

If the **Windows `ctg.exe` CLI itself is allowed** to run, CopyTyGo v5 also offers:

```powershell
ctg dev --wsl
```

This starts the existing application's project directory in the default WSL distribution, then executes the **Linux `ctg dev`**, not Windows `go.exe`. It requires the Linux CLI and toolchains already set up above. A Windows project path may work but has slower Linux file access. For reliability, prefer opening the project from the WSL home directory.

If Windows blocks `ctg.exe` itself, skip the convenience flag and run `ctg dev` **inside** WSL instead.

## Windows-native compatibility remains a separate release gate

WSL is a full Linux Native Runtime. It does not prove that unsigned Windows binaries will execute under Smart App Control. CopyTyGo's v5 release still requires independent Windows acceptance testing on a machine/policy that allows a trusted, appropriately signed Go toolchain and generated executables.

Do not change ownership or patch Windows CodeIntegrity policy files to make a developer tool run. Do not disable security protections globally just to build CopyTyGo.

## References

- [Microsoft: Smart App Control FAQ](https://support.microsoft.com/en-us/windows/security/threat-malware-protection/smart-app-control-frequently-asked-questions)
- [Microsoft: WSL setup and development](https://learn.microsoft.com/en-us/windows/wsl/setup/environment)
- [Microsoft: WSL networking and Windows/Linux file access](https://learn.microsoft.com/en-us/windows/wsl/interop)
