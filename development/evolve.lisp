; Chaos2 — development module
; Lisp: rewrites Chaos's own rules based on accumulated experience

(defun split-pipe (str)
  (let ((pos (position #\| str)))
    (when pos
      (list (subseq str 0 pos)
            (subseq str (1+ pos))))))

(defun parse-stat (line)
  (let ((parts (split-pipe line)))
    (when parts
      (let ((val (read-from-string (second parts) nil nil)))
        (when (numberp val)
          (list (first parts) val))))))

(defun evolve-rule (rule-name current-weight stat-value)
  (list rule-name (max 0.1 (min 2.0 (+ current-weight (* stat-value 0.05))))))

(defun default-rules ()
  '(("aggression-sensitivity" 1.0)
    ("trust-threshold"        1.0)
    ("silence-bias"           0.5)
    ("intensity-ceiling"      1.0)
    ("memory-retention"       0.8)))

(defun apply-stats (rules stats)
  (mapcar (lambda (rule)
    (let* ((name   (first rule))
           (weight (second rule))
           (stat   (assoc name stats :test #'string=)))
      (if stat
          (evolve-rule name weight (second stat))
          rule)))
    rules))

(defun print-rules (rules)
  (dolist (rule rules)
    (format t "~a|~,4f~%" (first rule) (second rule))))

(defun main ()
  (let ((stats '()))
    (loop for line = (read-line *standard-input* nil nil)
          while line
          when (> (length line) 0)
          do (let ((stat (parse-stat line)))
               (when stat (push stat stats))))
    (print-rules (apply-stats (default-rules) stats))))

(main)
