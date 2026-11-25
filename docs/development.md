# Development and releases

This repository publishes the release binaries, installers, and documentation for `tfy-local-ai-setup`. The source code is maintained separately and is not published here.

## Versions

Releases are tagged `vMAJOR.MINOR.PATCH` (for example `v3.0.1`). Each tag is a GitHub release with these assets:

| Asset | Platform |
|---|---|
| `tfy-local-ai-setup-darwin-arm64` | macOS, Apple Silicon |
| `tfy-local-ai-setup-darwin-amd64` | macOS, Intel |
| `tfy-local-ai-setup-linux-amd64` | Linux, x86-64 |
| `tfy-local-ai-setup-linux-arm64` | Linux, ARM64 |
| `tfy-local-ai-setup-windows-amd64.exe` | Windows, x86-64 (also runs on Windows on Arm) |
| `SHA256SUMS` | SHA-256 checksums of the binaries |
| `install.sh`, `install.ps1` | The installers |

The `latest` release is the newest non-prerelease tag.

## Verify a download

Download the binary and `SHA256SUMS` from the same release, then check the file:

```bash
grep ' tfy-local-ai-setup-darwin-arm64$' SHA256SUMS | shasum -a 256 -c -
```

On Linux, use `sha256sum -c -` in place of `shasum -a 256 -c -`. On Windows, compare `Get-FileHash` output with the matching line in `SHA256SUMS`.

## Pinning

Pin a tag in your MDM scripts and change it deliberately. The deploy scripts in the [MDM deployment guide](mdm-deployment.md) install only the tag set in `RELEASE_TAG`. Keep the three OS scripts on the same tag.

## Release

Maintainers publish a release by merging a change with **exactly one** release label:

| Label | Version bump | Example |
|---|---|---|
| `release-minor` | patch (`x.y.Z` to `x.y.Z+1`) | `v1.4.2` to `v1.4.3` |
| `release-major` | minor (`x.Y.z` to `x.Y+1.0`) | `v1.4.2` to `v1.5.0` |

A change with both labels, or with neither, publishes nothing. The release build produces the five binaries and `SHA256SUMS`, and attaches the two installers.

Installers are served from `releases/latest/download/`, so a change to an installer reaches users only with the next release.

## Before rolling out a new tag

- Read the release notes for changed defaults, flags and minimum client versions.
- Dry-run the new binary on one machine against your `mdm.toml`, and compare the output with the current tag.
- Confirm the fleet meets any new minimum client version. See the [MDM deployment guide](mdm-deployment.md#2-prerequisites).
- Roll out in rings. See [MDM deployment](mdm-deployment.md#8-roll-out-in-rings).

## Reporting problems

Include the tag, your OS, the output of `tfy-local-ai-setup status` and `doctor`, and the relevant lines of the `--log-file` output. Remove tokens and internal hostnames before sharing logs. The [troubleshooting guide](troubleshooting.md) lists common errors and their fixes.
