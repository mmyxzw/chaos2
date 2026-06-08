// Chaos2 — instinct classifier
// Replaces the Python TF-IDF + LogisticRegression classifier.
//
// Build:   make -C instinct/
// Input:   stdin, one message per line — format "context || message" or just "message"
// Output:  stdout, one intent per line
//          classes: aggression | curiosity | intimacy | philosophical |
//                   provocation | trust | withdrawal
//
// Generate model_weights.h first:
//   cd /path/to/chaos1 && python3 /path/to/chaos2/instinct/export_weights.py

#include <algorithm>
#include <array>
#include <cmath>
#include <iostream>
#include <string>
#include <unordered_map>
#include <vector>

#include "model_weights.h"

// ─── tokenizer ──────────────────────────────────────────────────────────────

static std::vector<std::string> tokenize(const std::string& text) {
    std::vector<std::string> tokens;
    std::string tok;
    for (unsigned char c : text) {
        if (std::isalnum(c)) {
            tok += static_cast<char>(std::tolower(c));
        } else if (!tok.empty()) {
            tokens.push_back(std::move(tok));
            tok.clear();
        }
    }
    if (!tok.empty()) tokens.push_back(std::move(tok));
    return tokens;
}

// ─── TF-IDF + logistic regression ───────────────────────────────────────────

static std::string classify(const std::string& input) {
    auto tokens = tokenize(input);
    if (tokens.empty()) return "curiosity";

    // term frequencies
    std::unordered_map<int, double> tf;
    for (const auto& t : tokens) {
        auto it = VOCABULARY.find(t);
        if (it != VOCABULARY.end()) tf[it->second] += 1.0;
    }

    // TF-IDF dense vector + L2 norm accumulator
    std::array<double, N_FEATURES> feat{};
    double norm = 0.0;
    for (auto& [idx, count] : tf) {
        double val = count * IDF[idx];
        feat[idx]  = val;
        norm      += val * val;
    }

    if (norm > 0.0) {
        norm = std::sqrt(norm);
        for (auto& v : feat) v /= norm;
    }

    // argmax(COEF[c] · feat + INTERCEPT[c])
    int    best  = 0;
    double score = -1e18;
    for (int c = 0; c < N_CLASSES; ++c) {
        double s = INTERCEPT[c];
        for (int i = 0; i < N_FEATURES; ++i) s += COEF[c][i] * feat[i];
        if (s > score) { score = s; best = c; }
    }

    return CLASS_NAMES[best];
}

// ─── main: persistent process, one message per line ─────────────────────────

int main() {
    std::ios::sync_with_stdio(false);
    std::cin.tie(nullptr);

    std::string line;
    while (std::getline(std::cin, line)) {
        if (line.empty()) continue;
        std::cout << classify(line) << '\n';
        std::cout.flush();
    }
    return 0;
}
