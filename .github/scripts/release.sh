#!/usr/bin/env bash
set -euo pipefail
IFS=$'\n\t'

SCRIPT_NAME="$(basename "$0")"
# Default to dry-run outside CI
if [[ "${CI:-}" == "true" ]]; then
  DRY_RUN=false
else
  DRY_RUN=true
fi

usage() {
  cat <<EOF
Usage: $SCRIPT_NAME [--dry-run]

Determine the next semver tag from conventional commits and create a GitHub release.

Options:
  --dry-run   Show what would happen without creating tags or releases
  -h, --help  Show this help message
EOF
}

log() { local IFS=' '; printf '%s\n' "$*"; }
warn() { printf '%s\n' "$*" >&2; }
die() { warn "error: $*"; exit 1; }

run() {
  if [[ "$DRY_RUN" == true ]]; then
    local IFS=' '
    log "[DRY-RUN] $*"
  else
    "$@"
  fi
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --dry-run) DRY_RUN=true; shift ;;
    -h|--help) usage; exit 0 ;;
    *) die "unknown option: $1" ;;
  esac
done

# Ensure we're in a git repo with full history
git rev-parse --is-inside-work-tree > /dev/null 2>&1 || die "not a git repository"

LAST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "")

if [[ -z "$LAST_TAG" ]]; then
  NEXT="v0.1.0"
  log "No existing tags, starting at $NEXT"
  run git tag -a "$NEXT" -m "Release $NEXT"
  run git push origin "$NEXT"
  run gh release create "$NEXT" --notes "Initial release" --title "$NEXT"
  exit 0
fi

log "Last tag: $LAST_TAG"

COMMIT_COUNT=$(git rev-list "${LAST_TAG}..HEAD" --count)
if [[ "$COMMIT_COUNT" -eq 0 ]]; then
  log "No new commits since $LAST_TAG, nothing to release"
  exit 0
fi

log "Commits since $LAST_TAG: $COMMIT_COUNT"

# Determine bump from conventional commits since last tag
BUMP="patch"
while IFS= read -r msg; do
  log "  $msg"
  if [[ "$msg" =~ ^[a-z]+(\(.+\))?!: ]] || [[ "$msg" =~ BREAKING[[:space:]]CHANGE ]]; then
    BUMP="major"
    break
  elif [[ "$msg" =~ ^feat(\(.+\))?: ]]; then
    BUMP="minor"
  fi
done < <(git log "${LAST_TAG}..HEAD" --pretty=format:"%s")

# Calculate next version
VERSION="${LAST_TAG#v}"
IFS='.' read -r MAJOR MINOR PATCH <<< "$VERSION"

case "$BUMP" in
  major) MAJOR=$((MAJOR + 1)); MINOR=0; PATCH=0 ;;
  minor) MINOR=$((MINOR + 1)); PATCH=0 ;;
  patch) PATCH=$((PATCH + 1)) ;;
esac

NEXT="v${MAJOR}.${MINOR}.${PATCH}"
log "Bump: $BUMP $LAST_TAG -> $NEXT"

run git tag -a "$NEXT" -m "Release $NEXT"
run git push origin "$NEXT"
run gh release create "$NEXT" --generate-notes --title "$NEXT"
