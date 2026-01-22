#!/bin/bash

echo "Building all..."

cd interpreter && cargo build 2> /dev/null

cd ../compiler && go build

echo -e "\nTesting compiler"

go test

cd ..

echo -e "\nTesting interpreter"

python3 testInterpreter.py