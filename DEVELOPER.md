# World CLI Developer Guide

This guide provides information for developers who want to contribute to the World CLI project.

## Prerequisites

- [Go 1.24.0](https://go.dev/doc/install) or later
- [Task](https://taskfile.dev/installation/) - Task runner used for build, test, and lint commands
- [Docker](https://docs.docker.com/get-docker/) - Required for running services managed by World CLI
- [Docker Compose](https://docs.docker.com/compose/install/) - Required for multi-container Docker applications
- **Windows Users:** Windows Subsystem for Linux 2 (WSL2) is required for running World CLI on Windows

## Getting Started

### Clone the Repository

```bash
git clone https://github.com/Argus-Labs/world-cli.git
cd world-cli
```

### Setup Development Environment

The World CLI uses Task as a task runner for common development tasks. You can install Task by following the instructions at [taskfile.dev](https://taskfile.dev/installation/).

```bash
# Install task dependencies (golangci-lint, gotestsum, goreleaser)
task lint # This will install golangci-lint if it's not already installed
task test # This will install gotestsum if it's not already installed
task build # This will install goreleaser if it's not already installed
```

## Development Workflow

### Project Structure

The World CLI is organized around several core systems:

- `cmd/world/` - Contains the main CLI command implementations
  - `main.go` - Entry point for the CLI application
  - `root/` - Root command and core commands like `create` and `doctor`
  - `forge/` - World Forge platform commands
  - `cardinal/` - Cardinal game shard management commands
  - `evm/` - EVM-related commands
- `common/` - Shared utility code used across the application
  - `config/` - Configuration loading and management
  - `docker/` - Docker client and service definitions
  - `globalconfig/` - Global configuration persistence
  - `logger/` - Logging utilities
  - `teacmd/` - Terminal UI command utilities
- `tea/` - Terminal UI components using Bubble Tea framework
  - `component/` - Reusable UI components
  - `style/` - Terminal styling utilities
- `telemetry/` - Telemetry integration for error tracking and analytics
- `taskfiles/` - Task definitions for building, testing, and linting
- `example-world.toml` - Example configuration file

### Building the CLI

To build the CLI from source:

```bash
task build
```

This will create the binary in the `./dist` directory.

### Installing for Local Development

To install your local build for testing:

```bash
task install
```

This will install the World CLI binary in your Go bin directory (typically `$GOPATH/bin` or `$HOME/go/bin`).

## Configuration

The World CLI uses a TOML configuration file to manage settings for various services. An example configuration file is provided in the repository: `example-world.toml`.

Key sections in the configuration file include:

- `[cardinal]` - Settings for the Cardinal game shard
- `[evm]` - Settings for the Ethereum Virtual Machine
- `[common]` - Common settings shared across components
- `[nakama]` - Settings for the Nakama game server

Create a `world.toml` file in your project directory based on the example:

```bash
cp example-world.toml world.toml
```

Then customize the settings as needed for your development environment.

## Testing

### Running Tests

To run all tests:

```bash
task test
```

### Running Tests with Coverage

To run tests with coverage reporting:

```bash
task test:coverage
```

This will generate a coverage report and output it to a file.

## Linting

### Running Linter

To run the linter:

```bash
task lint
```

### Fixing Linting Issues

To run the linter and automatically fix issues where possible:

```bash
task lint:fix
```

## Pull Request Process

1. Create a feature branch from the `main` branch
2. Make your changes
3. Ensure tests pass with `task test`
4. Ensure linting passes with `task lint`
5. Push your changes and create a pull request
6. PR titles must follow the [conventional commit](https://www.conventionalcommits.org/) format
7. All tests and linting checks must pass with adequate coverage

## Additional Tools

### TUI Development

The World CLI uses the following libraries for terminal user interface development:

- [bubbletea](https://github.com/charmbracelet/bubbletea) - A framework for building terminal apps
- [lipgloss](https://github.com/charmbracelet/lipgloss) - Style definitions for terminal applications

### Docker Services

The CLI manages several Docker services:

1. **Cardinal**: Core game shard service
2. **Cardinal Editor**: Development tool for Cardinal
3. **Nakama**: Game server for multiplayer functionality
4. **NakamaDB**: Database for Nakama
5. **Redis**: In-memory data store
6. **EVM**: Ethereum Virtual Machine chain
7. **Celestia DevNet**: Data Availability layer for EVM
8. **Jaeger**: Distributed tracing (optional)
9. **Prometheus**: Metrics collection (optional)

## Troubleshooting

### Doctor Command

The World CLI includes a `doctor` command to check your system for dependencies and configuration issues:

```bash
world doctor
```

This command checks for required tools and services, and provides guidance on fixing any issues found.
