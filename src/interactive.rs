use crate::run::Limits;
use anyhow::{bail, Context, Result};
use std::fs::File;
use std::io::{self, Read};
use std::os::unix::process::{CommandExt, ExitStatusExt};
use std::path::Path;
use std::process::{Child, Command, ExitStatus};
use std::thread;
use std::time::{Duration, Instant};
use tempfile::TempDir;

const JUDGE_CPU: Duration = Duration::from_secs(15);
const JUDGE_MEM_MIB: u32 = 1024;
const FILE_BYTES: u64 = 16 * 1024 * 1024;

pub fn run(
    interactor: &Path,
    solution: &Path,
    input: &Path,
    answer: &Path,
    limits: Limits,
    mem_mib: u32,
) -> Result<u8> {
    let work = TempDir::new()?;
    let (mut judge, mut contestant) = commands(interactor, solution)?;
    judge
        .arg(input)
        .arg(work.path().join("output.txt"))
        .arg(answer);
    let diagnostics = work.path().join("interactor.stderr");
    judge.stderr(File::create(&diagnostics)?);
    contestant.stderr(File::create(work.path().join("solution.stderr"))?);
    let mut judge = Process::spawn(judge, JUDGE_CPU, JUDGE_MEM_MIB)?;
    let mut contestant = Process::spawn(contestant, limits.cpu.unwrap_or(limits.wall), mem_mib)?;
    supervise(&mut judge, &mut contestant, limits, &diagnostics)
}

fn commands(interactor: &Path, solution: &Path) -> Result<(Command, Command)> {
    let (judge_input, solution_output) = io::pipe()?;
    let (solution_input, judge_output) = io::pipe()?;
    let mut judge = Command::new(interactor);
    judge.stdin(judge_input).stdout(judge_output);
    let mut contestant = Command::new(solution);
    contestant.stdin(solution_input).stdout(solution_output);
    Ok((judge, contestant))
}

fn supervise(
    judge: &mut Process,
    contestant: &mut Process,
    limits: Limits,
    diagnostics: &Path,
) -> Result<u8> {
    let deadline = Instant::now() + limits.wall;
    loop {
        let solution = contestant.poll()?;
        let verdict = judge
            .poll()?
            .map(|exit| judge_verdict(exit, diagnostics))
            .transpose()?;
        if verdict == Some(0) || solution.is_some_and(|exit| !exit.passed(limits.cpu)) {
            return Ok(0);
        }
        if let (Some(percent), Some(_)) = (verdict, solution) {
            return Ok(percent);
        }
        if Instant::now() >= deadline {
            if solution.is_some() && verdict.is_none() {
                bail!("interactor wall timeout: {}", diagnostic(diagnostics));
            }
            return Ok(0);
        }
        thread::sleep(Duration::from_millis(5));
    }
}

fn judge_verdict(exit: Exit, diagnostics: &Path) -> Result<u8> {
    if exit.cpu > JUDGE_CPU {
        bail!("interactor CPU limit: {}", diagnostic(diagnostics));
    }
    match exit.status.code() {
        Some(0) => Ok(100),
        Some(code @ 50..=150) => Ok((code - 50) as u8),
        Some(1 | 2 | 8) => Ok(0),
        Some(3) => bail!("interactor failure: {}", diagnostic(diagnostics)),
        Some(code) => bail!(
            "unsupported interactor exit {code}: {}",
            diagnostic(diagnostics)
        ),
        None if matches!(exit.status.signal(), Some(libc::SIGPIPE | libc::SIGXFSZ)) => Ok(0),
        None => bail!("interactor {}: {}", exit.status, diagnostic(diagnostics)),
    }
}

fn diagnostic(path: &Path) -> String {
    let mut bytes = Vec::new();
    if let Ok(file) = File::open(path) {
        file.take(4096).read_to_end(&mut bytes).ok();
    }
    String::from_utf8_lossy(&bytes).trim().to_owned()
}

#[derive(Clone, Copy)]
struct Exit {
    status: ExitStatus,
    cpu: Duration,
}

impl Exit {
    fn passed(self, cpu: Option<Duration>) -> bool {
        self.status.success() && cpu.is_none_or(|limit| self.cpu <= limit)
    }
}

struct Process {
    child: Child,
    exit: Option<Exit>,
}

impl Process {
    fn spawn(mut command: Command, cpu: Duration, mem_mib: u32) -> Result<Self> {
        command.process_group(0);
        // Only async-signal-safe syscalls run between fork and exec.
        unsafe {
            command.pre_exec(move || {
                set_limit(
                    libc::RLIMIT_CPU,
                    cpu.as_secs() + u64::from(cpu.subsec_nanos() > 0),
                )?;
                set_limit(libc::RLIMIT_AS, u64::from(mem_mib) * 1024 * 1024)?;
                set_limit(libc::RLIMIT_FSIZE, FILE_BYTES)?;
                set_limit(libc::RLIMIT_CORE, 0)
            });
        }
        let child = command
            .spawn()
            .with_context(|| format!("spawn {:?}", command.get_program()))?;
        Ok(Self { child, exit: None })
    }

    fn poll(&mut self) -> Result<Option<Exit>> {
        if self.exit.is_none() {
            self.exit = try_wait(self.child.id())?;
            if self.exit.is_some() {
                self.kill_group();
            }
        }
        Ok(self.exit)
    }

    fn kill_group(&self) {
        // Descendants inherit the group, including after the direct child exits.
        unsafe { libc::kill(-(self.child.id() as i32), libc::SIGKILL) };
    }
}

impl Drop for Process {
    fn drop(&mut self) {
        if self.exit.is_none() {
            self.kill_group();
            self.child.wait().ok();
        }
    }
}

fn try_wait(pid: u32) -> Result<Option<Exit>> {
    let mut status = 0;
    let mut usage = unsafe { std::mem::zeroed::<libc::rusage>() };
    let waited = unsafe { libc::wait4(pid as i32, &mut status, libc::WNOHANG, &mut usage) };
    if waited < 0 {
        let error = io::Error::last_os_error();
        if error.kind() != io::ErrorKind::Interrupted {
            return Err(error).context("wait interactive process");
        }
    }
    Ok((waited > 0).then(|| Exit {
        status: ExitStatus::from_raw(status),
        cpu: timeval(usage.ru_utime) + timeval(usage.ru_stime),
    }))
}

fn set_limit(resource: libc::__rlimit_resource_t, value: u64) -> io::Result<()> {
    let limit = libc::rlimit {
        rlim_cur: value,
        rlim_max: value,
    };
    if unsafe { libc::setrlimit(resource, &limit) } != 0 {
        return Err(io::Error::last_os_error());
    }
    Ok(())
}

fn timeval(time: libc::timeval) -> Duration {
    Duration::from_secs(time.tv_sec as u64) + Duration::from_micros(time.tv_usec as u64)
}
