#!/bin/sh
# smoke-test.sh — Run dupfind smoke test, verify output contains expected strings.
set -eu

output=$(bin/dupfind -tests . 2>&1 || true)
echo "$output"

echo "$output" | grep -q 'Exact Duplicates' || {
  echo "Missing 'Exact Duplicates' in output" >&2
  exit 1
}
echo "$output" | grep -q 'Near Duplicates' || {
  echo "Missing 'Near Duplicates' in output" >&2
  exit 1
}
