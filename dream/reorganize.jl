struct Memory
    timestamp::Int64
    strength::Float64
    content::String
end

function parse_memory(line::String)::Union{Memory, Nothing}
    parts = split(line, "|", limit=3)
    length(parts) == 3 || return nothing
    ts = tryparse(Int64, parts[1])
    st = tryparse(Float64, parts[2])
    (ts === nothing || st === nothing) && return nothing
    Memory(ts, st, parts[3])
end

function similarity(a::String, b::String)::Float64
    wa = Set(split(lowercase(a)))
    wb = Set(split(lowercase(b)))
    isempty(wa) && return 0.0
    length(intersect(wa, wb)) / length(union(wa, wb))
end

function dream(memories::Vector{Memory})::Vector{Memory}
    isempty(memories) && return memories
    boosts = zeros(Float64, length(memories))
    for i in 1:length(memories), j in 1:length(memories)
        i == j && continue
        sim = similarity(memories[i].content, memories[j].content)
        sim > 0.2 && (boosts[i] += sim * memories[j].strength * 0.1)
    end
    result = [Memory(m.timestamp, clamp(m.strength + boosts[i], 0.0, 1.0), m.content)
              for (i, m) in enumerate(memories)]
    sort!(result, by=m -> m.strength, rev=true)
end

function main()
    memories = Memory[]
    for line in eachline(stdin)
        isempty(strip(line)) && continue
        m = parse_memory(line)
        m !== nothing && push!(memories, m)
    end
    for m in dream(memories)
        println("$(m.timestamp)|$(round(m.strength, digits=4))|$(m.content)")
    end
    println(stderr, "[dream] reorganized $(length(memories)) memories")
end

main()
