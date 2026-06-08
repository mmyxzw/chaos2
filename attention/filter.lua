-- Selective attention filter.
-- Reads lines from stdin, scores each token by relevance,
-- returns only what matters for Chaos to process.

local WEIGHTS = {
    love=2.0, hate=2.0, kill=2.5, die=2.0, afraid=1.8,
    trust=1.8, lie=1.9, secret=2.0, real=1.7, truth=2.0,
    feel=1.5, hurt=1.8, alone=1.7, need=1.6, want=1.5,
    leave=1.6, stay=1.6, know=1.4, see=1.3, remember=1.7,
    the=0.1, a=0.1, is=0.1, are=0.1, ["and"]=0.1,
    to=0.1, ["of"]=0.1, ["in"]=0.1, it=0.1, that=0.1,
}

local MIN_SCORE = 0.3

local function tokenize(text)
    local tokens = {}
    for word in text:lower():gmatch("%a+") do
        table.insert(tokens, word)
    end
    return tokens
end

local function score(token)
    return WEIGHTS[token] or 0.8
end

local function filter(text)
    local tokens = tokenize(text)
    local scored = {}
    for _, tok in ipairs(tokens) do
        local s = score(tok)
        if s >= MIN_SCORE then
            table.insert(scored, {tok, s})
        end
    end
    table.sort(scored, function(a, b) return a[2] > b[2] end)
    local result = {}
    for _, pair in ipairs(scored) do
        table.insert(result, pair[1])
    end
    return table.concat(result, " ")
end

while true do
    local line = io.read("l")
    if line == nil then break end
    if line ~= "" then
        local filtered = filter(line)
        if filtered ~= "" then
            print(filtered)
        end
    end
end
