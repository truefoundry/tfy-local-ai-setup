#!/usr/bin/env bash
# Downloads a pinned tfy-local-ai-setup release binary, verifies it against the
# release's SHA256SUMS, and dumps the two catalogs the mdm-script-builder skill
# builds its questions from:
#   help.txt         the binary's own flag list (`-h`), the source of truth
#   config-keys.txt  the personal-mode settings keys (`config keys`)
#
# Usage: catalog.sh <tag> [asset]
#   asset defaults to tfy-local-ai-setup-darwin-arm64.
# Output goes to $MDM_CATALOG_DIR/<tag>/ (default ~/.cache/tfy-local-ai-setup-catalog).
# Requires gh (authenticated) and shasum or sha256sum.
set -euo pipefail

TAG="${1:?usage: catalog.sh <tag> [asset]}"
ASSET="${2:-tfy-local-ai-setup-darwin-arm64}"
CACHE="${MDM_CATALOG_DIR:-$HOME/.cache/tfy-local-ai-setup-catalog}"
DIR="${CACHE}/${TAG}"

if command -v shasum >/dev/null 2>&1; then SHA="shasum -a 256"; else SHA="sha256sum"; fi

mkdir -p "${DIR}"
gh release download "${TAG}" --repo truefoundry/tfy-local-ai-setup \
  --pattern "${ASSET}" --pattern SHA256SUMS --dir "${DIR}" --clobber
(cd "${DIR}" && grep " ${ASSET}\$" SHA256SUMS | ${SHA} -c -)
chmod +x "${DIR}/${ASSET}"

"${DIR}/${ASSET}" -h > "${DIR}/help.txt" 2>&1 || true
"${DIR}/${ASSET}" config keys > "${DIR}/config-keys.txt"

echo "tag=${TAG} asset=${ASSET}"
echo "flags=$(grep -c '^  -' "${DIR}/help.txt") config_keys=$(wc -l < "${DIR}/config-keys.txt" | tr -d ' ')"
echo "catalog written to ${DIR}"
