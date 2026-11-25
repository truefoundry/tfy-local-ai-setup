# Troubleshooting

This guide covers diagnostics, exit codes, log locations, WSL, and an error catalogue. Each error entry gives the message you will see, its usual cause, and the fix.

Behaviour is described for v3.0.1. Where a behaviour couldn't be confirmed against the release, it is marked *unverified*.

## Contents

- [Diagnostic commands](#diagnostic-commands)
- [Doctor checks](#doctor-checks)
- [Exit codes](#exit-codes)
- [Where logs are](#where-logs-are)
- [Error catalogue](#error-catalogue)
- [WSL](#wsl)
- [Scheduling](#scheduling)

## Diagnostic commands

| Command | Runs as | Network | What it tells you |
|---|---|---|---|
| `tfy-local-ai-setup status` | the developer | no (offline) | What is configured: tenant, gateway, sign-in state and PAT expiry, each agent's state, skills, and "Next" fix lines. Exit `1` if anything needs attention. `--json` for machine output. |
| `tfy-local-ai-setup doctor` | the developer | yes | Health checks, each with a fix. Exit `1` if any check fails. |
| `tfy-local-ai-setup token --mdm` | the developer | — | Prints only the token, which is useful for checking that the credential helper works. Never opens a browser. |
| `sudo tfy-local-ai-setup --config=<file> --dry-run` | root | yes | Shows the config the managed run would write, without writing it. |
| `tfy-local-ai-setup config show` | the developer | no | Saved settings, with secrets redacted. |

Run `status` first. It works offline and names what to fix.

## Doctor checks

`doctor` runs these checks, in order. Each failure line includes a fix.

| Check | Fails when | Fix |
|---|---|---|
| `configured` | No URL or tenant is saved. | Run `tfy-local-ai-setup personal`. |
| `keychain` | The keychain round trip fails. The error is shown. | Unlock the login keychain, or check the error text. |
| `token` | No usable token exists. | Run `tfy-local-ai-setup login`. |
| `gateway` | No gateway is saved, or the model list can't be fetched. Shows the HTTP status. | Check the gateway URL and network. `401` means sign in again. |
| `model <name>` | A pinned model is missing on the gateway. | Correct the model ID, or ask an admin to add the model. |
| `mcp <name>` | The server is not found (`404`). Other errors are shown. `401` and `403` count as reachable. | Correct the server name, or ask an admin. |
| `skill <ref>` | The skill is missing, has been removed from the registry, or is pinned behind a newer version. | Correct the reference, or run `update`. |
| `<agent> installed` | The agent's CLI is not on `PATH`. | Install the agent, or set its tool off. |

## Exit codes

| Entry point | `0` | `1` | Other |
|---|---|---|---|
| Managed run (`sudo tfy-local-ai-setup --config=…`) | Success, or nothing to configure (clean no-op). | Error. Details to stderr and `--log-file`. | `3`: all agents configured, but some Agent Skills need an admin. Usually the policy names a skill that is not in the registry, or a pinned version that doesn't exist. Transient skill download failures don't cause `3`. They retry on the next run. |
| `personal` / `update` | Success. | Any step failed. | — |
| `login` | Success. | Not set up, or sign-in failed. | `2`: bad flags. |
| `login --mdm` | Success. | Any error, or a skills-only failure. | `2`: bad flags. |
| `status` | Set up and nothing needs attention. | Not set up, or needs attention. | — |
| `doctor` | All checks pass. | At least one check failed. | — |
| `token` | Token printed. | Not configured, or no sign-in. | `2`: bad flags. |
| `config` | Success. | Load or save failed. | `2`: bad usage. |
| `uninstall` | Success. | Declined ("Aborted."), or prompt blocking wasn't fully removed. | `2`: no terminal and no `--yes`. |
| `cursor-monitor` | Always `0`. The hook helper's allow or deny decision is in its JSON output, not its exit code. | — | — |

For the managed run, scripts pass the binary's exit code through. Alert on `1` and `3` in your MDM job history.

## Where logs are

| Log | Location | Enabled by |
|---|---|---|
| MDM run log | `/var/log/tfy-mdm-setup.log` (macOS, Linux); `C:\Program Files\TrueFoundry\mdm-setup.log` (Windows) in the MDM scripts | `--log-file` |
| Personal debug log | `~/.tf/debug.log` | `--debug` (wizard default when `--log-file` isn't given) |
| Personal run reports | `~/.tf/state/runs/run-<id>.json` | Written by the personal hourly check |
| Managed run reports | macOS `/Library/Application Support/TrueFoundry/state/<user>/runs/`; Windows `%ProgramData%\TrueFoundry\state\<user>\runs\`; Linux `/var/lib/truefoundry/state/<user>/runs/` | Written by each managed run. The newest 48 are kept. |
| Desktop app log | macOS `~/Library/Logs/<identifier>/app.log`; Windows `%LOCALAPPDATA%\<identifier>\logs\app.log`; Linux `~/.local/share/<identifier>/logs/app.log` | Written by the desktop app |

`--debug` logs each control-plane and gateway call (method, URL and status, never headers or tokens), sign-in and PAT decisions, planned changes, and each file written. `--log-file` appends (mode `0600`) and also prints to stderr.

**Run reports** record the run ID, mode and trigger, the result (`ok`, `changed`, `warn` or `error`), sign-in state and reason, each agent's status, the gateway's reachability, the PAT's expiry, missing models and MCP servers, files changed with their backups, and any actions the user needs to take. Free text is redacted before it is written. Read them in the desktop app, or with the hidden `machine runs list` operation.

The Claude Desktop **Help → Troubleshooting → Copy Managed Configuration Report** menu is described in the [README](../README.md#claude-desktop-cowork-on-3p). *Unverified*: the repo doesn't contain that string. Check it against the app version you deploy.

## Error catalogue

Messages are grouped by the area that raises them. Where the repo has the exact text, it is quoted.

### Sign-in

| Message | Cause | Fix |
|---|---|---|
| `sign-in required: run "<bin> login" in a terminal first` | No cached sign-in, and no console session to prompt in. | Run `login` in a terminal as the developer. |
| `Your TrueFoundry session has ended. Sign in again.` / `You're not signed in` | The session expired or was revoked. | Run `login`. |
| `sign-in timed out before it was approved` | The device code expired before the browser approval finished. | Run the command again. A new browser window opens straight away. |
| `login required: run login --mdm` (manual PAT mode) | No PAT has been created yet for this user. | Run `tfy-local-ai-setup login --mdm` without `sudo`, in a terminal. |
| `TrueFoundry: <err>: sign in again with "… login"` (from `token`) | The stored credential is unusable. | Run `login`. |
| Gateway `HTTP 401` or `403` | The session token has expired or was never created. | Run the binary by hand to sign in again. Future scheduled runs keep the token fresh. |

### Personal access tokens

| Message | Cause | Fix |
|---|---|---|
| `team required: run login --mdm --pat-team <team-name> in a terminal` | The tenant requires a team for PATs. | Pass `--pat-team`, or set `auth.pat_team` in `mdm.toml`. |
| `you're in several teams… IT can set auth.pat_team` | The user is in more than one team, and no team is set. | Set `auth.pat_team`. |
| `--yes` or `update` returns the team error | Non-interactive runs can't pick a team. | Pass `--pat-team`. |
| `no available teams; ask your administrator for team access` | The user has no team to create a PAT under. | Ask an admin for team access. |
| `a token named "…" already exists` | The PAT name is taken. | Use `--pat-name` with a new name. |
| `not allowed to create a token` | The user lacks permission. *Inferred from the code path; the admin should check the role.* | Ask an admin. |
| `TrueFoundry no longer accepts this key` | The key was revoked or has expired. | Run `personal` to create a new key. `status` shows "token expired". |
| `could not verify managed PAT; retry later` | The verification call failed, or returned `401`/`403`. | Retry. If it persists, check the key, then sign in again. |

### Gateway and models

| Message | Cause | Fix |
|---|---|---|
| `POST <url>: HTTP 404` or a model-not-found `400` | The model ID doesn't exist on the gateway. | Correct the model ID, or have an admin add the model. Copy the exact ID from the control plane. |
| `HTTP 502`, `503` or `504` | The gateway is unreachable or overloaded. | Retry. Check the proxy and VPN. |
| `Couldn't reach <host> (timed out after …)` | Network path to the control plane is blocked. | Check connectivity, proxy and VPN. |
| `/v1/models returned 401` (from `doctor`) | Authentication failed. | Sign in again. |

### Agent Skills

| Message | Cause | Fix |
|---|---|---|
| `<ref> is not in the Skills Registry (or you can't see it)` | The reference is misspelled, or the user lacks access. | Correct the reference, or have an admin fix the policy. |
| `took too long to respond` | Registry timeout. | Retry. The next scheduled run tries again. |
| `refused access (sign in again)` | `401` or `403` from the registry. | Run `login`. |
| `no longer has it` | `404`. The skill has been removed. | Remove it from the policy. |
| `agents configured, but some Agent Skills could not be installed` | Exit code `3`. The policy names a skill that isn't in the registry, or a pinned version that doesn't exist. | Fix the policy's skill references. |

### Files and locks

| Message | Cause | Fix |
|---|---|---|
| `WARNING: chattr not available — file not immutable-locked` (Linux) | The filesystem doesn't support the immutable attribute. | Use ext4 or XFS for managed hosts. The config is still written, but unlocked. |
| `chflags schg failed` (macOS) | Setting the system immutable flag failed. | Re-run as root. If it persists, check the file's flags with `ls -lO`, and the error text in the log. |
| `<file> is not valid JSON … nothing was changed` | A developer's file is malformed. The tool won't overwrite it. | Fix the file, or move it aside and re-run. |
| `<tool> config would leave invalid TOML; nothing was changed` | Patching a TOML file would produce invalid output. | Fix or move the file aside. |
| `ignoring <policy>: not owned by an administrator, or writable by other users` | The policy file has the wrong owner or permissions, so personal mode ignores it. | Set owner `root` (or SYSTEM/Administrators), and remove write for other users. |
| `refusing to write <path> as root — not one of its managed files` | A write was requested for a path the binary doesn't manage. | Check any `--settings-file`, `--codex-settings-file` or `--config` path values. |

### Claude Desktop

| Message | Cause | Fix |
|---|---|---|
| `Claude Desktop is running: quit it and run setup again` | Desktop reads its preferences at launch, and the run stops while it is open. | Quit Desktop, then re-run. |
| `Claude Desktop didn't quit` | The quit request was refused or timed out. | Quit the app manually, then re-run. |
| Users locked out of Claude Desktop sign-in | `--force-login-org` was set to the wrong UUID. | Correct the UUID in `mdm.toml` and re-run. On Windows only the first UUID applies. Check the value against the Claude Admin Console. |

### Prompt blocking and Cursor

| Message | Cause | Fix |
|---|---|---|
| `MDM sign-in required. For manual login, run tfy-local-ai-setup login --mdm…` | `cursor-monitoring` is set with `require_login`, and no MDM credential exists. | The developer runs `login --mdm`. |
| `Prompt blocking was not fully removed; fix the error above and run uninstall again` | Removing a hook failed. | Fix the error above, then re-run `uninstall`. |

### Personal mode

| Message | Cause | Fix |
|---|---|---|
| `Not set up yet: run "<bin> personal" first.` | No saved settings. | Run `personal`. |
| `No agents saved: set them with "config set tools claude-code,codex"` | `tools` is empty. | Run the suggested `config set`. |
| `Not removing anything without a terminal to confirm in: re-run with --yes` | `uninstall` needs confirmation. | Pass `--yes`. |
| `Claude Desktop … managed profile … skipped` | A managed profile already sets more than app-behaviour keys. | Use `--force` only if you accept overriding it. |

### MDM run

| Message | Cause | Fix |
|---|---|---|
| `--url and --tenant are required` | No connection settings, and no `--config`. | Pass them, or use `--config`. |
| `could not determine logged-in user` | The run has no console user (Linux). | Set `TFY_MDM_USER` in the job. |
| `managed config written without a verified token — the device login must still be completed` | `write_config_on_login_failure` wrote the placeholder. | The user signs in. Until then the gateway returns `401`. |
| `MCP refresh failed …; keeping existing managed configuration` | Discovery failed. | Nothing needs fixing. The next run retries. |

### Self-update

| Message | Cause | Fix |
|---|---|---|
| `does not match its SHA256SUMS entry; kept the current version` | Download corrupted or tampered. | The current binary is kept. Retry the update. |
| `is not listed in SHA256SUMS` | The release doesn't publish checksums for this asset. | Pin a release that does. |

## WSL

The `--wsl` flag (Windows only) pushes the managed config into the logged-in user's WSL distros.

- It copies Claude Code and Codex configs into each distro through `/mnt/c`. In OIDC mode, helper paths are rewritten to `/mnt/<drive>/…`.
- The Windows binary runs `wsl.exe` as the console user, because `wsl.exe` refuses `LocalSystem`.
- Running distros are updated on every run. Stopped distros are updated at most once every 24 hours. Booting a stopped distro can start its services.
- Docker Desktop, Rancher Desktop and podman-machine distros are skipped.
- Limits: a root shell inside a distro is available to any user who can run `wsl.exe`. The config is routed there, not enforced. There's no immutable lock inside a distro.
- Claude Desktop, VS Code and Cursor are never configured inside distros.

Errors from WSL runs:

| Message | Fix |
|---|---|
| `tfy-local-ai-setup isn't installed in <distro>; run WSL setup first` | Install the Linux CLI inside the distro, using the [Linux deploy script](mdm-deployment.md#linux-ansible-chef-puppet-cron). |
| `the tfy-local-ai-setup in <distro> is too old` | Upgrade the distro's copy to the pinned release. |
| `the setup inside <distro> didn't finish in time` | Retry. Check that the distro starts. |
| `no such WSL distro` | The distro name is wrong, or the distro was removed. |
| `<distro> runs on X, which has no Linux CLI build` | The distro's architecture has no published CLI. |

Personal mode has no `--wsl` flag. WSL setup there is handled by the desktop app (`wsl.list` and `wsl.provision`).

## Scheduling

The binary can install its own schedules. Scheduled runs are hourly or daily. The MDM job is the usual way to schedule the managed run, but these are what the binary installs:

| Platform | Personal check | Admin run | Run-now trigger |
|---|---|---|---|
| macOS | LaunchAgent `com.truefoundry.ai-setup.check`, in `~/Library/LaunchAgents/` (every 3600 s for hourly, 86400 s for daily) | LaunchDaemon `com.truefoundry.ai-setup`, in `/Library/LaunchDaemons/` | LaunchDaemon `com.truefoundry.ai-setup.trigger`, watching `/Library/Application Support/TrueFoundry/spool/run-now` |
| Linux | systemd user units `truefoundry-ai-setup-check.service` and `.timer`, in `~/.config/systemd/user/` | systemd units `truefoundry-ai-setup.service` and `.timer`, in `/etc/systemd/system/` | `truefoundry-ai-setup-run-now.path` and `.service`, watching `/var/lib/truefoundry/spool/run-now` |
| Windows | Task `\TrueFoundry AI Setup Check`, running as the user | Task `\TrueFoundry AI Setup`, running as SYSTEM | Task `\TrueFoundry AI Setup Run Now`. Interactive users may start it. |

Timers use a randomised delay of up to 5 minutes (systemd `RandomizedDelaySec`, and Task Scheduler's `RandomDelay`). The Windows tasks have a 30-minute execution limit. *Unverified*: which component calls the admin installer. No caller in the repo was found, so the MDM job remains the expected scheduler.

For diagnosing a missed run, check the run reports first. Then check the scheduler itself, with `launchctl print system/com.truefoundry.ai-setup` on macOS, `systemctl list-timers` on Linux, or Task Scheduler on Windows.
