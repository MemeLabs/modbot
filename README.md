# modbot

### mod commands

| Command | Arguments | Example | Extra |
| --- | --- | --- | ---- |
| !modify | {service/username, username} [nsfw\|hidden\|afk\|promoted]... | !modify youtube/6n3pFFPSlW4 hidden !nsfw | To invert options (remove modifier), prefix with "!".
| !rename | oldUsername newUsername | !rename ihatememes ilovememes | User has to reconnect after. Alternatively ban for 1 second.
| !addcommand | [!]commandname [output\|\_] | !addcommand test i like tests | Using "\_" as output removes the given command.
| !say | string | !say something nice |
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
| !embed | link | !embed twitch.tv/admin | If using an unsupported service no response is sent.

All mod-commands can also be issued via PMs to Bot. E.g. `/w Bot !modify youtube/6n3pFFPSlW4 hidden !nsfw`. Responses will be via normal chat though!
