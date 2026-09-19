use anyhow::{Context, Result};
use serde::{Deserialize, Serialize};
use std::collections::BTreeMap;
use std::fs;
use std::io::Write;
use std::path::PathBuf;

/// On-disk configuration stored at `$XDG_CONFIG_HOME/taskzip/config.toml`
/// (falling back to `~/.config/taskzip/config.toml`).
#[derive(Debug, Default, Deserialize, Serialize)]
pub struct Config {
    /// Default AI provider used by commands that call a model.
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub provider: Option<String>,
    /// Per-provider settings keyed by the provider id (`openai`, `deepseek`).
    #[serde(default, skip_serializing_if = "BTreeMap::is_empty")]
    pub providers: BTreeMap<String, ProviderConfig>,
}

#[derive(Debug, Default, Clone, Deserialize, Serialize)]
pub struct ProviderConfig {
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub api_key: Option<String>,
    #[serde(default, skip_serializing_if = "Option::is_none")]
    pub model: Option<String>,
}

impl Config {
    pub fn provider_config(&self, id: &str) -> Option<&ProviderConfig> {
        self.providers.get(id)
    }
}

pub fn config_path() -> Result<PathBuf> {
    let base = std::env::var_os("XDG_CONFIG_HOME")
        .filter(|v| !v.is_empty())
        .map(PathBuf::from)
        .or_else(|| std::env::var_os("HOME").map(|h| PathBuf::from(h).join(".config")))
        .ok_or_else(|| anyhow::anyhow!("no config home"))?;
    Ok(base.join("taskzip").join("config.toml"))
}

pub fn load() -> Result<Config> {
    let path = config_path()?;
    if !path.is_file() {
        return Ok(Config::default());
    }
    let body = fs::read_to_string(&path).with_context(|| format!("read {}", path.display()))?;
    toml::from_str(&body).with_context(|| format!("parse {}", path.display()))
}

pub fn save(config: &Config) -> Result<PathBuf> {
    let path = config_path()?;
    let dir = path
        .parent()
        .ok_or_else(|| anyhow::anyhow!("invalid config path {}", path.display()))?;
    fs::create_dir_all(dir).with_context(|| format!("create {}", dir.display()))?;
    let body = toml::to_string_pretty(config).context("serialize config")?;
    let mut temp = tempfile::NamedTempFile::new_in(dir)?;
    temp.write_all(body.as_bytes())?;
    set_private_permissions(temp.as_file())?;
    temp.persist(&path)
        .with_context(|| format!("write {}", path.display()))?;
    Ok(path)
}

#[cfg(unix)]
fn set_private_permissions(file: &fs::File) -> Result<()> {
    use std::os::unix::fs::PermissionsExt;
    file.set_permissions(fs::Permissions::from_mode(0o600))
        .context("set config permissions")
}

#[cfg(not(unix))]
fn set_private_permissions(_file: &fs::File) -> Result<()> {
    Ok(())
}
