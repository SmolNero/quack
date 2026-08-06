<p align="center">
  <img src="assets/quack_logo_v2.png" alt="quack_logo" width="770" style="max-width: 100%; height: auto;" />
</p>

# quack

`quack` is a macOS terminal UI for identifying active OpenCode sessions, comparing their resource use, and terminating the right process.

## Features

- Shows each process's real OpenCode session title instead of guessing from its working directory.
- Uses Kitty's window title for resumed or idle sessions that have no recent OpenCode log event.
- Labels an idle OpenCode home screen as `No session selected`.
- Displays live resident memory (RSS), CPU, process age, and cumulative token-cache use.
- Sorts sessions by RAM use, highest first, to make cleanup decisions quick.
- Shows the current weekly Codex allowance, remaining percentage, and reset time.
- Refreshes process data every 4 seconds and weekly usage every minute.
- Refreshes everything immediately when `r` is pressed.
- Confirms the session title and PID before terminating a process.

`Cache` is the session's cumulative cached-token count, not reclaimable system memory. Use `RAM (RSS)` to judge how much memory terminating a process is likely to release.

## Install

### Go

```bash
go install github.com/SmolNero/quack/cmd/quack@latest
```

### Source

```bash
git clone https://github.com/SmolNero/quack.git
cd quack
go build -o quack ./cmd/quack
install -m 755 quack /usr/local/bin/quack
```

## Usage

```bash
quack
```

- `j` / `k` or arrow keys: move selection
- `r`: refresh sessions and weekly usage
- `c`: cancel selected session
- `q`: quit

Rows are ordered by resident memory. Selecting a row shows its full session ID, directory, token totals, cache reads/writes, update time, and command.

## Weekly Usage

Quack reads the existing OpenCode OpenAI OAuth credential and requests the current Codex balance directly from ChatGPT. It does not print or copy the access token. The reset time is shown in your local timezone.

Log in through OpenCode if the usage card reports that authentication is unavailable:

```bash
opencode providers login
```

Session monitoring continues if the usage request is offline or unavailable. Quack keeps the last successful value visible and marks a failed update.

## Kitty Session Titles

OpenCode does not log a session ID when an existing session is resumed without sending a new prompt. In Kitty, Quack resolves those processes through the title already shown by the terminal. Grant only the read-only `ls` action in `kitty.conf`:

```text
allow_remote_control password
remote_control_password "" ls
```

Reload Kitty's configuration with `ctrl+shift+f5` after changing it. Quack performs this title lookup before entering its terminal UI; no window-control or input actions are granted.

## Requirements

- macOS with `ps`
- Go 1.24+
- OpenCode 1.17.18+ available as `opencode` in `PATH`
- Optional: an OpenAI OAuth login in OpenCode for the weekly usage card

## Development

```bash
go test ./...
go vet ./...
go build ./cmd/quack
```

## License

Smol Nero License.

---
Smol Nero product.
