(define isprime (lambda (n) 
    (letrec ((aux (lambda (f n) (if (= f (sub1 n)) #t
        (and (not (= (modulo n f) 0)) (tailcall aux (+ f 1) n))))))
    (aux 2 n))))

(isprime 1166857)
