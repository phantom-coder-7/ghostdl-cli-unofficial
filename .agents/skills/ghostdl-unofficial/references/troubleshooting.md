# Diagnosis and limitations

**Auth**

- Pairing tokens cannot be revoked via the CLI — regenerate in Ghost Downloader settings if compromised.
- If the token is regenerated in Ghost Downloader, previously stored tokens stop working until re-pair.
- Only one Ghost Downloader instance paired at a time. `ghostdl auth login` prints already logged in when the stored token still works. Use `--force` or `--token` to replace that session, or logout before switching. A rejected token still re-pairs with plain `ghostdl auth login`.
- Wrong or unreachable `server.url` surfaces as `unauthorized`, `unreachable`, or `protocol_mismatch` — verify Browser Extension settings and `ghostdl config get server.url`, then `ghostdl auth check`.
- TLS handshake or certificate errors on `wss://` — fix the cert or CA trust, or use `--insecure` / `server.insecure` only when the user accepts skipping verification (no effect on `ws://`).

**Config**

- Changing `server.url` does not log the user in; task commands still require a valid pairing token.

**Auth (keychain)**

- When the OS keychain isn't reachable, run `ghostdl auth login --allow-credential-file` to store the pairing token in the credential file `login-data`. If the keychain's there but refuses access (locked, headless SSH, wrong password), login errors — no file fallback.

**Tasks**

- Guessed or remembered task id → resolve with `ghostdl task list --query "<fragment>" --format json` first; never invent ids from docs or examples.
- No CLI path to mutate create-time fields after the task exists.
- Snapshot URL may be empty even when a download exists; `--query` on URL is weak. Do not promise a URL before it appears in output.
- `--draft` tasks may be missing from `ghostdl task list` until approved in Ghost Downloader.
- No `--seeding` filter and no sort by upload or ratio — those fields are not on `task_snapshot`; use `--pack bt` to list torrents.
- "Not authenticated" or handshake failures → complete auth before retrying task commands.
