# Scheme Compiler

Bytecode compiler for a subset of the Scheme language.

Developed for Max Bernstein's *[Compilers for Functional Languages](https://bernsteinbear.com/dartmouth-compilers/)*, which roughly follows [Abdulaziz Ghuloum’s compiler construction tutorial](https://bernsteinbear.com/dartmouth-compilers/11-ghuloum.pdf).

## Building

For the compiler, it is enough to do `cd compiler && go build`.

For the interpreter, `cd interpreter && cargo build`.

Or, use the `buildAll.sh` script to build both.

This generates binaries `compiler/compiler` and `interpreter/target/debug/interpreter`, which can be used through stdin/stdout. Normal usage would be something like:

```bash
./compiler < script.scm > script.bc
./interpreter < script.bc
```

Alternatively, use `python run.py script.scm` to compile and run.


## Testing

To run the test suite for the compiler, do `cd compiler && go test`.

To run the test suite for the interpreter, make sure **both** the compiler and interpreter are built, and run `python3 testInterpreter.py`.

Run `testAll.sh` to build and test both.