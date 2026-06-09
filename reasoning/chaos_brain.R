#!/usr/bin/env Rscript
# Chaos2 — R brain daemon
# Persistent process. Reads one line per message from stdin:
#   session_id|intent|chaos_state|trust_level
# Writes one line per message to stdout:
#   plan=...|player_type=...|threat_level=...|...

sessions <- list()

make_session <- function() {
  list(
    counts = list(
      aggression=0, trust=0, withdrawal=0,
      philosophical=0, intimacy=0, provocation=0, curiosity=0
    ),
    total       = 0L,
    trust_level = 0.0
  )
}

decide <- function(sess, chaos_state) {
  counts <- sess$counts
  total  <- sess$total

  aggr         <- counts$aggression
  trust_cnt    <- counts$trust
  withdr       <- counts$withdrawal
  phil_cnt     <- counts$philosophical
  intimacy_cnt <- counts$intimacy
  provoc_cnt   <- counts$provocation
  curious_cnt  <- counts$curiosity

  plan            <- "observe"
  player_type     <- "unknown"
  threat_level    <- 0
  emotional_drift <- "stable"
  manipulation    <- FALSE
  confidence      <- 0.5
  dominant_intent <- "unknown"
  volatility      <- "low"

  if (total > 0) {
    cnt_vec <- c(
      aggression=aggr, trust=trust_cnt, withdrawal=withdr,
      philosophical=phil_cnt, intimacy=intimacy_cnt,
      provocation=provoc_cnt, curiosity=curious_cnt
    )
    dominant_intent <- names(which.max(cnt_vec))
  }

  if (total >= 4) {
    aggr_ratio     <- aggr         / total
    trust_ratio    <- trust_cnt    / total
    withdr_ratio   <- withdr       / total
    phil_ratio     <- phil_cnt     / total
    intimacy_ratio <- intimacy_cnt / total
    provoc_ratio   <- provoc_cnt   / total

    if      (aggr_ratio     > 0.4) player_type <- "aggressive"
    else if (intimacy_ratio > 0.3) player_type <- "vulnerable"
    else if (provoc_ratio   > 0.3) player_type <- "challenger"
    else if (trust_ratio    > 0.4) player_type <- "trusting"
    else if (withdr_ratio   > 0.3) player_type <- "avoidant"
    else if (phil_ratio     > 0.3) player_type <- "philosophical"
    else                            player_type <- "mixed"
  }

  threat_level    <- min(10, round(aggr * 1.5 + provoc_cnt * 1.0 + withdr * 0.5))

  if      (aggr > withdr + trust_cnt)      emotional_drift <- "escalating"
  else if (trust_cnt > aggr + withdr)      emotional_drift <- "calming"
  else if (withdr > aggr + trust_cnt)      emotional_drift <- "retreating"

  if      (aggr >= 3 && trust_cnt >= 2)    volatility <- "high"
  else if (total >= 6)                     volatility <- "medium"

  # intent-history-based plan
  if (total >= 6 && aggr >= 2 && trust_cnt >= 2) {
    manipulation <- TRUE
    plan         <- "mirror"
    confidence   <- 0.8
  } else if (total >= 6 && aggr >= 3) {
    plan         <- "confront"
    threat_level <- min(10, threat_level + 2)
    confidence   <- 0.75
  } else if (total >= 4 && intimacy_cnt >= 2) {
    plan         <- "seduce"
    confidence   <- 0.7
  } else if (total >= 4 && trust_cnt >= 3) {
    plan         <- "seduce"
    confidence   <- 0.7
  } else if (total >= 4 && provoc_cnt >= 2) {
    plan         <- "confront"
    confidence   <- 0.65
  } else if (total >= 4 && withdr >= 2) {
    plan         <- "observe"
    confidence   <- 0.55
  } else if (total >= 6 && phil_cnt >= 2) {
    plan         <- "philosophical"
    confidence   <- 0.6
  }

  # state-driven override — emotional moment plans
  if (chaos_state == "Redemptive") {
    plan       <- "reveal"
    confidence <- max(confidence, 0.7)
  } else if (chaos_state == "Obsessive_Hate" && threat_level >= 7) {
    plan       <- "collapse"
    confidence <- max(confidence, 0.85)
  } else if (chaos_state == "Hostile" && plan == "confront") {
    plan       <- "resist"
  }

  list(
    plan             = plan,
    player_type      = player_type,
    threat_level     = as.integer(threat_level),
    emotional_drift  = emotional_drift,
    manipulation     = tolower(as.character(manipulation)),
    confidence       = confidence,
    dominant_intent  = dominant_intent,
    trust_level      = sess$trust_level,
    volatility       = volatility,
    intimacy_signals = as.integer(trust_cnt + intimacy_cnt),
    aggression_count = as.integer(aggr)
  )
}

