(labels
  ((f_addx (code (y) (x)
            (+ x y))))
  (let (x 5)
    (let (add5 (closure f_addx x))
      (funcall add5 37))))