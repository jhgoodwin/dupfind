#!/bin/sh
# Commits the current phase of a phased implementation, but only if
# .devin/skills/grey-review/grey-approve.sh has recorded approval for the exact tree
# being committed. Usage: .devin/workflows/phase-commit.sh "commit message"
set -eu

cd "$(git rev-parse --show-toplevel)"

msg="${1:?usage: phase-commit.sh \"commit message\"}"

git add -A
tree=$(git write-tree)
marker=".grey-review/${tree}.ok"
fix_marker=".grey-review/${tree}.fix"

if [ -f "$fix_marker" ]; then
    echo "phase-commit: grey-review rejected tree ${tree}:" >&2
    cat "$fix_marker" >&2
    exit 1
fi

if [ ! -f "$marker" ]; then
    echo "phase-commit: no grey-review approval for tree ${tree}" >&2
    echo "phase-commit: run the grey-review skill, then .devin/skills/grey-review/grey-approve.sh" >&2
    exit 1
fi

echo "phase-commit: grey-review approved tree ${tree}:" >&2
cat "$marker" >&2

git commit -m "$msg"
rm -f "$marker"
