# tfy-local-ai-setup

Tools for deploying TrueFoundry-managed AI clients — **Claude Code**, **Codex**, and **Claude Desktop (Cowork on 3P)** — on developer machines via MDM (Mobile Device Management).

## tfy-local-ai-setup

A single binary that handles everything needed to configure Claude Code on a managed machine: authenticates the logged-in developer with TrueFoundry, builds the appropriate `managed-settings.json`, and locks it so it cannot be modified by developers. Designed to run on a schedule (hourly) via Jamf, Mosyle, Kandji, Intune, SCCM, or any MDM that supports script execution.

### How it works

On every run the binary does five things:

1. **Decides which tools to configure** — with no tool flag it auto-detects what is installed (`claude`/`codex` on `PATH`, `Claude.app` / registry for Claude Desktop); with explicit flags it honors each, but only for tools that are actually installed. A tool is **never** configured if it is not installed — an explicitly-requested-but-absent tool is skipped with a warning. If nothing is left to configure, the binary exits **0 immediately** (no token fetch, no login popup) so hourly MDM runs on machines without any AI client are a clean no-op.
2. **Detects the logged-in user** — platform-specific: `scutil` on macOS, `SUDO_USER`/`logname` on Linux, WMI on Windows.
3. **Gets a fresh auth token** — attempts a silent refresh from `~/.tf/refresh-token`. If the token is missing or expired, opens the browser device-authorization flow in the user's session. After the first login, subsequent runs complete silently. The obtained access token and refresh token are stored under `~/.tf/`.
4. **Writes `managed-settings.json`** — injects `ANTHROPIC_BASE_URL`, model IDs, and the token into `ANTHROPIC_CUSTOM_HEADERS`. If a file already exists on disk (or a `--settings-file` template is provided), it is patched in-place — all other keys are preserved. If there is no existing file, a secure default config is written.
5. **Locks the file** — `chflags schg` on macOS, `chattr +i` on Linux, `icacls` ACL on Windows. Developers cannot modify the file without root/Administrator access.

### Destination paths

| OS      | Managed settings path |
|---------|-----------------------|
| macOS   | `/Library/Application Support/ClaudeCode/managed-settings.json` |
| Linux   | `/etc/claude-code/managed-settings.json` |
| Windows | `C:\Program Files\ClaudeCode\managed-settings.json` |

---

## Binaries

Pre-built binaries for all supported platforms are in `bin/`:

| File | Platform |
|------|----------|
| `bin/tfy-local-ai-setup-darwin-arm64` | macOS — Apple Silicon (M1/M2/M3/M4) |
| `bin/tfy-local-ai-setup-darwin-amd64` | macOS — Intel |
| `bin/tfy-local-ai-setup-linux-amd64` | Linux — x86_64 |
| `bin/tfy-local-ai-setup-windows-amd64.exe` | Windows — x86_64 |

