#!/usr/bin/env escript

classify(S) when S < 5   -> "urgent";
classify(S) when S < 30  -> "fast";
classify(S) when S < 120 -> "normal";
classify(S) when S < 600 -> "slow";
classify(_)               -> "absent".

main([_SessionId, LastTsStr]) ->
    LastTs = list_to_integer(LastTsStr),
    Now = os:system_time(second),
    Diff = Now - LastTs,
    io:format("~s~n", [classify(Diff)]);
main(_) ->
    io:format("normal~n").
