-- Chaos2 — regulation module
-- Haskell: calibrates response intensity from strategy + threat
-- Input (stdin): "strategy threat_level"
-- Output (stdout): intensity as float 0.0–1.0

module Main where

import System.IO (hSetBuffering, stdin, stdout, BufferMode(..))

strategyBase :: String -> Double
strategyBase s = case s of
    "observe"            -> 0.3
    "mirror"             -> 0.4
    "resist"             -> 0.6
    "collapse"           -> 0.9
    "reveal"             -> 0.5
    "test_loyalty"       -> 0.45
    "existential_crisis" -> 0.8
    "seduce"             -> 0.55
    "destabilize"        -> 0.85
    "investigate_player" -> 0.35
    _                    -> 0.3

threatMultiplier :: Double -> Double
threatMultiplier t = 1.0 + (t / 10.0) * 0.4

clamp :: Double -> Double
clamp x = max 0.0 (min 1.0 x)

regulate :: String -> Double -> Double
regulate strategy threat =
    clamp $ strategyBase strategy * threatMultiplier threat

main :: IO ()
main = do
    hSetBuffering stdin  LineBuffering
    hSetBuffering stdout LineBuffering
    contents <- getContents
    mapM_ process (lines contents)
  where
    process line = case words line of
        [strategy, threatStr] ->
            case reads threatStr of
                [(threat, "")] -> putStrLn $ show (regulate strategy threat)
                _              -> putStrLn "0.3"
        _ -> putStrLn "0.3"
