# COSC294

## Building

For the compiler, it is enough to do `cd compiler && go build`.

For the interpreter, `cd interpreter && cargo build`.

Or, use the `buildAll.sh` script to build both.

This generates binaries `compiler/compiler` and `interpreter/target/debug/interpreter`, which can be used through stdin/stdout as would be expected.


## Testing

To run the test suite for the compiler, do `cd compiler && go test`.

To run the test suite for the interpreter, make sure **both** the compiler and interpreter are built, and run `python3 testInterpreter.py`.

Run `testAll.sh` to build and test both.