fallback_line <- function() {
  cat("plan=observe|player_type=unknown|threat_level=0|emotional_drift=stable|manipulation=false|confidence=0.50|dominant_intent=unknown|trust_level=0.00|volatility=low|intimacy_signals=0|aggression_count=0\n")
  flush(stdout())
}

con <- file("stdin", "r")
repeat {
  line <- tryCatch(readLines(con, n=1, warn=FALSE), error=function(e) character(0))
  if (length(line) == 0) break
  line <- trimws(line)
  if (nchar(line) == 0) next

  parts <- strsplit(line, "|", fixed=TRUE)[[1]]
  if (length(parts) < 3) { fallback_line(); next }

  sid         <- parts[1]
  intent      <- parts[2]
  chaos_state <- parts[3]
  trust_val   <- if (length(parts) >= 4) suppressWarnings(as.numeric(parts[4])) else 0.0
  if (is.na(trust_val)) trust_val <- 0.0

  if (is.null(sessions[[sid]])) sessions[[sid]] <- make_session()
  sess <- sessions[[sid]]

  if (intent %in% names(sess$counts)) {
    sess$counts[[intent]] <- sess$counts[[intent]] + 1L
  }
  sess$total       <- sess$total + 1L
  sess$trust_level <- trust_val
  sessions[[sid]]  <- sess

  r <- decide(sess, chaos_state)

  # prolog facts — R observes, Prolog infers
  counts <- sess$counts
  facts <- paste(c(
    paste0("aggression_count(",   as.integer(counts$aggression),   ")"),
    paste0("trust_count(",        as.integer(counts$trust),        ")"),
    paste0("withdrawal_count(",   as.integer(counts$withdrawal),   ")"),
    paste0("provocation_count(",  as.integer(counts$provocation),  ")"),
    paste0("intimacy_count(",     as.integer(counts$intimacy),     ")"),
    paste0("philosophical_count(",as.integer(counts$philosophical),")"),
    paste0("curiosity_count(",    as.integer(counts$curiosity),    ")"),
    paste0("total(",              as.integer(sess$total),          ")"),
    paste0("dominant(",           r$dominant_intent,               ")"),
    paste0("emotional_drift(",    r$emotional_drift,               ")"),
    paste0("volatility(",         r$volatility,                    ")"),
    sprintf("trust_level(%.2f)",  sess$trust_level)
  ), collapse = ";")

  cat(sprintf(
    "plan=%s|player_type=%s|threat_level=%d|emotional_drift=%s|manipulation=%s|confidence=%.2f|dominant_intent=%s|trust_level=%.2f|volatility=%s|intimacy_signals=%d|aggression_count=%d|prolog_facts=%s\n",
    r$plan, r$player_type, r$threat_level, r$emotional_drift,
    r$manipulation, r$confidence, r$dominant_intent,
    r$trust_level, r$volatility, r$intimacy_signals, r$aggression_count,
    facts
  ))
  flush(stdout())
}
