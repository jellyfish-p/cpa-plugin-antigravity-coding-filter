#!/usr/bin/env bash
set -euo pipefail

script_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)
work_dir=$(mktemp -d)
trap 'rm -rf -- "$work_dir"' EXIT

export GITHUB_REF_NAME=v1.2.3
export GITHUB_REPOSITORY=example/plugin

gh() {
  case "$1 $2" in
    'release view')
      [[ "$PUBLISH_TEST_ACTION" == upload ]]
      ;;
    'release create')
      [[ "$PUBLISH_TEST_ACTION" == create &&
        "$*" == "release create v1.2.3 dist/plugin.zip dist/checksums.txt --repo example/plugin --verify-tag --title v1.2.3 --notes-file -" ]] || return 1
      cat
      ;;
    'release upload')
      [[ "$PUBLISH_TEST_ACTION" == upload && "$*" == "release upload v1.2.3 dist/plugin.zip dist/checksums.txt --clobber --repo example/plugin" ]]
      ;;
    *) return 1 ;;
  esac
}
export -f gh

cd "$work_dir"
mkdir dist
touch dist/plugin.zip
printf 'checksum  plugin.zip\n' > dist/plugin.zip.sha256

cat > expected-notes.md <<'EOF'
Automated release for v1.2.3.

## Downloads

- Download the archive for your platform from the release assets.
- Verify the download against `checksums.txt`.
EOF

PUBLISH_TEST_ACTION=create bash "$script_dir/publish-release.sh" > actual-notes.md
diff -u expected-notes.md actual-notes.md
diff -u dist/plugin.zip.sha256 dist/checksums.txt
PUBLISH_TEST_ACTION=upload bash "$script_dir/publish-release.sh"
