<div align="center"> <!-- markdownlint-disable-line first-line-heading -->
<img alt="World CLI Logo" src="https://i.imgur.com/XM74ODi.png" width="378">
<p>A swiss army knife for creating, managing, and running World Engine projects</p>
  <p>
    <a href="https://github.com/Argus-Labs/world-cli/releases/latest">
    <img alt="Latest release" src="https://img.shields.io/github/v/release/Argus-Labs/world-cli">
    </a>
    <a href="https://t.me/worldengine_dev" target="_blank">
    <img alt="Telegram Chat" src="https://img.shields.io/endpoint?color=neon&logo=telegram&label=chat&url=https%3A%2F%2Ftg.sumanjay.workers.dev%2Fworldengine_dev">
    </a>
    <a href="https://x.com/WorldEngineGG" target="_blank">
    <img alt="Twitter Follow" src="https://img.shields.io/twitter/follow/WorldEngineGG">
    </a>
  </p>
</div>

> **This repository distributes the World CLI. It does not hold its source.**
>
> The CLI is developed in the Argus Labs monorepo; every release here is published from there by CI.
> Please open bug reports and feature requests as [issues](https://github.com/Argus-Labs/world-cli/issues) —
> code pull requests against this repository cannot be merged.

## Install

### macOS and Linux

```shell
# Latest version
curl https://install.world.dev/install.sh | sh

# A specific version
curl https://install.world.dev/install.sh | sh -s v2.4.8
```

The binary lands in `~/.worldcli/bin/world`. The installer appends that directory to your `PATH`
in `~/.bashrc` or `~/.zshrc` — restart your shell (or `source` the file) afterwards. Set
`WORLD_INSTALL` to install somewhere else.

### Windows

```powershell
# Latest version
iwr https://install.world.dev/install.ps1 -useb | iex

# A specific version
iex "& { $(iwr https://install.world.dev/install.ps1 -useb) } v2.4.8"
```

### Manual download

Grab an archive from [Releases](https://github.com/Argus-Labs/world-cli/releases), unpack it, and
put the `world` binary on your `PATH`:

```shell
curl -L https://github.com/Argus-Labs/world-cli/releases/latest/download/world-cli_Linux_x86_64.tar.gz | tar -xz
sudo mv world /usr/local/bin/
```

Archives are published for Linux, macOS, and Windows on `x86_64` and `arm64`, alongside a
`_checksums.txt` file.

### Verify and update

```shell
world --version   # print the installed version
world doctor      # check Git, Go, Docker, and the Docker daemon
world update      # self-update to the latest release
```

## Requirements

| Dependency    | Why |
| ------------- | --- |
| Docker        | Cardinal runs in containers; the daemon must be running |
| Go            | Building your game shards |
| Git           | Fetching project templates |

`world doctor` checks all four and tells you what is missing.

## Quick start

```shell
world setup my-game   # scaffold a project from a template
cd my-game
world start           # bring up the Cardinal game environment
world logs            # tail shard and platform logs
world stop            # shut down, keeping state
```

## Commands

### Getting started

| Command | Description |
| ------- | ----------- |
| `world setup [DIR] [-t basic\|demo\|bare]` | Scaffold a new World Engine project. Skips the interactive template picker when `-t` is given |
| `world doctor` | Check your development environment |
| `world docs` | Open the World CLI documentation |
| `world update` | Self-update to the latest release |

### Cardinal

| Command | Description |
| ------- | ----------- |
| `world start [--no-debug]` | Launch your Cardinal game environment |
| `world stop` | Gracefully shut it down; state is preserved |
| `world reload [INSTANCES...] [--purge]` | Rebuild and roll shards in the running cluster. Defaults to every instance; `--purge` wipes instance state (NATS JetStream) first |
| `world purge [--image]` | Reset to a clean state by removing all data and containers. `--image` also prunes shard images and per-reload registry tags |
| `world logs [--env ENV] [--shard NAME] [--tail N] [--previous]` | View and tail logs. Without `--env` it reads the local cluster; with it, a deployed environment |

### Debugger

Tick-level control of a running game environment (aliased as `world db`):

| Command | Description |
| ------- | ----------- |
| `world debug pause` | Pause tick execution |
| `world debug resume` | Resume after a pause |
| `world debug step` | Execute a single tick (only while paused) |
| `world debug reset` | Reset the world to its state before tick 0 |

### SDK generation

`world sdk generate` produces typed, reflection-free wire serialization for a backend's commands,
events, components, and system events — Go written back into the backend's own packages, and C#
for the client.

```shell
world sdk generate ./my-backend \
  --go-out ./my-backend/gen \
  --cs-out ../my-client/Assets/Generated
```

| Flag | Description |
| ---- | ----------- |
| `--go-out DIR` | Where the generated Go protobuf package goes. Must be inside the backend's Go module. Requires a local source checkout |
| `--cs-out DIR` | Where the generated C# client SDK goes |
| `--ref REF` | Branch, tag, or commit to clone when the source is a remote URL or `owner/repo` |
| `--no-emit` | Run discovery and print the report, then stop. Writes nothing, needs no Docker, and exits non-zero if anything blocks |

Pass at least one of `--go-out` or `--cs-out`; passing both in a single run keeps backend and
client generated from the same source state. A remote source is read-only, so it can produce the
C# client SDK only. Generation needs Docker, same as `world start`.

Every run prints a report of anything it cannot serialize — unbounded slices, maps, pointers,
`any` fields, duplicate wire names — and refuses to emit until those are fixed.

### Global flags

| Flag | Description |
| ---- | ----------- |
| `--version` | Show the CLI version |
| `-v`, `--verbose` | Enable debug logs |
| `--help` | Show help; works on any subcommand, e.g. `world sdk generate --help` |

## MCP: connect an AI assistant

The CLI ships an [MCP](https://modelcontextprotocol.io) server that hands your AI assistant tools
to inspect, query, and drive a Cardinal environment. It runs over stdio as `world mcp` — your MCP
client launches it, you never start it yourself. Running `world mcp` in a terminal just prints the
setup hint.

### Claude Code

```shell
claude mcp add cardinal -- world mcp
```

Add `--scope project` to write it to the project's `.mcp.json` and share it with your team.

### Cursor, Windsurf, Claude Desktop

Add the server to your client's config file:

| Client | Config file |
| ------ | ----------- |
| Cursor | `.cursor/mcp.json` in the project, or `~/.cursor/mcp.json` |
| Windsurf | `~/.codeium/windsurf/mcp_config.json` |
| Claude Desktop | `~/Library/Application Support/Claude/claude_desktop_config.json` (macOS), `%APPDATA%\Claude\claude_desktop_config.json` (Windows) |

```json
{
  "mcpServers": {
    "cardinal": {
      "command": "world",
      "args": ["mcp"]
    }
  }
}
```

### VS Code

`.vscode/mcp.json` uses a different shape:

```json
{
  "servers": {
    "cardinal": {
      "type": "stdio",
      "command": "world",
      "args": ["mcp"]
    }
  }
}
```

### Tools

| Tool | What it does |
| ---- | ------------ |
| `ping` | Health check for the server itself |
| `cluster` | Bring the world up, shut it down, or wipe the environment |
| `describe_world` | List the worlds deployed on the local cluster |
| `get_world_start_status` | Shard pools and pod instances: size, image tag, phase, readiness, restarts |
| `inspect_shard` | Runtime state of one shard's pool and pods |
| `get_shard_logs` | Recent logs for a shard's pods |
| `get_nats_logs` | Recent logs from the NATS pod |
| `introspect` | A shard's registered components, commands, and events, with schemas |
| `get_state` | Query world state: tick height, paused flag, entities and component data, filtered by component set or an `expr-lang` predicate such as `Health.HP > 50` |
| `send_command` | Send a command to a shard to trigger game logic |
| `debug_control` | Pause, resume, step one tick, or reset to the pre-tick-0 state |
| `reload` | Rebuild shards from local source and roll them into the cluster |
| `sdk_generate` | Regenerate the wire layer from Go source — the MCP counterpart of `world sdk generate` |

Every tool except `ping` needs a running cluster — start one with `world start`.

### If the server will not connect

Clients launch with a minimal `PATH` and often cannot find `world`. Use an absolute path instead:

```json
{
  "mcpServers": {
    "cardinal": {
      "command": "/Users/you/.worldcli/bin/world",
      "args": ["mcp"]
    }
  }
}
```

Run `which world` to get yours.

## Troubleshooting

**`world: command not found`** — restart your shell, or add the install directory to your `PATH`:
`export PATH="$HOME/.worldcli/bin:$PATH"`.

**Docker errors** — make sure the daemon is running, then re-run `world doctor`.

**A game environment that will not start cleanly** — `world purge` removes all containers and data
so `world start` begins from scratch.

**`world update` cannot write** — the install directory is not writable by your user. Re-run with
elevated permissions, or reinstall with the install script.

## Help

- Documentation — [world.dev](https://world.dev)
- Community — [Telegram](https://t.me/worldengine_dev)
- Bugs and feature requests — [GitHub Issues](https://github.com/Argus-Labs/world-cli/issues)

## License

[LGPL-3.0](LICENSE)