On macOS and Linux you need to make the binary executable before running:

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
| `--url` | **Yes** | — | Base URL of the TrueFoundry control plane (e.g. `https://app.example.truefoundry.com`) |
| `--tenant` | **Yes** | — | Your TrueFoundry tenant name |
| `--gateway` | No | value of `--url` | Common gateway URL for all tools. Used as the default for `--claude-gateway` and `--codex-gateway` if neither is set. |
| `--claude-code` | No | auto-detect | Configure Claude Code managed settings **if it is installed**. If neither `--claude-code` nor `--codex` is set, the binary auto-detects which tools are installed. An explicit `--claude-code` on a machine without Claude Code is skipped with a warning — no file is written. |
| `--codex` | No | auto-detect | Configure Codex managed settings **if it is installed**. If neither `--claude-code` nor `--codex` is set, the binary auto-detects which tools are installed. An explicit `--codex` on a machine without Codex is skipped with a warning — no file is written. |
| `--claude-gateway` | No | value of `--gateway` | Gateway URL for Claude Code (written to `ANTHROPIC_BASE_URL`). Overrides `--gateway` for Claude Code only. |
| `--codex-gateway` | No | value of `--gateway` | Gateway URL for Codex (written to `base_url` in the provider config). Defaults to `--gateway`. |
| `--claude-desktop` | No | auto-detect | Configure Claude Desktop (Cowork 3P) managed preferences **if it is installed**. If none of `--claude-code` / `--codex` / `--claude-desktop` is set, the binary auto-detects which are installed. An explicit `--claude-desktop` on a machine without Claude Desktop is skipped with a warning — no preferences are written. |
| `--claude-desktop-gateway` | No | value of `--gateway` | Gateway URL for Claude Desktop (written to `inferenceGatewayBaseUrl`). Overrides `--gateway` for Claude Desktop only. |
| `--desktop-opus-model` | No | value of `--opus-model` | Opus model ID for Claude Desktop (`inferenceModels`). Inherits the Claude Code opus model unless overridden. |
| `--desktop-sonnet-model` | No | value of `--sonnet-model` | Sonnet model ID for Claude Desktop (`inferenceModels`). Inherits the Claude Code sonnet model unless overridden. |
| `--desktop-haiku-model` | No | value of `--haiku-model` | Haiku model ID for Claude Desktop (`inferenceModels`). Inherits the Claude Code haiku model unless overridden. |
| `--desktop-header` | No | — | Extra custom header for Claude Desktop as `Name: Value`, written to `inferenceCustomHeaders`. Repeatable — pass it multiple times for multiple headers. |
| `--opus-model` | No | `claude-code/claude-opus` | Model ID written to `ANTHROPIC_DEFAULT_OPUS_MODEL` (Claude Code only) |
| `--sonnet-model` | No | `claude-code/claude-sonnet` | Model ID written to `ANTHROPIC_DEFAULT_SONNET_MODEL` (Claude Code only) |
| `--haiku-model` | No | `claude-code/claude-haiku` | Model ID written to `ANTHROPIC_DEFAULT_HAIKU_MODEL` (Claude Code only) |
| `--settings-file` | No | — | Path to a JSON template for `managed-settings.json` (Claude Code only). Patches token and model IDs; all other keys are preserved. Falls back to the existing file on disk, then the built-in default. |
| `--refresh-token-file` | No | `~/.tf/refresh-token` | Path where the refresh token is stored (0600, owner-only). Also read on startup to attempt a silent refresh. Written in the logged-in user's session. |
| `--access-token-file` | No | `~/.tf/access-token` | Path where the freshly obtained access token (JWT) is stored (0600, owner-only). Written in the logged-in user's session. |
| `--dry-run` | No | `false` | Print the resulting config to stdout instead of writing any files. Works for both Claude Code and Codex. |
| `--log-file` | No | — | Also append all log output to this file (0600). Useful for headless MDM runs where there is no console to read. |
| `--debug` | No | `false` | Verbose debug logging. Surfaces the device-login browser URL and streams the (Windows) device-login child process output, and keeps the child log file for inspection. |

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
    "ask": ["Bash(git push:*)", "Write(**)"]
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
    "ANTHROPIC_DEFAULT_OPUS_MODEL": "claude-code/claude-opus",
    "ANTHROPIC_DEFAULT_SONNET_MODEL": "claude-code/claude-sonnet",
    "ANTHROPIC_DEFAULT_HAIKU_MODEL": "claude-code/claude-haiku",
    "CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS": "1",
    "ANTHROPIC_CUSTOM_HEADERS": "X-TFY-API-KEY: <token>"
  }
}
```

Use `--dry-run` to inspect the exact output before deploying to a fleet.

### Codex default config

When `--codex` is set (or Codex is auto-detected), the binary writes `/etc/codex/config.toml` with just the provider section. Model selection and all other settings come from the user's own config or a separately managed `requirements.toml`.

```toml
model_provider = "truefoundry"

[model_providers.truefoundry]
name     = "TrueFoundry Gateway"
base_url = "<value of --gateway>"
wire_api = "responses"

