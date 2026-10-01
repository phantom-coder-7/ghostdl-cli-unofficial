# Tasks

Manage Ghost Downloader download jobs through BrowserService. Task state arrives via push snapshots; the CLI renders them in several formats.

**Choosing a read command**

| Need | Command |
|------|---------|
| Counts / traffic snapshot | `ghostdl overview` |
| Find an id by name/fragment | `ghostdl task list --query "<fragment>" --format json` |
| One task + capabilities | `ghostdl task get <id> --detail` (or `--format json`) |
| Progress / ETA once | `ghostdl task progress <id>` |
| Live progress bar | `ghostdl task progress <id> --watch` (ignores `--format`) |

**JSON parse cheat sheet**

| Command | Paths |
|---------|-------|
| `task list` | `.data.tasks[].id`, `.data.total` |
| `task get` | `.data.id`, `.data.status`, `.data.canPause`, `.data.canOpenFile`, `.data.canOpenFolder` |
| `task progress` | `.data.task_id` (snake_case), `.data.status`, `.data.progress` (0–1) |
| `overview` | `.data.counts`, `.data.inProgress[].id` |
| `version` | flat map, no `ok`/`data` envelope — `.version`, `.commit` |

**Resource model**

A **task** is one Ghost Downloader download job.

**Statuses (only these, case-sensitive):** `waiting`, `running`, `paused`, `completed`, `failed`. Unknown `--status` values (e.g. `downloading`, `Running`) are rejected with an error — not a silent empty list.

**`ghostdl task list` (client-side on snapshot):** BrowserService has no list query API. The CLI fetches one snapshot, filters and sorts locally, then applies `--limit` and `--offset`. Within each of `--status`, `--pack`, and `--ext`, repeat the flag or use comma-separated values for OR. Across `--status`, `--pack`, `--ext`, and `--query`, filters AND. Unknown pack or extension strings still match nothing; unknown **status** strings error. For `table`/`list`, when the page is partial, stderr shows how many rows are shown vs filtered `total`; `--limit 0` shows all.

`--pack` is exact `packName` (case-insensitive). Prefer `--pack bt` for BitTorrent, not `--query bt` (query also matches names containing "bt"). Common pack ids: `http`, `bt`, `m3u8`, `ytdlp`, `bili`, `ed2k`, `ftp`, `github`, `huggingface`, `ffmpeg`.

`--ext` is exact file extension (case-insensitive; `zip` and `.zip` are equivalent). `--query` / `-q` is a case-insensitive substring on id, name, url, pack name, or file extension. Snapshot `url` is often empty, so URL search via `--query` is unreliable.

`--sort` accepts `created`, `id`, `name`, `pack` (`packName`), `progress`, `received` (`receivedBytes`), `size`, `speed`, and `status` (status order: waiting, running, paused, completed, failed). `--order` is `asc` or `desc`; `--sort` alone defaults to asc; `--order` alone defaults to sort by `created`; omit both to keep Ghost Downloader newest-first order. Invalid sort field or order returns an error. JSON `data.total` is the filtered count before pagination.

Ghost Downloader tracks seeding internally, but `task_snapshot` omits seeding state, upload speed, and share ratio. The CLI cannot filter or sort by seeding, upload speed, or share ratio yet. Use `--pack bt` to list torrent tasks (`packName` is `bt` on the wire).

**Create:** Default `--source` is `resource` with `--url`. Only `download` and `resource` are accepted; `resource_merge` and `page_media` are rejected (the CLI does not send merge/page wire fields). `--draft` holds the task for approval in the Ghost Downloader UI; drafted tasks often do not appear in `ghostdl task list`.

**Open:** `ghostdl task open <task-id>` sends `open_file`; `--folder` sends `open_folder`. Ghost Downloader on the desktop performs the action (file manager or default app). Requires the desktop app to be running.

**Immutability:** BrowserService exposes only `subscribe_tasks`, `create_task`, and `task_action` — no update message. Fields set at `ghostdl task create` (URL, filename, path, headers, threads, and the rest of the `create_task` payload) cannot change on an existing task. `task_action` is lifecycle-only (`toggle_pause`, `cancel`, `remove`, `redownload`, `open_file`, `open_folder`); `redownload` reruns the original config. To change creation-time fields, delete and create again.

