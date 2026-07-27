use anyhow::{bail, Context, Result};
use reqwest::blocking::Client;
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};
use std::fs;
use std::io::Write;
use std::ops::AddAssign;
use std::path::{Path, PathBuf};
use std::time::Duration;
use std::time::Instant;

const API_URL: &str = "https://api.openai.com/v1/chat/completions";
const DEFAULT_MODEL: &str = "gpt-5.6-luna";
const SYSTEM: &str = include_str!("../prompts/lio2024/system.md");
const FORMAT: &str = include_str!("../prompts/lio2024/format.md");
const STORY: &str = include_str!("../prompts/lio2024/story.md");
const INPUT: &str = include_str!("../prompts/lio2024/input.md");
const OUTPUT: &str = include_str!("../prompts/lio2024/output.md");
const SUBTASKS: &str = include_str!("../prompts/lio2024/subtasks.md");
const METADATA: &str = include_str!("../prompts/lio2024/metadata.md");
const SOLUTION: &str = include_str!("../prompts/lio2024/solution.md");

pub struct StatementParts {
    pub story: String,
    pub input: String,
    pub output: String,
    pub subtasks: Vec<String>,
    pub metadata: TaskMetadata,
    pub solutions: Vec<SolutionEstimate>,
    pub usage: TokenUsage,
}

#[derive(Deserialize)]
#[serde(deny_unknown_fields)]
pub struct TaskMetadata {
    pub topics: Vec<String>,
    pub techniques: Vec<String>,
    pub data_structures: Vec<String>,
    pub difficulty: u8,
}

pub struct SolutionEstimate {
    pub fname: String,
    pub subtasks: Vec<u32>,
}

#[derive(Clone, Copy, Default, Deserialize, Serialize)]
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
    Start(String),
    Done {
        part: String,
        usage: TokenUsage,
        elapsed: Duration,
        cached: bool,
    },
}

pub fn import_statement(
    typ_source: &str,
    images: &[String],
    subtask_count: usize,
    cpu_ms: u32,
    solutions: &[(String, String)],
    mut on_progress: impl FnMut(StatementEvent),
) -> Result<StatementParts> {
    let system = render_system(typ_source, images)?;
    let mut chat = Chat::new(system)?;
    on_progress(StatementEvent::Model(chat.model.clone()));
    let mut usage = TokenUsage::default();
    let story = ask_counted(&mut chat, "story", STORY, &mut usage, &mut on_progress)?;
    let input = ask_counted(&mut chat, "input", INPUT, &mut usage, &mut on_progress)?;
    let output = ask_counted(&mut chat, "output", OUTPUT, &mut usage, &mut on_progress)?;
    let subtasks_prompt = render_subtasks(subtask_count)?;
    let subtasks_raw = ask_counted(
        &mut chat,
        "subtasks",
        &subtasks_prompt,
        &mut usage,
        &mut on_progress,
    )?;
    let subtasks = parse_subtasks(&subtasks_raw, subtask_count)?;
    let metadata = ask_metadata(&mut chat, &mut usage, &mut on_progress)?;
    let (solutions, solution_usage) =
        estimate_solutions(&mut chat, solutions, &subtasks, cpu_ms, &mut on_progress)?;
    usage += solution_usage;
    Ok(StatementParts {
        story,
        input,
        output,
        subtasks,
        metadata,
        solutions,
        usage,
    })
}

fn ask_metadata(
    chat: &mut Chat,
    usage: &mut TokenUsage,
    on_progress: &mut impl FnMut(StatementEvent),
) -> Result<TaskMetadata> {
    let raw = ask_counted(chat, "metadata", METADATA, usage, on_progress)?;
    parse_metadata(&raw)
}

fn ask_counted(
    chat: &mut Chat,
    part: &str,
    prompt: &str,
    usage: &mut TokenUsage,
    on_progress: &mut impl FnMut(StatementEvent),
) -> Result<String> {
    let (content, used) = ask_part(chat, part, prompt, on_progress)?;
    *usage += used;
    Ok(content)
}

