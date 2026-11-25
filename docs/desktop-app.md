# Desktop app

**TrueFoundry AI Setup** is a desktop app that sets up the same agents as the command-line tool, with a guided interface. It is a front end. It holds no tokens and writes no agent configuration itself. The command-line tool owns all state, so the app and the CLI always agree about what is configured.

## How it works

- The app drives the CLI through an internal interface. It discovers agents, signs in, shows a plan, applies changes, and lists run reports.
- The app bundles the CLI release that matches it, so a fresh install works with nothing else installed. Before its first apply, the app checks the bundled CLI against its own SHA-256 record.
- On Windows, the installer also includes the Linux CLI for the machine's architecture, so the app can set up WSL distros.

## Packaging

The installers below are the app's build targets. The current GitHub releases contain only the command-line binaries, so the app installers are not yet downloadable from this repository.

| Platform | Installer |
|---|---|
| macOS | Universal `.dmg`, for macOS 13 and later |
| Windows x64 | MSI and NSIS (per-user install) |
| Windows arm64 | NSIS |
| Linux x64 and arm64 | `.deb`, `.rpm` and AppImage. The deb and rpm depend on Ayatana AppIndicator for the tray icon. |

## Logs

The app writes its own log. Its location depends on the OS:

| OS | App log |
|---|---|
| macOS | `~/Library/Logs/<identifier>/app.log` |
| Windows | `%LOCALAPPDATA%\<identifier>\logs\app.log` |
| Linux | `~/.local/share/<identifier>/logs/app.log` |

The CLI's own logs and run reports are described in [troubleshooting](troubleshooting.md#where-logs-are).