[model_providers.truefoundry.http_headers]
Authorization = "Bearer <token>"
```

**Codex config paths:**

| OS | Path |
|----|------|
| macOS + Linux | `/etc/codex/config.toml` |
| Windows | Not supported (no system-level config path) |

### Claude Desktop (Cowork on 3P)

When `--claude-desktop` is set (or Claude Desktop is auto-detected), the binary writes the
[`com.anthropic.claudefordesktop`](https://claude.com/docs/cowork/3p/configuration) managed
preferences that point Claude Desktop's **Cowork on 3P** mode at the TrueFoundry gateway. Every
value is stored as a string; arrays and objects are JSON-encoded strings, as required by Anthropic's
configuration reference.

The auth token is delivered in the **`X-TFY-API-KEY`** header via `inferenceCustomHeaders` — the same
mechanism Claude Code uses via `ANTHROPIC_CUSTOM_HEADERS`. (`inferenceGatewayApiKey` is also set to the
token so the app's gateway mode is enabled; the gateway authenticates via either.) The model picker is
pinned to the explicit list and `modelDiscoveryEnabled` is set to `false`, so Claude Desktop never
auto-discovers models via the gateway's `GET /v1/models`.

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
  <key>inferenceGatewayAuthScheme</key>
  <string>bearer</string>
  <key>inferenceGatewayApiKey</key>
  <string>&lt;token&gt;</string>
  <key>inferenceCustomHeaders</key>
  <string>{"X-TFY-API-KEY":"&lt;token&gt;","X-TFY-METADATA":"{\"team\":\"platform\"}"}</string>
  <key>modelDiscoveryEnabled</key>
  <string>false</string>
  <key>inferenceModels</key>
  <string>[{"name":"claude-code/claude-opus","anthropicFamilyTier":"opus","isFamilyDefault":true}, ...]</string>
</dict>
</plist>
```

Each `--desktop-*-model` flag defaults to the corresponding Claude Code model flag
(`--opus-model` / `--sonnet-model` / `--haiku-model`), so Claude Desktop uses the same models you
configure for Claude Code unless you override a specific one. The picker is pinned to this explicit
list and `modelDiscoveryEnabled` is set to `false`, so Claude Desktop does **not** auto-discover
models from the gateway's `GET /v1/models`. Override the desktop flags for a direct provider or
Claude Enterprise account.

```bash
sudo /usr/local/bin/tfy-local-ai-setup \
  --url="https://app.example.truefoundry.com" \
  --tenant="myorg" \
  --gateway="https://gateway.example.truefoundry.com/api/llm" \
  --claude-desktop \
  --desktop-opus-model="tfy-ai-anthropic/claude-opus-4-6" \
  --desktop-sonnet-model="tfy-ai-anthropic/claude-sonnet-4-6" \
  --desktop-haiku-model="tfy-ai-anthropic/claude-haiku-4-5" \
  --desktop-header='X-TFY-METADATA: {"team":"platform"}'
```

Claude Desktop loads managed preferences at launch — **restart Claude Desktop** after a run for
changes to take effect. To verify what the app picked up, use **Help → Troubleshooting → Copy
Managed Configuration Report** (secrets redacted).

### Exit codes

| Code | Meaning |
|------|---------|
| `0` | Success — configured every installed tool, **or** nothing was installed to configure (clean no-op) |
| `1` | Error — details printed to stderr |

---

## MDM deployment

The recommended approach is to deploy a thin wrapper script via your MDM that downloads the correct binary for the platform (once, or when a new release tag is available) and runs it. Schedule the script to run hourly so the token stays fresh.

The wrapper scripts below don't pass `--claude-code` / `--codex` / `--claude-desktop`, so the binary **auto-detects** which clients are installed and configures each of them. To configure Claude Desktop with a pinned model list or extra custom headers, add the `--claude-desktop` and `--desktop-*` flags to the run command (see [Claude Desktop (Cowork on 3P)](#claude-desktop-cowork-on-3p)).

### macOS (Jamf / Mosyle / Kandji)

