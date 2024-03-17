<div align="center">
<h2>World CLI</h2>
<p>A swiss army knife for creating, managing, and deploying World Engine projects</p>
  <p>
    <a href="https://codecov.io/gh/Argus-Labs/world-cli" >
    <img alt="Codecov" src="https://codecov.io/gh/Argus-Labs/world-cli/branch/main/graph/badge.svg?token=XMH4P082HZ"/>
    </a>
    <a href="https://goreportcard.com/report/pkg.world.dev/world-cli">
    <img src="https://goreportcard.com/badge/pkg.world.dev/world-cli" alt="Go Report Card">
    </a>
    <a href="https://t.me/worldengine_dev" target="_blank">
    <img alt="Telegram Chat" src="https://img.shields.io/endpoint?color=neon&logo=telegram&label=chat&url=https%3A%2F%2Ftg.sumanjay.workers.dev%2Fworldengine_dev">
    </a>
    <a href="https://x.com/WorldEngineGG" target="_blank">
    <img alt="Twitter Follow" src="https://img.shields.io/twitter/follow/WorldEngineGG">
    </a>
  </p>
</div>

- **Create** — Initialize a new World Engine project based on [starter-game-template](https://github.com/Argus-Labs/starter-game-template)
- **Dev Mode** — Run your game shard in dev mode (with editor support) for fast iteration
- **[Soon] Deploy** — Get a prod-ready World Engine deployment in the cloud easier than deploying a smart contract.

**Need help getting started with World Engine?** Check out the [World Engine docs](https://world.dev)!

## Installation

World CLI has been rigorously tested on macOS and Linux.
If you are using Windows, you will need 
[WSL](https://docs.microsoft.com/en-us/windows/wsl/install-win10) to install and use the CLI.

**Install latest release**
```
curl https://install.world.dev/cli! | bash
```

**Install a specific release**
```
curl https://install.world.dev/cli@<release_tag>! | bash
```

## Development

This section is for devel developers who want to contribute to the World CLI.
If you want to develop a World Engine project using World CLI, see the
[World Engine quickstart guide](https://world.dev/quickstart)

**Building from source**

```
make build
```

**Testing your local build**

You can test your local build of World CLI by running the following command. 
This will install the World CLI binary in your `/usr/local/bin` directory.

```
make install
```