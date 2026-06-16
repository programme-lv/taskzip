use anyhow::{bail, Context, Result};
use serde::Deserialize;
use std::collections::{BTreeMap, HashMap};

#[derive(Debug, Clone)]
pub struct TaskMeta {
    pub taskzip: u32,
    pub id: String,
    pub name: HashMap<String, String>,
    pub testing: Testing,
    pub scoring: Scoring,
    pub solutions: Vec<Solution>,
    pub attached: Vec<Attached>,
    pub subtasks: Vec<Subtask>,
    pub groups: Vec<Group>,
    pub origin: Option<Origin>,
    pub metadata: Option<Metadata>,
    pub extensions: BTreeMap<String, toml::Table>,
}

#[derive(Debug, Clone, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Testing {
    #[serde(rename = "type")]
    pub kind: String,
    pub cpu_ms: u32,
    pub mem_mib: u32,
}

#[derive(Debug, Clone, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Scoring {
    #[serde(rename = "type")]
    pub kind: String,
    pub total: u32,
}

#[derive(Debug, Clone, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Solution {
    pub fname: String,
    #[serde(default)]
    pub subtasks: Vec<u32>,
    pub score: Option<u32>,
}

#[derive(Debug, Clone, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Attached {
    pub path: String,
}

#[derive(Debug, Clone, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Subtask {
    pub points: u32,
    #[serde(default)]
    pub vis_input: bool,
    pub description: Option<HashMap<String, String>>,
}

#[derive(Debug, Clone, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Group {
    pub id: u32,
    pub tests: String,
    pub points: u32,
    pub mode: String,
    pub subtask: Option<u32>,
    #[serde(default)]
    pub requires: Vec<u32>,
}

#[derive(Debug, Clone, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Origin {
    pub olymp: Option<String>,
    pub year: Option<i32>,
    pub stage: Option<String>,
    pub org: Option<String>,
    #[serde(default)]
    pub authors: Vec<String>,
    pub lang: Option<String>,
    pub contestants: Option<u32>,
    pub solvers: Option<u32>,
}

#[derive(Debug, Clone, Deserialize)]
#[serde(deny_unknown_fields)]
pub struct Metadata {
    #[serde(default)]
    pub topics: Vec<String>,
    #[serde(default)]
    pub techniques: Vec<String>,
    #[serde(default)]
    pub data_structures: Vec<String>,
    pub difficulty: Option<u8>,
}

pub fn parse(content: &str) -> Result<TaskMeta> {
    let root: toml::Table = toml::from_str(content).context("parse task.toml")?;
    if root.get("taskzip").and_then(|v| v.as_integer()) != Some(1) {
        bail!("taskzip must be 1");
    }
    let extensions = ext_tables(&root);
    let core = strip_ext(&root);
    let raw = toml::Value::Table(core);
    let body: CoreBody = raw.try_into().context("task.toml core fields")?;
    Ok(TaskMeta {
        taskzip: body.taskzip,
        id: body.id,
        name: body.name,
        testing: body.testing,
        scoring: body.scoring,
        solutions: body.solutions.unwrap_or_default(),
        attached: body.attached.unwrap_or_default(),
        subtasks: body.subtasks.unwrap_or_default(),
        groups: body.groups.unwrap_or_default(),
        origin: body.origin,
        metadata: body.metadata,
        extensions,
    })
}

#[derive(Debug, Deserialize)]
#[serde(deny_unknown_fields)]
struct CoreBody {
    taskzip: u32,
    id: String,
    name: HashMap<String, String>,
    testing: Testing,
    scoring: Scoring,
    solutions: Option<Vec<Solution>>,
    attached: Option<Vec<Attached>>,
    subtasks: Option<Vec<Subtask>>,
    groups: Option<Vec<Group>>,
    origin: Option<Origin>,
    metadata: Option<Metadata>,
}

fn ext_tables(root: &toml::Table) -> BTreeMap<String, toml::Table> {
    let mut out = BTreeMap::new();
    for (k, v) in root {
        if let Some(name) = k.strip_prefix("ext.") {
            if let toml::Value::Table(t) = v {
                out.insert(name.to_string(), t.clone());
            }
        }
    }
    out
}

fn strip_ext(root: &toml::Table) -> toml::Table {
    root.iter()
        .filter(|(k, _)| !k.starts_with("ext."))
        .map(|(k, v)| (k.clone(), v.clone()))
        .collect()
}