```bash
#!/bin/bash
# Must be run as root.
set -euo pipefail

[[ "$(id -u)" -ne 0 ]] && { echo "ERROR: Must be run as root." >&2; exit 1; }

# ---------------------------------------------------------------------------
# Config — update these values for your environment
# ---------------------------------------------------------------------------
GATEWAY_URL="<your-gateway-url>"
CONTROL_PLANE_URL="<your-control-plane-url>"
TENANT_NAME="<your-tenant-name>"

# Optional: override model IDs (defaults shown below — can be a direct provider model or a virtual model)
# OPUS_MODEL="claude-code/claude-opus"
# SONNET_MODEL="claude-code/claude-sonnet"
# HAIKU_MODEL="claude-code/claude-haiku"

# Optional: path to a JSON template to use as base config
# SETTINGS_FILE="/etc/tfy/base-settings.json"

BINARY_PATH="/usr/local/bin/tfy-local-ai-setup"
RELEASE_TAG="v1.2.0"
RELEASE_BASE="https://github.com/truefoundry/tfy-local-ai-setup/releases/download/${RELEASE_TAG}"
VERSION_FILE="${BINARY_PATH}.version"

log() { echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*"; }

# ---------------------------------------------------------------------------
# Install binary (skip if already on the correct release tag)
# ---------------------------------------------------------------------------
INSTALLED_TAG="$([[ -f "${VERSION_FILE}" ]] && cat "${VERSION_FILE}" || echo '')"

if [[ ! -f "${BINARY_PATH}" ]] || [[ "${INSTALLED_TAG}" != "${RELEASE_TAG}" ]]; then
  case "$(uname -m)" in
    arm64)  BINARY_URL="${RELEASE_BASE}/tfy-local-ai-setup-darwin-arm64" ;;
    x86_64) BINARY_URL="${RELEASE_BASE}/tfy-local-ai-setup-darwin-amd64" ;;
    *) echo "ERROR: Unsupported architecture: $(uname -m)" >&2; exit 1 ;;
  esac
  log "Installing tfy-local-ai-setup ${RELEASE_TAG} ($(uname -m))..."
  curl -fsSL "${BINARY_URL}" -o "${BINARY_PATH}" && chmod +x "${BINARY_PATH}"
  echo "${RELEASE_TAG}" > "${VERSION_FILE}"
  log "tfy-local-ai-setup installed at ${BINARY_PATH}."
else
  log "tfy-local-ai-setup ${RELEASE_TAG} already installed — skipping download."
fi

# ---------------------------------------------------------------------------
# Run
# ---------------------------------------------------------------------------
exec "${BINARY_PATH}" \
  --url="${CONTROL_PLANE_URL}" \
  --tenant="${TENANT_NAME}" \
  --gateway="${GATEWAY_URL}" \
  ${OPUS_MODEL:+--opus-model="${OPUS_MODEL}"} \
  ${SONNET_MODEL:+--sonnet-model="${SONNET_MODEL}"} \
  ${HAIKU_MODEL:+--haiku-model="${HAIKU_MODEL}"} \
  ${SETTINGS_FILE:+--settings-file="${SETTINGS_FILE}"}
```

### Linux (Ansible / Chef / Puppet / cron)

