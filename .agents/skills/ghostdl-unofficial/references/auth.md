# Auth

Pair with one Ghost Downloader BrowserService using a pairing token, not a password. How the token is stored (OS keychain vs `login-data`, file modes, Windows ACL): README pairing section. This repo uses zalando/go-keyring for the keychain; there is no second vault.

**How pairing works**

1. **Pairing request (interactive):** `ghostdl auth login` with no flags. If the stored token's `hello` still succeeds, the CLI prints already logged in and does not send `pair_request`. Otherwise it sends `pair_request` → approve the dialog in Ghost Downloader within **60 s** → the CLI receives `pair_result` with a token. `ghostdl auth login --force` sends `pair_request` even when that `hello` succeeds. Use when someone is at the machine.
2. **Copied Pairing Token:** Copy from Ghost Downloader → Settings → Browser Extension → `ghostdl auth login --token <pairing-token>`. Skips the approval dialog and replaces the stored token. `--force` does not apply. Preferred for automation or headless use.

Both paths are equivalent once logged in. Later commands authenticate with a `hello` handshake using the stored token.

**One instance at a time**

The CLI holds a single credential for one BrowserService. Logout clears it. To switch installs or machines, logout and log in against the new target, or replace a working token with `--force` or `--token`.

**Procedures**

- Interactive login: `ghostdl auth login` — already logged in when `hello` succeeds; otherwise approve within 60 s.
- Pair again while the token still works: `ghostdl auth login --force`.
- Token login: `ghostdl auth login --token <pairing-token>`.
- Switch target: `ghostdl auth logout`, then login against the new instance. `--force` or `--token` also replaces a session that still works.
- Token invalid or regenerated in Ghost Downloader: re-pair with `ghostdl auth login` or a fresh `--token`. The CLI cannot revoke tokens; regenerate in Ghost Downloader settings if compromised.

| Intent | Command |
|--------|---------|
| Log in with popup | `ghostdl auth login` |
| Pair again while the token still works | `ghostdl auth login --force` |
| Log in with copied token | `ghostdl auth login --token <pairing-token>` |
| Am I logged in? | `ghostdl auth status` |
| Is Ghost Downloader reachable? | `ghostdl auth check` |
| Log out | `ghostdl auth logout` |
| Change to another Ghost Downloader install | `ghostdl auth logout`, then login again |
