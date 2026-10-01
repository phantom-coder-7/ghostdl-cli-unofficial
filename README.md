# ghostdl-unofficial

Unofficial command-line client for [Ghost Downloader](https://github.com/XiaoYouChR/Ghost-Downloader-3). Pair it with the Desktop App, then list, create, and watch downloads from the terminal.

It talks to Ghost Downloader over the local **BrowserService** WebSocket API. It's not a downloader: it can't fetch URLs or save files on its own. If the app isn't running or paired, this CLI has nothing to control.

## Disclaimer

Most of this CLI was written by AI from short prompts, not from a spec with QA. Treat it as **unofficial** and **experimental**.

- It may hang, write somewhere unexpected, or fail in untested ways.
- CI tests on Linux and runs `make build-common` (five linux/darwin/windows targets). Android binaries are built only for tagged releases.
- Don't use it on data you can't afford to lose.
- No warranty, express or implied. You run it at your own risk.

## Install

```bash
# Method 1: Download precompiled binary
curl -sL https://github.com/phantom-coder-7/ghostdl-cli-unofficial/releases/latest/download/ghostdl-unofficial-linux-amd64 -o /usr/local/bin/ghostdl
chmod +x /usr/local/bin/ghostdl

# Method 2: Build from source (requires Go 1.26+)
git clone https://github.com/phantom-coder-7/ghostdl-cli-unofficial.git
cd ghostdl-unofficial
make build
sudo mv bin/ghostdl-unofficial /usr/local/bin/ghostdl
```

Verify a downloaded release file:

```bash
gh attestation verify <downloaded-file> -R <owner>/<repo>
```

Install it as `ghostdl` because `gd` collides with the common `git diff` shell alias (Oh My Zsh and many shell configs).

`make build` compiles for this machine; CI cross-compiles the five common OS/arch pairs (`make build-common`). When you ship a release, `make build-all` writes every supported target under `bin/ghostdl-unofficial-<os>-<arch>` (`.exe` on Windows).

**Heads-up:** Before you pair, Ghost Downloader must be running (tray is fine), and **Enable Browser Extension** must be on — Settings → Browser Extension → Enable Browser Extension.

## Pair with Ghost Downloader

Two equivalent paths:

**Ask Ghost Downloader** — run `ghostdl auth login` with no flags. If the stored token still works, the command prints already logged in and leaves the dialog closed. Otherwise a pairing dialog pops up; click Allow within 60 seconds. Fine when you're at the machine.

**Paste a pairing token** — copy it from Ghost Downloader → Settings → Browser Extension → Pairing Token, then:

```bash
ghostdl auth login --token <pairing-token>
```

This skips the dialog. Handy for scripts and headless machines.

The CLI stores one credential in the OS keychain when that service is available (macOS Keychain, Windows Credential Manager, Linux Secret Service). If the OS keychain isn't reachable — pretty common on minimal Linux — run `ghostdl auth login --allow-credential-file` to stash the pairing token in **`login-data`** (credential file in your config directory); login prints that path on stderr. That file's weaker than the OS keychain, FYI. On Unix it is mode **0600**. On Windows it is an owner-only DACL for the current user, not an OpenSSH host-key ACL. Do not copy host-key ACLs onto **`login-data`**. If the file can't be restricted, it's deleted and login fails. If the OS keychain is present but refuses access (locked, headless SSH), login errors and does not write the file.

You can pair with only one Ghost Downloader instance at a time. When the stored token still works, `ghostdl auth login` prints already logged in. `ghostdl auth login --force` opens the dialog again, and `ghostdl auth login --token` replaces the token. Logout clears the credential. If Ghost Downloader rejects the stored token, plain `ghostdl auth login` pairs again.

```bash
ghostdl auth check
```

## Common commands

```bash
ghostdl overview
ghostdl task list
ghostdl task list --status running
ghostdl task list --pack bt --sort size --order desc
ghostdl task create --url https://example.com/file.zip
ghostdl task progress tsk_abc123 --watch
ghostdl task pause tsk_abc123
ghostdl task resume tsk_abc123
```

### List

```bash
ghostdl task list
ghostdl task list --status running,paused
ghostdl task list --pack bt
ghostdl task list --ext zip
ghostdl task list --query report
ghostdl task list --limit 10 --offset 20
ghostdl task list --format json | jq '.data.tasks | length'
```

| Flag | Default | Description |
|------|---------|-------------|
| `--query` / `-q` | | Case-insensitive substring search in id, name, url, pack name, and file extension. AND with `--status`, `--pack`, and `--ext` when those flags are set. Prefer `--pack bt` over `--query bt` for BitTorrent (query also matches names containing "bt"). |
| `--status` | | Filter: `waiting`, `running`, `paused`, `completed`, `failed`. Repeat the flag or use comma-separated values for OR (for example `running,paused`). Unknown values are an error (not a silent empty list). |
| `--pack` | | Filter by pack name (exact match, case-insensitive). Repeat or comma-separate for OR. AND with `--status`, `--ext`, and `--query` when those flags are set. Use `bt` for BitTorrent. |
| `--ext` | | Filter by file extension (exact match, case-insensitive). Repeat or comma-separate for OR. Leading dot is optional (`zip` and `.zip` are equivalent). AND with `--status`, `--pack`, and `--query` when those flags are set. |
| `--sort` | | Sort field: `created`, `id`, `name`, `pack` (`packName`), `progress`, `received` (`receivedBytes` alias), `size`, `speed`, `status`. For `status`, order is waiting, running, paused, completed, then failed. |
| `--order` | | `asc` or `desc`. With `--sort` only, order defaults to `asc`. With `--order` only, sort defaults to `created`. Invalid sort field or order is an error. |
| `--limit` | `20` | Number of tasks per page (after filtering and sorting) |
| `--offset` | `0` | Pagination offset (after filtering and sorting) |

Without `--sort` and `--order`, the list keeps Ghost Downloader's order (newest first). Filter and sort run in the CLI on the full snapshot; `--limit` and `--offset` apply afterward. In JSON, `total` is the filtered count before the page slice. For `table` and `list` only, when the page is partial (`--offset` > 0 or filtered `total` exceeds the rows shown), the CLI prints a one-line notice on **stderr**; use `--limit 0` to show all matching tasks.

Ghost Downloader tracks BitTorrent seeding internally, but `task_snapshot` doesn't include seeding state, upload speed, or share ratio. The CLI can't filter or sort by those. Use `--pack bt` to list torrents (`packName` is `bt` on the wire).

> [!NOTE]
> Valid `--status` values are `waiting`, `running`, `paused`, `completed`, and `failed`
> (case-sensitive). Typos such as `downloading` or `Running` are rejected with an error —
> they do not return an empty list. Use `running` for in-progress downloads and `failed`
> for errors.

### Create

```bash
ghostdl task create --url https://example.com/file.zip
ghostdl task create --url https://example.com/file.zip --filename archive.zip --path ~/Downloads --threads 8
ghostdl task create --url https://example.com/file.zip --header "Referer: https://example.com/"
ghostdl task create --url https://example.com/file.zip --dry-run
```

This CLI doesn't support adding tasks in bulk yet.

| Flag | Default | Description |
|------|---------|-------------|
| `--url` | | Download URL (**required**) |
| `--filename` | | Target filename |
| `--path` | | Output folder |
| `--header` | | HTTP header in `key: value` form (repeatable) |
| `--threads` | `0` | Download threads (maps to `preBlockNum`) |
| `--title` | | Task title |
| `--source` | `resource` | Task source: `download` or `resource` only (`resource_merge` and `page_media` are rejected) |
| `--draft` | `false` | Create as a draft awaiting approval in Ghost Downloader |

> [!WARNING]
> **Breaking change:** `ghostdl task create` no longer previews by default. `--dry-run` used
> to default to `true`, so create only printed a preview. It now defaults to `false` and
> **creates the task**. Pass `--dry-run` if you want the old preview.

### Progress

```bash
ghostdl task get tsk_abc123
ghostdl task progress tsk_abc123
ghostdl task progress tsk_abc123 --watch
```

`--watch` follows Ghost Downloader's push stream (~1Hz). It exits 0 on Ctrl+C (interrupt) or when the task reaches `completed`; a failed task or a dead connection exits 1. In JSON, `ghostdl task progress` reports `progress` as a **0–1 fraction**; `task list` / `task get` keep **0–100** percent on `Task.progress`.

`ghostdl task get` and `ghostdl task progress` read the live `task_snapshot` stream.

### Pause, resume, delete, restart, open

```bash
ghostdl task pause tsk_abc123
ghostdl task resume tsk_abc123
ghostdl task restart tsk_abc123
ghostdl task delete tsk_abc123
ghostdl task delete tsk_abc123 --with-files
ghostdl task delete tsk_abc123 --with-files --yes
ghostdl task open tsk_abc123
ghostdl task open tsk_abc123 --folder
```

`pause` and `resume` check status first. They only send an action when the state needs to flip, and they refuse on completed or failed tasks (use `restart` there). Pausing an already-paused task, or resuming an already-running one, is a no-op. `waiting` can't be paused yet; resume on `waiting` is also a no-op (it starts on its own).

`ghostdl task delete` maps to Ghost Downloader's `remove` action (delete the task, keep the files). `--with-files` maps to `cancel` instead (delete the task and its files) and prompts for confirmation. `--yes` skips the prompt; it is required when stdin is not a terminal.

`ghostdl task open` asks Ghost Downloader to run `open_file` or `open_folder`. The app has to be running.

`--dry-run` is a global flag (default `false`). It prints the JSON that would go over the WebSocket and sends nothing. It works on `create`, `pause`, `resume`, `delete`, `restart`, and `open`. For `pause` and `resume` it still reads current status so it can say whether an action would be sent. `restart` and `open` print the wire message without connecting to BrowserService.

## Config

Default server URL is **`ws://127.0.0.1:14370`** — plaintext WebSocket on localhost, same as the official browser extension. Override it if you changed the port in the app:

```bash
ghostdl config set server.url ws://127.0.0.1:14370
ghostdl config unset server.url
```

For **`wss://`**, the CLI verifies TLS against your system CAs. To skip verification (self-signed or private CA), pass **`--insecure`** or set **`server.insecure`** to `true`. Neither applies to **`ws://`**.

```bash
ghostdl --insecure auth check
ghostdl config set server.insecure true
ghostdl config unset server.insecure
```

`config.yaml` lives under your OS user config directory, in a `ghostdl-unofficial` subdirectory. Linux: `$XDG_CONFIG_HOME/ghostdl-unofficial` if `XDG_CONFIG_HOME` is set, otherwise `~/.config/ghostdl-unofficial`. macOS: `~/Library/Application Support/ghostdl-unofficial`. Windows: `%AppData%\ghostdl-unofficial`. `ghostdl config list` prints the path this install uses. `GD_CLI_CONFIG_DIR` overrides the whole directory; a relative path is used as given.

If the OS config directory can't be determined, the CLI falls back under home: `AppData/Roaming` on Windows, `Library/Application Support` on macOS, or `.config` elsewhere, still ending in `ghostdl-unofficial`. If home can't be determined either, config commands error instead of writing to a relative `.config`. On Android, Go treats an empty `HOME` as `/sdcard`, which is where that fallback lands.

## Commands

| Command | Description |
|---------|-------------|
| `ghostdl auth login` | Pair with Ghost Downloader (interactive, `--force`, or `--token`) |
| `ghostdl auth logout` | Logout |
| `ghostdl auth status` | View login status |
| `ghostdl auth check` | Verify server reachability and token validity |
| `ghostdl overview` | Snapshot of counts, in-progress/failed/recent tasks, and download traffic |
| `ghostdl task create` | Create a download task |
| `ghostdl task list` | List download tasks |
| `ghostdl task get` | View task details |
| `ghostdl task progress` | Query download progress (`--watch` for continuous) |
| `ghostdl task pause` | Pause a running task (idempotent) |
| `ghostdl task resume` | Resume a paused task (idempotent) |
| `ghostdl task delete` | Delete a task and keep its files, or add `--with-files` to also delete them |
| `ghostdl task restart` | Restart a task (redownload) |
| `ghostdl task open` | Open a task's downloaded file or folder (`--folder`) via Ghost Downloader |
| `ghostdl config get` | View configuration item |
| `ghostdl config set` | Modify configuration item |
| `ghostdl config unset` | Remove override, restore default |
| `ghostdl config list` | List all configurations |
| `ghostdl version` | Print CLI build info (no login required) |
| `ghostdl license` | Print the full license (no login required) |

## Global flags

| Flag | Default | Description |
|------|---------|-------------|
| `--format` / `-f` | `table` | Output format: `table`, `json`, `ndjson`, `csv`, `list` |
| `--detail` | `false` | Show every field instead of the overview subset |
| `--utc` | `false` | Print timestamps in UTC instead of your local time zone |
| `--dry-run` | `false` | Print the request that would be sent, without sending it |
| `--verbose` / `-v` | `false` | Enable debug logs on stderr |
| `--insecure` | `false` | Skip TLS certificate verification for `wss://` (same as config `server.insecure`; no effect on `ws://`) |
| `--log-file` | | Write debug logs to a file as well as stderr (implies `--verbose`) |

Format precedence: **`--format`**, else **`GD_OUTPUT_FORMAT`**, else **`output.format`** in the config file, else **`table`**. `table` is for reading; `json` wraps success and failure in `{"ok":…}` on stdout (failures use `{"ok":false,"error":{type,message,hint}}` with a one-line summary on stderr); `ndjson` emits one raw JSON object per line (no envelope); `csv` for spreadsheets and `cut`/`awk`; `list` for vertical `key: value` blocks. `csv` prints a header plus one row per task.

```bash
ghostdl task list --format csv > tasks.csv
ghostdl task get tsk_abc123 --format list
```

By default a task shows five columns: ID, name, status, progress, and size. `--detail` adds received bytes, speed, creation time, capability flags, and the source URL when Ghost Downloader reports one. `--detail` affects `table`, `csv`, and `list` only. `json` and `ndjson` always emit every field. `ghostdl task progress --watch` draws a live bar and ignores both flags.

The CLI is silent by default. `--verbose` (or `-v`) logs the WebSocket connection, frames, timing, server errors, and the login check to **stderr only**. Tokens are never logged. `--log-file PATH` implies `--verbose` and appends the same lines to the file. Pipelines keep working: `ghostdl task list --format json --verbose | jq '.data.tasks | length'`.

### Reading timestamps

Ghost Downloader reports creation time as Unix seconds. The CLI prints it as RFC 3339 with an offset, in your local zone by default — `2026-09-21T07:34:38+08:00`. `--utc` prints the same instant as Zulu time — `2026-09-20T23:34:38Z`. Every format uses that same string.

`createdAt` is always present in `json` and `ndjson`. `table`, `csv`, and `list` show it only with `--detail`. `TZ` still picks the zone for local rendering (`TZ=America/New_York ghostdl task list --detail`).

`ghostdl version` is the exception: its `build_date` is always UTC and ends in `Z`. `--utc` does not change it. Plain output lists `name`, `homepage`, optional `version` (bare product semver such as `1.2.3` when the binary is stamped from a semver git tag), `build_date`, `commit`, `os`, and `arch`. `commit` may end with `+dirty` when `git status` was not clean at `make build`, or when Go's `vcs.modified` applies on unstamped builds.

## Limitations

This CLI doesn't install or embed Ghost Downloader — you run Ghost Downloader separately and pair with `ghostdl auth login`. Release Android binaries (`ghostdl-unofficial-android-armv7`, `ghostdl-unofficial-android-arm64`, `ghostdl-unofficial-android-386`, `ghostdl-unofficial-android-amd64`) are native executables, not APKs.

On Windows, `ghostdl task progress --watch` draws the progress bar with ASCII (`#` and `-`) because classic Windows 10 conhost does not enable VT processing or UTF-8 output. Windows Terminal is fine with that bar.

Release binaries for macOS, Windows, and Android are cross-compiled on Linux and are not executed in CI.

**You cannot modify an existing task.** BrowserService accepts only `subscribe_tasks`, `create_task`, and `task_action`. URL, filename, save path, headers, and thread count can be set only at create time. `pause`, `resume`, `delete`, `restart`, and `open` change lifecycle only — `restart` (aka `redownload`) reruns the original configuration.

Recreate instead:

```bash
ghostdl task delete tsk_abc123
ghostdl task create --url https://example.com/file.zip --filename new-name.zip --threads 8
```

Ghost Downloader's Aria2 JSON-RPC interface (default port `16800`) is a different protocol. This CLI doesn't use it.

## License

ghostdl-cli-unofficial is licensed under the GNU Affero General Public License, version 3 or any later version (AGPL-3.0-or-later). The full text is in [LICENSE](./LICENSE). `ghostdl license` prints that text and does not require login.
