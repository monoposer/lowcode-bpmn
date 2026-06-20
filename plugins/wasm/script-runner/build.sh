#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
docker run --rm \
  -v "$PWD:/src" -w /src \
  tinygo/tinygo:0.34.0 \
  tinygo build -o script.wasm -target=wasi -opt=2 main.go
echo "built $(wc -c < script.wasm) bytes -> script.wasm"
