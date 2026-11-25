# MDM deployment guide

This guide takes a managed fleet from zero to working Claude Code, Codex, Claude Desktop, and the other supported agents. It covers choosing an auth mode, writing `mdm.toml`, deploying the per-OS script through your MDM, scheduling it, rolling it out in rings, verifying it, upgrading, and rolling back.

For every flag, see the [CLI reference](cli-reference.md). For every key in `mdm.toml`, see the [MDM config reference](mdm-config-reference.md). For where files land, see [file locations](file-locations.md). For errors, see [troubleshooting](troubleshooting.md).

## Contents

1. [How a deployment works](#1-how-a-deployment-works)
2. [Prerequisites](#2-prerequisites)
3. [Choose an auth mode](#3-choose-an-auth-mode)
4. [Write mdm.toml](#4-write-mdmtoml)
5. [Deploy the script](#5-deploy-the-script)
6. [Dry-run and first sign-in](#6-dry-run-and-first-sign-in)
7. [Schedule the job](#7-schedule-the-job)
8. [Roll out in rings](#8-roll-out-in-rings)
9. [Verify a machine](#9-verify-a-machine)
10. [Upgrade and roll back](#10-upgrade-and-roll-back)
11. [Operational notes](#11-operational-notes)

---

## 1. How a deployment works

Each deployment has two parts:

1. **The script** (one per OS, in the [deployment scripts](#5-deploy-the-script) section). It runs as root or SYSTEM and does three things:
   - Installs the pinned release of `tfy-local-ai-setup` to an administrator-owned path, if that version isn't already there. It verifies the binary against the release's `SHA256SUMS` when the release publishes one.
   - Writes your `mdm.toml` to a root-owned path.
   - Runs `tfy-local-ai-setup --config=<mdm.toml> --log-file=<log>`.
2. **The binary run**. It reads `mdm.toml`, works out which clients to configure, signs each logged-in user in if needed, writes each client's managed config, and locks it.

The script is meant to run **hourly** through your MDM. Each run is idempotent. It silently refreshes the user's token, rewrites anything that drifted, and does nothing when nothing changed. A run on a machine with no configured clients is a clean no-op, so it is safe to send to every machine.

```
MDM (hourly, root/SYSTEM)
  └─ deploy script
       ├─ install pinned tfy-local-ai-setup  (checksum-verified)
       ├─ write mdm.toml                     (root-owned, 0644)
       └─ tfy-local-ai-setup --config=mdm.toml
            ├─ detect/select clients (claude-code, codex, claude-desktop, …)
            ├─ user auth: device login (cached refresh) or OIDC helper
            ├─ write managed config per client
            └─ lock file (chflags schg / chattr +i / icacls)
```

## 2. Prerequisites

| Requirement | Why |
|---|---|
| Root (macOS/Linux) or SYSTEM/Administrator (Windows) on every target machine | Managed paths are root-owned. Scripts must run with those rights. |
| A TrueFoundry control plane URL and tenant name | `url` and `tenant` in `mdm.toml`. |
| A gateway URL | `gateway` in `mdm.toml`. Defaults to `url` if omitted. |
| The gateway model IDs your tenant offers | Set `opus_model`, `sonnet_model`, `haiku_model`, or the Codex `model`. The defaults (`claude-code/claude-opus` and friends) only work on standard deployments. |
| Claude Desktop **2.7032.0 or later** on every managed Mac or PC | Needed for Desktop OIDC (`auth = "oidc"` under `[agents.claude-desktop]`). |
| Claude Desktop **1.2581.0 or later** | Needed for Desktop `managedMcpServers`. |
| Claude Code **2.1.295 or later** | Needed for prompt-blocking hooks (`onFailure = "block"`). |
| Claude Code **2.1.259 or later** | Needed for `managedMcpServers` in Claude Code. Older builds ignore the key. |
| For OIDC: a public client on your IdP with the loopback redirect URIs `http://127.0.0.1:8585/callback`, `:8586` and `:8587` | Claude Code and Codex OIDC helpers use these. Desktop uses `http://127.0.0.1/callback`. Register all of them on the same public client. |
| An outbound route to `github.com` on each machine | The script downloads the release binary and `SHA256SUMS`. Alternatively, mirror the release assets internally and point the script at your mirror. |

> Versions are minimums for the feature named. Check the [CLI reference](cli-reference.md) and the README's per-client sections for the exact gating.

## 3. Choose an auth mode

Each client has its own `auth` setting. The two modes are:

| Mode | What the user gets | What is written to the managed config | When to use it |
|---|---|---|---|
| `api-key` (default) | A device login, once. The binary keeps a refresh token in the user's `~/.tf/` and refreshes silently. | Claude Code: `ANTHROPIC_CUSTOM_HEADERS` with `X-TFY-API-KEY`. Codex: `http_headers`. Desktop: `inferenceGatewayApiKey` (with the header). | You don't have an OIDC client yet, or you want the simplest path. |
| `oidc` | The client runs its own sign-in (authorization code + PKCE) and refreshes through a helper. No token is stored in the managed file. | Claude Code: `apiKeyHelper`. Codex: `[model_providers.truefoundry.auth]` with `command` and `args`. Desktop: `inferenceCredentialKind = "external-idp"` and `inferenceIdpOidc`. | You want no bearer material in managed files, and you can register an OIDC public client. |

Mix them per client if you need to. For example, Claude Code and Desktop on `oidc` with Codex on `api-key`. Additional agents (OpenCode, Qwen Code and others) share one auth mode through their shell helper, so their per-agent `auth` values must agree.

For a full-OIDC rollout the binary needs `issuer` and `client_id` under `[auth.oidc]`. The client ID is public and is the same for every machine in the tenant.

## 4. Write mdm.toml

Start from one of these, then edit:

- [`docs/examples/mdm-reference.toml`](examples/mdm-reference.toml): every setting, with advanced options commented out.

A minimal, working file for a fleet that uses device login for Claude Code and Codex:

```toml
url = "https://app.example.com"
tenant = "acme"
gateway = "https://gateway.truefoundry.ai"
tools = ["claude-code", "codex"]   # explicit: no auto-detect

[auth]
mode = "api-key"

[agents.claude-code]
opus_model = "claude-code/claude-opus"
sonnet_model = "claude-code/claude-sonnet"
haiku_model = "claude-code/claude-haiku"

[agents.codex]
model = "gpt-5.6-sol"

[execution]
write_config_on_login_failure = true   # never leave a machine without policy
```

Rules to keep in mind:

- **Set `tools` explicitly** for a fleet. Without it the binary auto-detects installed clients, which is fine for a pilot and surprising at scale. `tools = []` disables auto-detection.
- **Keep the file free of secrets.** Don't put tokens, PATs or client secrets in it. The OIDC client ID is public and belongs in the file.
- **Keep the file root-owned and readable by all** (the scripts write it `0644`). It is the policy, and a user who can edit it can change what gets configured.
- **Validate before you deploy.** Run the dry run in [step 6](#6-dry-run-and-first-sign-in). The file is checked against the MDM schema before any authentication or write happens, so unknown keys fail early.
- Key-by-key details are in the [MDM config reference](mdm-config-reference.md).

## 5. Deploy the script

Paste your `mdm.toml` into the **Configuration** block at the top of the script, set `RELEASE_TAG`, and deploy through your MDM. Each script:

- Refuses to run as a non-root user.
- Installs the release only when `RELEASE_TAG` differs from the installed version, verifying the checksum when `SHA256SUMS` exists.
- Writes the config to its root-owned path.
- Runs the binary and exits with the binary's exit code (see [exit codes](cli-reference.md#exit-codes)).

Keep `RELEASE_TAG` identical across the whole fleet. When you change it, you upgrade everything on the next run. Pin the same tag in all three OS scripts. They have drifted before.

Preview a deployment without writing managed files by setting `DRY_RUN=true` (macOS, Linux) or `$DryRun = $true` (Windows). The run passes `--dry-run`, which prints the config instead of writing files.

### macOS (Jamf, Mosyle, Kandji)

Config path: `/Library/Application Support/TrueFoundry/mdm.toml`. Log: `/var/log/tfy-mdm-setup.log`. Run as root.

```bash
#!/bin/bash
# Installs tfy-local-ai-setup, writes mdm.toml, and applies it.
# Run on a schedule as root. Preview without applying: sudo DRY_RUN=true ./deploy-mdm.sh
set -euo pipefail

# ---- Configuration: edit this section ----
RELEASE_TAG="v3.0.1"
MDM_TOML=$(cat <<'TOML'
# Paste your mdm.toml here, between the two TOML lines.
TOML
)
# ---- End of configuration ----

BINARY="/usr/local/bin/tfy-local-ai-setup"
CONFIG="/Library/Application Support/TrueFoundry/mdm.toml"
LOG_FILE="/var/log/tfy-mdm-setup.log"
DRY_RUN="${DRY_RUN:-false}"

[[ "$(id -u)" -eq 0 ]] || { echo "Run as root" >&2; exit 1; }
case "$(uname -m)" in
  arm64) ASSET="tfy-local-ai-setup-darwin-arm64" ;;
  x86_64) ASSET="tfy-local-ai-setup-darwin-amd64" ;;
  *) echo "Unsupported CPU: $(uname -m)" >&2; exit 1 ;;
esac
WORK_DIR="$(mktemp -d)"
trap 'rm -rf "${WORK_DIR}"' EXIT

# 1. Install the pinned release if it is missing or a different version.
if [[ ! -x "${BINARY}" || "$(cat "${BINARY}.version" 2>/dev/null)" != "${RELEASE_TAG}" ]]; then
  BASE="https://github.com/truefoundry/tfy-local-ai-setup/releases/download/${RELEASE_TAG}"
  curl -fLsS --retry 3 -o "${WORK_DIR}/${ASSET}" "${BASE}/${ASSET}"
  if curl -fLsS --retry 3 -o "${WORK_DIR}/SHA256SUMS" "${BASE}/SHA256SUMS"; then
    (cd "${WORK_DIR}" && grep "  ${ASSET}\$" SHA256SUMS | shasum -a 256 -c -)
  else
    echo "WARNING: ${RELEASE_TAG} has no SHA256SUMS; skipping checksum verification" >&2
  fi
  mkdir -p "$(dirname "${BINARY}")"
  install -o root -g wheel -m 755 "${WORK_DIR}/${ASSET}" "${BINARY}.new"
  mv -f "${BINARY}.new" "${BINARY}"
  echo "${RELEASE_TAG}" > "${BINARY}.version"
fi

# 2. Write the configuration: root-owned, readable by everyone.
RUN_ARGS=("--log-file=${LOG_FILE}")
if [[ "${DRY_RUN}" == "true" ]]; then
  CONFIG="${WORK_DIR}/mdm.toml"
  RUN_ARGS+=(--dry-run)
fi
mkdir -p "$(dirname "${CONFIG}")"
printf '%s\n' "${MDM_TOML}" > "${WORK_DIR}/mdm.toml.new"
install -o root -g wheel -m 644 "${WORK_DIR}/mdm.toml.new" "${CONFIG}"

# 3. Apply it. The script exits with the binary's exit code.
"${BINARY}" --config="${CONFIG}" "${RUN_ARGS[@]}"
```

### Linux (Ansible, Chef, Puppet, cron)

Linux has no console session to infer the employee from, so the job must name the account with `TFY_MDM_USER`. Config path: `/etc/truefoundry/mdm.toml`. Log: `/var/log/tfy-mdm-setup.log`.

Linux runs the CLI only. Claude Desktop managed preferences are macOS and Windows only, and Claude Desktop 3P has no Linux build.

```bash
#!/bin/bash
# Installs tfy-local-ai-setup, writes mdm.toml, and applies it.
# Run on a schedule as root with TFY_MDM_USER set to the employee account.
# Preview without applying: sudo TFY_MDM_USER=alice DRY_RUN=true ./deploy-mdm.sh
set -euo pipefail

# ---- Configuration: edit this section ----
RELEASE_TAG="v3.0.1"
MDM_TOML=$(cat <<'TOML'
# Paste your mdm.toml here, between the two TOML lines.
TOML
)
# ---- End of configuration ----

BINARY="/usr/local/bin/tfy-local-ai-setup"
CONFIG="/etc/truefoundry/mdm.toml"
LOG_FILE="/var/log/tfy-mdm-setup.log"
DRY_RUN="${DRY_RUN:-false}"

[[ "$(id -u)" -eq 0 ]] || { echo "Run as root" >&2; exit 1; }
TARGET_USER="${TFY_MDM_USER:?Set TFY_MDM_USER to the employee account}"
[[ "$(id -u "${TARGET_USER}")" -ne 0 ]] || { echo "Select a non-root employee" >&2; exit 1; }
export SUDO_USER="${TARGET_USER}"
case "$(uname -m)" in
  x86_64) ASSET="tfy-local-ai-setup-linux-amd64" ;;
  aarch64|arm64) ASSET="tfy-local-ai-setup-linux-arm64" ;;
  *) echo "Unsupported CPU: $(uname -m)" >&2; exit 1 ;;
esac
WORK_DIR="$(mktemp -d)"
trap 'rm -rf "${WORK_DIR}"' EXIT

# 1. Install the pinned release if it is missing or a different version.
if [[ ! -x "${BINARY}" || "$(cat "${BINARY}.version" 2>/dev/null)" != "${RELEASE_TAG}" ]]; then
  BASE="https://github.com/truefoundry/tfy-local-ai-setup/releases/download/${RELEASE_TAG}"
  curl -fLsS --retry 3 -o "${WORK_DIR}/${ASSET}" "${BASE}/${ASSET}"
  if curl -fLsS --retry 3 -o "${WORK_DIR}/SHA256SUMS" "${BASE}/SHA256SUMS"; then
    (cd "${WORK_DIR}" && grep "  ${ASSET}\$" SHA256SUMS | sha256sum -c -)
  else
    echo "WARNING: ${RELEASE_TAG} has no SHA256SUMS; skipping checksum verification" >&2
  fi
  mkdir -p "$(dirname "${BINARY}")"
  install -o root -g root -m 755 "${WORK_DIR}/${ASSET}" "${BINARY}.new"
  mv -f "${BINARY}.new" "${BINARY}"
  echo "${RELEASE_TAG}" > "${BINARY}.version"
fi

# 2. Write the configuration: root-owned, readable by everyone.
RUN_ARGS=("--log-file=${LOG_FILE}")
if [[ "${DRY_RUN}" == "true" ]]; then
  CONFIG="${WORK_DIR}/mdm.toml"
  RUN_ARGS+=(--dry-run)
fi
mkdir -p "$(dirname "${CONFIG}")"
printf '%s\n' "${MDM_TOML}" > "${WORK_DIR}/mdm.toml.new"
install -o root -g root -m 644 "${WORK_DIR}/mdm.toml.new" "${CONFIG}"

# 3. Apply it. The script exits with the binary's exit code.
"${BINARY}" --config="${CONFIG}" "${RUN_ARGS[@]}"
```

### Windows (Intune, SCCM, Group Policy)

Run in 64-bit PowerShell as SYSTEM or Administrator. The x86-64 binary also runs on Windows on Arm. Config path: `C:\Program Files\TrueFoundry\mdm.toml`. Log: `C:\Program Files\TrueFoundry\mdm-setup.log`.

```powershell
# Installs tfy-local-ai-setup, writes mdm.toml, and applies it.
# Run on a schedule as SYSTEM or Administrator. To preview without applying, set $DryRun = $true.
$ErrorActionPreference = 'Stop'

# ---- Configuration: edit this section ----
$ReleaseTag = 'v3.0.1'
# Paste your mdm.toml between @' and '@. Keep '@ at the start of its line.
$MdmToml = @'
# Paste your mdm.toml here.
'@
# ---- End of configuration ----

$DryRun = $false
if (-not [Environment]::Is64BitProcess) { throw 'Use 64-bit PowerShell' }
$Asset = 'tfy-local-ai-setup-windows-amd64.exe'
$SetupDir = Join-Path $env:ProgramFiles 'TrueFoundry'
$Binary = Join-Path $SetupDir 'tfy-local-ai-setup.exe'
$WorkDir = Join-Path ([IO.Path]::GetTempPath()) ("tfy-mdm-" + [guid]::NewGuid())
New-Item -ItemType Directory -Force -Path $SetupDir, $WorkDir | Out-Null

try {
  # 1. Install the pinned release if it is missing or a different version.
  $installed = if (Test-Path "$Binary.version") { (Get-Content "$Binary.version" -Raw).Trim() } else { '' }
  if (-not (Test-Path $Binary) -or $installed -ne $ReleaseTag) {
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
    $base = "https://github.com/truefoundry/tfy-local-ai-setup/releases/download/$ReleaseTag"
    Invoke-WebRequest -UseBasicParsing "$base/$Asset" -OutFile "$WorkDir\$Asset"
    try { Invoke-WebRequest -UseBasicParsing "$base/SHA256SUMS" -OutFile "$WorkDir\SHA256SUMS" } catch { }
    if (Test-Path "$WorkDir\SHA256SUMS") {
      $line = Select-String -Path "$WorkDir\SHA256SUMS" -Pattern "  $([regex]::Escape($Asset))$"
      $actual = (Get-FileHash "$WorkDir\$Asset" -Algorithm SHA256).Hash
      if (-not $line -or $actual -ne $line.Line.Split(' ')[0]) { throw "Checksum mismatch for $Asset" }
    } else {
      Write-Warning "$ReleaseTag has no SHA256SUMS; skipping checksum verification"
    }
    # A running binary can be renamed but not overwritten.
    Remove-Item "$Binary.old" -Force -ErrorAction SilentlyContinue
    if (Test-Path $Binary) { Move-Item $Binary "$Binary.old" -Force }
    Move-Item "$WorkDir\$Asset" $Binary
    Set-Content -Path "$Binary.version" -Value $ReleaseTag
  }

  # 2. Write the configuration. Program Files is writable only by administrators.
  $config = if ($DryRun) { "$WorkDir\mdm.toml" } else { "$SetupDir\mdm.toml" }
  [IO.File]::WriteAllText($config, $MdmToml)
  $runArgs = @("--config=$config", "--log-file=$SetupDir\mdm-setup.log")
  if ($DryRun) { $runArgs += '--dry-run' }

  # 3. Apply it. The script exits with the binary's exit code.
  & $Binary @runArgs
  $code = $LASTEXITCODE
} finally {
  Remove-Item -Recurse -Force $WorkDir -ErrorAction SilentlyContinue
}
exit $code
```

> The scripts verify the binary against the release's `SHA256SUMS` when the release has one, and warn and continue when it doesn't. Confirm the pinned release lists `SHA256SUMS` in its assets before you roll it out to a fleet.

## 6. Dry-run and first sign-in

**Dry run first, on one machine.** Set `DRY_RUN=true` in the script (or run the binary directly with `--dry-run`). The output shows the resulting config, with secrets redacted, and nothing is written to managed locations.

```bash
sudo /usr/local/bin/tfy-local-ai-setup \
  --config="/Library/Application Support/TrueFoundry/mdm.toml" \
  --dry-run
```

**First sign-in is interactive.** With `auth.mode = "api-key"` the first run opens a browser device login for the logged-in user. After that, scheduled runs are silent until the refresh token expires. To trigger the first sign-in on demand, run the binary by hand with the same `mdm.toml` (see [Running manually](../README.md#running-manually)). A browser opens for the developer.

If the user isn't signed in when a scheduled run happens, the run can still write the managed policy when `write_config_on_login_failure = true` (see the [config reference](mdm-config-reference.md)). Without it, a failed login writes nothing, so the machine has no policy at all until the user signs in.

For `auth.login = "manual"` the scheduled run never opens a browser. The user runs `tfy-local-ai-setup login --mdm` without `sudo` once, and the MDM run then reuses the stored PAT. See [Manual PAT onboarding](../README.md#manual-pat-onboarding-for-scheduled-mdm-runs).

For `auth.mode = "oidc"` the client's own helper opens the browser the first time that client needs a token. Claude Desktop shows its organization sign-in on first launch.

## 7. Schedule the job

Schedule the deploy script hourly as root or SYSTEM, using your MDM's script-scheduling feature (Jamf policy with a custom trigger, Intune platform script with a repeat interval, Ansible or cron on Linux).

Guidance:

- **Hourly** is the intended cadence. It keeps refresh tokens and policy in step and picks up `mdm.toml` changes within an hour.
- **Don't run it more often** than your MDM's minimum. Each run may refresh tokens against the control plane.
- **Run it at login or boot as well**, if your MDM supports it. A laptop that was asleep through the hourly slot then catches up at once.
- **Keep the three OS scripts on the same `RELEASE_TAG`.** Compare the values as part of any change review.
- Exit code `0` is success, including the "nothing to configure" no-op. Exit code `1` is an error. Exit code `3` means every agent was configured but some Agent Skills could not be installed (not in the Skills Registry, or a pinned version that doesn't exist). Alert on `1` and `3` (see [exit codes](cli-reference.md#exit-codes)).

## 8. Roll out in rings

| Ring | Scope | Length | Gate to next ring |
|---|---|---|---|
| 0 | Lab or your own Macs, Linux VMs and a Windows VM | 1–2 days | Dry run is clean. Config written. Client launches. |
| 1 | 5–10 volunteer machines across OSes | 3–5 days | Logs have no `1` or `3` exit codes after the first sign-in. Claude Code and Codex answer through the gateway. |
| 2 | 10–20% of the fleet | 1 week | Failure rate under your threshold. Support tickets understood. |
| 3 | Everyone | — | — |

At each ring check:

- Exit codes from your MDM job history.
- The log file on a sample of machines (see [file locations](file-locations.md#logs)).
- A real prompt in Claude Code and Codex, and a chat in Claude Desktop.
- `tfy-local-ai-setup status` and `doctor` on a few machines (see [troubleshooting](troubleshooting.md#diagnostic-commands)).

Changing `auth` on a client after rollout moves users from one mode to the other. Test that in ring 1 first.

## 9. Verify a machine

Run these checks on a machine that has had at least one scheduled run.

**Managed file exists and is locked**

| OS | Check |
|---|---|
| macOS | `ls -lO "/Library/Application Support/ClaudeCode/managed-settings.json"` — shows `schg`. |
| Linux | `lsattr /etc/claude-code/managed-settings.json` — shows `i`. |
| Windows | `icacls "C:\Program Files\ClaudeCode\managed-settings.json"` — ACL shows SYSTEM and Administrators with full control and Users read-only. |

**Installed version**

```bash
cat /usr/local/bin/tfy-local-ai-setup.version    # macOS; Linux uses the same path
```

On Windows, read `C:\Program Files\TrueFoundry\tfy-local-ai-setup.exe.version`. The script writes this file after each install, so it shows the tag the fleet was last pinned to.

**Last run log**

```bash
tail -n 50 /var/log/tfy-mdm-setup.log
```

Look for `error:` lines. The [troubleshooting guide](troubleshooting.md) lists the common ones and their fixes.

**Client-level checks**

- **Claude Code:** start a session, run `/status` or send a prompt. The gateway should answer.
- **Codex:** `codex` should start and answer through the gateway. On Windows, check that `C:\ProgramData\OpenAI\Codex\requirements.toml` exists.
- **Claude Desktop:** quit and reopen the app. Open **Help → Troubleshooting → Copy Managed Configuration Report** and check the gateway URL and auth fields (secrets are redacted).

## 10. Upgrade and roll back

**Upgrade:** change `RELEASE_TAG` in all three scripts to the new release and redeploy. The next hourly run installs the new binary and rewrites the config. Users don't need to sign in again, because the refresh token lives under `~/.tf/` and survives upgrades.

Before you upgrade, read the release notes on the [binary repo's releases page](https://github.com/truefoundry/tfy-local-ai-setup/releases). Two notes matter most:

- **Major-behaviour changes** (for example, an auth default or a permission default). Test them in ring 1.
- **New minimum client versions.** Confirm the fleet meets them before rollout.

**Roll back:** set `RELEASE_TAG` to the previous tag and redeploy. Confirm the previous tag still publishes its `SHA256SUMS`. The binary rewrites managed files on each run, so the config follows the older binary's behaviour after the next run.

Keep an old tag's release available until the fleet has moved off it. Don't delete releases that machines still reference.

## 11. Operational notes

- **Managed files are locked.** A developer cannot edit them without root or Administrator rights. To change one by hand, use the unlock commands in [file locking](#file-locking-and-unlocking) below, then let the next run restore it.
- **Server-managed settings override local files.** In the Anthropic Admin Console, turn off any server-managed Claude Code or Claude Desktop settings that conflict with `mdm.toml`. They take precedence over file-based config.
- **Claude Desktop reads its managed preferences at launch.** A run that changes them takes effect after the user restarts Claude Desktop. The README's Desktop section documents the restart step.
- **Codex on Windows** reads `requirements.toml` and `config.toml` under `%ProgramData%\OpenAI\Codex`, and the binary locks that directory. See [security](security.md#managed-files).
- **Logs are plain text.** `--log-file` appends and is `0600`. Rotate it with your MDM's log tooling, or with `logrotate` on Linux.
- **Test on a clean VM** whenever you change the script's execution context. The MDM run must be root (or SYSTEM), with a minimal `PATH`. Reproduce it with:

  ```bash
  sudo env -i PATH="/usr/bin:/bin:/usr/sbin:/sbin" HOME=/var/root /bin/bash deploy-mdm.sh
  ```

  On Windows, use `psexec -s` or a scheduled task that runs as `SYSTEM`.

### File locking and unlocking

| OS | Lock | Manual unlock (only for a test or a deliberate fix) |
|---|---|---|
| macOS | `chflags schg` (system immutable) | `sudo chflags noschg <path>` |
| Linux | `chattr +i` (immutable) | `sudo chattr -i <path>` |
| Windows | `icacls` ACL: SYSTEM full control, Administrators and Users read-only (per the README's file-locking table) | Re-run the deploy script. It grants write access before updating the file. |

On a filesystem without immutable-attribute support (tmpfs, overlayfs, some NFS mounts), `chattr` fails. The binary logs a warning and still writes the file, unlocked. Use ext4 or XFS for managed hosts.
