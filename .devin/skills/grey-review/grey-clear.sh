#!/bin/sh
# Clears any prior verdict markers for the current tree before a new
# grey-review pass starts, so a stale .ok/.fix from an earlier run
# cannot be confused with the outcome of this run.
set -eu

cd "$(git rev-parse --show-toplevel)"

tree=$(git write-tree)
rm -f ".grey-review/${tree}.ok" ".grey-review/${tree}.fix"
echo "grey-review: cleared prior verdicts for tree ${tree}"
