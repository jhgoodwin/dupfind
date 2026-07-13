#!/bin/sh
# push-badges.sh — Push badge assets (coverage.json) to the master-badges branch.
#
# Usage: push-badges.sh <json-file-path>
#
# Creates or updates the master-badges orphan branch with the given JSON file,
# committed as coverage.json, and pushes it to origin.

set -eu

json_file="${1:?usage: push-badges.sh <json-file-path>}"

# Save original ref so we can restore it after mucking with branches.
if orig_branch=$(git symbolic-ref --short HEAD 2>/dev/null); then
  orig_ref="$orig_branch"
else
  orig_ref=$(git rev-parse HEAD)
fi

git config user.name "github-actions"
git config user.email "actions@github.com"

git fetch origin master-badges 2>/dev/null || true

if git show-ref --verify refs/remotes/origin/master-badges 2>/dev/null; then
  git checkout master-badges
else
  git checkout --orphan master-badges
  rm -f .git/index
  git clean -fdx
fi

cp "$json_file" coverage.json
git add coverage.json
git diff --cached --quiet || git commit -m "Update coverage badge"
git push origin master-badges

# Restore the original branch/commit the caller was on.
git checkout "$orig_ref"
