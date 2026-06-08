use std::time::{SystemTime, UNIX_EPOCH};

const LAMBDA: f64 = 0.1;
const MIN_STRENGTH: f64 = 0.05;

struct Memory {
    content: String,
    strength: f64,
    last_seen: u64,
}

fn decay(strength: f64, hours_elapsed: f64) -> f64 {
    strength * (-LAMBDA * hours_elapsed).exp()
}

fn hours_since(timestamp: u64, now: u64) -> f64 {
    now.saturating_sub(timestamp) as f64 / 3600.0
}

fn main() {
    let now = SystemTime::now()
        .duration_since(UNIX_EPOCH)
        .unwrap()
        .as_secs();

    let stdin = std::io::stdin();
    let mut surviving: Vec<Memory> = Vec::new();
    let mut forgotten = 0usize;

    for line in std::io::BufRead::lines(stdin.lock()) {
        let line = match line {
            Ok(l) if !l.trim().is_empty() => l,
            _ => continue,
        };
        let parts: Vec<&str> = line.splitn(3, '|').collect();
        if parts.len() != 3 { continue; }

        let timestamp: u64 = parts[0].parse().unwrap_or(0);
        let strength: f64  = parts[1].parse().unwrap_or(0.5);
        let content        = parts[2].to_string();
        let hours          = hours_since(timestamp, now);
        let new_strength   = decay(strength, hours);

        if new_strength >= MIN_STRENGTH {
            surviving.push(Memory { content, strength: new_strength, last_seen: timestamp });
        } else {
            forgotten += 1;
        }
    }

    for m in &surviving {
        println!("{}|{:.4}|{}", m.last_seen, m.strength, m.content);
    }
    eprintln!("[forgetting] {} survived, {} forgotten", surviving.len(), forgotten);
}
