#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 2 || $# -gt 3 ]]; then
  echo "usage: $0 <sudoku-binary> <database-path> [workers]" >&2
  exit 2
fi

binary=$1
database=$2
workers=${3:-$(getconf _NPROCESSORS_ONLN)}
commit=d8c8ebaee0c08c412cfba96af1923dfa61c83317
sha256=08553d0c1145ea4d7c13008040f47ea8205d21fe1eaf8f4ab17a1a6981928b35
url="https://raw.githubusercontent.com/grantm/sudoku-exchange-puzzle-bank/${commit}/diabolical.txt"
source="sudoku-exchange-diabolical@${commit}"

temporary=$(mktemp)
trap 'rm -f "$temporary"' EXIT
curl --fail --location --retry 3 --output "$temporary" "$url"
"$binary" import \
  --file "$temporary" \
  --format sudoku-exchange \
  --sha256 "$sha256" \
  --source "$source" \
  --workers "$workers" \
  --db "$database"
