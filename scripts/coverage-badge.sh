#!/bin/sh
# coverage-badge.sh — Run full test suite with coverage, emit $GITHUB_OUTPUT lines
# for pct and color, and write the badge JSON to the path given as the first argument.
#
# Usage: coverage-badge.sh <json-output-path>
#
# The json-output-path is written with a shields.io endpoint badge payload.

set -eu

json_out="${1:?usage: coverage-badge.sh <json-output-path>}"

go test -tags=slow -coverprofile=/tmp/coverage.out ./...
pct=$(go tool cover -func=/tmp/coverage.out | grep '^total:' | awk '{print $3}' | tr -d '%')

color="red"
if [ "$(echo "$pct >= 80" | bc -l)" -eq 1 ]; then
  color="brightgreen"
elif [ "$(echo "$pct >= 50" | bc -l)" -eq 1 ]; then
  color="yellowgreen"
elif [ "$(echo "$pct >= 30" | bc -l)" -eq 1 ]; then
  color="yellow"
fi

echo "pct=$pct"
echo "color=$color"

cat > "$json_out" <<EOF
{
  "schemaVersion": 1,
  "label": "coverage",
  "message": "${pct}%",
  "color": "${color}"
}
EOF