```bash
#!/bin/bash
# Must be run as root.
set -euo pipefail

[[ "$(id -u)" -ne 0 ]] && { echo "ERROR: Must be run as root." >&2; exit 1; }

# ---------------------------------------------------------------------------
# Config — update these values for your environment
# ---------------------------------------------------------------------------
GATEWAY_URL="<your-gateway-url>"
CONTROL_PLANE_URL="<your-control-plane-url>"
TENANT_NAME="<your-tenant-name>"

# Optional: override model IDs
# OPUS_MODEL="claude-code/claude-opus"
# SONNET_MODEL="claude-code/claude-sonnet"
# HAIKU_MODEL="claude-code/claude-haiku"

# Optional: path to a JSON template
# SETTINGS_FILE="/etc/tfy/base-settings.json"

BINARY_PATH="/usr/local/bin/tfy-local-ai-setup"
RELEASE_TAG="v1.2.0"
BINARY_URL="https://github.com/truefoundry/tfy-local-ai-setup/releases/download/${RELEASE_TAG}/tfy-local-ai-setup-linux-amd64"
VERSION_FILE="${BINARY_PATH}.version"

log() { echo "[$(date '+%Y-%m-%d %H:%M:%S')] $*"; }

# ---------------------------------------------------------------------------
# Install binary (skip if already on the correct release tag)
# ---------------------------------------------------------------------------
INSTALLED_TAG="$([[ -f "${VERSION_FILE}" ]] && cat "${VERSION_FILE}" || echo '')"

if [[ ! -f "${BINARY_PATH}" ]] || [[ "${INSTALLED_TAG}" != "${RELEASE_TAG}" ]]; then
  log "Installing tfy-local-ai-setup ${RELEASE_TAG} (linux-amd64)..."
  curl -fsSL "${BINARY_URL}" -o "${BINARY_PATH}" && chmod +x "${BINARY_PATH}"
  echo "${RELEASE_TAG}" > "${VERSION_FILE}"
  log "tfy-local-ai-setup installed at ${BINARY_PATH}."
else
  log "tfy-local-ai-setup ${RELEASE_TAG} already installed — skipping download."
fi

# ---------------------------------------------------------------------------
# Run
# ---------------------------------------------------------------------------
exec "${BINARY_PATH}" \
  --url="${CONTROL_PLANE_URL}" \
  --tenant="${TENANT_NAME}" \
  --gateway="${GATEWAY_URL}" \
  ${OPUS_MODEL:+--opus-model="${OPUS_MODEL}"} \
  ${SONNET_MODEL:+--sonnet-model="${SONNET_MODEL}"} \
  ${HAIKU_MODEL:+--haiku-model="${HAIKU_MODEL}"} \
  ${SETTINGS_FILE:+--settings-file="${SETTINGS_FILE}"}
```

### Windows (Intune / SCCM / Group Policy)

```powershell
# Must be run as Administrator.
#Requires -RunAsAdministrator
Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

# ---------------------------------------------------------------------------
# Config — update these values for your environment
# ---------------------------------------------------------------------------
$GatewayUrl      = "<your-gateway-url>"
$ControlPlaneUrl = "<your-control-plane-url>"
$TenantName      = "<your-tenant-name>"

# Optional: override model IDs
# $OpusModel   = "claude-code/claude-opus"
# $SonnetModel = "claude-code/claude-sonnet"
# $HaikuModel  = "claude-code/claude-haiku"

# Optional: path to a JSON template
# $SettingsFile = "C:\ProgramData\TrueFoundry\base-settings.json"

$BinaryDir   = "C:\Program Files\TrueFoundry"
$BinaryPath  = "$BinaryDir\tfy-local-ai-setup.exe"
$ReleaseTag  = "v1.2.0"
$BinaryUrl   = "https://github.com/truefoundry/tfy-local-ai-setup/releases/download/$ReleaseTag/tfy-local-ai-setup-windows-amd64.exe"
$VersionFile = "$BinaryPath.version"

function Write-Log {
  param([string]$Message)
  Write-Host "[$(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')] $Message"
}

# ---------------------------------------------------------------------------
# Install binary (skip if already on the correct release tag)
# ---------------------------------------------------------------------------
$InstalledTag = if (Test-Path $VersionFile) { (Get-Content $VersionFile -Raw).Trim() } else { "" }

if (-not (Test-Path $BinaryPath) -or $InstalledTag -ne $ReleaseTag) {
  Write-Log "Installing tfy-local-ai-setup $ReleaseTag (windows-amd64)..."
  if (-not (Test-Path $BinaryDir)) { New-Item -ItemType Directory -Path $BinaryDir | Out-Null }
  Invoke-WebRequest -Uri $BinaryUrl -OutFile $BinaryPath -UseBasicParsing
  Set-Content -Path $VersionFile -Value $ReleaseTag
  Write-Log "tfy-local-ai-setup installed at $BinaryPath."
} else {
  Write-Log "tfy-local-ai-setup $ReleaseTag already installed — skipping download."
}

# ---------------------------------------------------------------------------
# Run
# ---------------------------------------------------------------------------
$RunArgs = @("--url=$ControlPlaneUrl", "--tenant=$TenantName", "--gateway=$GatewayUrl")
if ($OpusModel)    { $RunArgs += "--opus-model=$OpusModel" }
if ($SonnetModel)  { $RunArgs += "--sonnet-model=$SonnetModel" }
if ($HaikuModel)   { $RunArgs += "--haiku-model=$HaikuModel" }
if ($SettingsFile) { $RunArgs += "--settings-file=$SettingsFile" }
& $BinaryPath @RunArgs
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
```

