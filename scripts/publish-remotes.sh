#!/usr/bin/env bash
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

GITHUB_URL="${1:?Usage: $0 <github-url> <gitea-url> [branch]}"
GITEA_URL="${2:?Usage: $0 <github-url> <gitea-url> [branch]}"
BRANCH="${3:-main}"

if [ ! -d .git ]; then
  git init -b "$BRANCH"
  git add -A
  git commit -m "Initial commit: OneX blockchain production stack"
fi

add_remote() {
  local name="$1" url="$2"
  git remote remove "$name" 2>/dev/null || true
  git remote add "$name" "$url"
  echo "Remote $name -> $url"
}

add_remote github "$GITHUB_URL"
add_remote gitea "$GITEA_URL"

git push -u github "$BRANCH"
git push -u gitea "$BRANCH"

echo "Published branch '$BRANCH' to GitHub and Gitea."
