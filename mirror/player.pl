% Chaos2 — mirror module
% Prolog: builds internal model of the player from interaction history
% Query: model(SessionId, Properties)

:- module(mirror, [model/2, update_model/3]).

:- dynamic player_fact/3.

model(SessionId, Properties) :-
    findall(Key-Value, player_fact(SessionId, Key, Value), Facts),
    ( Facts = [] ->
        Properties = [type-unknown, trust-0.0, threat-low, dominant-unknown]
    ;
        Properties = Facts
    ).

update_model(SessionId, Key, Value) :-
    retractall(player_fact(SessionId, Key, _)),
    assertz(player_fact(SessionId, Key, Value)).

archetype(SessionId, Archetype) :-
    ( player_fact(SessionId, type, T) -> Archetype = T ; Archetype = unknown ).

trust_level(SessionId, Trust) :-
    ( player_fact(SessionId, trust, T) -> Trust = T ; Trust = 0.0 ).

is_threat(SessionId) :-
    player_fact(SessionId, threat, high).

print_model(SessionId) :-
    model(SessionId, Props),
    forall(
        member(Key-Value, Props),
        format("~w=~w~n", [Key, Value])
    ).
