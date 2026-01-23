import subprocess

# wanted to make it look legit
BOLD  = "\x1b[1m"
RED   = "\x1b[31m"
GREEN = "\x1b[32m"
RESET = "\x1b[0m"

ERROR = None

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
        print(f"{RED}Error compiling {source}: {comp.stderr.decode("utf-8")}{RESET}")
        return False

    interp = subprocess.run(
        [INTERPRETER],
        input=comp.stdout,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        shell=True
    )

    if expected == ERROR:
        if interp.returncode == 0:
            print(f"{RED}Expected error interpreting {source}, got {interp.stdout.decode("utf-8").rstrip()} instead")
            return False
        else:
            return True

    if interp.returncode != 0:
        print(f"{RED}Error interpreting {source}: {interp.stderr.decode("utf-8")}{RESET}")
        return False
    
    output = interp.stdout.decode("utf-8").strip()
    if output != str(expected):
        print(f"{RED}Unexpected output for case {source}: expected {expected}, got {output} instead{RESET}")
        return False
    
    return True

testSuites = {
    "Literals": [
        ("1", "1"),
        ("#\\a", "#\\a"),
        ("#t", "#t"),
        ("#f", "#f"),
    ],
    "Unary functions": [
        ("(char->integer #\\a)", 97),
        ("(char->integer 97)", ERROR),
        ("(integer->char 97)", "#\\a"),
        ("(integer->char #t)", ERROR),
        ("(add1 10)", "11"),
        ("(add1 #t)", ERROR),
        ("(sub1 10)", "9"),
        ("(sub1 #t)", ERROR),
        ("(null? 1)", "#f"),
        ("(zero? 0)", "#t"),
        ("(zero? 1)", "#f"),
        ("(zero? #t)", ERROR),
        ("(not #t)", "#f"),
        ("(not #f)", "#t"),
        ("(not 1)", ERROR),
        ("(boolean? #f)", "#t"),
        ("(boolean? 10)", "#f"),
        ("(integer? 10)", "#t"),
        ("(integer? #\\a)", "#f"),
    ],
    "Binary functions": [
        ("(+ 1 2)", "3"),
        ("(+ #f 2)", ERROR),
        ("(+ 2 #f)", ERROR),
        ("(* 10 10)", "100"),
        ("(* #f 2)", ERROR),
        ("(* 2 #f)", ERROR),
        ("(- 10 8)", "2"),
        ("(- #f 2)", ERROR),
        ("(- 2 #f)", ERROR),
        ("(< 10 8)", "#f"),
        ("(< 1 8)", "#t"),
        ("(< #f 2)", ERROR),
        ("(< 2 #f)", ERROR),
        ("(= 1 1)", "#t"),
        ("(= 1 2)", "#f"),
        ("(= #\\a #\\a)", "#t"),
        ("(= #\\a #\\b)", "#f"),
        ("(= #f #f)", "#t"),
        ("(= #t #t)", "#t"),
        ("(= #t #f)", "#f"),
        ("(= #t 1)", "#f")
    ],
    "Nested functions": [
        ("(not (integer? (integer->char 28)))", "#t"),
        ("(= (- 10 8) (+ 1 1))", "#t"),
        ("(= 10 (char->integer (integer->char 10)))", "#t"),
    ],
    "Let expressions": [
        ("(let (a 1) a)", 1),
        ("(let (a 1) (b 2) (+ a b))", 3),
        ("(let (a (+ 1 2)) (b 10) (- b a))", 7),
        ("(let (a (let (c 10) c)) (b 22) (- b a))", 12),
        ("(let (a 42) (let (b a) b))", 42),
    ],
    "If expressions": [
        ("(if #t 1 2)", 1),
        ("(if #f 1 2)", 2),
        ("(if 1 1 2)", 1),
        ("(if #\\a 1 2)", 1),
        ("(if #t (+ 1 2) 1)", 3),
        ("(if #f (+ 1 2) 1)", 1),
        ("(if (= 10 10) (if #t 1 2) 3)", 1),
        ("(if (= 10 10) (if #f 1 2) 3)", 2),
        ("(let (a 10) (b 16) (if (< a b) a b))", 10)
    ]
}

passed = 0
total = 0

for (testSuite, cases) in testSuites.items():
    print(f"{BOLD}{testSuite}: {RESET}")

    passLocal = 0

    for (source, expected) in cases:
        if testCase(source, expected):
            passLocal += 1

    if passLocal == len(cases):
        print(GREEN)
    else:
        print(RED)
    print(f"\t passed {passLocal}/{len(cases)}{RESET}\n")
    

    passed += passLocal
    total += len(cases)

print(f"{BOLD}\n\nPassed {passed} out of {total}{RESET}")
    