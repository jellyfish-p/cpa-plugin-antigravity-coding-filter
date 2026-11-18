#!/usr/bin/env bash
set -euo pipefail

sort dist/*.sha256 > dist/checksums.txt
if gh release view "${GITHUB_REF_NAME}" --repo "${GITHUB_REPOSITORY}" >/dev/null 2>&1; then
  gh release upload "${GITHUB_REF_NAME}" dist/*.zip dist/checksums.txt --clobber --repo "${GITHUB_REPOSITORY}"
else
  gh release create "${GITHUB_REF_NAME}" dist/*.zip dist/checksums.txt \
    --repo "${GITHUB_REPOSITORY}" \
    --verify-tag \
    --title "${GITHUB_REF_NAME}" \
    --notes-file - <<EOF
Automated release for ${GITHUB_REF_NAME}.

## Downloads

- Download the archive for your platform from the release assets.
- Verify the download against \`checksums.txt\`.
EOF
fi
