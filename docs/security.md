# Security model

This page describes what `tfy-local-ai-setup` protects, what it doesn't, and what to decide before a rollout. Where a behaviour depends on the auth mode, both modes are covered.

Behaviour described here is from the v3.0.1 source. File modes and lock commands are as written by the binary.

## Summary

- Managed config is written by root (or SYSTEM) and locked, so developers can't edit it casually.
- **In `api-key` mode, the managed files contain the developer's device-login token, and those files are readable by every local user.** The token is short-lived, but it is still a bearer credential on disk. See [api-key mode](#api-key-mode-token-in-a-readable-file) below.
- `oidc` mode writes no bearer token into managed files. Clients fetch tokens through a helper at runtime.
- Personal credentials go to the OS credential store. If that's unavailable, they go to a `0600` file.
- Logs and run reports redact tokens and headers.

## Trust model

| Party | Trusted for |
|---|---|
| Root or SYSTEM | Writing managed files, the policy file, and the binary itself. |
| The logged-in developer | Their own user config and their own sign-in. The managed files are locked against them. |
| The control plane and gateway | Issuing tokens, enforcing model and MCP permissions. The gateway enforces permissions independently of anything written locally. |
| Local users other than the developer | Nothing. Assume they can read anything world-readable. |

## Managed files

Each managed file is written as root with mode `0644`, then locked:

| OS | Lock | Lift the lock (root only) |
|---|---|---|
| macOS | `chflags schg` (system-immutable flag) | `chflags noschg <path>` |
| Linux | `chattr +i` (immutable attribute) | `chattr -i <path>` |
| Windows | `icacls`: removes inherited permissions, grants SYSTEM full control, Administrators read, Users read | Re-run the deploy script. It grants write access before the update. |

The lock stops edits by ordinary users. It does not stop root, and it does not stop a user who can run `sudo` or an administrator shell.

On Linux, if the filesystem doesn't support the immutable attribute, the binary logs `WARNING: chattr not available — file not immutable-locked` and writes the file unlocked.

### Codex on Windows

Codex reads its managed config from `%ProgramData%\OpenAI\Codex`. Ordinary users can create folders in ProgramData, so before Codex runs, a user could plant a config there. The binary therefore takes over that directory:

- Owner is set to Administrators, recursively.
- Inherited permissions are removed.
- SYSTEM and Administrators get full control, inherited by children. Users get read and execute.

### Policy file

The policy file says what IT configured, and personal mode reads it to leave those agents alone. It lives where only administrators can write:

| OS | Path |
|---|---|
| macOS | `/Library/Application Support/TrueFoundry/agents-policy.json` |
| Linux | `/etc/truefoundry/agents-policy.json` |
| Windows | `%ProgramFiles%\TrueFoundry\agents-policy.json` |

Windows uses Program Files rather than ProgramData for the same reason as above: a user could create the folder first.

Personal mode ignores a policy file that is not owned by an administrator or that other users can write. It prints `ignoring <path>: not owned by an administrator, or writable by other users`.

The policy file holds configuration, never credentials.

## Credentials

### Personal mode

Credentials go to the OS credential store:

- Service name: `truefoundry-ai-setup`.
- Account: `<tenant>:<kind>`, so each tenant and credential type has its own entry.
- Backends: macOS Keychain, Windows Credential Manager, and the Secret Service (libsecret) on Linux.

If no usable credential store exists, the binary falls back to a file under `~/.tf` with mode `0600`, inside a `0700` directory. It prints a warning when it does. Only the credential's own file changes. The binary never creates or changes the user's default keychain.

### Managed mode, device login

Device-login tokens are cached under the logged-in user's home, in `~/.tf`:

- Files are `0600`, and directories are `0700`.
- The refresh-token and access-token cache files are `~/.tf/refresh-token` and `~/.tf/access-token`, or the `oidc-*` variants for the OIDC helper.

### api-key mode: token in a readable file

In `api-key` mode, the binary writes the developer's current device-login token into the managed file of each client:

- Claude Code: `ANTHROPIC_CUSTOM_HEADERS` with `X-TFY-API-KEY`.
- Codex: the provider's `http_headers`.
- Claude Desktop: `inferenceGatewayApiKey` and the `X-TFY-API-KEY` header.

The managed file is written `0644`, so **every local user can read the token**. It is refreshed on each scheduled run. It is short-lived, but a copy that another local user takes can be used against the gateway until it expires.

If that exposure matters for your fleet, use `auth.mode = "oidc"`. With OIDC, Claude Code uses an `apiKeyHelper` and Codex uses `auth.command`. Neither writes a token into the file. Claude Desktop uses its own native OIDC flow.

The OIDC helpers keep their refresh and access tokens under `~/.tf` (`oidc-refresh-token`, `oidc-access-token`), with the same `0600` and `0700` rules.

### `token` and shell helpers

`tfy-local-ai-setup token` prints the credential on stdout, for agents that read it through a shell block or a helper. It never opens a browser. Shell-block agents use `token --mdm` under managed mode.

Inside a WSL distro, the token-command file at `~/.tf/token-command` is written with `umask 077`.

## Logs and run reports

- `--log-file` appends with mode `0600`.
- `--debug` logs each control-plane and gateway call as method, URL and status. It never logs headers or tokens. It logs sign-in decisions, planned changes and each file written.
- Free text in run reports is redacted before it is written.
- Managed run reports are stored in the administrator-owned state directory, one folder per user. The newest 48 are kept.
- Skill download links are short-lived and signed. They never appear in output or debug logs.

## Self-update

`tfy-local-ai-setup` can replace itself with a newer release. Before swapping the binary, it:

1. Checks the downloaded file against the release's `SHA256SUMS`. A mismatch keeps the current binary.
2. Smoke-tests the new binary.
3. Swaps it in atomically.

Signature verification exists in the code path, but releases are not yet signed. Until they are, the checksum is the only check on a downloaded update. Pin the release in your MDM script, and don't rely on self-update for managed fleets.

## Personal restrictions are voluntary

Prompt blocking installed by `--block-agent` in personal mode is a hook the user can remove or disable. Codex also requires the user to review and trust a new hook before it runs. For enforcement, use managed mode, where hooks go in protected system locations.

## WSL

`--wsl` routes the managed config into WSL distros. A root shell inside a distro is available to any user who can run `wsl.exe`. The config is routed there, not enforced. There is no immutable lock inside a distro.

## Recommendations

- Use `auth.mode = "oidc"` for fleets where a readable device-login token is not acceptable.
- Keep `RELEASE_TAG` pinned in deploy scripts, and check the release's `SHA256SUMS` before rolling it out.
- Keep managed config and deploy scripts in your MDM's protected store. Anyone who can edit the script can change what runs as root.
- Don't put client secrets or PATs into `mdm.toml`. The OIDC client ID is public and is the only identifier it needs.
- Set `--force-login-org` only after confirming the org UUID in the Claude Admin Console. A wrong UUID locks users out.
- Send logs off-host with your normal log tooling. `--log-file` holds `0600` plaintext.
