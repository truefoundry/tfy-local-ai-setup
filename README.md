# tfy-local-ai-setup

Tools for deploying TrueFoundry-managed AI clients — **Claude Code**, **Codex**, **Claude Desktop (Cowork on 3P)**, and other supported agents — on developer machines via MDM (Mobile Device Management), and for setting the same agents up on a personal machine with no MDM.

## Documentation

| If you want to… | Read |
|---|---|
| Deploy to a managed fleet | [MDM deployment guide](docs/mdm-deployment.md) |
| Write or check `mdm.toml` | [MDM config reference](docs/mdm-config-reference.md) |
| Look up a command or flag | [CLI reference](docs/cli-reference.md) |
| Find where a file, lock, log or secret lives | [File locations](docs/file-locations.md) |
| Understand the security model | [Security](docs/security.md) |
| Fix an error or read a diagnostic | [Troubleshooting](docs/troubleshooting.md) |
| Set up one developer's machine (no MDM) | [Personal mode](#personal-mode-supported-agents) |
| Build, test or release the tool | [Development](docs/development.md) |
| See the desktop app | [Desktop app](docs/desktop-app.md) |

**Which mode do I need?**

- **Managed (MDM):** IT deploys one job to every machine. It runs as root or SYSTEM and writes locked, org-wide config. Start with the [MDM deployment guide](docs/mdm-deployment.md).
- **Personal:** a developer runs `tfy-local-ai-setup personal` as themselves. Nothing is locked, no root is needed, and no token is stored in a config file. Start with [Personal mode](#personal-mode-supported-agents).

**Quick start (personal):**

```bash
curl -fsSL https://github.com/truefoundry/tfy-local-ai-setup/releases/latest/download/install.sh | sh
tfy-local-ai-setup personal
```

**Quick start (MDM, dry run on one machine):**

```bash
sudo tfy-local-ai-setup --config=/path/to/mdm.toml --dry-run
```

## tfy-local-ai-setup

A single binary that configures the TrueFoundry AI gateway for the agents on a machine: Claude Code, Codex, Claude Desktop, and the additional agents listed under [Other agents](#other-agents). It authenticates the logged-in developer with TrueFoundry, builds each agent's managed config, and locks the file so developers cannot change it. Designed to run on a schedule (hourly) via Jamf, Mosyle, Kandji, Intune, SCCM, or any MDM that supports script execution.

Subcommands for personal setup and day-to-day operation (`personal`, `update`, `status`, `doctor`, `login`, `token`, `config`, `uninstall`) are listed in the [CLI reference](docs/cli-reference.md).

### How it works

On every run the binary does five things:

1. **Decides which tools to configure** — an explicit `--claude-code` / `--codex` / `--claude-desktop` is taken at face value and **configured whether or not the tool is installed**; detection does not even run. With no tool flag it auto-detects what is present: `claude` on `PATH`; Codex via the `codex` CLI on `PATH` **or** the Codex desktop app — on macOS matched by bundle ID `com.openai.codex` (OpenAI ships it as both `Codex.app` and `ChatGPT.app`), on Windows by the Store/MSIX package identity `OpenAI.Codex` (`Get-AppxPackage -AllUsers` / provisioned); Claude Desktop via `Claude.app` on macOS and the `Claude` MSIX/Appx package (`Get-AppxPackage -AllUsers` / provisioned) on Windows. If auto-detect finds nothing, the binary exits **0 immediately** (no token fetch, no login popup) so hourly MDM runs on machines without any AI client are a clean no-op.
2. **Detects the logged-in user** — platform-specific: `scutil` on macOS, `SUDO_USER`/`logname` on Linux, WMI on Windows.
3. **Auth per client** — Claude Code, Codex, and Claude Desktop default to **api-key** (this binary fetches a device-login JWT and writes it into config). OIDC is **opt-in** via `--claude-code-auth=oidc` (`apiKeyHelper` → `--_print-oidc-token`), `--codex-auth=oidc` (`auth.command` → `--_print-oidc-token`), or `--claude-desktop-auth=oidc` (native `inferenceIdpOidc`).
4. **Writes managed configs** — Claude Code: `ANTHROPIC_BASE_URL`, model IDs, and either `X-TFY-API-KEY` (api-key) or `apiKeyHelper` (oidc). Desktop: managed preferences / registry. Codex: gateway + either JWT `http_headers` (api-key) or OIDC `auth.command` (oidc). Existing templates are patched in-place where applicable.
5. **Locks the file** — `chflags schg` on macOS, `chattr +i` on Linux, `icacls` ACL on Windows. Developers cannot modify the file without root/Administrator access. For Codex on Windows the `C:\ProgramData\OpenAI\Codex` directory is locked too (owner `Administrators`, inheritance stripped, `Users` read-only) — ProgramData otherwise lets any authenticated user create files there, and Codex will load a squatted config without refusing it.

### Destination paths

| OS      | Managed settings path |
|---------|-----------------------|
| macOS   | `/Library/Application Support/ClaudeCode/managed-settings.json` |
| Linux   | `/etc/claude-code/managed-settings.json` |
| Windows | `C:\Program Files\ClaudeCode\managed-settings.json` |

---

## Personal mode: supported agents

`tfy-local-ai-setup personal` is the unprivileged setup for a developer without MDM: it writes each agent's *user* config, never a token. Agents read the Personal Access Token from `$TFY_AGENT_TOKEN` (exported by a block in your shell rc, which calls `tfy-local-ai-setup token`; for bash on macOS that is the first of `~/.bash_profile`, `~/.bash_login` and `~/.profile` that exists, as a login shell reads only that one) or run `tfy-local-ai-setup token` themselves: Claude Code through `apiKeyHelper`, and Codex through `[model_providers.truefoundry.auth]`, so the Codex desktop app, which is not started from a shell, gets the token too.

| Agent | `personal <agent>` | What gets written | Advanced flags |
|-------|--------------------|-------------------|----------------|
| Claude Code | `claude-code` | `~/.claude/settings.json`; MCP servers in `mcpServers` of `~/.claude.json` (user scope, the key `claude mcp add --scope user` writes; edited under Claude Code's own lock, rest of the file untouched) | — |
| Codex | `codex` | `~/.codex/config.toml` | — |
| Claude Desktop | `claude-desktop` | A saved configuration named "TrueFoundry" in Desktop's local `configLibrary/`, made the applied one (see below) | — |
| Grok Build | `grok-build` (`grok`) | `~/.grok/config.toml`: a `[model.tfy-…]` per model, `[models] default`, `[mcp_servers.*]` | `--api chat_completions\|responses\|messages`, `--header`, `--context-window`, `--max-tokens` |
| Pi | `pi` | `~/.pi/agent/models.json`, `settings.json`, `mcp.json` | `--api openai-completions\|openai-responses\|anthropic-messages`, `--header`, `--context-window`, `--max-tokens` |
| Qwen Code | `qwen-code` (`qwen`) | `~/.qwen/settings.json`: `modelProviders.openai`, auth type, `mcpServers` | `--api chat-completions\|responses`, `--header`, `--context-window` |
| OpenCode | `opencode` | `~/.config/opencode/opencode.json` (or `.jsonc`) | `--header`, `--context-window`, `--max-tokens`, `--gateway-only` |
| Gemini CLI | `gemini-cli` (`gemini`) | `~/.gemini/settings.json` + `GOOGLE_GEMINI_BASE_URL`/`GEMINI_API_KEY` in the shell block | `--provider-account` (native Gemini via `/gemini/<account>/proxy`), `--header` |
| Copilot CLI | `copilot-cli` (`copilot`) | `COPILOT_PROVIDER_*` in the shell block, `~/.copilot/mcp-config.json` | `--api completions\|responses`, `--header` (one), `--context-window`, `--max-tokens` |
| Goose | `goose` | `~/.config/goose/custom_providers/truefoundry.json`, `config.yaml` | `--header`, `--context-window` |
| VS Code | `vscode` (`code`) | `chatLanguageModels.json` (Copilot Chat Custom Endpoint), `chat.byokUtilityModelDefault`, `mcp.json` — the key is pasted once in VS Code (it keeps keys in its own secret storage) | `--api chat-completions\|responses\|messages`, `--header`, `--context-window`, `--max-tokens`, `--utility-model`, `--insiders` |
| Cursor | `cursor` | `~/.cursor/mcp.json` only — Cursor's model base URL and key are UI-only | — |

```bash
tfy-local-ai-setup personal                         # wizard for every detected agent
tfy-local-ai-setup personal pi                      # edit Pi with saved choices prefilled
tfy-local-ai-setup personal grok --model openai-main/gpt-4o --header 'X-TFY-METADATA: {"app":"grok"}'
tfy-local-ai-setup config set agents.pi.context_window 200000 && tfy-local-ai-setup update
```

**More models in the Claude pickers.** Claude Code and Claude Desktop take one model per tier (opus, sonnet, haiku). To offer more models, set `additional_models` under `[agents.claude-code]` in `~/.tf/config.toml` or the MDM config, or pass `--additional-model` (`personal claude-code --additional-model tfy-ai-openai/gpt-5.5`, or `config set agents.claude-code.additional_models …`); the setup form doesn't ask for it. Claude Code then gets a `modelPicker` in its settings: `/model` lists the tier models followed by the additional models, any gateway model, Claude or not, and nothing else (the built-in lineup and the gateway's discovered models are hidden). Rows you add to `modelPicker` yourself are kept. Claude Desktop gets the same models after its tier entries in `inferenceModels` (its own `additional_models` under `[agents.claude-desktop]` wins when it lists models); Desktop shows Claude models only, so the run names the ones it hides. `additional_models = ["@all"]` offers every chat model the user can access instead, read from TrueFoundry on each setup or MDM run (Claude models first, then the rest by name; MDM uses the user's own sign-in, OIDC included). If they can't be listed, `/model` keeps its current list until the next run. Without `additional_models`, nothing changes.

**Context windows.** Agents that declare a context window per model (OpenCode, Qwen Code, Grok Build, Pi, Goose, VS Code, and Copilot CLI for its one model) get each model's own context window and output limit, read from the control plane when setup runs (TrueFoundry's model catalog, the same limits the gateway enforces; a Virtual Model gets the smallest of its targets). A model the catalog doesn't cover, or a run that can't reach the control plane, gets 128,000 context and 16,384 output. `--context-window` / `--max-tokens` (or `agents.<agent>.context_window` / `max_tokens`) set one value for every model of that agent and always win. Limits are read once per setup run, not refreshed in the background: run `update` to pick up a change. Under MDM the same agents get the same per-model limits, read on every MDM run: as the signed-in user (device login, OIDC or the managed key), and by the root run too when it holds the user's token (api-key mode). A failed lookup keeps the last limits a run read (saved in `~/.tf/mdm-user.json`); only a machine that never had a lookup gets the defaults.

The wizard signs in (reusing the saved URL and session; press **esc** on the agent list to sign in to a different tenant), asks which detected agents to set up and which gateway to use, then goes step by step: first the **MCP servers and Agent Skills** every selected agent gets, then **one agent at a time** for its models (Claude Code, Claude Desktop, Codex, then the rest; an agent with no models to choose, like Cursor, has no step). The checklist above fills in as each step is done, and **esc** goes back a step with its answers kept. Nothing is written along the way. The steps are drawn from the personal setup schema: its fields, labels, help text and where each list comes from. Advanced options (`api`, `headers`, context windows, per-agent MCP lists) are never asked; set them with flags, `config set` or `--config`, and the form leaves them alone. Saved choices are prefilled. Enter on a setting opens a search list right under it; Claude Desktop follows Claude Code's tiers, and agents with a model list follow the first one, until you change them. A Claude tier left unset uses gateway model discovery. On a first run (nothing saved, and no `--mcp-server`, `--no-mcp` or `--config`), every MCP server the gateway lists starts ticked; **Everything I have access to** keeps the list up to date as servers are added. The model pickers list only chat and responses models when the gateway supports the `model_types` filter. For Codex, a model that isn't listed needs a Virtual Model whose slug is an OpenAI model id; the list links to the guide. **Continue** checks each step's answers against the schema (a missing Codex model, say, keeps you on Codex with the cursor on it). On the last step, **Review changes** shows every file that will change, together, and asks once before writing anything. Cancelling at any point writes nothing.

`personal <agent>` runs the same form for one agent with saved choices prefilled; its model, MCP and skill edits apply only to that agent. Explicit flags or values supplied through `--config` skip the corresponding questions. `update` reapplies saved choices; `--yes` uses saved choices without opening the editor. Connection and authentication settings remain shared.

Both the TOML and JSON formats accept an existing PAT:

```toml
[auth]
pat_token = "YOUR_EXISTING_PAT"
```

`auth.pat_token` selects PAT authentication for all selected agents and takes
precedence over stored credentials and OIDC settings. Setup validates it against
the control plane without opening a browser or creating a PAT, then stores it in
credential storage when you approve/apply. A dry run validates and previews without
storing it. Saved settings, generated agent configs, diffs and logs omit the token;
the original input file stays as supplied, so keep it private and outside source
control. Rerun with `--config` to supply a replacement token; later `update` runs
reuse the stored one.

For browser sign-in with PAT creation/reuse, omit `pat_token` and set
`auth.pat_name = "my-laptop"` instead; the two keys cannot be combined in a file.
An explicit `--pat-name` overrides a token supplied by the file. Optional
`auth.pat_team = "engineering"` sets the owning team when creating a PAT, with
`--pat-team` taking precedence. PAT tokens/names/teams are config or CLI inputs,
not model/MCP/skills questions in the wizard.

#### Key renewal

A key setup creates lasts 365 days. Change it with
`tfy-local-ai-setup config set auth.pat_expiry_days 90` (7–3650). The new
length applies to keys created or renewed from then on; existing keys keep
their date. When your TrueFoundry organization limits how long keys may last
(Settings › Security), setup creates them for that limit instead, and says so.

When the desktop app is installed, its hourly check renews the key before it
expires, using the key itself (no browser):

| Key lasts | Renewed |
|---|---|
| 30 days or less | halfway (a 30-day key on day 15) |
| longer | when a quarter is left (a 100-day key on day 75, a 365-day key on about day 274) |

The new key replaces the old one in the keychain, and agents that read it
through `tfy-local-ai-setup token` use it at once. The old key keeps working
for 7 days, so terminals opened earlier aren't cut off, and is then deleted
(or when it expires, if that is sooner). VS Code and Cursor hold a pasted copy,
so when they use the key the old one is kept until it expires and the app asks
you to paste the new one. Only keys setup made
are renewed: a key you supplied with `auth.pat_token` is yours to replace. A
key that already expired or was revoked can't renew itself: sign in again.

### Personal mode: Claude Desktop

Claude Desktop on 3P reads a per-user configuration library — `~/Library/Application Support/Claude-3p/configLibrary/` (macOS), `%LOCALAPPDATA%\Claude-3p\configLibrary\` (Windows), `~/.config/Claude-3p/configLibrary/` (Linux) — the same files its in-app **Configure Third-Party Inference** window saves ([where Desktop reads configuration](https://claude.com/docs/third-party/claude-desktop/configuration-format#where-claude-desktop-reads-configuration-on-a-device)). Personal setup adds one saved configuration, **TrueFoundry**, makes it the applied one and leaves every other saved configuration alone:

- **Gateway**: `inferenceProvider: gateway`, `inferenceGatewayBaseUrl`, the tier models as `inferenceModels`, and TrueFoundry MCP servers as `managedMcpServers` (connectors you added yourself are kept).
- **Import & export**: enables both buttons in Desktop settings with `claudeAiImport.enabled` and `claudeAiImport.exportEnabled`, and sets `bannerBehavior: detect` to prompt when earlier local sessions are detected. Users choose what to import; setup does not enable automatic history import.
- **Personal Access Token**: no key is stored in Desktop. It runs this binary as a [credential helper](https://claude.com/docs/third-party/claude-desktop/configuration#inferencecredentialhelper) (`inferenceCredentialKind: helper-script`, `inferenceCredentialHelper` + `inferenceCredentialHelperArgs: ["token", "--url", …, "--tenant", …]`), which prints the token from credential storage.
- **OIDC**: also a credential helper: this binary's `--_print-oidc-token --oidc-issuer=… --oidc-client-id=…`, the same helper Claude Code and Codex use, with `inferenceCredentialHelperTimeoutSec: 300` so a browser sign-in can finish. One sign-in flow and token cache for every agent.

Desktop reads its configuration at launch, so if it is running, setup asks to quit it before writing and reopens it afterwards (on macOS, Desktop asks you to confirm in its own dialog: click Quit there, and setup waits up to 2 minutes; run by a coding agent, setup stops and asks you to quit it). The `_meta.json` it had is copied once to `~/.tf/backups/claude-desktop/`. `uninstall` deletes the TrueFoundry configuration and re-applies the one that was applied before. If `_meta.json` isn't in the shape setup expects, nothing in the library is changed: setup writes `~/.tf/claude-desktop-truefoundry.json` for **Import configuration** in the app instead. A managed profile (MDM plist, `HKLM`/`HKCU\SOFTWARE\Policies\Claude`, or `/etc/claude-desktop/managed-settings.json`) that sets more than the app-behavior keys overrides local settings, so a managed Desktop is skipped unless `--force`.

The main `personal` form sets Claude Desktop's tiers in their own section, prefilled from Claude Code's, and saves choices that differ from the shared defaults under `[agents.<id>]`. Use `personal claude-code` or `personal claude-desktop` to customize one agent's models afterwards. `--model` (repeatable), `--default-model`, `--mcp-server` (repeatable), `--no-mcp`, `--skill` (repeatable), and `--no-skills` follow the same scope: shared in the main command, per-agent in `personal <agent>`. Claude tier flags apply to both Claude agents in the main command and only the selected Claude agent in its subcommand. Advanced options are saved under `[agents.<id>]`, so `update` keeps applying them.

```toml
url = "https://app.example.com"
tenant = "acme"
gateway = "https://gateway.truefoundry.ai"
tools = ["claude-code", "claude-desktop", "codex", "opencode"]

[auth]
pat_name = "my-laptop"

# Optional: otherwise discovered during setup.
[auth.oidc]
issuer = "https://login.example.com/acme"
client_id = "your-client-id"

[defaults]
models = ["model-a", "model-b"]
default_model = "model-a"

[mcp]
servers = ["gmail"]

[agents.claude-code]
opus_model = "claude-code/claude-opus"
sonnet_model = "claude-code/claude-sonnet"
haiku_model = "claude-code/claude-haiku"
mcp_servers = ["gmail", "github"]

[agents.claude-desktop]
opus_model = "claude-code/claude-opus"
sonnet_model = "claude-code/claude-sonnet"
haiku_model = "claude-code/claude-haiku"
mcp_servers = ["github"]

[agents.codex]
model = "gpt-5.6-sol"
mcp_servers = [] # Explicitly disable MCPs for Codex.

[agents.opencode]
models = ["model-c"]
default_model = "model-c"
# MCP omitted: inherit [mcp].servers.
```

A model-list override defaults to its first model unless `default_model` selects another model in that list. A default-only override keeps the inherited list and adds that default if needed. Without shared model defaults, list-based agents fall back to the Claude Code/Codex picks. Claude Code and Desktop have independent tier settings.

An agent's `mcp_servers` list replaces the shared list; `[]` disables MCPs; omitting the key inherits shared servers. Use `config set agents.codex.mcp_servers ""` to disable MCPs, or `config unset agents.codex.mcp_servers` to restore inheritance. To restore shared model choices, unset both `agents.opencode.default_model` and `agents.opencode.models`. Run `update` after config commands.

### Agent Skills

Setup can install [Agent Skills](https://agentskills.io) (folders with a `SKILL.md`) from your tenant's TrueFoundry Skills Registry. In the wizard, each agent's review shows its skills and its diff, and **Edit** ends with a **Skills** screen: a searchable list of the registry's skills, prefilled from what you saved or approved for the previous agent. Skills are written when you approve that agent (a skills folder shared by two agents is planned with both, so neither removes the other's skills). On the command line, `--skill` (repeatable or comma-separated) and `--no-skills` follow the same scope as `--mcp-server`/`--no-mcp`.

A skill is named `name`, `repo/name` (needed when two registry repos have a skill with the same name), or either form with `:N` to pin version N. The full `agent-skill:<tenant>/<repo>/<name>[:N]` FQN is also accepted. An unpinned skill follows the latest version, so `update` installs new versions and the review shows the file diff.

```toml
[skills]
names = ["ai-gateway", "truefoundry-skills/truefoundry-platform:4"]

[agents.codex]
skills = [] # No skills for Codex.

[agents.cursor]
skills = ["ai-gateway"] # Replaces the shared list.
```

`config set skills.names ai-gateway,truefoundry-platform`, `config set agents.codex.skills ""` and `config unset agents.codex.skills` work like the MCP keys.

Each skill is copied into each agent's own skills folder:

| Agent | Skills folder |
|-------|---------------|
| Claude Code | `~/.claude/skills` (`$CLAUDE_CONFIG_DIR/skills`) |
| Codex | `~/.codex/skills` (`$CODEX_HOME/skills`) |
| Grok Build | `~/.grok/skills` |
| Pi | `~/.pi/agent/skills` |
| Qwen Code | `~/.qwen/skills` |
| OpenCode | `~/.config/opencode/skills` |
| Gemini CLI | `~/.gemini/skills` |
| Copilot CLI, VS Code | `~/.copilot/skills` (`$COPILOT_HOME/skills` for Copilot CLI); the two share the folder |
| Goose | `~/.config/goose/skills` |
| Cursor | `~/.cursor/skills` |
| Claude Desktop | `~/.tf/skills/claude-desktop/<name>.zip`, which you upload once in **Customize › Skills › + › Upload a skill** (code execution must be on); setup lists this as a numbered step |

Setup only changes skill folders that it installed, and only files that still match what it wrote (recorded by SHA-256 in `~/.tf/personal.json`):

- A skill folder that already exists but wasn't installed by setup is skipped with a warning, even with `--force`.
- A file you edited is kept when the skill is updated or removed, with a warning.
- Removing a skill from one agent leaves a shared folder alone while another selected agent still wants it, as with VS Code and Copilot CLI.
- `uninstall` removes the skills setup installed, keeping edited files; Desktop uploads must be removed in the app.

Grok, OpenCode, Goose, VS Code and Cursor also read `~/.claude/skills`, and Cursor reads `~/.codex/skills`. Those agents load a skill name once, so the same skill in both folders is harmless. Setup adds a note when the two agents are set to different versions of a skill (the agent may use either copy), or when an agent opts out of a skill that it still sees through Claude Code's or Codex's folder. Downloads use short-lived signed links that never appear in output or debug logs. `status` lists each agent's skills and versions, and `doctor` reports skills that are missing, no longer in the registry, or pinned behind a newer version. If setup has no token yet (a first `--yes` run that signs in during apply), skills are listed under Skipped; run `update` afterwards.

Existing `[models]` and `[claude_desktop]` files still load. The next confirmed setup save writes the new format and retains the original backup. Migration copies the old Claude tier choices into both Claude sections, moves Codex's model into its own section, preserves effective shared model fallbacks, and moves OIDC settings into `[auth.oidc]`. Explicit new keys take precedence over legacy equivalents. `config keys` lists the new names; old dotted keys remain accepted as aliases (old Claude tier keys target Claude Code). No migration is written during a dry run or cancelled setup.

**Review and completion.** Each agent's review shows unified diffs of the files that agent will change, with credential values redacted, before anything is written; scroll long diffs with the arrow keys. `--dry-run`, `--yes` and plain output print every diff at once. Settings managed through an app or CLI show operation previews. The closing summary lists what was written, left unchanged or skipped per agent, plus backup, saved-settings and credential operations.

After applying, each agent reports its actual result. Required manual steps are numbered under that agent, separately from warnings and general information; they remain visible on unchanged runs because setup cannot verify steps completed in another app. Failed writes or state/settings saves return a nonzero exit code and identify successful changes that were kept.

**Troubleshooting.** `personal`, `personal <agent>`, `update` and `login` take `--debug` and `--log-file <path>`. `--debug` logs each control-plane/gateway call (method, URL, status — never headers or tokens), the sign-in and PAT decisions, MCP server URLs, the planned changes and every file written. `--log-file` also appends all log output to that file (0600). While the wizard is on screen, debug lines go only to the log file (`~/.tf/debug.log` when `--log-file` isn't given) so the form stays readable; the path is printed at the end. `--dry-run` shows the plan without writing anything or creating a token.

```bash
tfy-local-ai-setup update --debug --log-file ./tfy-personal.log
```

---

## Install

For personal mode, install the latest release for the current user. No root or administrator rights are needed.

```bash
# macOS, Linux
curl -fsSL https://github.com/truefoundry/tfy-local-ai-setup/releases/latest/download/install.sh | sh
```

```powershell
# Windows PowerShell
irm https://github.com/truefoundry/tfy-local-ai-setup/releases/latest/download/install.ps1 | iex
```

The installers (`install.sh` and `install.ps1`, attached to each release) pick the binary for the OS and CPU, check it against the release's `SHA256SUMS`, install it to `~/.local/bin` (`%LOCALAPPDATA%\TrueFoundry` on Windows) and add that directory to PATH: a line in `~/.zshrc`, `~/.bashrc` (on macOS, the first of `~/.bash_profile`, `~/.bash_login` and `~/.profile` that exists, else `~/.bash_profile`) or fish's `conf.d`, or the user PATH on Windows. Then run `tfy-local-ai-setup personal`.

- **A specific release:** `curl -fsSL …/install.sh | sh -s v2.3.0`, or `& ([scriptblock]::Create((irm …/install.ps1))) v2.3.0` in PowerShell. `TFY_VERSION` does the same.
- **Another directory:** set `TFY_INSTALL_DIR`.
- **Leave PATH alone:** set `TFY_NO_MODIFY_PATH=1`.

MDM deployments install to an administrator-owned path instead; see [MDM deployment](#mdm-deployment).

## Binaries

Each release has these assets:

| File | Platform |
|------|----------|
| `tfy-local-ai-setup-darwin-arm64` | macOS — Apple Silicon (M1/M2/M3/M4) |
| `tfy-local-ai-setup-darwin-amd64` | macOS — Intel |
| `tfy-local-ai-setup-linux-amd64` | Linux — x86_64 |
| `tfy-local-ai-setup-linux-arm64` | Linux — ARM64 |
| `tfy-local-ai-setup-windows-amd64.exe` | Windows — x86_64 (also runs on Windows on Arm) |
| `SHA256SUMS` | SHA-256 checksums of the binaries |
| `install.sh`, `install.ps1` | The installers above |

On macOS and Linux you need to make a downloaded binary executable before running:

```bash
chmod +x bin/tfy-local-ai-setup-darwin-arm64
```

---

## Usage

Must be run as **root** on macOS/Linux or **Administrator** on Windows.

```
tfy-local-ai-setup --url <control-plane-url> --tenant <tenant-name> [flags]
```

### Flags

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--url` | **Yes** | — | Base URL of the TrueFoundry control plane (e.g. `https://app.example.com`) |
| `--tenant` | **Yes** | — | Your TrueFoundry tenant name |
| `--gateway` | No | value of `--url` | Common gateway URL for all tools. Used as the default for `--claude-gateway` and `--codex-gateway` if neither is set. |
| `--claude-code` | No | auto-detect | Configure Claude Code managed settings, **installed or not** — passing the flag skips the install check. If no tool flag is set, the binary auto-detects which tools are installed. |
| `--claude-code-auth` | No | `api-key` | Claude Code auth flow. `api-key`: this binary fetches a token and writes `X-TFY-API-KEY` — the pre-OIDC behaviour. `oidc`: writes `apiKeyHelper` that runs `--_print-oidc-token` (silent refresh or auth-code + PKCE); `--oidc-issuer` / `--oidc-client-id` become required. **OIDC is opt-in**: leaving this unset changes nothing for an existing fleet. |
| `--codex` | No | auto-detect | Configure Codex managed settings, **installed or not** — passing the flag skips the install check. Covers both Codex surfaces: the `codex` CLI and the Codex desktop app share the same managed config. If no tool flag is set, the binary auto-detects which tools are installed. |
| `--claude-gateway` | No | value of `--gateway` | Gateway URL for Claude Code (written to `ANTHROPIC_BASE_URL`). Overrides `--gateway` for Claude Code only. |
| `--codex-gateway` | No | value of `--gateway` | Gateway URL for Codex (written to `base_url` in the provider config). Defaults to `--gateway`. |
| `--codex-default-model` | No | *(unset)* | Default model ID for Codex, written as the top-level `model` key in `managed_config.toml`. If not passed, no `model` key is written and Codex uses its own default (or a value from `--codex-settings-file`). |
| `--codex-auth` | No | `api-key` | Codex auth flow. `api-key`: this binary fetches a token and writes it into `http_headers` (header style from `--codex-auth-mode`) — the pre-OIDC behaviour. `oidc`: writes `[model_providers.truefoundry.auth]` `command` + `args` pointing at `--_print-oidc-token`; `--oidc-issuer` / `--oidc-client-id` become required. **OIDC is opt-in**: leaving this unset changes nothing for an existing fleet. |
| `--codex-auth-mode` | No | `api-key` | Codex JWT header style when `--codex-auth=api-key`. `api-key`: token sent as `Authorization: Bearer`. `chatgpt-subscription`: sets `requires_openai_auth = true` and sends the token as `x-tfy-api-key`. Ignored under `--codex-auth=oidc`. |
| `--claude-desktop` | No | auto-detect | Configure Claude Desktop (Cowork 3P) managed preferences, **installed or not** — passing the flag skips the install check. Auth mode is set by `--claude-desktop-auth`. Still a no-op on Linux, where Claude Desktop 3P does not exist. |
| `--claude-desktop-auth` | No | `api-key` | Claude Desktop auth flow. `api-key`: this binary fetches a token and writes `inferenceGatewayApiKey` plus the `X-TFY-API-KEY` header — the pre-v1.3.7 behaviour. `oidc`: the app runs authorization-code + PKCE itself via `inferenceIdpOidc` (Claude Desktop ≥ 2.7032.0), and `--oidc-issuer` / `--oidc-client-id` become required. **OIDC is opt-in**: leaving this unset changes nothing for an existing fleet. |
| `--disable-deployment-mode-chooser` | No | `false` | With `--claude-desktop-auth=oidc`, write `disableDeploymentModeChooser=true` so users cannot leave the managed gateway profile for Claude.ai. No effect in api-key mode. |
| `--claude-desktop-gateway` | No | value of `--gateway` | Gateway URL for Claude Desktop (written to `inferenceGatewayBaseUrl`). Overrides `--gateway` for Claude Desktop only. |
| `--codex-settings-file` | No | — | Path to a TOML template for Codex's config (the provider section is injected into it). Applies to the config-schema files only, not to Windows `requirements.toml`. |
| `--force-login-org` | No | — | Claude Desktop only. Restrict sign-in to this Claude org UUID (repeatable). Writes `forceLoginOrgUUID`. Off by default. A wrong UUID locks everyone out. See [Restrict Claude Desktop sign-in](#claude-desktop-cowork-on-3p). |
| `--wsl` | No | `false` | Windows only. Also push the managed config into the logged-in user's WSL distros. See [WSL](docs/troubleshooting.md#wsl). |
| `--config` | No | — | One standalone TOML or JSON file for every client. Explicit flags override its values. See [Full MDM configuration file](#full-mdm-configuration-file) and the [config reference](docs/mdm-config-reference.md). |
| `--login` | No | — | `manual`: create a PAT only with `tfy-local-ai-setup login --mdm`, so scheduled runs never open a browser. |
| `--pat-team` | No | — | Team that owns a managed PAT (manual login). |
| `--pat-expiry-days` | No | `365` | Days a managed PAT lasts, 7–3650 (manual login). |
| `--auto-renew-key` | No | `false` | Renew each user's PAT before it expires, using the PAT itself and no browser. See [Manual PAT onboarding](#manual-pat-onboarding-for-scheduled-mdm-runs). |
| `--cursor-monitoring` | No | `false` | Copy Cursor prompt, response, tool and session events to TrueFoundry through gateway hooks. `=false` removes the monitoring hooks. See [Monitor Cursor through MDM](#monitor-cursor-through-mdm). |
| `--cursor-require-login` | No | `false` | With `--cursor-monitoring`, block Cursor prompts and tool calls until an MDM credential is available. |
| `--block-agent` | No | — | Block AI prompts for `claude-code`, `codex` or `cursor` through their prompt-submission hooks. Repeatable or comma-separated. See [Block AI prompts with hooks](#block-ai-prompts-with-hooks). |
| `--unblock-agent` | No | — | Remove this setup's prompt-blocking hook for the named agent. |
| `--block-message` | No | — | Message shown when a prompt is blocked. Replaces the file's message. |
| `--oidc-issuer` | With `--claude-code-auth=oidc`, `--codex-auth=oidc`, or `--claude-desktop-auth=oidc` | — | OIDC issuer URL for the tenant's TrueFoundry public client (written to `inferenceIdpOidc.issuer` for Desktop OIDC). |
| `--oidc-client-id` | With `--claude-code-auth=oidc`, `--codex-auth=oidc`, or `--claude-desktop-auth=oidc` | — | OIDC public client ID. Same value for all machines in the tenant — not per user. |
| `--oidc-bearer-token-type` | No | `access_token` | Which token Claude Desktop sends as `Authorization: Bearer`: `access_token` or `id_token`. The TrueFoundry gateway validates the OAuth access token and rejects an ID token, so keep the default unless the gateway in front of Desktop is not TrueFoundry's. |
| `--oidc-scopes` | No | `openid profile email offline_access` | Space-separated OIDC scopes written to `inferenceIdpOidc.scopes`. Required by Claude Desktop for `access_token` mode (default applied automatically). Keep `openid`, or Desktop's sign-in fails. |
| `--desktop-opus-model` | No | value of `--opus-model` | Opus model ID for Claude Desktop (`inferenceModels`). Inherits the Claude Code opus model unless overridden. |
| `--desktop-sonnet-model` | No | value of `--sonnet-model` | Sonnet model ID for Claude Desktop (`inferenceModels`). Inherits the Claude Code sonnet model unless overridden. |
| `--desktop-haiku-model` | No | value of `--haiku-model` | Haiku model ID for Claude Desktop (`inferenceModels`). Inherits the Claude Code haiku model unless overridden. |
| `--desktop-header` | No | — | Extra custom header for Claude Desktop as `Name: Value`, written to `inferenceCustomHeaders`. Repeatable — pass it multiple times for multiple headers. |
| `--mcp-server` | No | — | TrueFoundry gateway MCP server to deploy to every configured tool, resolved to `<gateway>/<tenant>/mcp/<name>/server`. Repeatable (`--mcp-server github --mcp-server slack`) or comma-separated. Names: letters, digits, `-`, `_`. See [MCP servers](#mcp-servers). |
| `--claude-desktop-allow-local-mcp` | No | `true` | Claude Desktop only. Pass `=false` to write `isLocalDevMcpEnabled=false`, which blocks user-added and plugin local (stdio) servers; remote servers keep working. |
| `--opus-model` | No | `claude-code/claude-opus` | Model ID written to `ANTHROPIC_DEFAULT_OPUS_MODEL` (Claude Code only) |
| `--sonnet-model` | No | `claude-code/claude-sonnet` | Model ID written to `ANTHROPIC_DEFAULT_SONNET_MODEL` (Claude Code only) |
| `--haiku-model` | No | `claude-code/claude-haiku` | Model ID written to `ANTHROPIC_DEFAULT_HAIKU_MODEL` (Claude Code only) |
| `--additional-model` | No | — | More models Claude Code offers in `/model`, after the tier models (repeatable, or comma-separated; any gateway model, or `@all` for every chat model the user can access). Written as a `modelPicker` that replaces the built-in lineup. Claude Desktop offers them too unless `--desktop-additional-model` is set. Config file: `additional_models` under `[agents.claude-code]`. |
| `--desktop-additional-model` | No | value of `--additional-model` | More models in Claude Desktop's model menu, after the tier models (`inferenceModels`). Desktop shows Claude models only; the run names the ones it hides. Config file: `additional_models` under `[agents.claude-desktop]`. |
| `--settings-file` | No | — | Path to a JSON template for `managed-settings.json` (Claude Code only). Patches token and model IDs; all other keys are preserved. Falls back to the existing file on disk, then the built-in default. |
| `--refresh-token-file` | No | device login: `~/.tf/refresh-token`; OIDC helper: `~/.tf/oidc-refresh-token` when unset | Path for the refresh-token cache (0600). Device-login and OIDC helper use separate defaults so they do not overwrite each other. |
| `--access-token-file` | No | device login: `~/.tf/access-token`; OIDC helper: `~/.tf/oidc-access-token` when unset | Path for the access-token cache (0600). |
| `--claude-ai-import` | No | `false` | Enable claude.ai history import in Claude Desktop (`claudeAiImport`). Off by default in the app and **not user-settable** — managed config is the only way to turn it on. Requires Claude Desktop >= 1.10628.0; older builds accept and silently ignore the key. |
| `--import-export-enabled` | No | `true` | Show the `Export...` button alongside import. No effect unless `--claude-ai-import`. |
| `--import-banner-behavior` | No | `detect` | Import banner behavior: `off`, `detect` or `show`. No effect unless `--claude-ai-import`. |
| `--import-automatic-3p` | No | `false` | **Beta.** Silently imports the user's earlier 3P sessions with no prompt. The user is never asked, so opt in only with the customer's agreement. |
| `--import-encoding` | No | `json-string` | How `claudeAiImport` is encoded in the macOS plist: `json-string` (a `<string>` holding JSON, matching the verified `managedMcpServers` shape) or `native-dict` (a real `<dict>`, matching Anthropic's own example). The key is not yet verified on a real Mac — if Settings → Import & export stays greyed out on an otherwise-configured machine, try `native-dict`. Windows always uses the JSON string form. |
| `--write-config-on-login-failure` | No | `false` | When the device login fails, still write and lock the managed config, then exit 1. Without it the run exits 1 having written nothing, which leaves an MDM-managed machine with **no policy at all** rather than merely unauthenticated. The token used is the logged-in user's cached access token if one is on disk, otherwise `--placeholder-token`. Claude Desktop OIDC, Claude Code `apiKeyHelper`, and Codex `auth.command` do not use this device-login token. |
| `--placeholder-token` | No | `TFY_PLACEHOLDER_LOGIN_REQUIRED` | Token written into the managed config when `--write-config-on-login-failure` is set and no cached token is available. Deliberately not a valid credential — requests made with it get a 401. |
| `--dry-run` | No | `false` | Print the resulting config to stdout instead of writing any files. Works for both Claude Code and Codex. |
| `--log-file` | No | — | Also append all log output to this file (0600). Useful for headless MDM runs where there is no console to read. |
| `--debug` | No | `false` | Verbose debug logging. Surfaces the device-login browser URL and streams the (Windows) device-login child process output, and keeps the child log file for inspection. |
| `--agent` | No | — | Configure another agent (repeatable or comma-separated): any personal-mode agent ID or alias, e.g. `opencode`, `qwen-code`, `gemini-cli`, `grok`, `pi`, `goose`, `copilot-cli`, `vscode`, `cursor`. `claude-code`, `codex` and `claude-desktop` are the same as their own flags. See [Other agents](#other-agents). |
| `--model` | No | — | Gateway model ID offered to the `--agent` agents (repeatable or comma-separated). Required by agents that need a model list. |
| `--default-model` | No | first `--model` | Default model for the `--agent` agents. Must be one of `--model`. |
| `--agents-config` | No | — | TOML file in the same schema as personal mode's `~/.tf/config.toml` (`tools`, `[defaults]`, `[mcp]`, `[skills]`, `[agents.<id>]`). Fills in what the flags leave out; `tools` adds agents. Keys the MDM run takes from elsewhere (`url`, `tenant`, `[auth]`, and anything for `claude-code`, `codex` or `claude-desktop`) are ignored with a warning. |
| `--skill` | No | — | Agent Skill from the TrueFoundry Skills Registry to install for the `--agent` agents, as `[repo/]name[:version]` (repeatable or comma-separated). Overrides `--agents-config`'s `[skills]`. |
| `--no-skills` | No | `false` | Install no Agent Skills for the `--agent` agents, ignoring `[skills]` in `--agents-config`; removes the ones earlier runs installed. |
| `--no-mcp` | No | `false` | Give the `--agent` agents no shared MCP servers, ignoring `[mcp]` in `--agents-config`. An agent's own `mcp_servers` there still applies. Can't be combined with `--mcp-server`. |
| `--agents-auth` | No | `api-key` | How the `--agent` agents authenticate: `api-key` (the user's device-login token, written directly where the agent allows it) or `oidc` (needs `--oidc-issuer` and `--oidc-client-id`). |

### Default config

When no `--settings-file` is provided and no `managed-settings.json` exists on disk, the binary writes the following config. Model IDs use the flag defaults unless overridden.

```json
{
  "permissions": {
    "disableBypassPermissionsMode": "disable",
    "deny": [
      "Bash(curl:*)",
      "Bash(wget:*)",
      "Read(**/.env)",
      "Read(**/.env.*)",
      "Read(**/secrets/**)",
      "Read(**/.ssh/**)",
      "Read(**/credentials/**)"
    ],
    "ask": ["Bash(git push:*)", "Edit(**)"]
  },
  "allowManagedPermissionRulesOnly": true,
  "allowManagedHooksOnly": true,
  "transcriptRetentionDays": 14,
  "sandbox": {
    "enabled": true,
    "network": { "httpProxyPort": 8080, "socksProxyPort": 8081 }
  },
  "strictKnownMarketplaces": [],
  "env": {
    "ANTHROPIC_BASE_URL": "<value of --gateway>",
    "ANTHROPIC_CUSTOM_HEADERS": "X-TFY-API-KEY: <token>",
    "ANTHROPIC_DEFAULT_OPUS_MODEL": "claude-code/claude-opus",
    "ANTHROPIC_DEFAULT_SONNET_MODEL": "claude-code/claude-sonnet",
    "ANTHROPIC_DEFAULT_HAIKU_MODEL": "claude-code/claude-haiku",
    "CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS": "1"
  }
}
```

**Default (`--claude-code-auth=api-key`):** the binary fetches a device-login JWT and writes it as
`X-TFY-API-KEY` (shown above).

**Opt-in OIDC (`--claude-code-auth=oidc`):** instead of stuffing a JWT, the binary writes
`apiKeyHelper` pointing at `--_print-oidc-token`. Claude Code obtains credentials at runtime:
the cached access token while it has more than 10 minutes left, else a silent refresh from
`~/.tf/oidc-*` (transient failures are retried, and a still-valid cached token is kept rather
than opening a browser), or authorization-code + PKCE on the first free loopback port
among `http://127.0.0.1:8585/callback`, `:8586`, and `:8587`. Register all three redirect URIs on
the same public client used for Claude Desktop (Desktop itself keeps `http://127.0.0.1/callback`).
Legacy `X-TFY-API-KEY` headers are stripped from templates in this mode. Concurrent helper runs
(several Claude Code sessions plus Codex) take a lock beside the refresh token, so a single-use
refresh token (`refresh_token_max_uses: 1`) is spent once and the others reuse the result.

Use `--dry-run` to inspect the exact output before deploying to a fleet.

### Codex default config

When `--codex` is set (or Codex is auto-detected), the binary writes the provider section into Codex's system-managed config. A default model is written only when `--codex-default-model` is passed.

`--codex` governs **both Codex surfaces** — there is no separate desktop flag. The desktop app bundles the same Rust core (`Contents/Resources/codex-cli/`) and drives it over the app-server protocol, which builds its `Config` through the shared loader with the real managed config in place, so one write covers the CLI and the app. (The only escapes — `CODEX_APP_SERVER_DISABLE_MANAGED_CONFIG` and `CODEX_APP_SERVER_MANAGED_CONFIG_PATH` — are debug-build only.)

**On macOS and Linux** the target is `/etc/codex/managed_config.toml`. Codex still loads it, at higher precedence than the user's own `~/.codex/config.toml`, so the gateway `base_url` and auth cannot be overridden and are reapplied on every launch.

**On Windows** `managed_config.toml` is not read at all — Codex dropped it in v0.149.0 and ignores it with a startup warning. There is no registry equivalent. Two files under `%ProgramData%\OpenAI\Codex` replace it, and the binary writes both:

| File | Role |
|------|------|
| `requirements.toml` | Admin-enforced; beats the user's own config. Codex honors `model_provider` / `model_providers` here only from **v0.156.0**. Its schema has no top-level `model` key, so `--codex-default-model` is written under `[models.new_thread]`. |
| `config.toml` | Machine defaults. Read by every Codex back to ~v0.126.0, but it is the *lowest*-precedence layer, so a developer editing their own `~/.codex/config.toml` beats it. It exists so older Codex builds still route through the gateway — it does not enforce. |

A `--codex-settings-file` template applies to the config-schema files only; it is not merged into `requirements.toml`, whose schema differs.

**Default (`--codex-auth=api-key`):** the binary fetches a device-login JWT and writes it into `http_headers`. The header style is selected by `--codex-auth-mode` (mirrors the two auth tabs in the Codex docs):

**`--codex-auth-mode=api-key`** (default) — token sent as an `Authorization: Bearer` header:

```toml
model_provider = "truefoundry"
model = "<value of --codex-default-model>"   # only when --codex-default-model is passed

[model_providers.truefoundry]
name     = "TrueFoundry Gateway"
base_url = "<value of --gateway>"
wire_api = "responses"

[model_providers.truefoundry.http_headers]
Authorization = "Bearer <token>"
```

**`--codex-auth-mode=chatgpt-subscription`** — Codex sends the user's own ChatGPT OAuth to OpenAI (`requires_openai_auth = true`); the token is sent as `x-tfy-api-key` so TrueFoundry authenticates the user separately:

```toml
model_provider = "truefoundry"
model = "<value of --codex-default-model>"   # only when --codex-default-model is passed

[model_providers.truefoundry]
name     = "TrueFoundry Gateway"
base_url = "<value of --gateway>"
wire_api = "responses"
requires_openai_auth = true

[model_providers.truefoundry.http_headers]
x-tfy-api-key = "<token>"
```

**Opt-in OIDC (`--codex-auth=oidc`):** instead of stuffing a JWT, the binary writes command-backed auth. Codex execs the helper with no shell (`command` + `args`) when it needs a bearer token; the helper silent-refreshes or runs authorization-code + PKCE (same loopback as Claude Code). `timeout_ms` is set to 300000 so first-time PKCE can finish. No device-login JWT is required for Codex in this mode.

```toml
model_provider = "truefoundry"

[model_providers.truefoundry]
name     = "TrueFoundry Gateway"
base_url = "<value of --gateway>"
wire_api = "responses"

[model_providers.truefoundry.auth]
command = "/usr/local/bin/tfy-local-ai-setup"
args = ["--_print-oidc-token", "--oidc-issuer=<issuer>", "--oidc-client-id=<client-id>"]
timeout_ms = 300000
```

**Codex config paths:**

| OS | Path |
|----|------|
| macOS + Linux | `/etc/codex/managed_config.toml` |
| Windows | `C:\ProgramData\OpenAI\Codex\requirements.toml` (enforced, Codex ≥ v0.156.0) and `C:\ProgramData\OpenAI\Codex\config.toml` (defaults) |

### Claude Desktop (Cowork on 3P)

When `--claude-desktop` is set (or Claude Desktop is auto-detected), the binary writes the
[`com.anthropic.claudefordesktop`](https://claude.com/docs/cowork/3p/configuration) managed
preferences that point Claude Desktop's **Cowork on 3P** mode at the TrueFoundry gateway using
**native OIDC** (`inferenceCredentialKind=external-idp` + `inferenceIdpOidc`).
**Requires Claude Desktop 2.7032.0 or later on every managed device before rollout.**
These settings replace the deprecated `interactive` + `inferenceGatewayOidc` pair.
On refresh, macOS rewrites the managed plist and Windows removes the old
`inferenceGatewayOidc` registry value. The issuer, client ID, scopes, and bearer token
type stay the same. API-key mode is unchanged.

Opt in with `--claude-desktop-auth=oidc`, then pass the tenant's fixed public client via
`--oidc-issuer` and `--oidc-client-id`. Without that flag Claude Desktop keeps using the
token this binary fetches, exactly as it did before v1.3.7. Claude Desktop
runs authorization-code + PKCE in the browser; this binary does **not** fetch or paste a JWT into
Desktop prefs. (`inferenceGatewayApiKey` / `X-TFY-API-KEY` are no longer written for Desktop.)

Pass `--disable-deployment-mode-chooser` with Desktop OIDC to set `disableDeploymentModeChooser=true`
so users cannot leave the managed gateway profile for a personal Claude.ai account (fail closed).
Off by default.

Every value is stored as a string; arrays and objects are JSON-encoded strings, as required by
Anthropic's configuration reference. The model picker is pinned to the explicit list and
`modelDiscoveryEnabled` is set to `false`, so Claude Desktop never auto-discovers models via the
gateway's `GET /v1/models`.

**Claude Desktop config paths:**

| OS | Path | Lock |
|----|------|------|
| macOS | `/Library/Managed Preferences/<user>/com.anthropic.claudefordesktop.plist` | `chflags schg` (immutable) |
| Windows | `HKLM\SOFTWARE\Policies\Claude` (string values) | admin-only hive |
| Linux | Not supported (Claude Desktop 3P does not run on Linux) | — |

Example generated plist (macOS), with a model list and an extra custom header via
`--desktop-header 'X-TFY-METADATA: {"team":"platform"}'`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>inferenceProvider</key>
  <string>gateway</string>
  <key>inferenceGatewayBaseUrl</key>
  <string>https://gateway.example.truefoundry.com/api/llm</string>
  <key>inferenceCredentialKind</key>
  <string>external-idp</string>
  <key>inferenceIdpOidc</key>
  <string>{"bearerTokenType":"access_token","clientId":"cli_myorg_claude_desktop","issuer":"https://login.example.truefoundry.com/myorg","scopes":"openid profile email offline_access"}</string>
  <key>modelDiscoveryEnabled</key>
  <string>false</string>
  <key>inferenceCustomHeaders</key>
  <string>{"X-TFY-METADATA":"{\"team\":\"platform\"}"}</string>
  <key>inferenceModels</key>
  <string>[{"name":"claude-code/claude-opus","anthropicFamilyTier":"opus","isFamilyDefault":true}, ...]</string>
</dict>
</plist>
```

Each `--desktop-*-model` flag defaults to the corresponding Claude Code model flag
(`--opus-model` / `--sonnet-model` / `--haiku-model`), so Claude Desktop uses the same models you
configure for Claude Code unless you override a specific one.

```bash
sudo /usr/local/bin/tfy-local-ai-setup \
  --url="https://app.example.com" \
  --tenant="myorg" \
  --gateway="https://gateway.example.truefoundry.com/api/llm" \
  --claude-desktop \
  --claude-desktop-auth="oidc" \
  --oidc-issuer="https://login.example.truefoundry.com/myorg" \
  --oidc-client-id="cli_myorg_claude_desktop" \
  --desktop-opus-model="tfy-ai-anthropic/claude-opus-4-6" \
  --desktop-sonnet-model="tfy-ai-anthropic/claude-sonnet-4-6" \
  --desktop-haiku-model="tfy-ai-anthropic/claude-haiku-4-5" \
  --desktop-header='X-TFY-METADATA: {"team":"platform"}'
```

Claude Desktop loads managed preferences at launch — **restart Claude Desktop** after a run for
changes to take effect. The first launch shows organization sign-in; later launches refresh the
OIDC session inside the app. To verify what the app picked up, use **Help → Troubleshooting → Copy
Managed Configuration Report** (secrets redacted).

### MCP servers

Each `--mcp-server` name is deployed as a remote (HTTP) server on the gateway. Every tool runs the
OAuth sign-in itself, so no credential is written. Omitting a name on a later run removes it.

| Tool | Where | Notes |
|------|-------|-------|
| Claude Code | `managedMcpServers` in `managed-settings.json` (also pushed to WSL with `--wsl`) | Additive: users keep their own servers. Needs Claude Code v2.1.259+; older builds ignore the key. To lock users out of other servers, add `allowedMcpServers` / `deniedMcpServers` to a `--settings-file` template. |
| Codex | `[mcp_servers.<name>]` with `url` in the managed config (`config.toml` on Windows) | Users sign in once with `codex mcp login <name>`. |
| Claude Desktop | `managedMcpServers` in the managed prefs / registry | Needs Claude Desktop 1.2581.0+. Pair with `--claude-desktop-allow-local-mcp=false` to allow only remote servers. |

```bash
sudo ./tfy-local-ai-setup \
  --url="https://app.example.com" \
  --tenant="acme" \
  --gateway="https://gateway.example.truefoundry.com/api/llm" \
  --mcp-server github \
  --mcp-server slack \
  --claude-desktop-allow-local-mcp=false
```

### Discover accessible MCP servers during MDM refresh

To replace a fixed MCP list with per-user discovery, put this in an administrator-managed
TOML file passed through `--agents-config`:

```toml
[mcp]
discovery = "all-accessible"
```

For example, schedule the following command hourly through Intune (use your own paths):

```bash
sudo ./tfy-local-ai-setup \
  --url="https://app.example.com" \
  --tenant="acme" \
  --gateway="https://gateway.example.truefoundry.com/api/llm" \
  --claude-code --claude-desktop \
  --agents-config="/path/to/agents.toml" \
  --log-file="/path/to/mdm-refresh.log"
```

Discovery applies to Claude Code, Codex, Claude Desktop, and the shared MCP list for
other configured agents. An agent's explicit `mcp_servers` override still takes precedence.
Don't combine discovery with `[mcp].servers`, `--mcp-server`, or `--no-mcp`.
This setting is for MDM runs; personal mode continues using its saved server choices.

Each scheduled run silently refreshes the logged-in user's credentials, fetches every
registry page, checks `USE_MCP_SERVER` permissions, and configures servers the user can
both discover and use. It does not initiate MCP OAuth consent. Complete the initial
device/OIDC sign-in interactively before enabling discovery; scheduled discovery never
opens a browser or falls back to interactive sign-in. Missing/expired credentials,
network failures, permission API failures, and incomplete responses leave the existing
managed configuration untouched. Failures are logged with timestamps and return exit
code 1 for MDM monitoring; the next scheduled invocation retries. No scheduler is installed
by this TOML setting.

A successful response with no accessible servers removes the previously discovered
entries. The refresh does not restart applications: Claude Desktop picks up changes
when it next restarts. Gateway authorization continues enforcing permissions independently.

### Other agents

`--agent` brings personal mode's agents into the MDM run. Nothing new is configured unless it is named, so an existing fleet's run is unchanged. Each agent goes through one or both of two channels:

- **managed**: root writes the agent's machine-wide admin file, owned by root, mode 0644 and locked, the same as Claude Code's `managed-settings.json`.
- **user**: the run starts itself again as the logged-in console user (`sudo -H -u <user>` through their login shell on macOS/Linux, a scheduled task on Windows) and that process writes the agent's own user config with personal mode's code. It is reconciled on every run but not locked. With no one logged in the user channel is skipped and the next run catches up.

| Agent | Channel | Where |
|---|---|---|
| OpenCode | managed | `/Library/Application Support/opencode/`, `/etc/opencode/`, `%ProgramData%\opencode\` |
| Qwen Code | managed | `/Library/Application Support/QwenCode/settings.json`, `/etc/qwen-code/`, `%ProgramData%\qwen-code\` |
| Gemini CLI | managed + user | model, MCP and auth type in `/Library/Application Support/GeminiCli/settings.json` (`/etc/gemini-cli/`, `%ProgramData%\gemini-cli\`); base URL and key can only be env vars, so they go in a shell block |
| Grok Build | managed (macOS/Linux), user (Windows) | `/etc/grok/requirements.toml` (gateway model tables, enforced) and `/etc/grok/managed_config.toml` (default model and MCP, the user's own config wins) |
| Pi, Goose, Copilot CLI, VS Code, Cursor | user | the agent's own user config, as personal mode writes it |

Root also writes the machine policy (`/Library/Application Support/TrueFoundry/agents-policy.json`, `/etc/truefoundry/agents-policy.json`, `%ProgramFiles%\TrueFoundry\agents-policy.json`): what IT configures, read by the user-channel process, by `token --mdm` in the shell block, and by personal mode, which leaves any agent it lists to IT.

In `api-key` mode the user's device-login token is written straight into managed files (OpenCode, Qwen, Grok) and refreshed every run; shell-block agents fetch it with `tfy-local-ai-setup token --mdm`. Dropping an agent or MCP server on a later run removes what MDM wrote for it.

Agent Skills (`--skill`, or `[skills]` and per-agent `skills` in `--agents-config`) are installed by the user-context run into each agent's own skills folder, for managed agents too, because skills folders are per user. Their record is kept in `~/.tf/mdm-user.json`, apart from personal mode's, so dropping a skill or an agent removes only what MDM installed. With `--config` (or an explicit `--skill` list), Claude Code and Codex also get skills in their user skills folders; Claude Desktop gets upload-ready ZIPs for manual upload in the app.

For automatic skill installation, replace the shared fixed list in `--config` or
`--agents-config` with:

```toml
[skills]
discovery = "all-accessible"
```

Use `--config` to enable discovery for Claude Code, Codex, and Claude Desktop too;
`--agents-config` retains its existing additional-agent scope.

Each MDM user-context run lists all accessible published registry skills and installs
repository-qualified, unpinned refs, following the latest versions. Per-agent `skills`
overrides still apply. Do not combine this with a nonempty `[skills].names`, `--skill`,
or `--no-skills`. The policy persists this choice as `skills_discovery`.
Discovery silently uses the user's managed PAT for manual login, the device-login
cache for API-key mode, or the OIDC cache for OIDC mode. It never initiates login;
dry runs do not list or download skills in discovery mode. Missing credentials,
failed/incomplete listings, and conflicting skill names across repositories leave
installed skills and ownership records untouched for the next scheduled retry.
Skill discovery failures do not block MCP reconciliation. A successful empty listing
removes previously managed skills; updates/removals preserve locally edited files.
Claude Desktop still requires manual upload of the generated ZIPs.

A skill that can't be installed never holds back the agents' config, and an earlier installed version stays in place. When the Skills Registry times out, can't be reached or answers with a server error, the run logs a warning and still succeeds; the next scheduled run retries. A skill the registry doesn't have, or a pinned version that doesn't exist, makes the run exit `3` so the policy gets fixed.

```bash
sudo ./tfy-local-ai-setup \
  --url="https://app.example.com" \
  --tenant="acme" \
  --gateway="https://gateway.example.truefoundry.com/api/llm" \
  --agent opencode,grok,pi \
  --model openai-main/gpt-4.1 --model anthropic-main/claude-sonnet-4-5 \
  --mcp-server github
```

### Exit codes

| Code | Meaning |
|------|---------|
| `0` | Success — configured every requested (or detected) tool, **or** auto-detect found nothing to configure (clean no-op) |
| `1` | Error — details printed to stderr, and to `--log-file` when set |
| `3` | Every agent was configured, but some Agent Skills could not be installed because they need an administrator (not in the Skills Registry, or a pinned version that doesn't exist) |

---

## Full MDM configuration file

`--config` accepts one standalone TOML or JSON file for **every client**, including
Claude Code, Codex and Claude Desktop. It does not read the administrator's personal
settings. A file with `control_plane_url` is in the MDM setup format
(the MDM setup schema),
the one the admin UI produces, and is checked against that schema:

```toml
control_plane_url = "https://acme.truefoundry.cloud"
tenant_name = "acme"
gateway_url = "https://gateway.truefoundry.ai"
mcp_servers = ["@all"]          # everything users can access; or ["github", "linear"], or []
skills = ["eng-skills/code-review"]

[auth]
mode = "oidc"                   # or "api-key" (device login, the default)

[auth.oidc]
issuer = "https://acme.truefoundry.cloud/acme"
client_id = "YOUR_PUBLIC_OIDC_CLIENT_ID"

[agents.claude-code.models]
opus = "anthropic-main/claude-opus-4-5"
sonnet = "anthropic-main/claude-sonnet-4-5"
haiku = "anthropic-main/claude-haiku-4-5"

[agents.codex]
model = "gpt-5.4"

[agents.opencode]
models = ["anthropic-main/claude-sonnet-4-5", "openai-main/gpt-5"]
default_model = "openai-main/gpt-5"

[agents.cursor]
```

Any other MDM setting below (for example `[restrictions]`, `[monitoring]`, `auth.login`,
or an agent's `auth`, `mcp_servers`, `skills` or `managed_settings`) can be added to the
same file by hand.

The original layout, described below, is still supported. See
[the annotated MDM reference](docs/examples/mdm-reference.toml) for all
supported settings with advanced options commented out.
A complete rollout example selects
OIDC for those three clients, device login with refresh for OpenCode and Qwen Code,
`gpt-5.6-sol` as the additional agents' shared default, five MCP servers and two pinned
registry skills. Add further skills to `[skills].names` when the rollout list is finalized.

```bash
go build -o bin/tfy-local-ai-setup .
sudo ./bin/tfy-local-ai-setup --config=mdm.toml \
  --log-file=/var/log/tfy-mdm-test.log
```

Keep the installed binary at a stable, administrator-owned location: clients call it
later to obtain credentials. The relative paths above are for a local test; MDM should
use absolute paths to both the installed binary and its administrator-controlled input.

The original layout is the canonical personal settings layout, with explicit auth modes
(in the setup format, `url`, `tenant`, `gateway`, `tools`, `[defaults]`, `[mcp]`,
`[skills]` and `opus_model`-style keys are replaced by the fields above):

- `url`, `tenant`, `gateway` and `tools` describe the rollout. A present `tools` list
  configures exactly those clients; `tools = []` disables auto-detection.
- `[auth] mode = "api-key"` means per-user device login with refresh. `"oidc"` selects
  OIDC; `[auth.oidc]` supplies `issuer` and `client_id`.
- `[agents.claude-code]`, `[agents.codex]` and `[agents.claude-desktop]` can each set
  `auth` independently. Additional agents share one auth mode because their shell
  credential helper is shared; per-agent `auth` values for them must agree.
- `[defaults] models` and `default_model` apply to list-based agents. Claude Code and
  Desktop use their own `opus_model`, `sonnet_model`, `haiku_model`; Codex uses `model`.
- `[mcp] servers` and `[skills] names` are inherited by every selected client. Per-agent
  `mcp_servers` or `skills` replaces the shared list; `[]` disables it. Desktop skills
  are staged as ZIP files under `~/.tf/skills/claude-desktop` for manual upload in the app.
- `[mcp] discovery = "all-accessible"` is this layout's spelling of `mcp_servers = ["@all"]`,
  with the same silent sign-in requirements described above. Explicit per-agent MCP lists
  still win.
- `[restrictions] blocked_agents` and `message` block AI prompts for `claude-code`,
  `codex` or `cursor`, as described in [Block AI prompts with hooks](#block-ai-prompts-with-hooks).
  `--block-agent`, `--unblock-agent` and `--block-message` apply on top of the file.

For Codex with a ChatGPT subscription, see
the Codex subscription example in the [config reference](docs/mdm-config-reference.md):

```toml
[agents.codex]
auth = "api-key"
auth_mode = "chatgpt-subscription"
```

`auth_mode` maps to `--codex-auth-mode`: `api-key` (the default) or
`chatgpt-subscription`. Subscription mode writes the TrueFoundry token as
`x-tfy-api-key` and sets `requires_openai_auth = true`; users sign in to ChatGPT
in Codex separately. It requires effective Codex `auth = "api-key"`, not `oidc`.
No credentials belong in the input TOML.

Explicit CLI flags override the corresponding file defaults. For example,
`--codex-default-model=another-model` replaces Codex's configured model, and
`--claude-desktop=false` deselects Desktop. `--agent` replaces the file's additional-agent
list; the three primary clients retain their own selection flags. `--mcp-server` replaces
only the shared MCP list, preserving per-agent overrides. `--no-mcp` suppresses the shared
MCP list; `--no-skills` removes all skills installed by MDM, including per-agent selections.
File syntax and unknown keys are checked before authentication or configuration writes.

`--agents-config` retains its older, partial-config behavior for existing deployments;
it cannot be combined with `--config`. All-CLI invocations remain supported.

### Inline managed settings and Desktop context

See [the annotated reference](docs/examples/mdm-reference.toml) for a complete
standalone configuration. These options are specific to the MDM `--config` input:

```toml
[agents.claude-code.managed_settings]
transcriptRetentionDays = 14
allowManagedHooksOnly = true

[agents.claude-code.managed_settings.permissions]
ask = ["Bash(git push:*)", "Read(**/.env)"]
deny = []

[agents.claude-desktop]
tool_search_enabled = true
prefer_1m_context = true

[agents.claude-desktop.context.opus]
supports_1m = true
prefer_1m = true

[agents.claude-desktop.context.sonnet]
supports_1m = true
prefer_1m = true
```

`managed_settings` uses Claude Code's JSON setting names and accepts nested tables,
arrays, numbers and booleans. It is a complete base template, using the same merge
behavior as `--settings-file`: the binary patches gateway URL, auth, model IDs and
gateway MCP entries while preserving other settings. An explicit `--settings-file`
takes precedence over the inline table; otherwise the inline table replaces the
existing on-disk template. With neither supplied, the existing file and built-in
fallback behavior are unchanged. An empty inline table starts from an empty base.
Arbitrary keys are accepted inside `managed_settings` for client compatibility;
unknown keys elsewhere in the MDM config are still rejected.

Desktop's `tool_search_enabled` writes `toolSearchEnabled`; `prefer_1m_context`
writes `modelPrefer1mContext`. On macOS these are native plist booleans; on Windows
they use the existing string policy convention. Omit either option to leave it to
the app's default, or use `false` to explicitly disable it. Removing an option on
a later run removes the corresponding managed preference.

Per-tier `context` entries (`opus`, `sonnet`, `haiku`) add `supports1m` and `prefer1m`
boolean metadata to the configured `inferenceModels` entries. Only set capabilities
your gateway models support; `prefer_1m = true` requires `supports_1m = true` in the
same entry. Unspecified tiers receive no extra metadata, and tiers without model IDs
are omitted. Desktop model IDs do not need Claude Code's `[1m]` suffix.

The binary renders these preferences before writing and locking the file, so
`--dry-run` includes them and no post-run plist editing is needed.

### Deployment behavior in TOML

The standalone MDM input also supports the existing login-failure and Desktop
history-import controls, plus independent gateway URLs for the three primary
clients. Explicit CLI flags still override these values, including `false` and
empty gateway overrides (which fall back to the shared gateway).

```toml
[execution]
write_config_on_login_failure = true
# Optional placeholder when no cached device-login token is available:
# placeholder_token = "TFY_PLACEHOLDER_LOGIN_REQUIRED"

[agents.claude-code]
gateway = "https://claude-gateway.example.com"

[agents.codex]
gateway = "https://codex-gateway.example.com"

[agents.claude-desktop]
gateway = "https://desktop-gateway.example.com"

[agents.claude-desktop.history_import]
enabled = true
encoding = "json-string" # or "native-dict" on macOS
export_enabled = true
banner_behavior = "detect" # off, detect or show
automatic_3p = false
```

Omitting these fields preserves existing CLI defaults. History-import options
other than `enabled` take effect only when import is enabled. Desktop needs a
compatible app version and workspace export permission; terminal Claude Code
history is not imported. Per-client `gateway` is supported only for Claude Code,
Codex and Claude Desktop; other clients use the shared gateway.

`write_config_on_login_failure` retains the existing device-login fallback behavior;
it does not bypass failed MCP discovery or initiate sign-in for scheduled discovery.
OIDC client ID remains under `[auth.oidc] client_id`.

## MDM deployment

The full deployment guide is [docs/mdm-deployment.md](docs/mdm-deployment.md). It covers choosing an auth mode, writing `mdm.toml`, the per-OS deploy scripts, scheduling, ring-based rollout, verification, upgrades and rollback. The public step-by-step guide is [MDM setup](https://www.truefoundry.com/docs/ai-gateway/mdm-setup).

Each script installs a pinned release (checked against the release's `SHA256SUMS` when it has one), writes your [`mdm.toml`](#full-mdm-configuration-file) to a root/Administrator-owned path, and runs the binary with `--config` and `--log-file`. Run it hourly as root or SYSTEM, and keep `RELEASE_TAG` identical across the three OS scripts.

| OS | Script (in the guide) | `mdm.toml` | Log |
|----|----------------------|------------|-----|
| macOS (Jamf, Mosyle, Kandji) | bash, run as root | `/Library/Application Support/TrueFoundry/mdm.toml` | `/var/log/tfy-mdm-setup.log` |
| Linux (Ansible, Chef, Puppet, cron) | bash, run as root with `TFY_MDM_USER` | `/etc/truefoundry/mdm.toml` | `/var/log/tfy-mdm-setup.log` |
| Windows (Intune, SCCM, Group Policy) | PowerShell, run as SYSTEM or Administrator | `C:\Program Files\TrueFoundry\mdm.toml` | `C:\Program Files\TrueFoundry\mdm-setup.log` |

Preview any script with `DRY_RUN=true` (macOS, Linux) or `$DryRun = $true` (Windows). See [Dry run](#dry-run) below.

## Running manually

If you need to trigger setup outside the MDM schedule — for example, to complete a first-time login or re-authenticate after a token expiry — run the binary directly as root/Administrator with the `mdm.toml` the deployment script wrote. A browser window will open for the developer to complete the device-flow login. After that, future MDM runs will be silent. All-flag invocations (`--url`, `--tenant`, `--gateway`, ...) remain supported.

```bash
# macOS
sudo /usr/local/bin/tfy-local-ai-setup \
  --config="/Library/Application Support/TrueFoundry/mdm.toml" \
  --log-file=/var/log/tfy-mdm-setup.log

# Linux
sudo /usr/local/bin/tfy-local-ai-setup \
  --config=/etc/truefoundry/mdm.toml \
  --log-file=/var/log/tfy-mdm-setup.log
```

```powershell
# Windows (PowerShell as Administrator)
& "C:\Program Files\TrueFoundry\tfy-local-ai-setup.exe" `
  --config="C:\Program Files\TrueFoundry\mdm.toml" `
  --log-file="C:\Program Files\TrueFoundry\mdm-setup.log"
```

### Dry run

Use `--dry-run` to inspect the config that would be written without touching the managed files — useful for validating config before a fleet rollout. The deployment scripts above do this with `DRY_RUN=true` (`$DryRun = $true` on Windows):

```bash
sudo /usr/local/bin/tfy-local-ai-setup \
  --config="/Library/Application Support/TrueFoundry/mdm.toml" \
  --dry-run
```

---

## Using custom model IDs

The default model IDs (`claude-code/claude-opus` etc.) work for standard TrueFoundry deployments. If your setup uses different model IDs — for example a direct provider model, a Claude Enterprise account, or a virtual model — pass them via the model flags:

```bash
sudo /usr/local/bin/tfy-local-ai-setup \
  --url="https://app.example.com" \
  --tenant="myorg" \
  --gateway="https://gateway.example.truefoundry.com" \
  --opus-model="claude-enterprise/claude-opus-4-6" \
  --sonnet-model="claude-enterprise/claude-sonnet-4-6" \
  --haiku-model="claude-enterprise/claude-haiku-4-5"
```

Copy the exact model IDs from **Integrations → Providers** or **Virtual Models** in the TrueFoundry control plane.

---

## Using a custom settings template

If you need to deploy settings beyond what the built-in default config provides (additional `deny` rules, `allowedMcpServers`, etc.), put your full desired config in a JSON file and pass it via `--settings-file`. The binary will read it as the base, then patch gateway URL, model IDs, and auth fields:

```bash
sudo /usr/local/bin/tfy-local-ai-setup \
  --url="https://app.example.com" \
  --tenant="myorg" \
  --settings-file="/etc/tfy/base-settings.json"
```

The binary always updates `ANTHROPIC_BASE_URL` and `ANTHROPIC_DEFAULT_*_MODEL`. In **api-key** mode it
writes `X-TFY-API-KEY` and removes `apiKeyHelper`; in **oidc** mode it writes `apiKeyHelper` and
strips `X-TFY-API-KEY`. All other keys in your template are passed through unchanged.

**Priority order for the base config:**
1. `--settings-file` (if provided)
2. Existing `managed-settings.json` on disk (patched in-place)
3. Built-in default config (written fresh)

---

## File locking

After writing, the binary locks the file against modifications by non-root/non-admin users:

| OS | Lock mechanism | How to unlock for editing |
|----|---------------|--------------------------|
| macOS | `chflags schg` (system immutable flag) | `sudo chflags noschg <path>` |
| Linux | `chattr +i` (immutable ext attribute) | `sudo chattr -i <path>` |
| Windows | `icacls` ACL (SYSTEM:F, Admins:R, Users:R) | Re-run the MDM script — it grants write access automatically before updating |

---

## Troubleshooting

The full guide, with diagnostic commands, exit codes and an error catalogue, is [docs/troubleshooting.md](docs/troubleshooting.md). The most common cases are below.

**Browser login appears on every MDM run**

The device-flow browser prompt should only appear on the first run or when the refresh token at `~/.tf/refresh-token` expires. If it appears on every run, the token is not being persisted — check that the MDM script re-execs the binary as the logged-in user (not as root) and that the user's home directory is writable.

**I accidentally closed the browser login window**

The expired device code is abandoned immediately. Run the binary manually as shown above — a new browser window opens straight away. You do not need to wait for the next MDM sync.

**`401 Unauthorized` or `403 Forbidden` in Claude Code**

The session token has expired or was never initialized. Run the binary manually to re-authenticate. After completing the login, future MDM runs will keep the token fresh automatically.

**`chattr: Operation not supported` on Linux**

`chattr +i` requires a filesystem that supports the immutable attribute (ext2/3/4, xfs with `attr` support). On other filesystems (tmpfs, overlayfs, NFS) the binary logs a warning and continues — the file is written correctly but is not immutable-locked.

---

## Building from source

The source code for `tfy-local-ai-setup` is not published in this repository. This repository carries the release binaries, the installers, and the documentation. To verify a release, compare its binary with the `SHA256SUMS` file attached to the same release, and read the [development notes](docs/development.md) for how releases are produced.

## Releasing

Releases are published to this repository as GitHub releases. Each release carries the five binaries, `SHA256SUMS`, `install.sh` and `install.ps1`. The release process is described in [development](docs/development.md#release).

## Advanced MDM features

### Manual PAT onboarding for scheduled MDM runs

Set `auth.login = "manual"` to keep browser sign-in out of scheduled runs.
The MDM process saves the organization policy first and applies configuration
that does not need credentials. Skills and authenticated MCP discovery wait
until the user signs in. Existing configuration is retained when authentication
or discovery is unavailable.

A full MDM TOML file can be supplied with `--config`:

```toml
url = "https://your-control-plane.example.com"
tenant = "your-tenant"
gateway = "https://your-gateway.example.com"
tools = ["claude-code", "codex", "goose"]

[auth]
mode = "api-key"
login = "manual"
# Optional: skip team selection when creating the PAT.
# pat_team = "engineering"
# Optional: how long each user's key lasts (7–3650, default 365 days).
# pat_expiry_days = 90
# Optional: renew each user's key before it expires (default off).
# auto_renew = true

[defaults]
models = ["your-model"]

[agents.codex]
model = "your-model"

[mcp]
discovery = "all-accessible"

[skills]
names = ["your-repo/your-skill:1"]
```

MDM runs as administrator:

```sh
tfy-local-ai-setup --config=/path/to/mdm.toml
```

Alternatively, add `--login=manual` and optionally `--pat-team=engineering`
to existing managed CLI arguments. `--agents-config` also accepts the new
`auth.login` and `auth.pat_team` settings. Full `--config` maps connection,
tool and legacy-agent settings to managed flags; explicit CLI flags take
precedence. Shared model, skill and MCP settings use the existing policy loader.

The user runs, without `sudo`:

```sh
tfy-local-ai-setup login --mdm
```

This reads the saved organization policy, opens browser sign-in when needed,
creates a PAT (365 days, or `auth.pat_expiry_days`), and securely saves it. If the server requires a team
and IT did not configure one, the command lists available teams in an interactive
selector. `login --mdm --pat-team=engineering` supplies a team explicitly.
The server validates permission to create the PAT for the selected team.

The same command discovers accessible MCP servers, applies user-owned settings,
and installs missing skills immediately. Administrator-owned settings containing
literal credentials are completed by the next privileged MDM run. Repeating the
login command reuses a valid PAT; expired or revoked PATs are replaced after
sign-in. A temporary verification failure does not create another PAT.

Future scheduled runs reuse the PAT without browser login and refresh discovery.

With `auth.auto_renew = true` (or `--auto-renew-key`), the user part of each
scheduled run also renews the PAT before it expires, with the PAT itself and
no browser: halfway through its life for a key of 30 days or less, when a
quarter is left for a longer one. The user's own agent files get the new key in
that run and the administrator-owned files on the next run; the old key keeps
working for 7 days (or until it expires) and is then deleted. Shortening `auth.pat_expiry_days` does
not renew existing keys early. A PAT that already expired or was revoked still
needs `login --mdm`. Off by default.

#### Device login with `auth.auto_renew`

`auth.auto_renew = true` also works without `login = "manual"`, with device
login (`mode = "api-key"`). There it swaps device login's refresh token for a
key: a long-lived credential in place of hour-long tokens. On the next run, a
user with no key is signed in as device login always does: silently with a
cached sign-in, otherwise in the browser. That sign-in makes a `tfy-mdm-*` key
in the user's keychain (lasting `auth.pat_expiry_days`, default 365), and the
device-login token files are removed. From then on the run works as manual
login with `auto_renew`: the key goes into the managed files, renews itself,
and no browser opens. A key that is revoked or expires is replaced the same
way on the next run.

If TrueFoundry refuses the key (the user's role may not create keys, or the
user is in several teams and `auth.pat_team` isn't set), that run uses device
login's token as before and the next run tries again. Not with `mode =
"oidc"`. Off by default, so existing fleets are unchanged.
Cached skill versions are reused; registry metadata checks can still occur.
Failed discovery never replaces existing MCP settings with an empty list.

Storage:

- Desired policy: `agents-policy.json` in `/Library/Application Support/TrueFoundry`
  (macOS), `/etc/truefoundry` (Linux), or `%ProgramFiles%\TrueFoundry` (Windows).
  This administrator-owned file contains configuration, never PAT values.
- Credentials: the user's OS credential store, isolated by control-plane URL
  and tenant from personal-mode credentials. If unavailable, the existing
  owner-only fallback is `~/.tf/personal-mdm-<account-hash>-pat`.
- Applied user state and pending work: `~/.tf/mdm-user.json`.
- Download cache: `~/.tf/cache/skills/<tenant>/<version-id>`; installed skills
  remain in each agent's own skills directory.

Omitting `login = "manual"` preserves the existing managed authentication flow.
Manual PAT login requires `auth.mode = "api-key"`; existing OIDC mode is unchanged.

### Block AI prompts with hooks

Claude Code, Codex and Cursor can be blocked through their prompt-submission
hooks. These restrictions do not prevent launching the editor, manual editing,
Cursor Tab completion, or using another client.

```sh
# Personal: applies immediately, without gateway login or wizard questions.
./tfy-local-ai-setup personal --block-agent cursor --block-message "Use the approved agent."
# Repeat --block-agent, or separate IDs with commas.
./tfy-local-ai-setup personal --block-agent claude-code,codex --dry-run
# Remove only the hook installed by this setup.
./tfy-local-ai-setup personal --unblock-agent cursor

# Managed: run as root / Administrator. No gateway credentials are needed
# when applying only restrictions.
sudo ./tfy-local-ai-setup --block-agent claude-code,codex,cursor --block-message "Contact IT."
sudo ./tfy-local-ai-setup --unblock-agent cursor
```

Restrictions also work alongside regular setup flags. Personal `--config`,
managed `--config` and managed `--agents-config` accept:

```toml
[restrictions]
blocked_agents = ["claude-code", "codex", "cursor"]
message = "Use the approved agent."
```

An explicit `blocked_agents = []` removes this setup's restrictions. CLI blocks
and unblocks modify the saved list and `--block-message` replaces the file's
message; omitted restrictions retain the last applied list. A managed `--config`
applies its `[restrictions]` during the regular setup run, so the file still
needs `url` and `tenant`; use the CLI flags or `--agents-config` for a
restrictions-only run. Existing unrelated hooks and settings are preserved.
Personal mode keeps first-write `.bak` copies unless `--no-backup` is passed.

| Agent | Hook | Personal location | Managed location |
| --- | --- | --- | --- |
| Claude Code | `UserPromptSubmit` | `$CLAUDE_CONFIG_DIR/settings.json` or `~/.claude/settings.json` | Existing platform managed-settings path |
| Codex | `UserPromptSubmit` | `$CODEX_HOME/hooks.json` or `~/.codex/hooks.json` | `/etc/codex/hooks.json`; Windows `%ProgramData%/OpenAI/Codex/hooks.json` |
| Cursor | `beforeSubmitPrompt` | `~/.cursor/hooks.json` | macOS `/Library/Application Support/Cursor/hooks.json`; Linux `/etc/cursor/hooks.json`; Windows `%ProgramData%/Cursor/hooks.json` |

The hook prints the configured message and exits with code 2. Claude Code uses
`onFailure = "block"` (requires Claude Code 2.1.295 or later); Cursor uses
`failClosed: true`. Codex's `features.hooks` is enabled in personal config or
managed requirements while blocked; unblocking restores its previous value if
it has not subsequently been changed.

**Personal restrictions are voluntary:** users can remove or disable them.
Codex requires reviewing and trusting the new hook through `/hooks` before it
runs; setup does not bypass that review. Claude Code's local hook-disable settings
can also prevent personal hooks from running. Managed hooks require a compatible
client version and protected system configuration. Restart the client or open a
new session after changing hooks. Windows restrictions apply to native clients;
WSL clients need restrictions installed separately inside each distro.

References: [Claude Code hooks](https://code.claude.com/docs/en/hooks),
[Codex hooks](https://developers.openai.com/codex/hooks),
[Cursor hooks](https://cursor.com/docs/agent/hooks).


### Monitor Cursor through MDM

Opt in to copying Cursor prompt, response, tool and session events to TrueFoundry without changing
Cursor's model routing. Add this to the standalone `--config` TOML (or use the
same structure in JSON):

```toml
url = "https://app.example.com"
tenant = "acme"
gateway = "https://gateway.truefoundry.ai"

[auth]
mode = "api-key"

[monitoring.cursor]
enabled = true
```

```sh
sudo tfy-local-ai-setup --config=/path/to/mdm.toml

# Equivalent flags, using the existing per-user MDM authentication flow:
sudo tfy-local-ai-setup --url=https://app.example.com \
  --tenant=internal --gateway=https://gateway.truefoundry.ai \
  --agents-auth=api-key --cursor-monitoring

# Remove only TrueFoundry's monitoring hooks; explicit flags override TOML:
sudo tfy-local-ai-setup --config=/path/to/mdm.toml --cursor-monitoring=false
```

Monitoring defaults to off. It works independently of `tools` / `--agent` and
requires no Cursor model settings or MCP configuration. With monitoring explicitly
configured, other clients are configured only when selected through `tools` or
client flags; the monitoring-only example does not auto-detect and configure them.
For OIDC, use `auth.mode = "oidc"` and the existing `[auth.oidc]` issuer/client ID
(or `--agents-auth=oidc --oidc-issuer=... --oidc-client-id=...`). Manual PAT login
(`auth.login = "manual"`) also uses the existing `login --mdm` flow.

MDM merges nine hooks into Cursor's system configuration:
`beforeSubmitPrompt`, `afterAgentResponse`, `beforeMCPExecution`,
`afterMCPExecution`, `preToolUse`, `postToolUse`, `stop`, `sessionStart` and
`sessionEnd`. Locations are macOS
`/Library/Application Support/Cursor/hooks.json`, Linux `/etc/cursor/hooks.json`,
and Windows `%ProgramData%/Cursor/hooks.json`. Existing hooks, including prompt
blocking, are preserved. Native Windows hooks do not cover Cursor running inside
WSL or cloud agents. The installed setup binary must stay at its deployed path;
rerunning MDM reconciles the command after a binary path change.

Each hook invokes `tfy-local-ai-setup cursor-monitor --event=<event>`, which receives JSON on stdin,
loads the protected machine policy and retrieves the user's credential through
`token --mdm`. It sends the original JSON to `POST <gateway>/hooks/cursor` with
`Authorization: Bearer <user-token>` and `Content-Type: application/json`, without
`x-tfy-hook-guardrails`. A gateway path prefix is preserved: for example,
`https://host/api/llm` becomes `https://host/api/llm/hooks/cursor`.

Supported event payloads are recorded on gateway traces when tracing is enabled,
subject to tracing redaction and size limits. Forwarded fields include model,
model ID/parameters when available, conversation/generation IDs,
prompt/response/tool content and session lifecycle data. This is event monitoring;
it does not capture the complete underlying inference traffic or authoritative
token usage/cost.

The helper shares a two-second deadline across authentication and HTTP delivery,
does not follow redirects or retry, and rejects payloads over 4 MiB. Delivery is
best effort: failures produce a generic stderr diagnostic without credentials or
event contents. Gateway deny/mutation responses are discarded. By default the helper
returns the valid allow/no-op response for the event; hook entries use
`failClosed: false` and a three-second timeout. Other installed hooks can still
block actions. Disabling monitoring removes only this installation's monitoring
commands and preserves unrelated hooks.


To block Cursor agent prompts and tool execution until the developer signs in
through the CLI, keep the URL/tenant/gateway settings above and use:

```toml
[auth]
mode = "api-key"
login = "manual"

[monitoring.cursor]
enabled = true
require_login = true
```

```sh
sudo tfy-local-ai-setup --config=/path/to/mdm.toml
# Developer runs this without sudo:
tfy-local-ai-setup login --mdm

# CLI override to enable the gate, or =false to return to monitoring only:
sudo tfy-local-ai-setup --config=/path/to/mdm.toml --cursor-require-login
```

`require_login` defaults to false and applies only when monitoring is enabled.
`login = "manual"` makes sign-in an explicit developer action; other existing
auth modes gate on the credential available through their existing MDM flow.
The prompt, pre-tool and pre-MCP hooks deny with sign-in instructions when
`token --mdm` cannot obtain a credential. These hooks use `failClosed: true` so a missing
policy, helper crash or timeout cannot silently bypass the gate. Response, post-tool,
post-MCP, stop and session hooks remain observational and never enforce login.

After a credential is available, forwarding remains best effort: gateway
outages and guardrail verdicts do not block Cursor. The gate checks token-helper
availability, not gateway validation of the credential on every event. It covers
agent prompt/tool actions; it does not block all Cursor network traffic or Tab
completion. Setting `require_login = false` (or passing
`--cursor-require-login=false`) keeps monitoring while removing the login gate.

The additional observational events require a gateway version that supports
`afterAgentResponse`, `afterMCPExecution`, `stop`, `sessionStart` and `sessionEnd`
on `POST /hooks/cursor`. Upgrade the gateway before rolling out these hooks.
No guardrail selectors are sent by the monitoring helper. Generic tool and MCP
hooks can describe the same operation; correlate them using the IDs available
in each payload rather than counting each hook as a distinct tool call.
