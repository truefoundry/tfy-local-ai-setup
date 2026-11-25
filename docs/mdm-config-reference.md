# MDM config reference

The `--config` file tells a managed run what to configure. It is TOML or JSON, and it's read by the binary itself, not by your MDM. This page lists every key. For the complete commented example, see [`examples/mdm-reference.toml`](examples/mdm-reference.toml).

Two input formats are supported. Both are validated before any sign-in or write, and unknown keys are rejected.

| Format | Recognised by | Use it when |
|---|---|---|
| **Flat** | the file has `url`, `tenant`, `gateway` and `tools` | You are writing a file by hand, or following the [MDM deployment guide](mdm-deployment.md). This is the format the deploy scripts' examples use. |
| **Setup** | the file has `control_plane_url`, `tenant_name` and `gateway_url` | The file was produced by the TrueFoundry admin UI. Required top-level keys: `control_plane_url`, `tenant_name`, `gateway_url`, `agents`. |

Explicit command-line flags override values from the file. For example, `--claude-desktop=false` deselects Desktop even when the file lists it.

## Contents

- [Flat format](#flat-format)
- [Setup format](#setup-format)
- [Agent tables](#agent-tables)
- [Restrictions, monitoring and execution](#restrictions-monitoring-and-execution)
- [Settings that are flags only](#settings-that-are-flags-only)
- [Personal config keys](#personal-config-keys)

## Flat format

### Top level

| Key | Type | Default | Meaning |
|---|---|---|---|
| `url` | string | — | Control-plane URL. Required. |
| `tenant` | string | — | Tenant name. Required. |
| `gateway` | string | value of `url` | Gateway URL for all clients. |
| `tools` | list of strings | auto-detect | Clients to configure: `claude-code`, `codex`, `claude-desktop`, plus any additional agent ID. A present list configures exactly those. `tools = []` disables auto-detection and selects none. |

**Always set `tools` explicitly for a fleet.** Without it the binary auto-detects installed clients.

### `[auth]`

| Key | Values | Default | Meaning |
|---|---|---|---|
| `mode` | `api-key`, `oidc` | `api-key` | `api-key` is per-user device login with refresh. `oidc` uses the OIDC helpers. Clients can override it under their own `[agents.*]` table. |
| `login` | `manual` | — | Defers PAT creation to `login --mdm`, so scheduled runs never open a browser. Requires `mode = "api-key"`. |
| `pat_team` | string | — | Team that owns a PAT created for manual login. |
| `pat_expiry_days` | integer 7–3650 | `365` | Lifetime of each PAT. Existing PATs keep their date until they are renewed. |
| `auto_renew` | boolean | `false` | Renew each user's key from the scheduled run, before it expires, with no browser. |

### `[auth.oidc]`

| Key | Meaning |
|---|---|
| `issuer` | OIDC issuer URL for the tenant's public client. Required when any client uses `oidc`. |
| `client_id` | Public client ID. The same value for every machine. No client secret is used. |

### `[defaults]`

| Key | Meaning |
|---|---|
| `models` | Shared model list for agents with a model picker: Grok Build, Pi, Qwen Code, OpenCode, Gemini CLI, Copilot CLI, Goose and VS Code. Gateway model or Virtual Model IDs. |
| `default_model` | Default from that list. Must be in `models`. If omitted, the first model is used. |

Claude Code, Claude Desktop and Codex don't use `[defaults]`. They use their own model keys.

### `[mcp]`

| Key | Meaning |
|---|---|
| `servers` | Gateway MCP server names deployed to every selected client. `[]` means none. |
| `discovery` | `"all-accessible"`. Deploys every MCP server the signed-in user can use, refreshed on each run. Cannot be combined with a non-empty `servers` list. |

Per-agent `mcp_servers` replaces the shared list for that agent. `[]` disables MCP for that agent.

### `[skills]`

| Key | Meaning |
|---|---|
| `names` | Skills from the TrueFoundry Skills Registry, as `[repo/]name[:version]`. A pinned `:version` is recommended for rollouts. |
| `discovery` | `"all-accessible"`. Installs every published skill the user can access, following the latest versions. Cannot be combined with `names`. |

Per-agent `skills` replaces the shared list. Claude Desktop skills are packaged as ZIP files for manual upload in the app.

### `[execution]`

| Key | Default | Meaning |
|---|---|---|
| `write_config_on_login_failure` | `false` | When device login fails, still write and lock the managed config, then exit `1`. Without it, a failed login writes nothing. |
| `placeholder_token` | `TFY_PLACEHOLDER_LOGIN_REQUIRED` | Token written in that case when no cached token exists. Requests with it get `401`. |

### `[monitoring.cursor]`

| Key | Default | Meaning |
|---|---|---|
| `enabled` | `false` | Copy Cursor prompt, response, tool and session events to the gateway through hooks. Doesn't change model routing. `false` removes the monitoring hooks. |
| `require_login` | `false` | Block Cursor agent prompts and tool calls until an MDM credential is available. Applies only when `enabled` is true. |

### `[restrictions]`

Not supported in the standalone `--config` file. Use the flags `--block-agent`, `--unblock-agent` and `--block-message`, or `--agents-config`.

## Setup format

This format is produced by the admin UI. Field names differ from the flat format.

| Key | Type | Required | Meaning |
|---|---|---|---|
| `control_plane_url` | string | yes | Same as `url`. |
| `tenant_name` | string | yes | Same as `tenant`. |
| `gateway_url` | string | yes | Same as `gateway`. |
| `agents` | table | yes | Per-agent settings. See [agent tables](#agent-tables). The `agents` map is closed: only the listed agent IDs are accepted. |
| `mcp_servers` | list of strings | no | Shared MCP names. `["@all"]` means everything the user can access. |
| `skills` | list of strings | no | Shared skills, as `[repo/]name[:version]`. |
| `auth.mode` | `oidc` or `api-key` | no | Same as flat `[auth] mode`. |
| `auth.oidc.issuer`, `auth.oidc.client_id` | strings | when `auth.oidc` is set | Same as flat `[auth.oidc]`. |

The schema also accepts hand-added MDM settings, such as `[restrictions]`, `[monitoring]`, `auth.login` and per-agent `auth`, `mcp_servers` and `skills`. The `auth.oidc` table and the Claude `models` table are closed. Other keys are checked by the binary, which rejects unknown ones before it writes anything.

```toml
control_plane_url = "https://app.example.com"
tenant_name = "acme"
gateway_url = "https://gateway.example.com"
mcp_servers = ["@all"]
skills = ["eng-skills/code-review"]

[auth]
mode = "oidc"

[auth.oidc]
issuer = "https://login.example.com/acme"
client_id = "YOUR_PUBLIC_OIDC_CLIENT_ID"

[agents.claude-code.models]
opus = "anthropic-main/claude-opus-4-5"
sonnet = "anthropic-main/claude-sonnet-4-5"
haiku = "anthropic-main/claude-haiku-4-5"

[agents.codex]
model = "gpt-5.4"
```

## Agent tables

Each agent has its own `[agents.<id>]` table. Keys that don't apply to an agent are rejected.

### Claude Code: `[agents.claude-code]`

| Key | Meaning |
|---|---|
| `auth` | `oidc` or `api-key`. Overrides `[auth] mode` for this client. |
| `opus_model`, `sonnet_model`, `haiku_model` | One model per tier. Default `claude-code/claude-opus`, `claude-code/claude-sonnet`, `claude-code/claude-haiku`. |
| `additional_models` | Extra models in `/model`, after the tier models. Any gateway model. `["@all"]` offers every chat model the user can access, read on each run. |
| `mcp_servers`, `skills` | Replace the shared lists. `[]` disables. |
| `gateway` | Override the gateway for this client only. |
| `managed_settings` | Inline Claude Code managed-settings template (see below). |

`managed_settings` uses Claude Code's own setting names, including nested tables. The binary patches the gateway URL, auth, model IDs and MCP entries into it, and keeps everything else. An explicit `--settings-file` takes precedence over this table. Without either, the existing file on disk is the base, and failing that the built-in default is used.

```toml
[agents.claude-code.managed_settings]
allowManagedHooksOnly = true
transcriptRetentionDays = 14

[agents.claude-code.managed_settings.permissions]
disableBypassPermissionsMode = "disable"
deny = []
ask = ["Bash(git push:*)", "Read(**/.env)"]
```

### Codex: `[agents.codex]`

| Key | Meaning |
|---|---|
| `auth` | `oidc` or `api-key`. |
| `model` | The one default model. Written as the top-level `model`. Not written when omitted. |
| `auth_mode` | `api-key` (default) or `chatgpt-subscription`. Subscription mode sends the token as `x-tfy-api-key` and requires `auth = "api-key"`. |
| `gateway` | Override the gateway for Codex only. |
| `mcp_servers`, `skills` | Replace the shared lists. |

### Claude Desktop: `[agents.claude-desktop]`

| Key | Meaning |
|---|---|
| `auth` | `oidc` or `api-key`. Desktop OIDC requires Claude Desktop 2.7032.0 or later. |
| `opus_model`, `sonnet_model`, `haiku_model` | One model per tier. Defaults follow Claude Code's. |
| `additional_models` | Extra models in Desktop's menu. Desktop shows Claude models only. |
| `mcp_servers`, `skills` | Replace the shared lists. |
| `gateway` | Override the gateway for Desktop only. |
| `tool_search_enabled` | `true` or `false`. Writes `toolSearchEnabled`. |
| `prefer_1m_context` | `true` or `false`. Writes `modelPrefer1mContext`. |
| `context.opus`, `context.sonnet`, `context.haiku` | Per-tier capabilities: `supports_1m` and `prefer_1m`. `prefer_1m = true` requires `supports_1m = true` in the same entry. |
| `history_import` | Table: `enabled`, `encoding` (`json-string` or `native-dict`, macOS only), `export_enabled`, `banner_behavior` (`off`, `detect` or `show`), `automatic_3p`. Other keys take effect only when `enabled` is true. Requires a compatible app and workspace export permission. |

Desktop custom headers are set with the flag `--desktop-header 'Name: Value'`. The `headers` key under `[agents.claude-desktop]` is rejected in the current source.

### Other agents

Each agent below is written only when it is listed in `tools` or named with `--agent`. Agents that take a model list accept `models` and `default_model`. The default must be in the list.

| Agent ID | Keys beyond the common ones (`auth`, `models`, `default_model`, `mcp_servers`, `skills`) |
|---|---|
| `opencode` | `headers`, `context_window`, `max_tokens`, `gateway_only` |
| `qwen-code` | `api` (`chat-completions` or `responses`), `headers`, `context_window` |
| `grok-build` | `api` (`chat_completions`, `responses` or `messages`), `headers`, `context_window`, `max_tokens` |
| `pi` | `api` (`openai-completions`, `openai-responses` or `anthropic-messages`), `headers`, `context_window`, `max_tokens` |
| `gemini-cli` | `provider_account` (native Gemini through `<gateway>/gemini/<account>/proxy`), `headers`. Only the resolved default is written. |
| `copilot-cli` | `api` (`completions` or `responses`), `headers`, `context_window`, `max_tokens`. Only the resolved default is written. |
| `goose` | `headers`, `context_window` |
| `vscode` | `api` (`chat-completions`, `responses` or `messages`), `headers`, `context_window`, `max_tokens`, `utility_model` (`mainAgent`, `copilot` or `none`), `insiders` |
| `cursor` | `mcp_servers`, `skills` only. No model settings are written. |

All additional agents share one credential-helper auth mode in a run. Don't mix `api-key` and `oidc` across them.

## Restrictions, monitoring and execution

The `[monitoring.cursor]` and `[execution]` tables are described under [flat format](#monitoringcursor). Prompt blocking for Claude Code, Codex and Cursor is set by flags in the standalone file:

| Flag | Effect |
|---|---|
| `--block-agent claude-code,codex,cursor` | Blocks new prompts for the listed agents through their prompt-submission hooks. |
| `--unblock-agent cursor` | Removes this setup's block for that agent. |
| `--block-message 'Contact IT.'` | Message shown when a prompt is blocked. |

## Settings that are flags only

These have no key in the file. Pass them on the command line.

| Flag | Purpose |
|---|---|
| `--dry-run` | Print the resulting configuration instead of writing it. |
| `--log-file PATH` | Append log output to a file, mode `0600`. |
| `--debug` | Verbose diagnostics. |
| `--refresh-token-file PATH`, `--access-token-file PATH` | Override the token cache paths for one run. |
| `--settings-file PATH` | Claude Code managed-settings template. |
| `--codex-settings-file PATH` | Codex config template. |
| `--force-login-org UUID` | Restrict Claude Desktop sign-in to a Claude org. Repeatable. A wrong UUID locks users out. |
| `--claude-desktop-allow-local-mcp=false` | Allow only remote MCP servers in Desktop. |
| `--disable-deployment-mode-chooser` | Hide the Claude.ai escape in Desktop. Applies with Desktop OIDC only. |
| `--oidc-scopes 'openid profile email offline_access'` | Desktop OIDC scopes. Keep `openid`. |
| `--oidc-bearer-token-type access_token` | Desktop bearer token type. Keep the default. The gateway rejects `id_token`. |
| `--wsl` | Also configure the logged-in user's WSL distros. Windows only. |
| `--no-mcp`, `--no-skills` | Suppress shared MCP servers or skills for the run. Per-agent settings still apply. |
| `--agents-auth oidc` | Override the shared auth for additional agents. |
| `--claude-gateway`, `--codex-gateway`, `--claude-desktop-gateway` | Per-client gateways, as flags. |
| `--claude-ai-import`, `--import-export-enabled`, `--import-banner-behavior`, `--import-automatic-3p`, `--import-encoding` | Claude Desktop history import, as flags. |
| `--write-config-on-login-failure`, `--placeholder-token`, `--cursor-monitoring`, `--cursor-require-login` | Also available as keys in the file. The flag overrides the file. |

`--agents-config` is a separate input mode. It cannot be combined with `--config`.

## Personal config keys

These keys are for personal mode: `tfy-local-ai-setup config set`, `config get` and `config unset`. They are also what `--agents-config` accepts. The list below is generated from the output of `tfy-local-ai-setup config keys` for v3.0.1.

| Key | Meaning |
|---|---|
| `url` | Control-plane URL |
| `tenant` | Tenant name |
| `gateway` | Gateway URL the agents use |
| `tools` | Agents to configure: claude-code, codex, claude-desktop, grok-build, pi, qwen-code, opencode, gemini-cli, copilot-cli, goose, vscode, cursor |
| `auth.pat_team` | Team owner for a newly created Personal Access Token |
| `auth.pat_name` | Personal Access Token the agents use (reused if it exists, else created) |
| `auth.pat_expiry_days` | Days a Personal Access Token setup creates or renews lasts (7-3650, default 365; existing keys keep their date) |
| `defaults.models` | Models offered by Grok Build, Pi, Qwen Code, OpenCode, Gemini CLI, Copilot CLI, Goose, VS Code |
| `defaults.default_model` | Default model for Grok Build, Pi, Qwen Code, OpenCode, Gemini CLI, Copilot CLI, Goose, VS Code |
| `mcp.servers` | MCP server names |
| `mcp.discovery` | all-accessible: keep MCP servers up to date with what you can use (set by the desktop app) |
| `skills.names` | Agent Skills from the TrueFoundry Skills Registry: [repo/]name[:version] |
| `skills.discovery` | all-accessible: keep Agent Skills up to date with what you can use (set by the desktop app) |
| `auth.oidc.issuer` | OIDC issuer URL the agents sign in with (Claude Code, Codex, Claude Desktop) |
| `auth.oidc.client_id` | OIDC client ID the agents sign in with (Claude Code, Codex, Claude Desktop) |
| `agents.claude-code.additional_models Claude Code: Model offered in the model picker besides the tier models (repeatable)` |  |
| `agents.claude-code.mcp_servers` | Claude Code: MCP servers for this agent; unset inherits, empty disables |
| `agents.claude-code.skills` | Claude Code: Agent Skills for this agent ([repo/]name[:version]); unset inherits, empty disables |
| `agents.claude-code.opus_model` | Claude Code: Model used by this agent |
| `agents.claude-code.sonnet_model` | Claude Code: Model used by this agent |
| `agents.claude-code.haiku_model` | Claude Code: Model used by this agent |
| `agents.codex.mcp_servers` | Codex: MCP servers for this agent; unset inherits, empty disables |
| `agents.codex.skills` | Codex: Agent Skills for this agent ([repo/]name[:version]); unset inherits, empty disables |
| `agents.codex.model` | Codex: Model used by this agent |
| `agents.claude-desktop.additional_models Claude Desktop: Model offered in the model picker besides the tier models (repeatable)` |  |
| `agents.claude-desktop.mcp_servers Claude Desktop: MCP servers for this agent; unset inherits, empty disables` |  |
| `agents.claude-desktop.skills` | Claude Desktop: Agent Skills for this agent ([repo/]name[:version]); unset inherits, empty disables |
| `agents.claude-desktop.opus_model Claude Desktop: Model used by this agent` |  |
| `agents.claude-desktop.sonnet_model Claude Desktop: Model used by this agent` |  |
| `agents.claude-desktop.haiku_model Claude Desktop: Model used by this agent` |  |
| `agents.grok-build.api` | Grok Build: Wire protocol the agent speaks to the gateway |
| `agents.grok-build.headers` | Grok Build: Extra request header as 'Name: Value' (repeatable), e.g. X-TFY-METADATA |
| `agents.grok-build.context_window Grok Build: Context window (tokens) for every model, instead of each model's own from TrueFoundry` |  |
| `agents.grok-build.max_tokens` | Grok Build: Max output tokens for every model, instead of each model's own from TrueFoundry |
| `agents.grok-build.mcp_servers` | Grok Build: MCP servers for this agent; unset inherits, empty disables |
| `agents.grok-build.skills` | Grok Build: Agent Skills for this agent ([repo/]name[:version]); unset inherits, empty disables |
| `agents.grok-build.models` | Grok Build: Models offered by this agent; unset to inherit shared defaults |
| `agents.grok-build.default_model` | Grok Build: Default model for this agent |
| `agents.pi.api` | Pi: Wire protocol the agent speaks to the gateway |
| `agents.pi.headers` | Pi: Extra request header as 'Name: Value' (repeatable), e.g. X-TFY-METADATA |
| `agents.pi.context_window` | Pi: Context window (tokens) for every model, instead of each model's own from TrueFoundry |
| `agents.pi.max_tokens` | Pi: Max output tokens for every model, instead of each model's own from TrueFoundry |
| `agents.pi.mcp_servers` | Pi: MCP servers for this agent; unset inherits, empty disables |
| `agents.pi.skills` | Pi: Agent Skills for this agent ([repo/]name[:version]); unset inherits, empty disables |
| `agents.pi.models` | Pi: Models offered by this agent; unset to inherit shared defaults |
| `agents.pi.default_model` | Pi: Default model for this agent |
| `agents.qwen-code.api` | Qwen Code: Wire protocol the agent speaks to the gateway |
| `agents.qwen-code.headers` | Qwen Code: Extra request header as 'Name: Value' (repeatable), e.g. X-TFY-METADATA |
| `agents.qwen-code.context_window` | Qwen Code: Context window (tokens) for every model, instead of each model's own from TrueFoundry |
| `agents.qwen-code.mcp_servers` | Qwen Code: MCP servers for this agent; unset inherits, empty disables |
| `agents.qwen-code.skills` | Qwen Code: Agent Skills for this agent ([repo/]name[:version]); unset inherits, empty disables |
| `agents.qwen-code.models` | Qwen Code: Models offered by this agent; unset to inherit shared defaults |
| `agents.qwen-code.default_model` | Qwen Code: Default model for this agent |
| `agents.opencode.headers` | OpenCode: Extra request header as 'Name: Value' (repeatable), e.g. X-TFY-METADATA |
| `agents.opencode.context_window` | OpenCode: Context window (tokens) for every model, instead of each model's own from TrueFoundry |
| `agents.opencode.max_tokens` | OpenCode: Max output tokens for every model, instead of each model's own from TrueFoundry |
| `agents.opencode.gateway_only` | OpenCode: Hide every other provider so only the gateway can be picked |
| `agents.opencode.mcp_servers` | OpenCode: MCP servers for this agent; unset inherits, empty disables |
| `agents.opencode.skills` | OpenCode: Agent Skills for this agent ([repo/]name[:version]); unset inherits, empty disables |
| `agents.opencode.models` | OpenCode: Models offered by this agent; unset to inherit shared defaults |
| `agents.opencode.default_model` | OpenCode: Default model for this agent |
| `agents.gemini-cli.provider_account Gemini CLI: Gemini provider account on the gateway: native Gemini models through <gateway>/gemini/<account>/proxy` |  |
| `agents.gemini-cli.headers` | Gemini CLI: Extra request header as 'Name: Value' (repeatable), e.g. X-TFY-METADATA |
| `agents.gemini-cli.mcp_servers` | Gemini CLI: MCP servers for this agent; unset inherits, empty disables |
| `agents.gemini-cli.skills` | Gemini CLI: Agent Skills for this agent ([repo/]name[:version]); unset inherits, empty disables |
| `agents.gemini-cli.models` | Gemini CLI: Models offered by this agent; unset to inherit shared defaults |
| `agents.gemini-cli.default_model` | Gemini CLI: Default model for this agent |
| `agents.copilot-cli.api` | Copilot CLI: Wire protocol the agent speaks to the gateway |
| `agents.copilot-cli.headers` | Copilot CLI: Extra request header as 'Name: Value' (repeatable), e.g. X-TFY-METADATA |
| `agents.copilot-cli.context_window Copilot CLI: Context window (tokens) for every model, instead of each model's own from TrueFoundry` |  |
| `agents.copilot-cli.max_tokens` | Copilot CLI: Max output tokens for every model, instead of each model's own from TrueFoundry |
| `agents.copilot-cli.mcp_servers` | Copilot CLI: MCP servers for this agent; unset inherits, empty disables |
| `agents.copilot-cli.skills` | Copilot CLI: Agent Skills for this agent ([repo/]name[:version]); unset inherits, empty disables |
| `agents.copilot-cli.models` | Copilot CLI: Models offered by this agent; unset to inherit shared defaults |
| `agents.copilot-cli.default_model Copilot CLI: Default model for this agent` |  |
| `agents.goose.headers` | Goose: Extra request header as 'Name: Value' (repeatable), e.g. X-TFY-METADATA |
| `agents.goose.context_window` | Goose: Context window (tokens) for every model, instead of each model's own from TrueFoundry |
| `agents.goose.mcp_servers` | Goose: MCP servers for this agent; unset inherits, empty disables |
| `agents.goose.skills` | Goose: Agent Skills for this agent ([repo/]name[:version]); unset inherits, empty disables |
| `agents.goose.models` | Goose: Models offered by this agent; unset to inherit shared defaults |
| `agents.goose.default_model` | Goose: Default model for this agent |
| `agents.vscode.api` | VS Code: Wire protocol the agent speaks to the gateway |
| `agents.vscode.headers` | VS Code: Extra request header as 'Name: Value' (repeatable), e.g. X-TFY-METADATA |
| `agents.vscode.context_window` | VS Code: Context window (tokens) for every model, instead of each model's own from TrueFoundry |
| `agents.vscode.max_tokens` | VS Code: Max output tokens for every model, instead of each model's own from TrueFoundry |
| `agents.vscode.utility_model` | VS Code: What VS Code uses for background tasks: mainAgent (the gateway model you chat with), copilot, or none |
| `agents.vscode.insiders` | VS Code: Configure VS Code Insiders instead of VS Code |
| `agents.vscode.mcp_servers` | VS Code: MCP servers for this agent; unset inherits, empty disables |
| `agents.vscode.skills` | VS Code: Agent Skills for this agent ([repo/]name[:version]); unset inherits, empty disables |
| `agents.vscode.models` | VS Code: Models offered by this agent; unset to inherit shared defaults |
| `agents.vscode.default_model` | VS Code: Default model for this agent |
| `agents.cursor.mcp_servers` | Cursor: MCP servers for this agent; unset inherits, empty disables |
| `agents.cursor.skills` | Cursor: Agent Skills for this agent ([repo/]name[:version]); unset inherits, empty disables |

Changes apply after `tfy-local-ai-setup update`.
