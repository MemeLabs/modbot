# modbot

Chat moderation bot for [strims.gg](https://strims.gg).

### running

| Flag | Default | Purpose |
| --- | --- | --- |
| `-cookie` | | jwt used for chat auth and API access (required) |
| `-chat` | `wss://chat.strims.gg/ws` | chat websocket url |
| `-api` | `https://strims.gg/api` | backend api base url |
| `-log` | `/tmp/chatlog/chatlog.log` | chat log file, rotated on `SIGHUP` (see `modbot-rotate`) |
| `-commands` | `commands.json` | static `!command` store, written by `!addcommand` |
| `-attoken` | | angelthump admin token, needed for `!(un)drop` |
| `-omdbkey` | | OMDb api key, needed for `!imdb` |
| `-logonly` | `false` | reply to the log instead of chat, for debugging |
| `-version` | | print the built commit and exit |

See [monitoring](#monitoring) for `-metrics`, `-pinginterval` and `-healthcheck`.

`SIGINT`/`SIGTERM` shut down cleanly; `SIGHUP` reopens the log file.

### development

Requires Go (version pinned in `go.mod`) and [`just`](https://github.com/casey/just).

```
just          # list recipes
just check    # fmt, vet, test -race, lint, tidy -- what CI runs
just build
just image    # build the container image
```

### mod commands

| Command | Arguments | Example | Extra |
| --- | --- | --- | ---- |
| !modify | {service/username, username} [nsfw\|hidden\|afk\|promoted]... | !modify youtube/6n3pFFPSlW4 hidden !nsfw | To invert options (remove modifier), prefix with "!".
| !rename | oldUsername newUsername | !rename ihatememes ilovememes | User has to reconnect after. Alternatively ban for 1 second.
| !addcommand | [!]commandname output | !addcommand test i like tests | Overwrites an existing command of the same name.
| !delcommand | [!]commandname | !delcommand test | Removes the given command.
| !say | string | !say something nice | Always replies in public chat, even when issued over PM.
| !mute | username | | Limited functionality, default 10m duration.
| !nuke | string | !nuke badword123 | default 10m duration.
| !nukeregex | regexp | !nukeregex (MiyanoHype ){10,} | default 10m duration.
| !aegis | _ | | undo all past nukes.
| !(un)drop | AT_name | !undrop test | Ban or unban user from angelthump service.

### public commands

| Command | Arguments | Example | Extra |
| --- | --- | --- | ---- |
| !stream\|!strim | _ | | Prints lists of top streams.
| !check | AT_name | !check test | Check status of an AT stream.
| !imdb | [-tv\|-s] title [year] | !imdb dune 2021 | Looks up a movie by default; `-tv` looks up a series instead; `-s` searches and lists up to 5 matches. Requires `-omdbkey` to be set.

### monitoring

`/metrics` (prometheus) and `/healthz` are served on `-metrics` (default `:9090`).
The container publishes no ports; it is scraped by name over the `strims` docker
network.

| Flag | Default | Meaning |
| --- | --- | --- |
| `-metrics` | `:9090` | listen address for `/metrics` and `/healthz` |
| `-pinginterval` | `30s` | keepalive PING interval, which drives `modbot_connected` |
| `-healthcheck` | `false` | probe a running instance and exit; used by the container `HEALTHCHECK` |

Health is the websocket being up, not chat being busy -- an idle chat would
otherwise look like a dead socket and get itself restarted. `SendPing` fails when
the socket is gone, so a ping write landing is the signal; the PONG is never read.

Series: `modbot_connected`, `modbot_last_message_received_timestamp_seconds`,
`modbot_ws_reconnects_total`, `modbot_command_persist_failures_total`, plus the
usual `go_*` and `process_*`. Nothing is labelled by nick -- unbounded
cardinality, and it would make the metrics store a record of user behaviour.

The image is `FROM scratch`, so `HEALTHCHECK` runs the binary's own
`-healthcheck` mode. Change it alongside `-metrics` if you move the port.

All mod-commands can also be issued via PMs to Bot. E.g. `/w Bot !modify youtube/6n3pFFPSlW4 hidden !nsfw`. Replies come back on the channel the command arrived on, so a command sent over PM is answered over PM. `!say` is the exception and always speaks in public chat.
