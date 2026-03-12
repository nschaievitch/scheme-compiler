(letrec ((map (lambda (f l) 
            (if (null? l) '()
                (cons (f (car l)) (map f (cdr l))))))
         (foldl (lambda (f init l)
            (if (null? l)
                 init
                 (foldl f (f (car l) init) (cdr l)))))
         (foldr (lambda (f init l)
            (if (null? l)
                init
                (f (car l) (foldr f init (cdr l))))))
         (list (lambda args args)))                
        (main))
