import subprocess, tqdm

COMPILER = "./compiler/compiler"
INTERPRETER = "./interpreter/target/debug/interpreter"

def testCase(source, expected):
    comp = subprocess.run(
        [COMPILER],
        input=source.encode("utf-8"),
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )
    if comp.returncode != 0:
        print(f"Error compiling {source}: {comp.stderr}")
        return False

    interp = subprocess.run(
        [INTERPRETER],
        input=comp.stdout,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        check=True,
        shell=True
    )
    
    output = interp.stdout.decode("utf-8").strip()
    if output != str(expected):
        print(f"Unexpected output: expected {expected}, got {output} instead")
        return False
    
    return True

testCases = [
    ("1", "1"),
    ("#\\a", "#\\a"),
    ("#t", "#t"),
    ("#f", "#f"),
    ("(+ 1 2)", "3"),
    ("(char->integer #\\a)", 97),
    ("(integer->char 97)", "#\\a"),
    ("(add1 10)", "11"),
    ("(sub1 10)", "9"),
    ("(null? 1)", "#f"),
    ("(zero? 0)", "#t"),
    ("(zero? 1)", "#f"),
    ("(not #t)", "#f"),
    ("(not #f)", "#t"),
    ("(boolean? #f)", "#t"),
    ("(boolean? 10)", "#f"),
    ("(integer? 10)", "#t"),
    ("(integer? #\\a)", "#f"),
    ("(* 10 10)", "100"),
    ("(- 10 8)", "2"),
    ("(< 10 8)", "#f"),
    ("(< 1 8)", "#t"),
    ("(= 1 1)", "#t"),
    ("(= 1 2)", "#f"),
    ("(= #\\a #\\a)", "#t"),
    ("(= #\\a #\\b)", "#f"),
    ("(= #f #f)", "#t"),
    ("(= #t #t)", "#t"),
    ("(= #t #f)", "#f"),
    ("(not (integer? (integer->char 28)))", "#t"),
    ("(= (- 10 8) (+ 1 1))", "#t"),
    ("(= 10 (char->integer (integer->char 10)))", "#t")
]

passed = 0

for (source, expected) in testCases:
    if testCase(source, expected):
        passed += 1

print(f"\n\nPassed {passed} out of {len(testCases)}")
    