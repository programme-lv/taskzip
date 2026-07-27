use std::io::{self, Write};
use std::time::Duration;

pub enum Event {
    Step { depth: usize, message: String },
    Detail { depth: usize, message: String },
}

impl Event {
    pub fn step(depth: usize, message: impl Into<String>) -> Self {
        Self::Step {
            depth,
            message: message.into(),
        }
    }

    pub fn detail(depth: usize, message: impl Into<String>) -> Self {
        Self::Detail {
            depth,
            message: message.into(),
        }
    }
}

pub fn print(event: Event) {
    let (depth, message) = match event {
        Event::Step { depth, message } | Event::Detail { depth, message } => (depth, message),
    };
    println!("{:width$}{message}", "", width = (depth + 1) * 2);
    let _ = io::stdout().flush();
}

pub fn bytes(bytes: u64) -> String {
    const UNITS: [&str; 4] = ["B", "KiB", "MiB", "GiB"];
    let mut value = bytes as f64;
    let mut unit = 0;
    while value >= 1024.0 && unit < UNITS.len() - 1 {
        value /= 1024.0;
        unit += 1;
    }
    if unit == 0 {
        format!("{bytes} B")
    } else {
        format!("{value:.1} {}", UNITS[unit])
    }
}

pub fn duration(duration: Duration) -> String {
    if duration.as_secs() > 0 {
        format!("{:.1}s", duration.as_secs_f64())
    } else {
        format!("{}ms", duration.as_millis())
    }
}
