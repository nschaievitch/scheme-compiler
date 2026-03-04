(labels
  ((f_inc (code (n) ()
           (+ n 1)))
   (f_twice (code (f x) ()
             (let ((t (funcall f x)))
               (funcall f t)))))
  (let ((inc (closure f_inc))
        (twice (closure f_twice)))
    (funcall twice inc 10)))