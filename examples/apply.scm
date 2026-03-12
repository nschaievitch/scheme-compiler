(let ((compose
  (lambda (f g)
    (lambda args
      (f (apply g args))))))

((compose add1 -) 20 10))