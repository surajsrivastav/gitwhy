#!/bin/sh
# install.sh — install ghw via go install
set -eu

if ! command -v go >/dev/null 2>&1; then
    echo "error: go is required but not found in PATH"
    echo "install Go: https://go.dev/doc/install"
    exit 1
fi

echo "installing ghw..."
go install github.com/surajsrivastav/gitwhy@latest

echo ""
echo "done. verify with: ghw version"
echo ""
echo "next steps:"
echo "  cd your-repo && ghw init"
echo "  git commit -m '...'  # auto-capture enabled"
