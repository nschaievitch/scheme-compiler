import subprocess, tqdm

COMPILER = "./compiler/compiler"
INTERPRETER = "./interpreter/target/debug/interpreter"

def testCase(source, expected):
    comp = subprocess.run(
        [COMPILER],
        input=source.encode("utf-8"),
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=True
    )

    interp = subprocess.run(
        [INTERPRETER],
        input=comp.stdout,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=True,
        shell=True
    )
    
    output = interp.stdout.decode("utf-8").strip()
    if output != expected:
        print(f"Unexpected output: expected {expected}, got {output} instead")
        return False
    
    return True

testCases = [
    ("1", "1"),
    ("#\\a", "#\\a"),
    ("#t", "#t"),
    ("#f", "#f"),
    ("(+ 1 2)", "3"),
    ("(not (integer? (integer->char 28)))", "#t"),
    ("(= (- 10 8) (+ 1 1))", "#t")
]

passed = 0

for (source, expected) in testCases:
    if testCase(source, expected):
        passed += 1

print(f"\n\nPassed {passed} out of {len(testCases)}")
    