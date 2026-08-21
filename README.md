# Dinex

Dinex is a lightweight, self-hostable server management platform built with Go.

Manage your servers through a modern web panel and connect them to Dinex Agents running on your nodes.

## ✨ Features

- 🖥️ Modern web panel
- 🖥️ Dinex Agent for server nodes
- ⚡ Lightweight Go backend
- 🔌 Node management
- 🚀 Server start, stop, and restart actions
- 📡 Live server status
- 🖥️ Live console
- 📁 Server file management
- 🔐 Authentication
- 📱 Responsive UI
- 🐧 Linux support
- 📱 Termux/Android support
- 🧩 Multi-node architecture

📦 Installation

Linux

Install the latest Dinex binaries with:

curl -fsSL https://raw.githubusercontent.com/Ribco/dinex/main/linux.sh | bash

The installer automatically detects your CPU architecture and downloads the appropriate release binaries.

Installed files:

/opt/dinex/dinex-panel
/opt/dinex/dinex-agent

Termux

On Termux:

curl -fsSL https://raw.githubusercontent.com/Ribco/dinex/main/termux.sh | bash

The binaries are installed to:

~/.local/bin/

If necessary, add it to your PATH:

export PATH="$HOME/.local/bin:$PATH"

🚀 Running Dinex

Panel

/opt/dinex/dinex-panel

Agent

/opt/dinex/dinex-agent

The exact configuration required depends on how your Dinex deployment is set up.

📥 Releases

Prebuilt binaries are available from the GitHub Releases page.

Current release:

v0.1.0

Available architectures:

Linux AMD64
Linux ARM64

Release assets:

dinex-agent-linux-amd64
dinex-agent-linux-arm64
dinex-panel-linux-amd64
dinex-panel-linux-arm64

🏗️ Building From Source

Requirements

- Go 1.25+
- Git

Clone the repository:

git clone https://github.com/Ribco/dinex.git
cd dinex

Dinex uses a Go workspace containing the Panel and Agent modules.

Test everything:

go test ./agent/... ./panel/...

Build the Agent:

go build -o dinex-agent ./agent/cmd/dinex-agent

Build the Panel:

go build -o dinex-panel ./panel/cmd/dinex-panel

📁 Project Structure

dinex/
├── agent/
│   ├── cmd/
│   │   └── dinex-agent/
│   ├── internal/
│   │   ├── api/
│   │   ├── config/
│   │   └── manager/
│   └── go.mod
│
├── panel/
│   ├── cmd/
│   │   └── dinex-panel/
│   ├── internal/
│   │   ├── api/
│   │   ├── auth/
│   │   ├── database/
│   │   ├── nodes/
│   │   └── servers/
│   ├── web/
│   │   ├── static/
│   │   └── templates/
│   └── go.mod
│
├── linux.sh
├── termux.sh
├── go.work
└── README.md

🔐 Configuration

Configuration will be handled through environment variables and configuration files as the project develops.

Do not commit passwords, API tokens, private keys, database files, or other secrets to the repository.

🛠️ Development

Run tests:

go test ./agent/... ./panel/...

Build both components:

go build -o dinex-agent ./agent/cmd/dinex-agent
go build -o dinex-panel ./panel/cmd/dinex-panel

For development, you can run the Panel and Agent directly from the repository.

🤝 Contributing

Contributions, bug reports, feature requests, and improvements are welcome.

Before submitting a change:

1. Test your changes.
2. Make sure sensitive information is not committed.
3. Keep changes focused.
4. Make sure the project still builds successfully.

📜 License

See the repository for the current license information.

⭐ Support Dinex

If you find Dinex useful, consider giving the repository a ⭐ on GitHub.

GitHub: https://github.com/Ribco/dinex
