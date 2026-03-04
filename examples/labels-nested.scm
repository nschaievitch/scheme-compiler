(labels (
    (f0 (code (x y) (- x y)))
    (f1 (code (x y) (+ (labelcall f0 x 1) y)))) 
(labelcall f1 10 100))
