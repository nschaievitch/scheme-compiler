(define isprime (lambda (n) 
    (letrec ((aux (lambda (f n) (if (= f (sub1 n)) #t
        (and (not (= (modulo n f) 0)) (aux (+ f 1) n))))))
    (aux 2 n))))

(define range (lambda (a b) (letrec ((aux (lambda (a b acc) 
    (if (= a b) acc (aux a (sub1 b) (cons b acc)))))) (aux a b '()))))

(filter isprime (range 2 100))