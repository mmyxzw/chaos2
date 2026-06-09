#!/usr/bin/env Rscript
# Chaos2 — R brain (reasoning module)
# Args: history.csv  profile.csv
# Output: pipe-separated key=value pairs on one line

suppressMessages(suppressWarnings({
  args <- commandArgs(trailingOnly = TRUE)
}))

# ── defaults ──────────────────────────────────────────────────────────────────
plan            <- "observe"
player_type     <- "unknown"
threat_level    <- 0
emotional_drift <- "stable"
manipulation    <- FALSE
confidence      <- 0.5
dominant_intent <- "unknown"
trust_level     <- 0.0
volatility      <- "low"
intimacy_signals <- 0
aggression_count <- 0

# ── load profile CSV ──────────────────────────────────────────────────────────
if (length(args) >= 2 && file.exists(args[2])) {
  tryCatch({
    prof <- read.csv(args[2], stringsAsFactors = FALSE)
    if (all(c("intent", "count") %in% names(prof))) {
      counts       <- setNames(as.numeric(prof$count), prof$intent)
      total        <- sum(counts, na.rm = TRUE)
      aggr         <- ifelse("aggression" %in% names(counts), counts["aggression"], 0)
      trust_cnt    <- ifelse("trust"      %in% names(counts), counts["trust"],      0)
      withdr       <- ifelse("withdrawal" %in% names(counts), counts["withdrawal"], 0)
      need_cnt     <- ifelse("need"       %in% names(counts), counts["need"],       0)
      phil_cnt     <- ifelse("philosophical" %in% names(counts), counts["philosophical"], 0)

      aggression_count <- aggr
      trust_level      <- min(1.0, trust_cnt / max(total, 1) * 2)

      # dominant intent
      if (total > 0) {
        dominant_intent <- names(which.max(counts))
      }

      # player type classification
      if (total >= 4) {
        aggr_ratio  <- aggr  / total
        trust_ratio <- trust_cnt / total
        withdr_ratio <- withdr / total
        phil_ratio  <- phil_cnt / total

        if (aggr_ratio > 0.4) {
          player_type <- "aggressive"
        } else if (trust_ratio > 0.5) {
          player_type <- "trusting"
        } else if (withdr_ratio > 0.3) {
          player_type <- "avoidant"
        } else if (phil_ratio > 0.3) {
          player_type <- "philosophical"
        } else {
          player_type <- "mixed"
        }
      }

      # threat level (0-10)
      threat_level <- min(10, round(aggr * 1.5 + withdr * 0.5))

      # manipulation detection: high aggression + trust mix = manipulation
      if (total >= 6 && aggr >= 2 && trust_cnt >= 2) {
        manipulation <- TRUE
        plan         <- "mirror"
        confidence   <- 0.8
      } else if (total >= 6 && aggr >= 3) {
        plan         <- "confront"
        threat_level <- min(10, threat_level + 2)
        confidence   <- 0.75
      } else if (total >= 4 && trust_cnt >= 3) {
        plan         <- "seduce"
        confidence   <- 0.7
      } else if (total >= 4 && withdr >= 2) {
        plan         <- "seduce"
        confidence   <- 0.65
      } else if (total >= 6 && phil_cnt >= 2) {
        plan         <- "philosophical"
        confidence   <- 0.6
      }

      # emotional drift
      if (aggr > withdr + trust_cnt) {
        emotional_drift <- "escalating"
      } else if (trust_cnt > aggr + withdr) {
        emotional_drift <- "calming"
      } else if (withdr > aggr + trust_cnt) {
        emotional_drift <- "retreating"
      }

      # volatility
      if (aggr >= 3 && trust_cnt >= 2) {
        volatility <- "high"
      } else if (total >= 6) {
        volatility <- "medium"
      }

      intimacy_signals <- as.integer(trust_cnt + need_cnt)
    }
  }, error = function(e) {})
}

# ── output ────────────────────────────────────────────────────────────────────
cat(sprintf(
  "plan=%s|player_type=%s|threat_level=%d|emotional_drift=%s|manipulation=%s|confidence=%.2f|dominant_intent=%s|trust_level=%.2f|volatility=%s|intimacy_signals=%d|aggression_count=%d\n",
  plan, player_type, as.integer(threat_level), emotional_drift,
  tolower(as.character(manipulation)), confidence, dominant_intent,
  trust_level, volatility, as.integer(intimacy_signals), as.integer(aggression_count)
))
