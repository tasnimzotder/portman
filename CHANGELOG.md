# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-10-03

### Added

- Interactive list, detail, find, kill, wait, and watch views using Bubble Tea v2.
- Port ranges, process grouping, shell completions, and development-port conflict inspection.
- CSV and TSV output with proper escaping and binding addresses.
- Regression tests covering actual TCP/UDP sockets, disposable process signaling, CLI results, and interactive model behavior.

### Changed

- Use machine-readable lsof output and preserve distinct bindings by protocol, address, port, and PID.
- Keep wait polling lightweight and cancel scans at the configured deadline.
- Exit interactive waits immediately when the condition is met or polling fails.
- Make `kill --force` try SIGTERM before escalating after `--timeout`; use `--signal KILL` for immediate termination.
- Require Go 1.25.6 or newer and derive the CI/release toolchain from go.mod.

### Fixed

- Detect bound UDP sockets, including connected UDP sockets.
- Distinguish scan failures from unused ports and reject ambiguous owners in single-port actions.
- Prevent wait --exec from running after timeout, cancellation, or failed scanning.
- Honor kill timeouts, verify ownership after confirmation, and propagate interactive failures.
- Preserve empty JSON results, validate flags, and use consistent inspection output formats.
- Recover watch mode after scan errors, serialize refreshes, and compare against the previous scan.
- Report real duplicate owners without treating process-name differences as conflicts.

## [0.0.1] - 2026-01-20

### Added

- Add PID and port commands for detailed process and port information

### Other

- Add initial documentation and setup files for portman project
- Create LICENSE
## [0.0.1-beta.4] - 2026-01-20

### Other

- Fix archive format definition and add post-install hook for Homebrew cask
## [0.0.1-beta.3] - 2026-01-20

### Other

- Add README.md with installation, usage, commands, and flags
## [0.0.1-beta.2] - 2026-01-20

### Other

- Update CI and Release workflows to use macOS for builds and enable tag-based releases
- Remove branch trigger from release workflow to focus on tag-based releases
## [0.0.1-beta.1] - 2026-01-20

### Other

- Initial chaos
- Add CI and Release workflows with Go setup and build configurations
- Refactor Homebrew configuration in goreleaser.yaml
