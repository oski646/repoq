#!/bin/sh
set -eu

if ! command -v go >/dev/null 2>&1; then
  echo "Go is required to install repoq."
  echo "Install Go from https://go.dev/dl/ and run this script again."
  exit 1
fi

echo "Installing repoq..."
go install github.com/oski646/repoq@latest

go_bin="$(go env GOPATH)/bin"
repoq_bin="$go_bin/repoq"

if [ ! -x "$repoq_bin" ]; then
  echo "repoq installation finished, but binary was not found at $repoq_bin."
  exit 1
fi

echo "repoq installed to $repoq_bin"

if command -v repoq >/dev/null 2>&1; then
  echo "repoq is available in PATH."
  exit 0
fi

echo
echo "$go_bin is not in PATH."
echo "Add this line to your shell config:"
echo
echo "export PATH=\"\$PATH:$go_bin\""
echo
echo "Then restart your terminal or reload your shell config."
