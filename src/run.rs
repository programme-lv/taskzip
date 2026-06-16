use anyhow::{bail, Context, Result};
use std::fs;
use std::io::Read;
use std::path::{Path, PathBuf};
use std::process::{Command, ExitStatus, Stdio};
use std::thread;
use std::time::Duration;
use wait_timeout::ChildExt;

const TIME_BIN: &str = "/usr/bin/time";

#[derive(Clone, Copy)]
pub struct Limits {
    pub wall: Duration,
    pub cpu: Option<Duration>,
}

pub struct Output {
    pub status: ExitStatus,
    pub stdout: Vec<u8>,
    pub stderr: Vec<u8>,
    pub cpu: Duration,
    pub timed_out: bool,
}

pub fn wall_for_cpu(cpu_ms: u32) -> Duration {
    Duration::from_millis((cpu_ms as u64).saturating_mul(3).max(5000))
}

pub fn run(
    program: &Path,
    args: &[&str],
    stdin: Option<PathBuf>,
    limits: Limits,
) -> Result<Output> {
    let time_file = tempfile::NamedTempFile::new().context("time temp")?;
    let time_path = time_file.path().to_path_buf();
    let mut cmd = Command::new(TIME_BIN);
    cmd.arg("-f")
        .arg("%U %S %x")
        .arg("-o")
        .arg(&time_path)
        .arg(program)
        .args(args);
    match stdin {
        Some(path) => {
            cmd.stdin(fs::File::open(&path).with_context(|| format!("open {}", path.display()))?);
        }
        None => {
            cmd.stdin(Stdio::null());
        }
    }
    cmd.stdout(Stdio::piped()).stderr(Stdio::piped());
    let mut child = cmd.spawn().with_context(|| format!("run {}", program.display()))?;
    let out_pipe = child.stdout.take();
    let err_pipe = child.stderr.take();
    let out_handle = out_pipe.map(|mut p| thread::spawn(move || read_pipe(&mut p)));
    let err_handle = err_pipe.map(|mut p| thread::spawn(move || read_pipe(&mut p)));
    match child.wait_timeout(limits.wall).context("wait")? {
        Some(status) => {
            let stdout = join_pipe(out_handle);
            let stderr = join_pipe(err_handle);
            let cpu = read_cpu(&time_path)?;
            Ok(Output {
                status,
                stdout,
                stderr,
                cpu,
                timed_out: false,
            })
        }
        None => {
            child.kill().ok();
            let status = child.wait().context("wait after kill")?;
            Ok(Output {
                status,
                stdout: join_pipe(out_handle),
                stderr: join_pipe(err_handle),
                cpu: Duration::ZERO,
                timed_out: true,
            })
        }
    }
}

pub fn cpu_exceeded(cpu: Duration, limit: Duration) -> bool {
    cpu > limit
}

pub fn ensure_time() -> Result<()> {
    if Path::new(TIME_BIN).is_file() {
        Ok(())
    } else {
        bail!("{TIME_BIN} missing");
    }
}

fn read_pipe(pipe: &mut impl Read) -> Vec<u8> {
    let mut buf = Vec::new();
    pipe.read_to_end(&mut buf).ok();
    buf
}

fn join_pipe(handle: Option<thread::JoinHandle<Vec<u8>>>) -> Vec<u8> {
    handle.map(|h| h.join().unwrap_or_default()).unwrap_or_default()
}

fn read_cpu(path: &Path) -> Result<Duration> {
    let text = fs::read_to_string(path).context("read time output")?;
    let mut parts = text.split_whitespace();
    let user: f64 = parts
        .next()
        .context("time user")?
        .parse()
        .context("time user parse")?;
    let sys: f64 = parts
        .next()
        .context("time sys")?
        .parse()
        .context("time sys parse")?;
    Ok(Duration::from_secs_f64(user + sys))
}
