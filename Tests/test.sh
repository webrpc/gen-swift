#!/bin/sh
set -eu

script_dir=$(CDPATH= cd -- "$(dirname "$0")" && pwd)
repo_root=$(CDPATH= cd -- "$script_dir/.." && pwd)

mkdir -p "$script_dir/Sources/Generated"

go -C "$repo_root/tools" tool webrpc-gen \
  -schema="$script_dir/test.ridl" \
  -target="$repo_root" \
  -client \
  -out="$script_dir/Sources/Generated/Generated.swift"

swift test --package-path "$script_dir"
