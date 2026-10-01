---
name: ghostdl-unofficial
description: Covers the Ghost Downloader `ghostdl` CLI — `ghostdl auth` pairing and credentials, `ghostdl config` keys and `GD_*` overrides, `ghostdl version`, `ghostdl license` (no login), and `ghostdl task` download control with `--format`, `--detail`, `--utc`, `--watch`, `--dry-run`, `ghostdl task open`, and `ghostdl task list` filters (`--query`, `--status`, `--pack`, `--ext`, `--sort` including `received`, `--order`). Use when the user logs in or re-pairs, pastes a Pairing Token, logs out, checks auth or server reachability, fixes token errors, changes BrowserService WebSocket URL or default output format, lists/creates/pauses/resumes/deletes/watches/restarts/opens downloads, or diagnoses CLI behavior. Triggers include Ghost Downloader CLI, ghostdl auth login, pairing token, logout, auth status, token invalid, not authenticated, server URL, config key, env override, task list, BitTorrent, pack bt, file extension, progress, pause, resume, delete, restart, redownload, open file, watch, dry-run, ghostdl version, ghostdl license. Not for Aria2, kubectl, Feishu, or unrelated downloaders.
---

## What this covers

Operate Ghost Downloader from the terminal via the `ghostdl` binary: pair with BrowserService, manage persistent settings, control download tasks, and print build info with `ghostdl version`.

**Auth:** `ghostdl auth login`, `login --force`, `login --token`, `logout`, `status`, `check`

**Config:** `ghostdl config get`, `set`, `unset`, `list`

**Tasks:** `ghostdl task create`, `list`, `get`, `progress`, `pause`, `resume`, `delete`, `restart`, `open`

**Overview:** `ghostdl overview` — one hello plus one task snapshot with counts, aggregate download traffic from running tasks, in-progress/failed/recent completed slices. Full filtered lists stay on `ghostdl task list`. Upload and seeding totals are not in the snapshot.

**Version:** `ghostdl version` — CLI build/version (no login required).

**License:** `ghostdl license` — prints the full license (no login required).

## When not to use this skill

- Aria2, generic curl/wget workflows, or other download tools — not Ghost Downloader BrowserService.
- kubectl, Helm, or cluster operations — unrelated to `ghostdl`.

## Decision tree

| User need | Area | First commands |
|-----------|------|----------------|
| Login, pair, token invalid, logout, reachability | Auth | `ghostdl auth status` (local), `ghostdl auth login` if needed, then `ghostdl auth check` (live) |
| Server URL, default output | Config | `ghostdl config list` / `get` / `set` (no login). Set `server.url` then `ghostdl auth login` when needed; `GD_SERVER_URL` overrides |
| Counts, in-progress, traffic snapshot | Overview | Pairing/server up (`ghostdl auth status`, `ghostdl auth check` if unsure), then `ghostdl overview` |
| List/create/pause/resume/delete/watch downloads | Tasks | Pairing/server up, then **read** (list/get) before mutate — see [Read before write](references/read-before-write.md) |

Task commands always require a valid pairing token. `ghostdl config` does not — it only reads/writes local `config.yaml`. Config changes do not log the user in. Only `auth`, `config`, `help`, `completion`, `version`, and `license` skip the login gate.

## Read before write

Look up before you mutate. Prefer read commands when the user gives a name or fragment rather than a task id, and before pause/resume/delete/restart/open.

Full procedure (login order, resolve id, inspect, status→verb, `--dry-run`): [Read before write](references/read-before-write.md).

## References

| Area | File |
|------|------|
| Read before write | [references/read-before-write.md](references/read-before-write.md) |
| Auth | [references/auth.md](references/auth.md) |
| Config | [references/config.md](references/config.md) |
| Tasks | [references/tasks.md](references/tasks.md) |
| Overview and version | [references/overview.md](references/overview.md) |
| Diagnosis and limitations | [references/troubleshooting.md](references/troubleshooting.md) |
