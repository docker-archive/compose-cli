# Top-Level Docker CLI Commands for Beta 1

In order for the new CLI to feel familiar to existing Docker users while we are
developing it, we need to ensure that we communicate which existing commands
are not yet implemented and which have been removed.

There are four possible states for existing commands:

| State | Description |
|:------|:------------|
| Implemented         | Shown in help, implemented but may not include all subcommands or flags |
| Hidden              | Not shown in help, implemented |
| Not yet implemented | Shown in help, command returns "not yet implemented" error |
| Removed             | Not shown in help, command returns "please use a legacy context for this command" error |

## Existing management commands

| Command     | Behavior |
|:------------|:---------|
| `builder`   | Not yet implemented |
| `config`    | Not yet implemented |
| `context`   | Implemented |
| `manifest`  | Removed |
| `network`   | Removed |
| `node`      | Removed |
| `plugin`    | Removed |
| `secret`    | Not yet implemented |
| `service`   | Not yet implemented |
| `stack`     | Removed |
| `swarm`     | Removed |
| `system`    | Removed |
| `trust`     | Not yet implemented |
| `volume`    | Not yet implemented |

## Existing restructuring commands

| Command     | Behavior |
|:------------|:---------|
| `container` | Hidden |
| `image`     | Not yet implemented |

## Existing commands

| Command   | Behavior |
|:----------|:---------|
| `attach`  | Not yet implemented |
| `build`   | Not yet implemented |
| `commit`  | Removed |
| `cp`      | Removed |
| `create`  | Not yet implemented |
| `diff`    | Removed |
| `events`  | Removed |
| `exec`    | Implemented |
| `export`  | Removed |
| `history` | Removed |
| `images`  | Not yet implemented |
| `import`  | Removed |
| `info`    | Implemented |
| `inspect` | Implemented |
| `kill`    | Removed |
| `load`    | Not yet implemented |
| `login`   | Implemented |
| `logout`  | Implemented |
| `logs`    | Implemented |
| `pause`   | Removed |
| `port`    | Removed |
| `ps`      | Implemented |
| `pull`    | Not yet implemented |
| `push`    | Not yet implemented |
| `rename`  | Removed |
| `restart` | Not yet implemented |
| `rm`      | Implemented |
| `rmi`     | Not yet implemented |
| `run`     | Implemented |
| `save`    | Not yet implemented |
| `search`  | Not yet implemented |
| `start`   | Removed |
| `stats`   | Removed |
| `stop`    | Not yet implemented |
| `tag`     | Not yet implemented |
| `top`     | Not yet implemented |
| `unpause` | Removed |
| `update`  | Removed |
| `version` | Implemented |
| `wait`    | Removed |

## Existing CLI plugins

| Command  | Behavior |
|:---------|:---------|
| `app`    | Removed |
| `buildx` | Removed |

## Existing flags

| Flag              | Behavior |
|:------------------|:---------|
| `--config`        | Implemented |
| `-c, --context`   | Implemented |
| `-D, --debug`     | Implemented |
| `-H, --host`      | Removed |
| `-l, --log-level` | Removed |
| `--tls`           | Removed |
| `--tlscacert`     | Removed |
| `--tlscert`       | Removed |
| `--tlskey`        | Removed |
| `--tlsverify`     | Removed |
| `-v, --version`   | Implemented |

## New commands

| Command   | Behavior |
|:----------|:---------|
| `compose` | Implemented |
| `help`    | Implemented |
| `serve`   | Implemented |

## New flags

| Flag         | Behavior |
|:-------------|:---------|
| `-h, --help` | Implemented |
