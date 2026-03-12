import subprocess, sys

BOLD = "\x1b[1m"
RED = "\x1b[31m"
GREEN = "\x1b[32m"
RESET = "\x1b[0m"

COMPILER = "./compiler/compiler"
INTERPRETER = "./interpreter/target/debug/interpreter"


def testCase(source):
    print("compiling...")
    comp = subprocess.run(
        [COMPILER],
        input=source.encode("utf-8"),
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )

    if comp.returncode != 0:
        print(f"{RED}Error compiling {source}: {comp.stderr.decode('utf-8')}{RESET}")
        return False

    print("interpretting...")
    interp = subprocess.run(
        [INTERPRETER],
        input=comp.stdout,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        shell=True,
    )

    if interp.returncode != 0:
        print(
            f"{RED}Error interpreting {source}: {interp.stderr.decode('utf-8')}{RESET}"
        )
        return False

    output = interp.stdout.decode("utf-8").strip()
    print(output)


path = sys.argv[1]

source = open(path).read()

testCase(source)