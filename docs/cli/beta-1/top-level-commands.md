# Top-Level Docker CLI Commands for Beta 1

In order for the new CLI to feel familiar to existing Docker users while we are
developing it, we need to ensure that we communicate which existing commands
are not yet implemented and which have been removed.

There are five possible states for existing commands, flags, or environment
variables:

| State               | Description |
|:--------------------|:------------|
| Implemented         | Shown in help, implemented but may not include all subcommands or flags |
| Hidden              | Not shown in help, implemented |
| Not yet implemented | Shown in help, command returns "not yet implemented" error |
| Removed             | Not shown in help, command returns "please use a legacy context for this command" error |
| Ignored             | Not shown in help, setting has no effect |

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
| `service`   | Removed |
| `stack`     | Removed |
| `swarm`     | Removed |
| `system`    | Removed |
| `trust`     | Not yet implemented |
| `volume`    | Not yet implemented |

## Existing restructuring commands

These are commands that do not have different functionality but just restructure
the help. e.g.: `docker run` (command) vs `docker container run` (restructured
command).

| Command     | Behavior            | Comment |
|:------------|:--------------------|:--------|
| `container` | Hidden              | Allow existing scripts to work |
| `image`     | Not yet implemented | |

## Existing commands

| Command   | Behavior |
|:----------|:---------|
| `attach`  | Not yet implemented |
| `build`   | Not yet implemented |
| `commit`  | Removed |
| `cp`      | Removed |
| `create`  | Removed |
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
| `stop`    | Removed |
| `tag`     | Not yet implemented |
| `top`     | Not yet implemented |
| `unpause` | Removed |
| `update`  | Removed |
| `version` | Implemented |
| `wait`    | Removed |

## Existing CLI plugins

| Command  | Behavior | Comment |
|:---------|:---------|:--------|
| `app`    | Removed  | |
| `buildx` | Removed  | New `build` command will be `buildx` based |

## Existing flags

| Flag              | Behavior    | Comment |
|:------------------|:------------|:--------|
| `--config`        | Implemented | |
| `-c, --context`   | Implemented | |
| `-D, --debug`     | Implemented | |
| `-H, --host`      | Removed     | Replaced by contexts |
| `-l, --log-level` | Removed     | Replaced by `--debug` |
| `--tls`           | Removed     | Replaced by contexts |
| `--tlscacert`     | Removed     | Replaced by contexts |
| `--tlscert`       | Removed     | Replaced by contexts |
| `--tlskey`        | Removed     | Replaced by contexts |
| `--tlsverify`     | Removed     | Replaced by contexts |
| `-v, --version`   | Implemented | |

## New commands

| Command   | Behavior |
|:----------|:---------|
| `compose` | Implemented |
| `help`    | Implemented |
| `serve`   | Implemented |

## New flags

| Flag         | Behavior    | Comment |
|:-------------|:------------|:--------|
| `-h, --help` | Implemented | |
| `--verbose`  | Implemented | Same as `--debug` |

## Environment variables

| Variable                       | Behavior    |
|:-------------------------------|:------------|
| `DOCKER_API_VERSION`           | Ignored |
| `DOCKER_BUILDKIT`              | Ignored |
| `DOCKER_CONFIG`                | Implemented |
| `DOCKER_CERT_PATH`             | Ignored |
| `DOCKER_CLI_EXPERIMENTAL`      | Ignored |
| `DOCKER_DRIVER`                | Ignored |
| `DOCKER_HOST`                  | Ignored |
| `DOCKER_NOWARN_KERNEL_VERSION` | Ignored |
| `DOCKER_RAMDISK`               | Ignored |
| `DOCKER_STACK_ORCHESTRATOR`    | Ignored |
| `DOCKER_TLS`                   | Ignored |
| `DOCKER_TLS_VERIFY`            | Ignored |
| `DOCKER_CONTENT_TRUST`         | Ignored |
| `DOCKER_CONTENT_TRUST_SERVER`  | Ignored |
| `DOCKER_HIDE_LEGACY_COMMANDS`  | Ignored |
| `DOCKER_TMPDIR`                | Ignored |
| `DOCKER_CONTEXT`               | Implemented |
| `DOCKER_DEFAULT_PLATFORM`      | Ignored |
