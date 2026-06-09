% Chaos2 — mirror module
% R observes and remembers. Prolog interprets and infers.
% Facts are asserted per-call from R's exported state.
% Prolog is stateless by design — it only reasons, never stores.

:- module(mirror, [print_inferences/1]).

:- dynamic player_fact/3.

% ── Inferences ────────────────────────────────────────────────────────────────

% Deliberate escalation: sustained aggression + trajectory going up
escalating_deliberately(S) :-
    player_fact(S, aggression_count, A), A > 5,
    player_fact(S, emotional_drift, escalating).

% Testing limits: provocation is the dominant pattern
testing_limits(S) :-
    player_fact(S, provocation_count, P), P > 3,
    player_fact(S, dominant, provocation).

% Manipulation: aggression and trust alternating in close ratio
manipulating(S) :-
    player_fact(S, aggression_count, A), A > 2,
    player_fact(S, trust_count, T), T > 2,
    player_fact(S, total, N), N > 5,
    Ratio is abs(A - T) / N,
    Ratio < 0.2.

% Attached: intimacy + trust building
attached(S) :-
    player_fact(S, intimacy_count, I), I > 3,
    player_fact(S, trust_count, T), T > 2.

% Withdrawing: pulling back consistently
withdrawing(S) :-
    player_fact(S, withdrawal_count, W), W > 3,
    player_fact(S, total, N), N > 0,
    Ratio is W / N,
    Ratio > 0.4.

% Using philosophy as refuge: ideas instead of contact
using_philosophy(S) :-
    player_fact(S, philosophical_count, P), P > 3,
    player_fact(S, intimacy_count, I), I < 2,
    player_fact(S, trust_count, T), T < 2.

% Oscillating: erratic pattern across intent types
oscillating(S) :-
    player_fact(S, volatility, erratic).

% ── Output ────────────────────────────────────────────────────────────────────

active_inference(S, escalating_deliberately) :- escalating_deliberately(S).
active_inference(S, testing_limits)          :- testing_limits(S).
active_inference(S, manipulating)            :- manipulating(S).
active_inference(S, attached)                :- attached(S).
active_inference(S, withdrawing)             :- withdrawing(S).
active_inference(S, using_philosophy)        :- using_philosophy(S).
active_inference(S, oscillating)             :- oscillating(S).

print_inferences(S) :-
    findall(I, active_inference(S, I), List),
    ( List = [] ->
        format("none~n")
    ;
        atomic_list_concat(List, ',', Atom),
        format("~w~n", [Atom])
    ).
