# SNet

[![Go Version](https://img.shields.io/github/go-mod/go-version/Stuko0/SNet?style=flat-square&logo=go)](https://go.dev/)
[![License](https://img.shields.io/github/license/Stuko0/SNet?style=flat-square)](LICENSE)
[![Build Status](https://img.shields.io/github/actions/workflow/status/Stuko0/SNet/ci.yml?branch=main&style=flat-square&logo=github)](https://github.com/Stuko0/SNet/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/Stuko0/SNet?style=flat-square)](https://goreportcard.com/report/github.com/Stuko0/SNet)
[![Release](https://img.shields.io/github/v/release/Stuko0/SNet?include_prereleases&style=flat-square&logo=git)](https://github.com/Stuko0/SNet/releases)

**SNet** — A beautiful TUI for managing NetworkManager connections, written in Go.

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lipgloss](https://github.com/charmbracelet/lipgloss), SNet lets you scan Wi-Fi networks, manage saved connections, configure VPNs, and control hotspots — all from your terminal.

## Features

- 📡 **Dashboard** — Real-time network status: connectivity, IP, gateway, DNS, signal strength
- 📶 **Wi-Fi Manager** — Scan networks, see signal strength & security, connect with password
- 💾 **Saved Connections** — View, edit, and delete known networks
- 🔒 **VPN Manager** — List, add, connect/disconnect VPNs (OpenVPN, WireGuard, L2TP, SSTP)
- 🔥 **Hotspot** — Create and control Wi-Fi hotspots with custom SSID & password
- 🎨 **Beautiful TUI** — Tabbed navigation, color-coded status, responsive layout

## Installation

### Prerequisites

- **NetworkManager** (`nmcli`) — usually pre-installed on Fedora, Ubuntu, Arch
- Go 1.26+ (to build from source)

### From Source

```bash
git clone https://github.com/Stuko0/SNet.git
cd SNet
go build -o snet ./cmd/snet/
sudo cp snet /usr/local/bin/
```

### From Releases

Download the latest binary from the [Releases page](https://github.com/Stuko0/SNet/releases).

## Usage

```bash
snet
```

### Keybindings

| Key              | Action                |
|------------------|-----------------------|
| `Tab`/`Shift+Tab`| Navigate tabs         |
| `↑`/`↓` or `k`/`j`| Navigate lists      |
| `Enter`          | Select / connect      |
| `r`              | Refresh / rescan      |
| `e`              | Edit connection       |
| `d`              | Delete connection     |
| `Ctrl+n`         | New (network/VPN/hotspot) |
| `?`              | Help                  |
| `Ctrl+q` or `q`  | Quit                  |

## Project Structure

```
SNet/
├── cmd/
│   └── snet/
│       └── main.go              # Application entry point
├── internal/
│   ├── config/                  # Persistent configuration (TOML)
│   ├── network/                 # NetworkManager interaction layer
│   │   ├── types.go             # Domain types + Client interface
│   │   ├── nmcli.go             # nmcli command execution & parsing
│   │   ├── wifi.go              # Wi-Fi scan & connect
│   │   ├── connection.go        # Connection CRUD
│   │   ├── vpn.go               # VPN management
│   │   └── hotspot.go           # Hotspot create/control
│   └── tui/                     # Terminal UI layer
│       ├── app.go               # Main Bubble Tea model
│       ├── keys.go              # Keybinding definitions
│       ├── theme/
│       │   └── theme.go         # Colors & Lipgloss styles
│       └── views/
│           ├── dashboard.go     # Status overview
│           ├── wifilist.go      # Wi-Fi scanning & connection
│           ├── saved.go         # Saved connections
│           ├── vpnlist.go       # VPN management
│           ├── hotspot.go       # Hotspot control
│           └── editor.go        # Generic connection editor
├── .github/
│   ├── dependabot.yml
│   ├── CODEOWNERS
│   ├── ISSUE_TEMPLATE/
│   └── workflows/
│       └── ci.yml
├── LICENSE                      # Apache 2.0
├── CONTRIBUTING.md
├── SECURITY.md
└── README.md
```

## Development

```bash
# Run tests
go test ./...

# Build
go build -o snet ./cmd/snet/

# Install locally
go install ./cmd/snet/
```

## License

Apache 2.0 — see [LICENSE](LICENSE).

---

Built with ❤️ by [Stuko0](https://github.com/Stuko0)
