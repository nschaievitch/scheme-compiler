#!/bin/bash

echo "Building all..."

cd interpreter && cargo build 2> /dev/null

cd ../compiler && go build