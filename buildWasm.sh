#!/bin/bash

cd compiler && GOOS=wasip1 GOARCH=wasm go build -o compiler.wasm

cd ../interpreter && cargo build --target wasm32-wasip1

cd ../web-demo/scheme-web-demo && pnpm run build