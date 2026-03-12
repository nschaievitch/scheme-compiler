import subprocess

# wanted to make it look legit
BOLD = "\x1b[1m"
RED = "\x1b[31m"
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
        if expected == ERROR:
            return True

        print(f"{RED}Error compiling {source}: {comp.stderr.decode('utf-8')}{RESET}")
        return False

    interp = subprocess.run(
        [INTERPRETER],
        input=comp.stdout,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        shell=True,
    )

    if expected == ERROR:
        if interp.returncode == 0:
            print(
                f"{RED}Expected error interpreting {source}, got {interp.stdout.decode('utf-8').rstrip()} instead"
            )
            return False
        else:
            return True

    if interp.returncode != 0:
        print(
            f"{RED}Error interpreting {source}: {interp.stderr.decode('utf-8')}{RESET}"
        )
        return False

    output = interp.stdout.decode("utf-8").strip()
    if output != str(expected):
        print(
            f"{RED}Unexpected output for case {source}: expected {expected}, got {output} instead{RESET}"
        )
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
        ("(= #t 1)", "#f"),
    ],
    "Nested functions": [
        ("(not (integer? (integer->char 28)))", "#t"),
        ("(= (- 10 8) (+ 1 1))", "#t"),
        ("(= 10 (char->integer (integer->char 10)))", "#t"),
    ],
    "Let expressions": [
        ("(let ((a 1)) a)", 1),
        ("(let ((a 1) (b 2)) (+ a b))", 3),
        ("(let ((a (+ 1 2)) (b 10)) (- b a))", 7),
        ("(let ((a (let ((c 10)) c)) (b 22)) (- b a))", 12),
        ("(let ((a 42)) (let ((b a)) b))", 42),
        ("(let ((a 1)) (let ((a 2)) a))", 2),
        ("(cons (let ((a 10)) (+ a a)) 30)", "(20 . 30)")
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
        ("(let ((a 10) (b 16)) (if (< a b) a b))", 10),
    ],
    "cons/car/cdr": [
        ("(car (cons 1 2))", 1),
        ("(cdr (cons 1 2))", 2),
        ("(cdr (car (cons (cons 1 2) 3)))", 2),
        ("(let ((pair (cons 10 20))) (car pair))", 10),
        ("(let ((pair (cons 10 (cons 20 30)))) (car (cdr pair)))", 20),
    ],
    "Vector operations": [
        ("(vector-ref (vector 1 2 3) 1)", 2),
        ("(vector-ref (vector 1 2 3) 0)", 1),
        ("(vector-ref (vector 1 2 3) 3)", ERROR),
        ("(let ((v (vector 1 2 3))) (vector-ref v 2))", 3),
        ("(vector-ref (vector-set! (vector 1 2 3) 1 10) 1)", 10),
        ("(vector-set! (vector 1 2 3) 3 10)", ERROR),
        ("(let ((v1 (vector 1 2 3)) (v2 (vector 4 5 6))) (vector-ref v1 2))", 3),
        ("(let ((v1 (vector 1 2 3)) (v2 (vector 40 50 60))) (vector-ref v2 1))", 50),
        (
            "(let ((v1 (vector 1 2 3)) (v2 (vector 4 5 6))) (vector-ref (vector-append v1 v2) 4))",
            5,
        ),
        (
            "(let ((v1 (vector 1 2 3)) (v2 (vector 4 5 6))) (vector-ref (vector-append v1 v2) 2))",
            3,
        ),
    ],
    "String operations": [
        ('(string-ref "hola" 2)', "#\\l"),
        (
            "(string-ref (string #\\h #\\o #\\l #\\a #\\m #\\u #\\n #\\d #\\o) 2)",
            "#\\l",
        ),
        (
            "(string-ref (string #\\h #\\o #\\l #\\a #\\m #\\u #\\n #\\d #\\o) 6)",
            "#\\n",
        ),
        ('(string-ref "holamundo" 9)', ERROR),
        ('(string-ref (string-set! "hola" 3 #\\i) 3)', "#\\i"),
        (
            "(string-ref (string-append (string #\\h #\\o #\\l #\\a) (string #\\m #\\u #\\n #\\d #\\o)) 6)",
            "#\\n",
        ),
        (
            '(string-ref (string-append "hola" (string #\\m #\\u #\\n #\\d #\\o)) 0)',
            "#\\h",
        ),
    ],
    "Printing boxed heap values": [
        ('"hola\\n"', '"hola\n"'),
        ("(cons 1 2)", "(1 . 2)"),
        ("(cons 1 (cons 2 3))", "(1 . (2 . 3))"),
        ('(vector 1 "hola" 10)', '#(1 "hola" 10)'),
    ],
    "begin": [
        ('(let ((s "hola")) (begin (string-set! s 3 #\\i) (string-ref s 3)))', "#\\i"),
        (
            '(let ((s "hola") (v (vector 1 2 3))) (begin (string-set! s 3 #\\i) (vector-set! v 2 5) (string-ref s 3)))',
            "#\\i",
        ),
        (
            '(let ((s "hola") (v (vector 1 2 3))) (begin (string-set! s 3 #\\i) (vector-set! v 2 5) (vector-ref v 2)))',
            "5",
        ),
    ],
    "labels": [("(labels ((f0 (code (x y) (- x y)))) (labelcall f0 25 15))", 10),
               ("""
                (labels (
                    (f0 (code (x y) (- x y)))
                    (f1 (code (x y) (+ (labelcall f0 x 1) y)))) 
                (labelcall f1 10 100))
                """, 109)],
    "closure and funcall":
        [
            ("(labels ((f0 (code (x) (y) (- x y)))) (funcall (closure f0 4) 10))", 6),
            ("(labels ((f_addx (code (y) (x) (+ x y)))) (let ((x 5)) (let ((add5 (closure f_addx x))) (funcall add5 37))))", 42),
            ("(labels ((f_adder (code (y) (x) (+ x y))) (f_make (code (x) () (closure f_adder x)))) (let ((make-adder (closure f_make))) (let ((add7 (funcall make-adder 7))) (funcall add7 10))))", 17),
            ("(labels ((f_inc (code (n) () (+ n 1))) (f_twice (code (f x) () (let ((t (funcall f x))) (funcall f t))))) (let ((inc (closure f_inc)) (twice (closure f_twice))) (funcall twice inc 10)))", 12)
         ],
    "lambdas":
        [
           ("((lambda (x) (+ x x)) 21)", 42),
           ("(let ((adder (lambda (x) (lambda (y) (+ x y))))) ((adder 10) 20))", 30),
           ("(let ((b 2)) (let ((a (lambda (y) (+ y b)))) (+ (a 1) (a 1))))", 6),
           ("(let ((f (lambda (y) (cons 10 y)))) (cdr (f 20)))", 20),
           ("(let ((first (lambda (p) (car p))) (app (lambda (f) (f 10)))) (app (first (cons (lambda (x) (+ x 100)) (lambda (x) (+ x x))))))", 110),
           ("(let ((b 2)) (let ((a (lambda (y) (cons (let ((z 30)) (lambda () (+ y z))) (lambda (c) (- c b)))))) (let ((x (cdr (a 20)))) (x 3))))", 1),
           ("(let ((b 2)) (let ((a (lambda (y) (cons (let ((z 11)) (lambda () (+ y z))) (lambda (a) (- a b)))))) ((car (a 50)))))", 61),
           ("(let ((b 2)) (let ((a (lambda (y) (cons (let ((z 11)) (lambda () (+ y z))) (lambda (a) (- a b)))))) ((cdr (a 50)) 10)))", 8),

        ],
    "letrec":
    [
        ("(letrec ((fact (lambda (x) (if (= x 0) 1 (* x (fact (- x 1))))))) (fact 5))", 120),
        ("(letrec ((even? (lambda (n) (if (= n 0) #t (odd? (- n 1))))) (odd? (lambda (n) (if (= n 0) #f (even? (- n 1)))))) (even? 4))", "#t"),
        ("(letrec ((even? (lambda (n) (if (= n 0) #t (odd? (- n 1))))) (odd? (lambda (n) (if (= n 0) #f (even? (- n 1)))))) (odd? 7))", "#t"),
        ("(letrec ((even? (lambda (n) (if (= n 0) #t (odd? (- n 1))))) (odd? (lambda (n) (if (= n 0) #f (even? (- n 1)))))) (even? 5))", "#f"),
        ("(letrec ((even? (lambda (n) (if (= n 0) #t (odd? (- n 1))))) (odd? (lambda (n) (if (= n 0) #f (even? (- n 1)))))) (odd? 6))", "#f"),
    ],
    "tailcall":
    [
        ("(letrec ((tri (lambda (n acc) (if (= n 0) acc (funcall tri (- n 1) (+ acc n)))))) (tri 1000 0))", ERROR),
        ("(letrec ((tri (lambda (n acc) (if (= n 0) acc (tailcall tri (- n 1) (+ acc n)))))) (tri 1000 0))", 500500),
        (
            "(letrec ((even-sum (lambda (n acc) (if (= n 0) acc (odd-sum (- n 1) (+ acc n))))) (odd-sum (lambda (n acc) (if (= n 0) acc (even-sum (- n 1) (+ acc n)))))) (even-sum 1000 0))",
            500500,
        ),
        (
            "(letrec ((even-sum (lambda (n acc) (if (= n 0) acc (tailcall odd-sum (- n 1) (+ acc n))))) (odd-sum (lambda (n acc) (if (= n 0) acc (tailcall even-sum (- n 1) (+ acc n)))))) (even-sum 1000 0))",
            500500,
        ),
        ("(letrec ((tri (lambda (n acc) (if (= n 0) acc (tailcall tri (- n 1) (+ acc n)))))) (tri 4 0))", 10),
        ("(letrec ((tri (lambda (n acc) (if (= n 0) acc (tri (- n 1) (+ acc n)))))) (tri 4 0))", 10),
        ("(letrec ((countdown (lambda (n) (if (= n 0) 0 (countdown (- n 1)))))) (countdown 10000))", 0)
    ],
    "stdlib": [
        ("(map (lambda (x) (* 2 x)) (cons 1 (cons 2 (cons 3 '()))))", "(2 . (4 . (6 . '())))"),
        ("(foldl + 0 (cons 1 (cons 2 (cons 3 (cons 4 '())))))", 10),
        ("(foldl cons '() (cons 1 (cons 2 (cons 3 (cons 4 '())))))", "(4 . (3 . (2 . (1 . '()))))"), 
        ("(foldr + 0 (cons 1 (cons 2 (cons 3 (cons 4 '())))))", 10),
        ("(foldr cons '() (cons 1 (cons 2 (cons 3 (cons 4 '())))))", "(1 . (2 . (3 . (4 . '()))))"), 
    ],
    "quotes": [
        ("(let ((a (quote (cons 1 2))) (b (quote (cons 1 2)))) (eq? a b))", "#t"),
        ("(quote (cons 1 2))", "(1 . 2)"),
        ("(car (quote (cons 10 20)))", 10),
        ("(cdr (quote (cons 10 20)))", 20),
        ("(let ((a (quote (vector 1 2 3)))) (vector-ref a 1))", 2),
        ("(let ((a (quote (cons 1 (cons 2 (cons 3 '())))))) (car (cdr a)))", 2),
        ("(let ((a (quote (cons 1 2))) (b (quote (cons 3 4)))) (+ (car a) (car b)))", 4),
        ("(let ((a (quote (vector 1 2 3))) (b (quote (vector 1 2 3)))) (eq? a b))", "#t"),
        ("(let ((a (quote (cons 1 2))) (b (quote (cons 1 2))) (c (quote (cons 1 2)))) (if (eq? a b) (eq? b c) #f))", "#t"),
        ("(let ((a (quote (cons 1 2))) (b (quote (cons 3 4)))) (eq? a b))", "#f"),
        ('(let ((a (quote "hi")) (b (quote "hi"))) (eq? a b))', "#t"),
        ('(let ((a (quote "hi")) (b (quote "bye"))) (eq? a b))', "#f"),
        ('(let ((a (quote (cons "ok" 1))) (b (quote (cons "ok" 1)))) (eq? a b))', "#t"),
    ],
    "variadic": [
        ("((lambda (a . b) (cons a (car (cdr b)))) 1 2 3)", "(1 . 3)"), 
        ("(list 1 2 3 4)", "(1 . (2 . (3 . (4 . '()))))")
    ],
    "apply": [
        ("(let ((compose (lambda (f g) (lambda args (f (apply g args)))))) ((compose add1 -) 20 10))", 11),
        ("(apply add1 (cons 41 '()))", 42),
        ("(apply + (cons 20 (cons 22 '())))", 42),
    ],
    "define": [
        ("(define a 10) a", 10),
        ("(define a 10) (define b 20) (cons a b)", "(10 . 20)"),
        ("(define a b) (define b 20) a", ERROR),
        ("(define a 10) (define b (* 2 a)) b", 20)
    ],
    "let*": [
        ("(let* ((a 10) (b (* 2 a))) b)", 20),
        ("(let* ((a b) (b 10)) a)", ERROR),
        ("(let* ((a 10) (b (* 2 a)) (c (- b 5))) c)", 15)
    ],
    "boolean ops and shortcircuitting": [
        ("(or #t #t)", "#t"),
        ("(or #t #f)", "#t"),
        ("(or #f #t)", "#t"),
        ("(or #f #f)", "#f"),
        ("(define v (vector 1)) (or #t (vector-set! v 0 2)) (vector-ref v 0)", "1"),
        ("(define v (vector 1)) (or #f (vector-set! v 0 2)) (vector-ref v 0)", "2"),
        ("(define v (vector 1)) (or #f #t (vector-set! v 0 2)) (vector-ref v 0)", "1"),
        ("(and #t #t)", "#t"),
        ("(and #t #f)", "#f"),
        ("(and #f #t)", "#f"),
        ("(and #f #f)", "#f"),
        ("(define v (vector 1)) (and #f (vector-set! v 0 2)) (vector-ref v 0)", "1"),
        ("(define v (vector 1)) (and #t (vector-set! v 0 2)) (vector-ref v 0)", "2"),
        ("(define v (vector 1)) (and #t #f (vector-set! v 0 2)) (vector-ref v 0)", "1"),
    ],
    "alpha conversion": [
        ("(define a 10) (set! a 20) a", 20),
        ("(let ((a 20)) (begin (set! a 10) a))", 10),
        ("(let ((f (lambda (c) (cons (lambda (v) (set! c v)) (lambda () c))))) (let ((p (f 0))) (begin ((car p) 12) ((cdr p)))))", 12)
    ]
}

passed = 0
total = 0

for testSuite, cases in testSuites.items():
    print(f"{BOLD}{testSuite}: {RESET}")

    passLocal = 0

    for source, expected in cases:
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
