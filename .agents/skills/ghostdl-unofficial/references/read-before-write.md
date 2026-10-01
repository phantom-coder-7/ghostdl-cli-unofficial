# Read before write

Look up before you mutate. Prefer read commands whenever the user gives a name or fragment rather than a task id, and before any pause/resume/delete/restart/open (or create when targeting an existing download).

**Login order**

1. `ghostdl auth status` — local, cheap (`Status: Logged in` / `Not logged in`).
2. If not logged in → `ghostdl auth login` (or `--token`).
3. Before blaming task or config failures on the server → `ghostdl auth check` (`Server is up and authenticated.`).

`ghostdl overview` and `ghostdl task` require stored credentials. `ghostdl config` does not — it only touches local `config.yaml`. Set `server.url` with `ghostdl config set` before `ghostdl auth login` when needed; `GD_SERVER_URL` remains a valid override.

**Resolve id**

When the user names a download instead of an id:

```
ghostdl task list --query "<fragment>" --format json
```

Read `.data.tasks[].id` and `.data.total`. Narrow with `--status`, `--pack`, `--sort` / `--order` if several match. If `total` is 0, say so — never invent an id from docs or examples.

Default page is `--limit 20` (offset 0). `data.total` is the filtered count **before** pagination. Raise `--limit` or walk `--offset` when the match might be past the first page. `--limit 0` returns all rows after offset — use only when a full dump is intentional.

**Inspect**

```
ghostdl task get <id> --format json
```

Use `.data.id`, `.data.status`, `.data.canPause`, `.data.canOpenFile`, `.data.canOpenFolder`. Human alternative: `ghostdl task get <id> --detail --format list`. Missing id → `task not found: <id>`.

**Choose the verb from status**

| Status | `ghostdl task pause` | `ghostdl task resume` |
|--------|-----------------|------------------|
| `waiting` | refuses — nothing to pause yet | no-op — queued, starts on its own |
| `running` | pauses the task | no-op — already running |
| `paused` | no-op — already paused | resumes the task |
| `completed` | refuses — already completed | refuses — already completed |
| `failed` | refuses — use `ghostdl task restart` | refuses — use `ghostdl task restart` |

Open a file or folder only when the matching `canOpen*` field is true.

Preview destructive or uncertain mutations with `ghostdl task <verb> <id> --dry-run`.
