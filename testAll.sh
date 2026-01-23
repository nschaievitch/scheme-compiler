#!/bin/bash

echo "Building all..."

cd interpreter && cargo build 2> /dev/null

cd ../compiler && go build

echo -e "\n----------------"
echo -e "Testing compiler"
echo -e "----------------\n"

go test

cd ..

echo -e "\n-------------------"
echo -e "Testing interpreter"
echo -e "-------------------\n"

python3 testInterpreter.py