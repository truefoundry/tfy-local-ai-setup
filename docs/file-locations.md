# File locations

Where `tfy-local-ai-setup` reads and writes, by OS. Paths are from the v3.0.1 source. Some agents also honour their own override variables, noted where they apply.

## Contents

- [Managed config (root or SYSTEM)](#managed-config-root-or-system)
- [Policy and MDM input](#policy-and-mdm-input)
- [Personal and per-user state](#personal-and-per-user-state)
- [Credentials](#credentials)
- [Logs](#logs)
- [Run reports](#run-reports)
- [Scheduled jobs](#scheduled-jobs)

## Managed config (root or SYSTEM)

The managed run writes these, then locks them. See [security](security.md#managed-files).

### Claude Code

| OS | Path |
|---|---|
| macOS | `/Library/Application Support/ClaudeCode/managed-settings.json` |
| Linux | `/etc/claude-code/managed-settings.json` |
| Windows | `C:\Program Files\ClaudeCode\managed-settings.json` |

### Codex

| OS | Path |
|---|---|
| macOS, Linux | `/etc/codex/managed_config.toml` |
| Windows | `C:\ProgramData\OpenAI\Codex\requirements.toml` (enforced, Codex 0.156.0 or later) and `C:\ProgramData\OpenAI\Codex\config.toml` (defaults). The whole directory is locked. |

### Claude Desktop

| OS | Path |
|---|---|
| macOS | `/Library/Managed Preferences/com.anthropic.claudefordesktop.plist` (the managed-preferences plist for the user) |
| Windows | `HKLM\SOFTWARE\Policies\Claude` (string values) |
| Linux | Not supported. Claude Desktop 3P doesn't run on Linux. |

Claude Desktop reads these at launch. A change takes effect when the user restarts the app.

### Other agents

Written when the agent is named with `--agent`. Files marked *managed* are root-owned, locked and installed by MDM.

| Agent | macOS | Linux | Windows |
|---|---|---|---|
| OpenCode (managed) | `/Library/Application Support/opencode/` | `/etc/opencode/` | `%ProgramData%\opencode\` |
| Qwen Code (managed) | `/Library/Application Support/QwenCode/settings.json` | `/etc/qwen-code/` | `%ProgramData%\qwen-code\` |
| Gemini CLI (managed) | `/Library/Application Support/GeminiCli/settings.json` | `/etc/gemini-cli/` | `%ProgramData%\gemini-cli\` |
| Grok Build (managed on macOS and Linux) | `/etc/grok/requirements.toml`, `/etc/grok/managed_config.toml` | same | user config only |
| Copilot CLI | `/etc/github-copilot/managed-settings.json` | — | — |
| Cursor (hooks) | `/Library/Application Support/Cursor/hooks.json` | `/etc/cursor/hooks.json` | `%ProgramData%\Cursor\hooks.json` |
| Codex (prompt-block hook) | `/etc/codex/hooks.json` | `/etc/codex/hooks.json` | `%ProgramData%\OpenAI\Codex\hooks.json` |
| Claude Code (prompt-block hook) | the platform managed-settings path above | same | same |

Pi, Goose, VS Code and Cursor's model settings are user-level only. They are written by personal mode and by the user-context run. See the [README](../README.md#other-agents).

## Policy and MDM input

| File | Written by | Path |
|---|---|---|
| Agents policy (what IT configured) | Root, during the managed run | macOS `/Library/Application Support/TrueFoundry/agents-policy.json`; Linux `/etc/truefoundry/agents-policy.json`; Windows `%ProgramFiles%\TrueFoundry\agents-policy.json` |
| MCP policy | The administrator (read by the binary) | macOS `/Library/Application Support/TrueFoundry/policy.json`; Linux `/etc/truefoundry/policy.json`; Windows `%ProgramData%\TrueFoundry\policy.json` |
| MDM config (`--config` input) | You, through your MDM script | macOS `/Library/Application Support/TrueFoundry/mdm.toml`; Linux `/etc/truefoundry/mdm.toml`; Windows `C:\Program Files\TrueFoundry\mdm.toml` |

The MDM config path is the one the deploy scripts in the [MDM deployment guide](mdm-deployment.md) use. The binary reads any path you pass to `--config`.

## Personal and per-user state

All of these live under the user's home. Directories are `0700`, and files are `0600` unless noted.

| Path | Holds |
|---|---|
| `~/.tf/config.toml` | Saved personal settings. Written by `personal`, `config set` and `update`. Not a credential store. |
| `~/.tf/personal.json` | State that personal mode uses to track what it installed, with SHA-256 records of the files it wrote. |
| `~/.tf/refresh-token` | Device-login refresh token (managed mode, api-key). |
| `~/.tf/access-token` | Device-login access-token cache. |
| `~/.tf/oidc-refresh-token`, `~/.tf/oidc-access-token` | OIDC helper caches. |
| `~/.tf/mdm-user.json` | The managed run's user-context record: applied settings, pending work, and installed skills, kept apart from personal state. |
| `~/.tf/debug.log` | Wizard debug output when `--log-file` isn't given. |
| `~/.tf/backups/claude-desktop/` | One-time copy of Claude Desktop's `_meta.json`, and `.bak` copies made on first write when `--no-backup` isn't passed. |
| `~/.tf/claude-desktop-truefoundry.json` | Written for **Import configuration** in Claude Desktop when the library's shape isn't recognised. |
| `~/.tf/skills/claude-desktop/<name>.zip` | Skills packaged for manual upload in Claude Desktop. |
| `~/.tf/cache/skills/<tenant>/<version-id>` | Downloaded skill cache. |
| `~/.tf/state/runs/` | Personal run reports. |
| `~/.tf/token-command` | Inside a WSL distro, the token helper command for agents. |

### Agent-owned user config

Written by personal mode or the user-context run. The agent reads it on its own:

| Agent | Path |
|---|---|
| Claude Code | `~/.claude/settings.json` (or `$CLAUDE_CONFIG_DIR`); MCP servers in `~/.claude.json` |
| Codex | `~/.codex/config.toml` (or `$CODEX_HOME`) |
| Claude Desktop | `~/Library/Application Support/Claude-3p/configLibrary/` (macOS), `%LOCALAPPDATA%\Claude-3p\configLibrary\` (Windows), `~/.config/Claude-3p/configLibrary/` (Linux) |
| Grok Build | `~/.grok/config.toml` |
| Pi | `~/.pi/agent/models.json`, `settings.json`, `mcp.json` |
| Qwen Code | `~/.qwen/settings.json` |
| OpenCode | `~/.config/opencode/opencode.json` |
| Gemini CLI | `~/.gemini/settings.json` |
| Copilot CLI | `~/.copilot/mcp-config.json` |
| Goose | `~/.config/goose/custom_providers/truefoundry.json`, `config.yaml` |
| VS Code | `chatLanguageModels.json` and `mcp.json` (see the README for the VS Code paths) |
| Cursor | `~/.cursor/mcp.json`, and `~/.cursor/hooks.json` for prompt blocking |

Agent skills go to each agent's own skills folder, for example `~/.claude/skills` or `~/.codex/skills`. Setup only changes folders it installed into, and only files that still match what it wrote.

## Credentials

| Store | Where |
|---|---|
| OS credential store | Service `truefoundry-ai-setup`, account `<tenant>:<kind>`. macOS Keychain, Windows Credential Manager, Linux Secret Service. |
| Fallback file | `~/.tf/` under mode `0600`. Used when no credential store is available. The binary warns when it falls back. |

See [security](security.md#credentials) for what is and isn't protected.

## Logs

| Log | Path | Enabled by |
|---|---|---|
| MDM run | macOS and Linux `/var/log/tfy-mdm-setup.log`; Windows `C:\Program Files\TrueFoundry\mdm-setup.log` | `--log-file`, set by the deploy scripts |
| Personal debug | `~/.tf/debug.log` | `--debug` in the wizard, when `--log-file` isn't given |
| Any run | the path you pass to `--log-file` | `--log-file` (mode `0600`) |
| Desktop app | macOS `~/Library/Logs/<identifier>/app.log`; Windows `%LOCALAPPDATA%\<identifier>\logs\app.log`; Linux `~/.local/share/<identifier>/logs/app.log` | Always written by the app |

## Run reports

| Scope | Path |
|---|---|
| Personal | `~/.tf/state/runs/run-<id>.json` |
| Managed (macOS) | `/Library/Application Support/TrueFoundry/state/<user>/runs/` |
| Managed (Windows) | `%ProgramData%\TrueFoundry\state\<user>\runs\` |
| Managed (Linux) | `/var/lib/truefoundry/state/<user>/runs/` |

The newest 48 reports are kept. See [troubleshooting](troubleshooting.md#where-logs-are) for the contents.

## Scheduled jobs

The binary can install its own schedules. The MDM job is the usual scheduler for managed machines.

| Platform | Personal check | Admin run | Run-now trigger |
|---|---|---|---|
| macOS | `~/Library/LaunchAgents/com.truefoundry.ai-setup.check.plist` | `/Library/LaunchDaemons/com.truefoundry.ai-setup.plist` | `com.truefoundry.ai-setup.trigger`, watching `/Library/Application Support/TrueFoundry/spool/run-now` |
| Linux | `~/.config/systemd/user/truefoundry-ai-setup-check.{service,timer}` | `/etc/systemd/system/truefoundry-ai-setup.{service,timer}` | `truefoundry-ai-setup-run-now.{path,service}`, watching `/var/lib/truefoundry/spool/run-now` |
| Windows | Task `\TrueFoundry AI Setup Check` (user) | Task `\TrueFoundry AI Setup` (SYSTEM) | Task `\TrueFoundry AI Setup Run Now` |

See [troubleshooting](troubleshooting.md#scheduling) for the schedules' intervals.
