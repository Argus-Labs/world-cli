# World CLI (deprecated)

World CLI now lives in [argus-labs/world-engine](https://github.com/argus-labs/world-engine/tree/main/cli)
and ships as a Go tool inside the World Engine module, so a game's CLI is always the same version as
its World Engine. This repository no longer receives releases, and the `install.world.dev` install
scripts are deprecated. You need Go 1.27.1 or later.

Install the `world` command and create a project:

```sh
go install github.com/argus-labs/world-engine/cli/cmd/world@latest
world setup my-game
```

Inside a project, `world` runs the version the project pins. To add World CLI to an existing game at
its current World Engine version:

```sh
go mod edit -tool=github.com/argus-labs/world-engine/cli/cmd/world
go mod tidy
```

The binaries attached to past [releases](https://github.com/Argus-Labs/world-cli/releases) stay
available for projects pinned to them. The source in this repository predates World CLI v2 and is
not maintained.
