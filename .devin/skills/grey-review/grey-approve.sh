#!/bin/sh
# Records grey-beard review approval for the current staged tree.
# Run by the grey-review skill after a clean review. Consumed by
# .devin/workflows/phase-commit.sh to gate incremental phase commits.
# Usage: grey-approve.sh "<review comments>"
set -eu

comments="${1:?usage: grey-approve.sh \"<review comments>\"}"
min_len=50
if [ "${#comments}" -lt "$min_len" ]; then
    echo "grey-approve: review comments must be at least ${min_len} chars" >&2
    exit 1
fi

cd "$(git rev-parse --show-toplevel)"

tree=$(git write-tree)
mkdir -p .grey-review
printf '%s\n' "$comments" > ".grey-review/${tree}.ok"
rm -f ".grey-review/${tree}.fix"
echo "grey-review: approved tree ${tree}"
