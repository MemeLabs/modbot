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
| !aegis | | | undo all past nukes.

### public commands

| Command | Arguments | Example | Extra |
| --- | --- | --- | ---- |
| !stream\|!strim | | | Prints lists of top streams.
| !check | AT_name | | Check status of an AT stream.

All mod-commands can also be issued via PMs to Bot. E.g. `/w Bot !modify youtube/6n3pFFPSlW4 hidden !nsfw`. Responses will be via normal chat though!
