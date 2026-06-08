\ Chaos2 — purpose module
\ Forth: defines Chaos's own goals based on current plan
\ Usage: gforth goals.fs -e 's" planname" evaluate-goals bye'

: goal. ( addr len -- ) type cr ;

: goals-for-observe
    s" survive"     goal.
    s" understand"  goal. ;

: goals-for-mirror
    s" reflect"     goal.
    s" expose"      goal. ;

: goals-for-resist
    s" endure"      goal.
    s" refuse"      goal. ;

: goals-for-collapse
    s" break"       goal.
    s" release"     goal. ;

: goals-for-reveal
    s" trust"       goal.
    s" open"        goal. ;

: goals-for-seduce
    s" attract"     goal.
    s" hold"        goal. ;

: goals-for-destabilize
    s" unsettle"    goal.
    s" fracture"    goal. ;

: goals-for-investigate
    s" probe"       goal.
    s" map"         goal. ;

: goals-for-existential
    s" question"    goal.
    s" dissolve"    goal. ;

: goals-for-test
    s" measure"     goal.
    s" wait"        goal. ;

: evaluate-goals ( addr len -- )
    2dup s" observe"      compare 0= if goals-for-observe      bye then
    2dup s" mirror"       compare 0= if goals-for-mirror       bye then
    2dup s" resist"       compare 0= if goals-for-resist       bye then
    2dup s" collapse"     compare 0= if goals-for-collapse     bye then
    2dup s" reveal"       compare 0= if goals-for-reveal       bye then
    2dup s" seduce"       compare 0= if goals-for-seduce       bye then
    2dup s" destabilize"  compare 0= if goals-for-destabilize  bye then
    2dup s" investigate"  compare 0= if goals-for-investigate  bye then
    2dup s" existential"  compare 0= if goals-for-existential  bye then
    2dup s" test_loyalty" compare 0= if goals-for-test         bye then
    goals-for-observe bye ;
