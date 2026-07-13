#!/bin/sh
# unstable-version.sh — Determine the next unstable pre-release version string.
#
# Usage: unstable-version.sh
#
# Prints the version string to stdout in the format:
#   v<base>-unstable.<yyyymmdd>.<build>
#
# The base is the latest v0.*.* tag, or v0.1.0 if none exists.
# The build number is the count of existing tags for the same base+date
# incremented by one.

set -eu

# Latest stable tag (v0.*.*) determines the base minor/patch.
stable=$(git tag --sort=v:refname | grep -E '^v0\.[0-9]+\.[0-9]+$' | tail -1)

if [ -z "$stable" ]; then
  base="v0.1.0"
else
  base="$stable"
fi

# Today's date
today=$(date -u +%Y%m%d)

# Count existing unstable tags for this base and day
count=$(git tag | grep -cE "^${base}-unstable\\.${today}\\." || true)
build=$((count + 1))

version="${base}-unstable.${today}.${build}"
echo "$version"
