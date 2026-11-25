# CLI reference

`tfy-local-ai-setup` has two modes and one catalogue of settings:

- **Managed (MDM) mode.** The default when no subcommand is given. Runs as root or Administrator. Writes and locks each client's managed config. Driven by flags or by `--config`.
- **Personal mode.** Runs as the developer, never elevates, and writes user-level config. Started with the `personal` subcommand.
- **Subcommands.** Day-to-day operations such as `status`, `doctor`, `login`, `token`, `config`, `update` and `uninstall`.

The flag list below is taken from `tfy-local-ai-setup -h` on **v3.0.1**. Check it against your release with `tfy-local-ai-setup -h`. The Go `flag` package prints the usage text to stderr and exits `0`.

> There is no `--version` flag in v3.0.1. Read the version from the `.version` file that the deploy script writes (see [MDM deployment](mdm-deployment.md#9-verify-a-machine)).

## Contents

- [Synopsis](#synopsis)
- [Exit codes](#exit-codes)
- [Managed-mode flags](#managed-mode-flags)
- [Subcommands](#subcommands)
- [Hidden and internal flags](#hidden-and-internal-flags)

## Synopsis

```
sudo tfy-local-ai-setup --config=<file> [flags]          # managed, file-driven (recommended for fleets)
sudo tfy-local-ai-setup --url=<url> --tenant=<name> ...  # managed, flag-driven
tfy-local-ai-setup <subcommand> [args]                   # personal mode and operations
```

Managed mode requires root or Administrator. `--url` and `--tenant` are required unless `--config` supplies them. The exception is a restrictions-only run (see [prompt blocking](../README.md#block-ai-prompts-with-hooks)).

## Exit codes

| Code | Meaning |
|---|---|
| `0` | Success. Every requested client, or every detected client, is configured. Also returned when auto-detect finds nothing to configure. |
| `1` | Error. The message goes to stderr and, when set, to `--log-file`. |
| `3` | Every agent was configured, but some Agent Skills could not be installed. Causes: the skill is not in the Skills Registry, or the pinned version does not exist. |

These codes apply to managed mode. Subcommands use their own result codes, which are listed under each subcommand where they differ.

## Managed-mode flags

Flags are grouped by purpose. Each row's default comes from the binary. Empty means the flag has no default and is unset.

### Connection

| Flag | Default | Description |
|---|---|---|
| `--url` | — | Base URL of the TrueFoundry control plane. Required unless `--config` supplies it. |
| `--tenant` | — | Tenant name. Required unless `--config` supplies it. |
| `--gateway` | value of `--url` | Gateway URL for all clients. Per-client overrides below take precedence. |
| `--claude-gateway` | value of `--gateway` | Gateway URL for Claude Code only. Written to `ANTHROPIC_BASE_URL`. |
| `--codex-gateway` | value of `--gateway` | Gateway URL for Codex only. Written to the provider `base_url`. |
| `--claude-desktop-gateway` | value of `--gateway` | Gateway URL for Claude Desktop only. Written to `inferenceGatewayBaseUrl`. |

### Clients and sign-in

| Flag | Default | Description |
|---|---|---|
| `--claude-code` | auto-detect | Configure Claude Code even if it is not installed. Passing the flag skips the install check. |
| `--claude-code-auth` | `api-key` | `api-key` writes `X-TFY-API-KEY` from a device-login token. `oidc` writes `apiKeyHelper` that runs `--_print-oidc-token`. Requires `--oidc-issuer` and `--oidc-client-id`. |
| `--codex` | auto-detect | Configure Codex, whether the CLI or the desktop app. Both share the managed config. |
| `--codex-auth` | `api-key` | `api-key` writes a token into the provider's headers. `oidc` writes `auth.command` that runs `--_print-oidc-token`. Requires `--oidc-issuer` and `--oidc-client-id`. |
| `--codex-auth-mode` | `api-key` | Header style when `--codex-auth=api-key`. `api-key` sends `Authorization: Bearer`. `chatgpt-subscription` sets `requires_openai_auth = true` and sends `x-tfy-api-key`. |
| `--codex-default-model` | empty | Top-level `model` in `managed_config.toml`. Not written when empty. |
| `--codex-settings-file` | — | TOML template for Codex's config. The provider section is injected into it. |
| `--claude-desktop` | auto-detect | Configure Claude Desktop (Cowork on 3P) even if it is not installed. Still a no-op on Linux. |
| `--claude-desktop-auth` | `api-key` | `api-key` writes `inferenceGatewayApiKey` and the `X-TFY-API-KEY` header. `oidc` makes the app sign in itself through `inferenceIdpOidc`, and requires Claude Desktop 2.7032.0 or later. |
| `--oidc-issuer` | — | OIDC issuer for TrueFoundry's public client. Required with any `…-auth=oidc` flag. |
| `--oidc-client-id` | — | OIDC public client ID. Required with any `…-auth=oidc` flag. Same for every machine in the tenant. |
| `--oidc-bearer-token-type` | `access_token` | Token Claude Desktop sends as the bearer. Keep the default. The TrueFoundry gateway rejects `id_token`. |
| `--oidc-scopes` | `openid profile email offline_access` | Scopes for Claude Desktop. Keep `openid`, or sign-in fails. |
| `--agents-auth` | `api-key` | Auth for the `--agent` agents: `api-key` or `oidc`. `oidc` requires `--oidc-issuer` and `--oidc-client-id`. |
| `--login` | empty | `manual`: a PAT is created only by `login --mdm`, so scheduled runs never open a browser. |
| `--pat-team` | empty | Team that owns a managed PAT. Used with manual login. |
| `--pat-expiry-days` | `365` | Lifetime of a managed PAT, 7 to 3650 days. Used with manual login. |
| `--auto-renew-key` | `false` | Renew each user's PAT before it expires. Uses the PAT and no browser. |
| `--refresh-token-file` | `~/.tf/refresh-token` (device login) or `~/.tf/oidc-refresh-token` (OIDC helper) | Path of the refresh-token cache, mode `0600`. |
| `--access-token-file` | `~/.tf/access-token` (device login) or `~/.tf/oidc-access-token` (OIDC helper) | Path of the access-token cache, mode `0600`. |
| `--write-config-on-login-failure` | `false` | When device login fails, still write and lock the managed config, then exit `1`. Without it, a failed login writes nothing. |
| `--placeholder-token` | `TFY_PLACEHOLDER_LOGIN_REQUIRED` | Token written when `--write-config-on-login-failure` is set and there is no cached token. Not a valid credential. Requests with it get `401`. |

### Models

| Flag | Default | Description |
|---|---|---|
| `--opus-model` | `claude-code/claude-opus` | `ANTHROPIC_DEFAULT_OPUS_MODEL` (Claude Code). |
| `--sonnet-model` | `claude-code/claude-sonnet` | `ANTHROPIC_DEFAULT_SONNET_MODEL` (Claude Code). |
| `--haiku-model` | `claude-code/claude-haiku` | `ANTHROPIC_DEFAULT_HAIKU_MODEL` (Claude Code). |
| `--desktop-opus-model` | value of `--opus-model` | Opus entry in Claude Desktop's `inferenceModels`. |
| `--desktop-sonnet-model` | value of `--sonnet-model` | Sonnet entry in Claude Desktop's `inferenceModels`. |
| `--desktop-haiku-model` | value of `--haiku-model` | Haiku entry in Claude Desktop's `inferenceModels`. |
| `--additional-model` | — | Models Claude Code offers in `/model` after the tier models. Repeatable or comma-separated. Accepts `@all`. Writes a `modelPicker`. |
| `--desktop-additional-model` | value of `--additional-model` | Extra models in Claude Desktop's menu. Desktop shows Claude models only. |
| `--model` | — | Gateway model offered to the `--agent` agents. Repeatable or comma-separated. |
| `--default-model` | first `--model` | Default model for the `--agent` agents. Must be one of `--model`. |

### MCP servers and Agent Skills

| Flag | Default | Description |
|---|---|---|
| `--mcp-server` | — | Gateway MCP server to deploy to Claude Code, Codex and Claude Desktop. Repeatable or comma-separated. Resolves to `<gateway>/<tenant>/mcp/<name>/server`. |
| `--no-mcp` | `false` | Give the `--agent` agents no shared MCP servers. Cannot be combined with `--mcp-server`. |
| `--claude-desktop-allow-local-mcp` | `true` | Let users run their own local (stdio) MCP servers in Claude Desktop. `=false` allows remote servers only. |
| `--skill` | — | Agent Skill for the `--agent` agents, as `[repo/]name[:version]`. Repeatable or comma-separated. |
| `--no-skills` | `false` | Install no skills for the `--agent` agents. Removes the ones earlier runs installed. |

### Claude Desktop extras

| Flag | Default | Description |
|---|---|---|
| `--desktop-header` | — | Extra header as `Name: Value`, written to `inferenceCustomHeaders`. Repeatable. |
| `--force-login-org` | — | Restrict sign-in to this Claude org UUID. Repeatable. Writes `forceLoginOrgUUID`. A wrong UUID locks everyone out. |
| `--disable-deployment-mode-chooser` | `false` | Write `disableDeploymentModeChooser=true`, so users cannot switch the app to a personal Claude.ai profile. Applied only with `--claude-desktop-auth=oidc`. |
| `--claude-ai-import` | `false` | Enable claude.ai history import (`claudeAiImport`). Requires Claude Desktop 1.10628.0 or later. |
| `--import-export-enabled` | `true` | Show the Export button. No effect without `--claude-ai-import`. |
| `--import-banner-behavior` | `detect` | `off`, `detect` or `show`. No effect without `--claude-ai-import`. |
| `--import-automatic-3p` | `false` | Beta. Imports earlier 3P sessions silently. Use only with the customer's agreement. |
| `--import-encoding` | `json-string` | `json-string` or `native-dict`, for the macOS plist. Windows always uses `json-string`. |

### Files, templates and config

| Flag | Default | Description |
|---|---|---|
| `--config` | — | One standalone TOML or JSON file for every client. Explicit flags override its values. See the [config reference](mdm-config-reference.md). |
| `--agents-config` | — | TOML in personal mode's `config.toml` format, for the `--agent` agents. Flags win over it. Cannot be combined with `--config`. |
| `--settings-file` | — | JSON template for `managed-settings.json` (Claude Code). The binary patches the token and model IDs and keeps the rest. |
| `--codex-settings-file` | — | TOML template for Codex's config. See above. |

### Other agents

| Flag | Default | Description |
|---|---|---|
| `--agent` | — | Agents to configure, by ID. Repeatable or comma-separated: `claude-code`, `codex`, `claude-desktop`, `grok-build`, `pi`, `qwen-code`, `opencode`, `gemini-cli`, `copilot-cli`, `goose`, `vscode`, `cursor`. The first three are the same as their own flags. The rest are configured only when named here. |

### Prompt blocking and Cursor monitoring

| Flag | Default | Description |
|---|---|---|
| `--block-agent` | — | Block AI prompts for `claude-code`, `codex` or `cursor`. Repeatable or comma-separated. |
| `--unblock-agent` | — | Remove this setup's prompt block for the named agent. |
| `--block-message` | — | Message shown when a prompt is blocked. |
| `--cursor-monitoring` | `false` | Copy Cursor prompt, response, tool and session events to TrueFoundry through gateway hooks. Doesn't change model routing. `=false` removes the monitoring hooks. |
| `--cursor-require-login` | `false` | With `--cursor-monitoring`, block Cursor prompts and tool calls until an MDM credential is available. |

### Windows

| Flag | Default | Description |
|---|---|---|
| `--wsl` | `false` | Also push the managed config into the logged-in user's WSL distros. Windows only. |

### Logging and testing

| Flag | Default | Description |
|---|---|---|
| `--dry-run` | `false` | Print the resulting config to stdout instead of writing files. |
| `--log-file` | — | Also append all log output to this file, mode `0600`. |
| `--debug` | `false` | Verbose logging. Shows the device-login browser URL and child-process output. |

## Subcommands

These run before managed-mode flag parsing. Anything that isn't a subcommand name is managed mode. Flags below come from running `tfy-local-ai-setup <subcommand> -h` on v3.0.1.

| Subcommand | Runs as | Purpose |
|---|---|---|
| `personal` | developer | Interactive setup for the developer's own machine, or `personal <agent>` for one agent. |
| `update` | developer | Reapply saved personal choices. |
| `status` | developer | Show what is configured: agents, MCP servers, skills and versions, and sign-in state. |
| `doctor` | developer | Run health checks on the setup and report problems. |
| `login` | developer | Sign in. `login --mdm` finishes manual-PAT onboarding on a managed machine. |
| `token` | developer | Print the credential for agents and shell blocks. |
| `config` | developer | Read and write saved settings. |
| `uninstall` | developer | Remove what personal setup installed. |
| `cursor-monitor` | Cursor hooks | Internal entry point for Cursor hooks. Not for manual use. |

### `personal`, `update`, `login`

These three share one flag set:

| Flag | Description |
|---|---|
| `-url`, `-tenant`, `-gateway` | Control plane, tenant and gateway. |
| `-config` | Read settings from a `.toml` or `.json` file. The wizard asks only for what the file doesn't set. |
| `-tool` | Agent to set up. |
| `-model`, `-default-model`, `-codex-model` | Model IDs. `-codex-model` is the Virtual Model for Codex. |
| `-opus-model`, `-sonnet-model`, `-haiku-model` | Claude tier models. |
| `-mcp-server`, `-no-mcp` | MCP servers to add, or none. |
| `-skill`, `-no-skills` | Agent Skills to add, or none. |
| `-oidc-issuer`, `-oidc-client-id` | OIDC settings. |
| `-pat-name`, `-pat-team` | Name and team for the Personal Access Token. |
| `-block-agent`, `-unblock-agent`, `-block-message` | Prompt blocking. |
| `-dry-run` | Preview without writing or creating a token. |
| `-force` | Proceed even when managed (MDM) settings are present. |
| `-yes` | Apply without asking for confirmation. |
| `-no-backup` | Turn off first-write `.bak` copies. |
| `-log-file`, `-debug` | Logging. |

Explicit flags skip the matching wizard questions. `personal <agent>` runs the same form for one agent. Behaviour is described in the [README](../README.md#personal-mode-supported-agents).

### `update`

Same flag set as `personal`. It reapplies saved choices. Use `-yes` to skip the prompts.

### `status`

| Flag | Description |
|---|---|
| `-json` | Emit JSON instead of text. |

### `doctor`

| Flag | Description |
|---|---|
| `--json` | Emit the checks as JSON. Must be the first argument. |

Exits `1` if any check fails. The check list is in the [troubleshooting guide](troubleshooting.md#doctor-checks).

### `login`

Same flag set as `personal`. `login --mdm` (without `sudo`) reads the managed policy, signs in if needed, creates a PAT, and then discovers MCP servers and installs skills.

### `token`

| Flag | Description |
|---|---|
| `-mdm` | Use the organization's sign-in (the MDM policy on this machine), not personal mode's. |
| `-url`, `-tenant` | Control plane and tenant. |

### `config`

| Usage | Description |
|---|---|
| `config show [--json]` | Print the saved settings. |
| `config get <key>` | Print one value. |
| `config set <key> <value...>` | Change a value. Lists take comma- or space-separated values. |
| `config unset <key>` | Clear a value. |
| `config keys` | List every settable key with its description. |

Settings are stored in `~/.tf/config.toml`. Run `update` after a change to apply it. The full key list is in the [config reference](mdm-config-reference.md#personal-config-keys).

### `uninstall`

| Flag | Description |
|---|---|
| `-yes` | Do not ask for confirmation. |

Removes what personal setup installed: the TrueFoundry configuration, the Claude Desktop configuration that was applied before, and the skills setup installed (edited files are kept). Desktop uploads must be removed in the app. Managed machines are cleaned up through the MDM job, not this subcommand. See [MDM deployment](mdm-deployment.md#11-operational-notes).

### Internal subcommands

`_mdm-pat`, `_mdm-user` and `_scheduled-check` are internal. The MDM run and the personal hourly check start them. Don't call them by hand.

## Hidden and internal flags

These flags exist for the binary's own use and are not part of the public interface. They are omitted from `-h`'s description, but they appear in the help output for flags that exist as internal helpers.

| Flag | Used by |
|---|---|
| `--_get-token` | Legacy device auth. Prints the token to stdout. |
| `--_print-oidc-token` | Claude Code `apiKeyHelper`, Codex `auth.command`, Claude Desktop credential helper. Prints a fresh access token to stdout. |
| `--_silent` | With `--_print-oidc-token`. Returns a cached or refreshed token only. Never opens a browser. |

The client helpers run these flags for you, so the written config contains them. Don't pass them yourself.
