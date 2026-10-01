# Config

Read and write persistent CLI settings. Individual commands can still override output with flags such as `--format`.

`ghostdl config` reads and writes local `config.yaml` — no stored credentials required. Set `server.url` before `ghostdl auth login` when BrowserService is not on the default address. `GD_SERVER_URL` overrides `server.url` when set.

**Keys**

| Key | Default | Meaning |
|-----|---------|---------|
| `server.url` | `ws://127.0.0.1:14370` | BrowserService WebSocket address (plaintext localhost default, same as the official browser extension) |
| `server.insecure` | `false` | Skip TLS certificate verification for `wss://` only (same as global `--insecure`; ignored for `ws://`) |
| `output.format` | `table` | Default output: `table`, `json`, `ndjson`, `csv`, or `list` |

**Precedence:** `GD_*` environment variables > config file > built-in defaults.

Example: `export GD_SERVER_URL=ws://127.0.0.1:14370` overrides `server.url`. Map env names with the `GD_` prefix (server URL → `GD_SERVER_URL`).

**Output format precedence:** `--format` / `-f` when passed > `GD_OUTPUT_FORMAT` > `output.format` in the config file > built-in `table`.

Older `config.yaml` files may still contain **`task.max_count`** or **`watch.interval`**; the CLI ignores them. Remove them with `ghostdl config unset task.max_count` or `ghostdl config unset watch.interval` if you want a clean file.

Pairing tokens are not stored in config keys. `ghostdl auth login` stores the token in the OS keychain when that service is available. The credential file `login-data` is written only with `--allow-credential-file` when the OS keychain isn't reachable. If the keychain's present but refuses access, login errors — it won't write the file.

**Procedures**

- Show one value: `ghostdl config get server.url`
- Non-default BrowserService port: check Ghost Downloader → Settings → Browser Extension, then `ghostdl config set server.url ws://127.0.0.1:<port>`
- TLS WebSocket (`wss://`): certificates are verified by default; self-signed or private CA → `ghostdl --insecure …` or `ghostdl config set server.insecure true`
- Default JSON output: `ghostdl config set output.format json`
- Show everything: `ghostdl config list`
- Restore default for a key: `ghostdl config unset server.url`

| Intent | Command |
|--------|---------|
| Where is the server configured? | `ghostdl config get server.url` |
| Point CLI at another port | `ghostdl config set server.url ws://127.0.0.1:<port>` |
| Always use JSON in the terminal | `ghostdl config set output.format json` |
| JSON for one listing only | `ghostdl task list --format json` |
| Reset server URL to default | `ghostdl config unset server.url` |
