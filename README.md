# Dinex

|| OPEN SOURCE, FEEL FREE TO EDIT FILES BUT REMAIN DINEX TEXTS FOR CREDITS. ||

Dinex is a lightweight, self-hostable server management platform built with Go.
## ✨ Features

- 🖥️ Modern web panel
- 🤖 Dinex Agent for server nodes
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
 ## 📦 Installation

## 📦 Installation

### Linux

    curl -fsSL https://raw.githubusercontent.com/Ribco/dinex/main/linux.sh | bash

### Termux

    curl -fsSL https://raw.githubusercontent.com/Ribco/dinex/main/termux.sh | bash
## Create an Admin Account
``./dinex-panel admin:create -username YOUR_USER -password 'YOUR_PASSWORD'``
## 📥 Releases

The latest prebuilt binaries are available on the GitHub Releases page.

### Current Release

**v0.1.0**

### Supported Architectures

- Linux AMD64
- Linux ARM64

### Release Assets

- `dinex-agent-linux-amd64`
- `dinex-agent-linux-arm64`
- `dinex-panel-linux-amd64`
- `dinex-panel-linux-arm64`
## 🏗️ Building From Source

### Requirements

- Go 1.25+
- Git

Clone the repository:

    git clone https://github.com/Ribco/dinex.git
    cd dinex

Run tests:

    go test ./agent/... ./panel/...

Build the Agent:

    go build -o dinex-agent ./agent/cmd/dinex-agent

Build the Panel:

    go build -o dinex-panel ./panel/cmd/dinex-panel
## 🔐 Security

Never commit sensitive information to the repository.

This includes:

- Passwords
- API tokens
- Authentication tokens
- Private keys
- Credentials
- Database files
- Other secrets
## 🤝 Contributing

Contributions, bug reports, feature requests, and improvements are welcome.

Before submitting a change:

1. Test your changes.
2. Make sure sensitive information is not committed.
3. Keep changes focused.
4. Make sure the project still builds successfully.
## ⭐ Support Dinex

If you find Dinex useful, consider giving the repository a ⭐ on GitHub.

**GitHub:** https://github.com/Ribco/dinex

## 🦖 Dinex Support/Discord
You may get support & give us a suggestion & chat with us at:
**Discord**: https://discord.gg/E3EyuzxX94
