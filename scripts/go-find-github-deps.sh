#!/usr/bin/env bash
set -ueo pipefail

# Print usage
if [[ "$#" -eq 0 ]]; then
  echo "Usage: go-find-deps.sh /path/to/repo1 /path/to/repo2 ..."
  echo
  exit 0
fi

(
  # shellcheck disable=SC2086
  find $* -name "Godeps*" -exec cat {} \; \
    | grep -vE "^#" | awk '{ print $1 }' \
    | grep "github.com" \
    | sed -E "s/.*(github.com[\/:][^/]+\/[^/]+)[/\"]?.*/\1/g"
  # shellcheck disable=SC2086
  find $* -name "*.go" -exec cat {} \; \
    | grep -E "[\s\t]*\"github.com" \
    | sed -E "s/.*(github.com\/[^/]+\/[^/\"]+)[/\"]?.*/\1/g"
) | sed -E "s/github.com:/github.com\//g" \
  | grep -E "^github.com\/[^/]+\/[a-zA-Z0-9_\.-]+$" \
  | sort \
  | uniq