fn ask_part(
    chat: &mut Chat,
    part: &str,
    prompt: &str,
    on_progress: &mut impl FnMut(StatementEvent),
) -> Result<(String, TokenUsage)> {
    on_progress(StatementEvent::Start(part.to_string()));
    let start = Instant::now();
    let (content, usage, cached) = chat.ask(prompt)?;
    on_progress(StatementEvent::Done {
        part: part.to_string(),
        usage,
        elapsed: start.elapsed(),
        cached,
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

    fn ask(&mut self, prompt: &str) -> Result<(String, TokenUsage, bool)> {
        self.messages.push(Message::new("user", prompt.trim()));
        let request = ChatRequest {
            model: &self.model,
            messages: &self.messages,
        };
        let cache_path = response_cache_path(&request)?;
        if let Some(response) = read_cached_response(&cache_path)? {
            self.messages
                .push(Message::new("assistant", &response.content));
            return Ok((response.content, response.usage, true));
        }
        let response = self
            .client
            .post(API_URL)
            .bearer_auth(&self.api_key)
            .json(&request)
            .send()
            .context("OpenAI request")?;
        let response = parse_response(response)?;
        write_cached_response(&cache_path, &response)?;
        self.messages
            .push(Message::new("assistant", &response.content));
        Ok((response.content, response.usage, false))
    }

    fn reset(&mut self) {
        self.messages.truncate(1);
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

#[derive(Deserialize, Serialize)]
struct CachedResponse {
    content: String,
    usage: TokenUsage,
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

fn parse_response(response: reqwest::blocking::Response) -> Result<CachedResponse> {
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
    Ok(CachedResponse { content, usage })
}

fn response_cache_path(request: &ChatRequest<'_>) -> Result<PathBuf> {
    let body = serde_json::to_vec(request)?;
    let mut hash = Sha256::new();
    hash.update(API_URL);
    hash.update([0]);
    hash.update(body);
    Ok(user_cache_root()?.join(format!("{:x}.json", hash.finalize())))
}

fn user_cache_root() -> Result<PathBuf> {
    let base = std::env::var_os("XDG_CACHE_HOME")
        .filter(|v| !v.is_empty())
        .map(PathBuf::from)
        .or_else(|| std::env::var_os("HOME").map(|h| PathBuf::from(h).join(".cache")))
        .ok_or_else(|| anyhow::anyhow!("no cache home"))?;
    Ok(base.join("taskzip").join("openai"))
}

fn read_cached_response(path: &Path) -> Result<Option<CachedResponse>> {
    if !path.is_file() {
        return Ok(None);
    }
    let body =
        fs::read_to_string(path).with_context(|| format!("read AI cache {}", path.display()))?;
    let response = serde_json::from_str(&body)
        .with_context(|| format!("parse AI cache {}", path.display()))?;
    Ok(Some(response))
}

fn write_cached_response(path: &Path, response: &CachedResponse) -> Result<()> {
    let dir = path.parent().unwrap();
    fs::create_dir_all(dir).with_context(|| format!("create AI cache {}", dir.display()))?;
    let mut temp = tempfile::NamedTempFile::new_in(dir)?;
    temp.write_all(&serde_json::to_vec(response)?)?;
    temp.persist(path)
        .with_context(|| format!("write AI cache {}", path.display()))?;
    Ok(())
}

fn parse_subtasks(raw: &str, expected: usize) -> Result<Vec<String>> {
    let text = strip_json_fence(raw);
    let items: Vec<String> =
        serde_json::from_str(text).context("parse subtask descriptions JSON")?;
    if items.len() != expected {
        bail!(
            "subtask description count {}, expected {}",
            items.len(),
            expected
        );
    }
    if items.iter().any(|s| s.trim().is_empty()) {
        bail!("empty subtask description");
    }
    Ok(items.into_iter().map(|s| s.trim().to_string()).collect())
}

fn parse_metadata(raw: &str) -> Result<TaskMetadata> {
    let metadata: TaskMetadata =
        serde_json::from_str(strip_json_fence(raw)).context("parse task metadata JSON")?;
    if !(1..=5).contains(&metadata.difficulty) {
        bail!("task difficulty out of range");
    }
    if metadata.topics.is_empty()
        || metadata.topics.len() > 2
        || metadata.techniques.len() > 4
        || metadata.data_structures.len() > 3
    {
        bail!("too many or no classification tags");
    }
    Ok(metadata)
}

fn estimate_solutions(
    chat: &mut Chat,
    sources: &[(String, String)],
    subtasks: &[String],
    cpu_ms: u32,
    on_progress: &mut impl FnMut(StatementEvent),
) -> Result<(Vec<SolutionEstimate>, TokenUsage)> {
    let mut estimates = Vec::new();
    let mut usage = TokenUsage::default();
    for (fname, source) in sources {
        chat.reset();
        let prompt = render_solution(fname, source, subtasks, cpu_ms)?;
        let part = format!("solution {fname}");
        let (raw, used) = ask_part(chat, &part, &prompt, on_progress)?;
        usage += used;
        estimates.push(SolutionEstimate {
            fname: fname.clone(),
            subtasks: parse_solution_subtasks(&raw, subtasks.len(), fname)?,
        });
    }
    Ok((estimates, usage))
}

fn parse_solution_subtasks(raw: &str, count: usize, fname: &str) -> Result<Vec<u32>> {
    let mut ids: Vec<u32> =
        serde_json::from_str(strip_json_fence(raw)).context("parse solution subtasks JSON")?;
    ids.sort_unstable();
    if ids.iter().any(|id| *id == 0 || *id as usize > count) {
        bail!("{fname}: solution subtask out of range");
    }
    if ids.windows(2).any(|pair| pair[0] == pair[1]) {
        bail!("{fname}: duplicate solution subtask");
    }
    Ok(ids)
}

fn strip_json_fence(raw: &str) -> &str {
    let trimmed = raw.trim();
    let Some(rest) = trimmed.strip_prefix("```") else {
        return trimmed;
    };
    let rest = rest
        .strip_prefix("json")
        .or_else(|| rest.strip_prefix("JSON"))
        .unwrap_or(rest)
        .trim_start();
    rest.strip_suffix("```").map(str::trim).unwrap_or(trimmed)
}

fn render_system(typ_source: &str, images: &[String]) -> Result<String> {
    let values = [
        ("format_rules", FORMAT.trim()),
        ("typ_source", typ_source),
        ("image_names", &image_list(images)),
    ];
    render_template(SYSTEM, &values)
}

fn render_subtasks(count: usize) -> Result<String> {
    let count = count.to_string();
    render_template(SUBTASKS, &[("count", &count)])
}

fn render_solution(fname: &str, source: &str, subtasks: &[String], cpu_ms: u32) -> Result<String> {
    let cpu_ms = cpu_ms.to_string();
    let subtasks = subtasks
        .iter()
        .enumerate()
        .map(|(i, text)| format!("{}. {text}", i + 1))
        .collect::<Vec<_>>()
        .join("\n");
    render_template(
        SOLUTION,
        &[
            ("fname", fname),
            ("source", source),
            ("subtasks", &subtasks),
            ("cpu_ms", &cpu_ms),
        ],
    )
}

fn render_template(template: &str, values: &[(&str, &str)]) -> Result<String> {
    let mut check = template.to_string();
    for (name, _) in values {
        let marker = format!("{{{{{name}}}}}");
        if check.matches(&marker).count() != 1 {
            bail!("prompt placeholder {marker}");
        }
        check = check.replace(&marker, "");
    }
    if check.contains("{{") || check.contains("}}") {
        bail!("unknown prompt placeholder");
    }
    let mut out = template.to_string();
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
