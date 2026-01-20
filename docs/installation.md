# Installation

## Homebrew (Recommended)

```bash
brew install tasnimzotder/tap/portman
```

## Manual Download

Download the latest binary from [GitHub Releases](https://github.com/tasnimzotder/portman/releases).

```bash
# Extract and move to PATH
tar -xzf portman_*.tar.gz
sudo mv portman /usr/local/bin/
```

## Build from Source

Requires Go 1.21+

```bash
git clone https://github.com/tasnimzotder/portman.git
cd portman
go build -o portman ./cmd/portman
sudo mv portman /usr/local/bin/
```

## Verify Installation

```bash
portman --version
```

## Platform Support

| Platform | Status |
|----------|--------|
| macOS (Apple Silicon) | Supported |
| macOS (Intel) | Supported |
| Linux | Coming soon |
| Windows | Not planned |
