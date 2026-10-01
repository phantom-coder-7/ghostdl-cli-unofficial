# Overview and version

## Overview

`ghostdl overview` opens one WebSocket session (hello, then a single `subscribe_tasks` snapshot). It prints server/app/CLI lines, status counts from the full snapshot, aggregate download rate and received bytes summed over **running** tasks only (plus a count of running BitTorrent tasks), and three task sections: in progress (waiting, running, paused), failed, and recent completed. `--limit` caps failed and recent completed only (default 5); in progress is uncapped in JSON and limited to 20 table rows with a hint in human formats. Upload speed and seeding ratio are not on `task_snapshot`, so they never appear here — use `ghostdl task list` for full lists and filters.

| Intent | Command |
|--------|---------|
| Quick dashboard | `ghostdl overview` |
| More failed/completed rows | `ghostdl overview --limit 10` |
| Scriptable snapshot | `ghostdl overview --format json` |

JSON: `.data.counts`, `.data.inProgress[].id`.

## Version

`ghostdl version` prints CLI build/version. No login required (same login-gate skip as `auth`, `config`, `help`, and `completion`).

JSON is a flat map with no `ok`/`data` envelope — `.version`, `.commit`.
