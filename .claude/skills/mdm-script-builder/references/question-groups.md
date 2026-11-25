# Question groups (mapping from the binary's catalog)

The binary is the source of truth. `scripts/catalog.sh <tag>` writes two files under `$MDM_CATALOG_DIR/<tag>/`:

- `help.txt`: the flag list from `tfy-local-ai-setup -h`. Each flag is a line `  -name` followed by its help text. The help text becomes the question's description.
- `config-keys.txt`: the personal-mode key list from `tfy-local-ai-setup config keys`.

Every flag and key must land in exactly one group below. A flag or key that is not in this table goes into **Other binary settings** and is asked as a plain question. A new binary release can therefore add a setting without the skill silently dropping it.

Internal flags are never asked: `_get-token`, `_print-oidc-token`, `_silent`.

## Groups

| # | Group | Flags (from `-h`) | Config keys (from `config keys`) |
|---|---|---|---|
| A | Connection | `url`, `tenant`, `gateway`, `claude-gateway`, `codex-gateway`, `claude-desktop-gateway` | `url`, `tenant`, `gateway` |
| B | Tools and sign-in | `claude-code`, `codex`, `claude-desktop`, `claude-code-auth`, `codex-auth`, `codex-auth-mode`, `claude-desktop-auth`, `oidc-issuer`, `oidc-client-id`, `oidc-bearer-token-type`, `oidc-scopes`, `agents-auth`, `login`, `pat-expiry-days`, `pat-team`, `auto-renew-key` | `tools`, `auth.pat_name`, `auth.oidc.issuer`, `auth.oidc.client_id` |
| C | Models | `opus-model`, `sonnet-model`, `haiku-model`, `desktop-opus-model`, `desktop-sonnet-model`, `desktop-haiku-model`, `codex-default-model`, `additional-model`, `desktop-additional-model`, `default-model`, `model` | `defaults.models`, `defaults.default_model`, `agents.claude-code.*_model`, `agents.claude-desktop.*_model`, `agents.codex.model`, `agents.<id>.models`, `agents.<id>.default_model` |
| D | MCP servers | `mcp-server`, `no-mcp`, `claude-desktop-allow-local-mcp` | `mcp.servers`, `agents.<id>.mcp_servers` |
| E | Agent Skills | `skill`, `no-skills` | `skills.names`, `agents.<id>.skills` |
| F | Claude Desktop extras | `claude-ai-import`, `import-export-enabled`, `import-banner-behavior`, `import-automatic-3p`, `import-encoding`, `disable-deployment-mode-chooser`, `force-login-org`, `desktop-header` | none |
| G | Files and templates | `settings-file`, `codex-settings-file`, `config`, `agents-config` | none |
| H | Prompt blocking and Cursor | `block-agent`, `unblock-agent`, `block-message`, `cursor-monitoring`, `cursor-require-login` | `agents.cursor.*` where listed |
| I | Other agents | `agent` | `agents.<id>.*` for OpenCode, Qwen Code, Grok Build, Pi, Goose, Copilot CLI, VS Code, Gemini CLI, Cursor |
| J | Failure handling, logs, dry run | `write-config-on-login-failure`, `placeholder-token`, `log-file`, `debug`, `refresh-token-file`, `access-token-file`, `dry-run` | none |
| K | Windows WSL | `wsl` | none |
| X | Other binary settings | any flag or key not in the table above | any key not in the table above |

Coverage check: the table covers 64 of the 67 flags in `-h` for v3.0.1. The other 3 are the internal flags above. Re-check the count on every new release.

## Question sets

### POC set (always asked, in this order)

Keep the POC set to six questions. Use defaults wherever a default exists.

1. **Control-plane URL** (group A `url`). Default: the `host` value from `~/.truefoundry/credentials.json`, if present. Otherwise ask.
2. **Tenant** (group A `tenant`). No default. Ask.
3. **Gateway URL** (group A `gateway`). Default: same as the control-plane URL.
4. **Tools** (group B `tools`, from the binary's `tools` key). Multi-select. Default: `claude-code` and `codex`.
5. **Claude model IDs** (group C `opus-model`, `sonnet-model`, `haiku-model`). Default: `claude-code/claude-opus`, `claude-code/claude-sonnet`, `claude-code/claude-haiku`. Ask only if the user says the tenant uses other IDs. Remind them to copy the exact IDs from the control plane.
6. **Target OS and first run** (groups J `dry-run`, plus the OS list). Multi-select macOS, Linux, Windows. Ask whether the first deployment should be a dry run. Default: yes.

POC fixed values, not asked: `auth.mode = "api-key"` (device login), no MCP servers, no skills, no OIDC, no restrictions, no Cursor monitoring, default release tag.

### Complete set

Ask groups A to K in order, one group per round. Skip any group the user declines in the advanced offer. Use the flag's help text from `help.txt` as the question description. Ask at most four questions per AskUserQuestion call.

### Advanced offer (after a POC)

After the POC scripts are written, offer the groups that the POC did not cover: B (OIDC and sign-in extras), C (extra models), D, E, F, G, H, I, J and K. Ask the user which to configure, as a multi-select. Ask questions only for the groups they choose. If they choose none, the POC output stands.

## Rules for specific settings

- **Secrets:** never ask for or write a PAT, access token, refresh token, client secret or API key. The OIDC client ID is public and may be written.
- **`force-login-org`:** a wrong UUID locks every user out of Claude Desktop sign-in. Ask for the UUID, then ask the user to confirm it against the Claude Admin Console.
- **`claude-enterprise/...` model IDs:** add a comment to the output reminding the admin that the Anthropic provider on TrueFoundry must have an empty API key.
- **`oidc` auth:** requires `oidc-issuer` and `oidc-client-id`. Remind the user to register the loopback redirect URIs `http://127.0.0.1:8585/callback`, `:8586` and `:8587` on the public client. Claude Desktop uses `http://127.0.0.1/callback`.
- **`claude-desktop-auth = oidc`:** requires Claude Desktop 2.7032.0 or later on every managed machine.
- **`dry-run`:** never put `--dry-run` into the production script. Only the preview run uses it.
- **`tools`:** always write it explicitly in `mdm.toml`. Without it the binary auto-detects installed clients.
- **`config` and `agents-config`:** these take a file path. Write the matching TOML alongside the generated `mdm.toml`, not inside it.