**Output**

`--format` / `-f`: `table` (default), `json`, `ndjson`, `csv`, `list`.

| Format | Use when |
|--------|----------|
| `table` | Human-readable terminal output |
| `json` | Parsing full envelope `{"ok": true, "data": ...}` |
| `ndjson` | One JSON object per line |
| `csv` | Spreadsheet or `cut`/`awk` pipelines |
| `list` | Wide rows as vertical `key: value` blocks |

`ghostdl task get <task-id> --format csv` emits a header plus one data row.

`--detail` adds fields beyond the overview (ID, Name, Status, Progress%, Size): received bytes, speed, creation time, capability flags, URL when present. Applies to `table`, `csv`, and `list` only — `json`/`ndjson` already include all fields; do not add `--detail` when parsing JSON.

Timestamps are RFC 3339 with offset. Default is local zone; global `--utc` prints UTC (including in `json`/`ndjson`). Report timestamps as printed; do not convert to relative time.

`ghostdl task progress --watch` shows an interactive progress bar and ignores `--format` and `--detail`. Exit 0 on interrupt or `completed`; exit 1 on task `failed` or connection loss while watching.

With `--format json`, command failures print `{"ok":false,"error":{…}}` on stdout and a one-line `Error: [type] message` on stderr. Connect/auth failures from `ghostdl auth check` and task commands return structured errors (no long hint text on stdout).

**Safety and dry-run**

- Global `--dry-run` defaults to `false`. When set, prints the JSON that would be sent and sends nothing. Supported on `create`, `pause`, `resume`, `delete`, `restart`, and `open`.
- `ghostdl task create` creates a real task by default; pass `--dry-run` only when the user wants a preview.
- `ghostdl task delete --with-files` deletes downloaded files and prompts for confirmation. Use `--yes` to skip the prompt; `--yes` is required when stdin is not a terminal.

**Procedures**

- List downloads: `ghostdl task list` — `--status running`, `--pack bt`, `--ext zip`, `--query report`, `--sort received --order desc`, etc.
- Count completed tasks without parsing JSON: `ghostdl task list --status completed --format csv` then `tail -n +2 | wc -l`.
- Create: `ghostdl task create --url https://example.com/file.zip` — optional `--filename`, `--threads`, `--header "Name: value"`, etc.
- Open download: `ghostdl task open <task-id>` — `--folder` for the containing directory; `--dry-run` to preview the wire message.
- Watch until done: `ghostdl task progress <task-id> --watch`
- Pause / resume: `ghostdl task pause <task-id>` / `ghostdl task resume <task-id>` — no-op if already paused/running; refuse on `completed` or `failed`.
- Delete: `ghostdl task delete <task-id>` keeps files; add `--with-files` to remove files (confirm or `--yes`).
- Change URL, path, headers, or threads: no edit command — `ghostdl task delete <task-id>` then `ghostdl task create …` with new flags.

| Intent | Command |
|--------|---------|
| Show my downloads | `ghostdl task list` |
| Running only | `ghostdl task list --status running` |
| Running or paused | `ghostdl task list --status running,paused` |
| BitTorrent tasks only | `ghostdl task list --pack bt` |
| Largest BitTorrent tasks | `ghostdl task list --pack bt --sort size --order desc` |
| Search by name/url/etc. | `ghostdl task list --query report` |
| Zip files only | `ghostdl task list --ext zip` |
| Completed torrent zips, most received first | `ghostdl task list --pack bt --ext zip --status completed --sort received --order desc` |
| Largest first | `ghostdl task list --sort size --order desc` |
| Oldest created first | `ghostdl task list --order asc` |
| Start a download | `ghostdl task create --url https://example.com/file.zip` |
| Preview create payload | `ghostdl task create --url … --dry-run` |
| Task details | `ghostdl task get <task-id>` |
| All fields, readable | `ghostdl task get <task-id> --detail --format list` |
| Progress once | `ghostdl task progress <task-id>` |
| Redownload | `ghostdl task restart <task-id>` |
| Remove task, keep files | `ghostdl task delete <task-id>` |
| Open downloaded file | `ghostdl task open <task-id>` |
| Open download folder | `ghostdl task open <task-id> --folder` |
