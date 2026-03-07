(letrec ((map (lambda (f l) 
            (if (null? l) ()
                (cons (f (car l)) (cdr l))))) (map (lambda (x) (+ x 1)) (cons 1 (cons 2 3))))