---

## Running manually

If you need to trigger setup outside the MDM schedule — for example, to complete a first-time login or re-authenticate after a token expiry — run the binary directly as root/Administrator with the same flags. A browser window will open for the developer to complete the device-flow login. After that, future MDM runs will be silent.

```bash
# macOS (Apple Silicon)
sudo /usr/local/bin/tfy-local-ai-setup \
  --url="https://app.example.truefoundry.com" \
  --tenant="myorg" \
  --gateway="https://gateway.example.truefoundry.com"

# Linux
sudo /usr/local/bin/tfy-local-ai-setup \
  --url="https://app.example.truefoundry.com" \
  --tenant="myorg" \
  --gateway="https://gateway.example.truefoundry.com"
```

```powershell
# Windows (PowerShell as Administrator)
& "C:\Program Files\TrueFoundry\tfy-local-ai-setup.exe" `
  --url="https://app.example.truefoundry.com" `
  --tenant="myorg" `
  --gateway="https://gateway.example.truefoundry.com"
```

### Dry run

Use `--dry-run` to inspect the JSON that would be written without touching the filesystem — useful for validating config before a fleet rollout:

```bash
sudo /usr/local/bin/tfy-local-ai-setup \
  --url="https://app.example.truefoundry.com" \
  --tenant="myorg" \
  --dry-run
```

---

## Using custom model IDs

The default model IDs (`claude-code/claude-opus` etc.) work for standard TrueFoundry deployments. If your setup uses different model IDs — for example a direct provider model, a Claude Enterprise account, or a virtual model — pass them via the model flags:

```bash
sudo /usr/local/bin/tfy-local-ai-setup \
  --url="https://app.example.truefoundry.com" \
  --tenant="myorg" \
  --gateway="https://gateway.example.truefoundry.com" \
  --opus-model="claude-enterprise/claude-opus-4-6" \
  --sonnet-model="claude-enterprise/claude-sonnet-4-6" \
  --haiku-model="claude-enterprise/claude-haiku-4-5"
```

Copy the exact model IDs from **Integrations → Providers** or **Virtual Models** in the TrueFoundry control plane.

---

## Using a custom settings template

If you need to deploy settings beyond what the built-in default config provides (additional `deny` rules, `allowedMcpServers`, etc.), put your full desired config in a JSON file and pass it via `--settings-file`. The binary will read it as the base, then inject the token and model IDs:

```bash
sudo /usr/local/bin/tfy-local-ai-setup \
  --url="https://app.example.truefoundry.com" \
  --tenant="myorg" \
  --settings-file="/etc/tfy/base-settings.json"
```

The binary only writes `ANTHROPIC_CUSTOM_HEADERS`, `ANTHROPIC_DEFAULT_*_MODEL`, and (on a fresh file) `ANTHROPIC_BASE_URL`. All other keys in your template are passed through unchanged.

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

Requires Go 1.24+. No external dependencies — stdlib only.

```bash
# Clone the source (source lives in the tfy-claude-mdm POC)
git clone https://github.com/truefoundry/tfy-local-ai-setup.git

# Native build
go build -o tfy-local-ai-setup main.go

# Cross-compile all platforms
GOOS=darwin  GOARCH=arm64 go build -o bin/tfy-local-ai-setup-darwin-arm64       main.go
GOOS=darwin  GOARCH=amd64 go build -o bin/tfy-local-ai-setup-darwin-amd64       main.go
GOOS=linux   GOARCH=amd64 go build -o bin/tfy-local-ai-setup-linux-amd64        main.go
GOOS=windows GOARCH=amd64 go build -o bin/tfy-local-ai-setup-windows-amd64.exe  main.go
```
