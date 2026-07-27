use anyhow::{bail, Context, Result};
use reqwest::blocking::Client;
use serde::{Deserialize, Serialize};
use std::ops::AddAssign;
use std::time::Duration;
use std::time::Instant;

const API_URL: &str = "https://api.openai.com/v1/chat/completions";
const DEFAULT_MODEL: &str = "gpt-5.6-luna";
const SYSTEM: &str = include_str!("../prompts/lio2024/system.md");
const FORMAT: &str = include_str!("../prompts/lio2024/format.md");
const STORY: &str = include_str!("../prompts/lio2024/story.md");
const INPUT: &str = include_str!("../prompts/lio2024/input.md");
const OUTPUT: &str = include_str!("../prompts/lio2024/output.md");

pub struct StatementParts {
    pub story: String,
    pub input: String,
    pub output: String,
    pub usage: TokenUsage,
}

#[derive(Clone, Copy, Default)]
pub struct TokenUsage {
    pub input: u64,
    pub output: u64,
}

impl AddAssign for TokenUsage {
    fn add_assign(&mut self, rhs: Self) {
        self.input += rhs.input;
        self.output += rhs.output;
    }
}

pub enum StatementEvent {
    Model(String),
    Start(&'static str),
    Done {
        part: &'static str,
        usage: TokenUsage,
        elapsed: Duration,
    },
}

pub fn import_statement(
    typ_source: &str,
    images: &[String],
    mut on_progress: impl FnMut(StatementEvent),
) -> Result<StatementParts> {
    let system = render_system(typ_source, images)?;
    let mut chat = Chat::new(system)?;
    on_progress(StatementEvent::Model(chat.model.clone()));
    let (story, story_usage) = ask_part(&mut chat, "story", STORY, &mut on_progress)?;
    let (input, input_usage) = ask_part(&mut chat, "input", INPUT, &mut on_progress)?;
    let (output, output_usage) = ask_part(&mut chat, "output", OUTPUT, &mut on_progress)?;
    let mut usage = story_usage;
    usage += input_usage;
    usage += output_usage;
    Ok(StatementParts {
        story,
        input,
        output,
        usage,
    })
}

fn ask_part(
    chat: &mut Chat,
    part: &'static str,
    prompt: &str,
    on_progress: &mut impl FnMut(StatementEvent),
) -> Result<(String, TokenUsage)> {
    on_progress(StatementEvent::Start(part));
    let start = Instant::now();
    let (content, usage) = chat.ask(prompt)?;
    on_progress(StatementEvent::Done {
        part,
        usage,
        elapsed: start.elapsed(),
    });
    Ok((content, usage))
}

struct Chat {
    client: Client,
    api_key: String,
    model: String,
    messages: Vec<Message>,
}

impl Chat {
    fn new(system: String) -> Result<Self> {
        let _ = dotenvy::dotenv();
        let api_key = env("OPENAI_API_KEY")?;
        let model = std::env::var("OPENAI_MODEL")
            .ok()
            .filter(|v| !v.trim().is_empty())
            .unwrap_or_else(|| DEFAULT_MODEL.into());
        let client = Client::builder()
            .timeout(Duration::from_secs(120))
            .build()
            .context("build OpenAI client")?;
        Ok(Self {
            client,
            api_key,
            model,
            messages: vec![Message::new("system", system)],
        })
    }

    fn ask(&mut self, prompt: &str) -> Result<(String, TokenUsage)> {
        self.messages.push(Message::new("user", prompt.trim()));
        let request = ChatRequest {
            model: &self.model,
            messages: &self.messages,
        };
        let response = self
            .client
            .post(API_URL)
            .bearer_auth(&self.api_key)
            .json(&request)
            .send()
            .context("OpenAI request")?;
        let (content, usage) = parse_response(response)?;
        self.messages.push(Message::new("assistant", &content));
        Ok((content, usage))
    }
}

#[derive(Clone, Serialize)]
struct Message {
    role: String,
    content: String,
}

impl Message {
    fn new(role: &str, content: impl Into<String>) -> Self {
        Self {
            role: role.into(),
            content: content.into(),
        }
    }
}

#[derive(Serialize)]
struct ChatRequest<'a> {
    model: &'a str,
    messages: &'a [Message],
}

#[derive(Deserialize)]
struct ChatResponse {
    choices: Vec<Choice>,
    usage: ApiUsage,
}

#[derive(Deserialize)]
struct ApiUsage {
    prompt_tokens: u64,
    completion_tokens: u64,
}

#[derive(Deserialize)]
struct Choice {
    message: ResponseMessage,
}

#[derive(Deserialize)]
struct ResponseMessage {
    content: String,
}

#[derive(Deserialize)]
struct ErrorResponse {
    error: ApiError,
}

#[derive(Deserialize)]
struct ApiError {
    message: String,
}

fn parse_response(response: reqwest::blocking::Response) -> Result<(String, TokenUsage)> {
    let status = response.status();
    let body = response.text().context("read OpenAI response")?;
    if !status.is_success() {
        let detail = serde_json::from_str::<ErrorResponse>(&body)
            .map(|e| e.error.message)
            .unwrap_or_else(|_| body.trim().to_string());
        bail!("OpenAI HTTP {status}: {detail}");
    }
    let response: ChatResponse = serde_json::from_str(&body).context("parse OpenAI response")?;
    let content = response
        .choices
        .into_iter()
        .next()
        .map(|c| c.message.content.trim().to_string())
        .unwrap_or_default();
    if content.is_empty() {
        bail!("empty OpenAI response");
    }
    let usage = TokenUsage {
        input: response.usage.prompt_tokens,
        output: response.usage.completion_tokens,
    };
    Ok((content, usage))
}

fn render_system(typ_source: &str, images: &[String]) -> Result<String> {
    let values = [
        ("format_rules", FORMAT.trim()),
        ("typ_source", typ_source),
        ("image_names", &image_list(images)),
    ];
    let mut check = SYSTEM.to_string();
    for (name, _) in &values {
        let marker = format!("{{{{{name}}}}}");
        if check.matches(&marker).count() != 1 {
            bail!("prompt placeholder {marker}");
        }
        check = check.replace(&marker, "");
    }
    if check.contains("{{") || check.contains("}}") {
        bail!("unknown prompt placeholder");
    }
    let mut out = SYSTEM.to_string();
    for (name, value) in values {
        out = out.replace(&format!("{{{{{name}}}}}"), value);
    }
    Ok(out)
}

fn image_list(images: &[String]) -> String {
    if images.is_empty() {
        return "(none)".into();
    }
    images
        .iter()
        .map(|name| format!("- {name}"))
        .collect::<Vec<_>>()
        .join("\n")
}

fn env(name: &str) -> Result<String> {
    let value = std::env::var(name).with_context(|| format!("{name} missing"))?;
    if value.trim().is_empty() {
        bail!("{name} empty");
    }
    Ok(value)
}
