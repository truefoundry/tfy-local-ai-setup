---
name: mdm-script-builder
description: Interview-driven builder for a TrueFoundry tfy-local-ai-setup MDM deployment. Compiles its questions from the pinned release binary's own flags and config keys (the source of truth), offers a POC path (six questions) or a complete path, then generates mdm.toml and per-OS deploy scripts. Offers advanced fields only after the simple setup, and asks them only if the user wants. Use when the user wants to create, scaffold, or update an MDM script, an mdm.toml, or a POC or complete rollout of tfy-local-ai-setup.
---

Build an MDM deployment for `tfy-local-ai-setup` by interviewing the user. Every question comes from the pinned binary's own flags and keys, so the output matches the release being deployed. Never invent a setting that the binary does not have.

## Source of truth

1. **The binary decides what can be asked.** Run `scripts/catalog.sh <tag>` (from this skill folder). It downloads the release binary, checks it against `SHA256SUMS`, and writes `help.txt` (`-h`) and `config-keys.txt` (`config keys`) under `$MDM_CATALOG_DIR/<tag>/`. Build every question from those two files, using the grouping in `references/question-groups.md`.
   - Use the binary for the tag being deployed. Don't use an older copy installed on a machine: its flags may differ.
2. **The docs explain meaning and defaults.** The repository's `README.md`, `docs/mdm-deployment.md` (scripts and rollout), `docs/mdm-config-reference.md` and `docs/cli-reference.md`. If a doc and the binary disagree, the binary wins. Report the mismatch to the user.
3. **`~/.truefoundry/credentials.json` is TrueFoundry's own login file.** Read only its `host` key, to suggest the control-plane URL as a default. Never print, copy or write `access_token` or `refresh_token`, and never copy any part of that file into an output.

## Step 0: Preflight (no questions)

1. Choose the release tag. Default to the latest: `gh release view --repo truefoundry/tfy-local-ai-setup --json tagName -q .tagName`. Ask the user to confirm or pin another tag.
2. Run `scripts/catalog.sh <tag>`. If it fails the checksum, stop and report.
3. Read `help.txt` and `config-keys.txt` from the catalog folder. Map each entry to a group in `references/question-groups.md`. Anything unmapped goes into group X, asked as a plain question.
4. Read the `host` value from `~/.truefoundry/credentials.json`, if it exists. Keep it only as the default for the control-plane URL.

## Step 1: Path

Ask with AskUserQuestion:

- **POC (recommended to start):** six questions, device login, no advanced features. Produces a working `mdm.toml` and scripts. An advanced offer follows.
- **Complete setup:** groups A to K, in order. Use this for a full rollout.

## Step 2: POC questions

Ask the six POC questions in `references/question-groups.md`, in one or two AskUserQuestion rounds. Use the binary's help text as each description, and use the defaults given there.

Then confirm a summary of the answers before generating anything.

## Step 2 (alternative): Complete questions

Ask groups A to K, one group per round, at most four questions per call. For each question:

- Use the flag's help text from `help.txt` as the description.
- Offer the default from `help.txt`, or from `config-keys.txt` where there is one.
- Ask only the questions that the chosen group contains. Skip a group the user declines.
- Apply the rules in `references/question-groups.md` (secrets, `force-login-org`, OIDC redirect URIs, Desktop version floors, `tools` explicit).

## Step 3: Generate

Write everything under `mdm-deploy/<slug>/` in the current working directory, where `<slug>` is a short lowercase name for the rollout:

| File | Contents |
|---|---|
| `mdm.toml` | The answers, in the MDM config format. Keep `tools` explicit. Keep secrets out. |
| `macos-deploy.sh` | The macOS script from `docs/mdm-deployment.md`, section "macOS". Paste `mdm.toml` into its configuration block and set `RELEASE_TAG`. |
| `linux-deploy.sh` | The Linux script, same method. Only if Linux was selected. |
| `windows-deploy.ps1` | The Windows script, same method. Only if Windows was selected. |
| `SUMMARY.md` | The answers, the release tag, the catalog counts (flags and keys) and the date. Start the file with `**Date**: YYYY-MM-DD`. |

Templates come from the docs. Do not write a script from memory. The script bodies must match the docs exactly, apart from the configuration block and the release tag.

Preview: with a dry run, keep `DRY_RUN=true` in the preview copy only. Never ship it in the production copy.

## Step 4: Validate

- `bash -n` every generated `.sh` file.
- For the `.ps1` file, run a PowerShell parse check if `pwsh` is installed. If it isn't, say that the Windows script is unverified.
- For `mdm.toml`, tell the user to run `--config <file> --dry-run` on one test machine. The binary checks the file's syntax and unknown keys before it signs in or writes anything. Do not run it on a developer machine as part of this skill.

## Step 5: Advanced offer (POC only)

After the POC files are written, ask one AskUserQuestion: whether to configure advanced fields now. Offer the groups the POC did not cover (B extras, C extra models, D MCP, E skills, F Desktop extras, G files and templates, H prompt blocking and Cursor, I other agents, J failure handling and logs, K WSL). Multi-select.

- If the user declines, stop. The POC output stands.
- If the user chooses groups, ask only those groups' questions, regenerate `mdm.toml` and the scripts, and validate again.

## Step 6: Hand-over

Report, in this order:

1. The files written, with paths.
2. The catalog version, and any binary settings that were not mapped.
3. Anything that needs a manual step: for example, registering the OIDC redirect URIs, or turning off server-managed settings in the Anthropic Admin Console.
4. The rollout plan. Point to `docs/mdm-deployment.md`, sections 7 to 9: hourly schedule, rings, and verification.

Do not push, deploy or change any MDM system. The user runs the scripts through their MDM.
