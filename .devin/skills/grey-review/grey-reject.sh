#!/bin/sh
# Records grey-beard review comments for the current tree when the
# review finds blocking issues. Run by the grey-review skill instead
# of grey-approve.sh when changes are not clean.
# Usage: grey-reject.sh "<blocking issue comments>"
set -eu

comments="${1:?usage: grey-reject.sh \"<blocking issue comments>\"}"
min_len=50
if [ "${#comments}" -lt "$min_len" ]; then
    echo "grey-reject: review comments must be at least ${min_len} chars" >&2
    exit 1
fi

cd "$(git rev-parse --show-toplevel)"

tree=$(git write-tree)
mkdir -p .grey-review
printf '%s\n' "$comments" > ".grey-review/${tree}.fix"
rm -f ".grey-review/${tree}.ok"
echo "grey-review: rejected tree ${tree}, see .grey-review/${tree}.fix"
