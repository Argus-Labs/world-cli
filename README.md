<div align="center"> <!-- markdownlint-disable-line first-line-heading -->
<img alt="World CLI Logo" src="https://i.imgur.com/XM74ODi.png" width="378">
<p>A swiss army knife for creating, managing, and deploying World Engine projects</p>
  <p>
    <a href="https://codecov.io/gh/Argus-Labs/world-cli" >
    <img alt="Codecov" src="https://codecov.io/gh/Argus-Labs/world-cli/branch/main/graph/badge.svg?token=XMH4P082HZ"/>
    </a>
    <a href="https://goreportcard.com/report/pkg.world.dev/world-cli">
    <img alt="Go Report Card" src="https://goreportcard.com/badge/pkg.world.dev/world-cli">
    </a>
    <a href="https://t.me/worldengine_dev" target="_blank">
    <img alt="Telegram Chat" src="https://img.shields.io/endpoint?color=neon&logo=telegram&label=chat&url=https%3A%2F%2Ftg.sumanjay.workers.dev%2Fworldengine_dev">
    </a>
    <a href="https://x.com/WorldEngineGG" target="_blank">
    <img alt="Twitter Follow" src="https://img.shields.io/twitter/follow/WorldEngineGG">
    </a>
  </p>
</div>

## Overview

Key features:

- **Create** — Initialize a new World Engine project based on [starter-game-template](https://github.com/Argus-Labs/starter-game-template)
- **Dev Mode** — Run your game shard in dev mode (with editor support) for fast iteration
- **[Soon] Deploy** — Get a prod-ready World Engine deployment in the cloud easier than deploying a smart contract.

**Need help getting started with World Engine?** Check out the [World Engine docs](https://world.dev)!

<br/>

## Installation

Before installing World CLI, you'll need to have Go installed on your system.
If you haven't installed Go yet, follow the official [Go installation guide](https://go.dev/doc/install) to get started.

### World CLI Installation

**Install latest release:**

```shell
go install pkg.world.dev/world-cli/cmd/world@latest
```

**Install a specific release:**

```shell
go install pkg.world.dev/world-cli/cmd/world@<tag>
```

<br/>

## Development

This section is for devel developers who want to contribute to the World CLI.
If you want to develop a World Engine project using World CLI, see the
[World Engine quickstart guide](https://world.dev/quickstart)

**Building from source:**

```shell
make build
```

**Testing your local build:**

You can test your local build of World CLI by running the following command.
This will install the World CLI binary in your `/usr/local/bin` directory.

```shell
make install
```
