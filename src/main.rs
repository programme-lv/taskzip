use taskzip::archive;
use taskzip::assist;
use taskzip::check;
use taskzip::exec;
use taskzip::generate;
use taskzip::import;
use taskzip::package;
use taskzip::progress;

use anyhow::{bail, Context, Result};
use chrono::{Datelike, Local};
use clap::{Parser, Subcommand, ValueEnum};
use dialoguer::{Input, Select};
use std::path::{Path, PathBuf};
use std::time::Duration;

#[derive(Parser)]
#[command(name = "taskzip", about = "TaskZip package tooling")]
struct Cli {
    #[command(subcommand)]
    cmd: Command,
}

#[derive(ValueEnum, Clone)]
enum ExternalFormat {
    Lio2024,
}

impl std::fmt::Display for ExternalFormat {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.write_str(self.to_possible_value().unwrap().get_name())
    }
}

#[derive(ValueEnum, Clone, Copy)]
enum LioStage {
    School,
    Municipal,
    National,
    Selection,
}

impl std::fmt::Display for LioStage {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        f.write_str(self.to_possible_value().unwrap().get_name())
    }
}

#[derive(Subcommand)]
enum Command {
    #[command(about = "Pack a directory to .zip or unpack a .zip")]
    Archive {
        input: PathBuf,
        #[arg(long)]
        out: Option<PathBuf>,
    },
    #[command(about = "Validate package structure and metadata without running task code")]
    Check {
        #[arg(default_value = ".")]
        package: PathBuf,
    },
    #[command(about = "Generate inputs, generate answers, or validate tests")]
    Tests {
        #[command(subcommand)]
        cmd: TestsCommand,
    },
    #[command(about = "Convert an external task package into TaskZip")]
    Import {
        #[arg(value_enum)]
        format: ExternalFormat,
        src: PathBuf,
        dest: PathBuf,
        #[arg(long)]
        skip_ai_import: bool,
        #[arg(long, value_parser = parse_lio_year)]
        year: Option<i32>,
        #[arg(long, value_enum)]
        stage: Option<LioStage>,
        #[arg(long, help = "Comma-separated task authors; empty allowed")]
        authors: Option<String>,
    },
    #[command(about = "Run check, validator, solutions, and compare expected solution scores")]
    Verify {
        #[arg(default_value = ".")]
        package: PathBuf,
    },
}

#[derive(Subcommand)]
enum TestsCommand {
    #[command(about = "Build inputs from testspec/tests.txt using generator or manual cases")]
    Generate {
        #[arg(default_value = ".")]
        package: PathBuf,
        #[arg(long)]
        write: bool,
        #[arg(long)]
        force: bool,
        #[arg(long, default_value = ".taskzip/generated")]
        out: PathBuf,
        #[arg(long, default_value_t = 60)]
        timeout: u64,
    },
    #[command(about = "Run the model solution to write answer files for generated inputs")]
    Answers {
        #[arg(default_value = ".")]
        package: PathBuf,
        #[arg(
            long = "in",
            default_value = ".taskzip/generated",
            help = "Directory with NNNi.txt inputs"
        )]
        input: PathBuf,
        #[arg(long, help = "Directory for NNNo.txt answers; defaults to --in")]
        out: Option<PathBuf>,
        #[arg(long, help = "Read inputs from and write answers to package tests/")]
        write: bool,
        #[arg(long, help = "Registered solution filename under solutions/")]
        solution: Option<String>,
    },
    #[command(about = "Run testspec/validator.cpp, when present, on every official input")]
    Validate {
        #[arg(default_value = ".")]
        package: PathBuf,
    },
}

fn main() -> Result<()> {
    let cli = Cli::parse();
    match cli.cmd {
        Command::Archive { input, out } => run_archive(input, out),
        Command::Check { package } => run_check(package),
        Command::Tests { cmd } => run_tests(cmd),
        Command::Verify { package } => run_verify(package),
        Command::Import {
            format,
            src,
            dest,
            skip_ai_import,
            year,
            stage,
            authors,
        } => run_import(format, src, dest, skip_ai_import, year, stage, authors),
    }
}

fn run_archive(input: PathBuf, out: Option<PathBuf>) -> Result<()> {
    let report = archive::run(&input, out.as_deref())?;
    let action = match report.action {
        archive::Action::Packed => "packed",
        archive::Action::Unpacked => "unpacked",
    };
    println!("ok: {action} to {}", report.output.display());
    Ok(())
}

fn run_check(package: PathBuf) -> Result<()> {
    let pkg = package::open(&package)?;
    let warns = check::check(&pkg)?;
    for w in warns {
        eprintln!("warn: {w}");
    }
    println!("ok: {}", pkg.id);
    Ok(())
}

fn run_tests(cmd: TestsCommand) -> Result<()> {
    match cmd {
        TestsCommand::Generate {
            package,
            write,
            force,
            out,
            timeout,
        } => run_generate(package, write, force, out, timeout),
        TestsCommand::Answers {
            package,
            input,
            out,
            write,
            solution,
        } => run_answers(package, input, out, write, solution),
        TestsCommand::Validate { package } => run_validate(package),
    }
}

fn run_generate(
    package: PathBuf,
    write: bool,
    force: bool,
    out: PathBuf,
    timeout: u64,
) -> Result<()> {
    let pkg = package::open(&package)?;
    check::preflight_generate(&pkg)?;
    let dst = if write { pkg.root.join("tests") } else { out };
    let report = generate::generate(&pkg, &dst, force, Duration::from_secs(timeout))?;
    println!(
        "ok: wrote inputs to {} (cached {}, regenerated {})",
        dst.display(),
        report.cached,
        report.regenerated
    );
    Ok(())
}

fn run_validate(package: PathBuf) -> Result<()> {
    let pkg = package::open(&package)?;
    check::check(&pkg)?;
    exec::validate_tests(&pkg, false)?;
    println!("ok: validator passed");
    Ok(())
}

fn run_answers(
    package: PathBuf,
    input: PathBuf,
    out: Option<PathBuf>,
    write: bool,
    solution: Option<String>,
) -> Result<()> {
    let pkg = package::open(&package)?;
    let input = if write { pkg.root.join("tests") } else { input };
    let out = if write {
        pkg.root.join("tests")
    } else {
        out.unwrap_or_else(|| input.clone())
    };
    let report = exec::generate_answers(&pkg, &input, &out, solution.as_deref())?;
    println!(
        "ok: wrote {} answers to {} using {}",
        report.written,
        out.display(),
        report.solution
    );
    Ok(())
}

fn run_verify(package: PathBuf) -> Result<()> {
    let pkg = package::open(&package)?;
    let warns = check::check(&pkg)?;
    for w in warns {
        eprintln!("warn: {w}");
    }
    exec::validate_tests(&pkg, true)?;
    let rows = exec::run_solutions(&pkg)?;
    for r in &rows {
        check_expected_score(r)?;
    }
    println!("ok: {} (solutions: {})", pkg.id, rows.len());
    Ok(())
}

fn check_expected_score(r: &exec::SolutionRun) -> Result<()> {
    if let Some(exp) = r.expected {
        if exp != r.score {
            anyhow::bail!(
                "{}: score {}/{} != expected {}",
                r.fname,
                r.score,
                r.total,
                exp
            );
        }
    }
    Ok(())
}

fn run_import(
    format: ExternalFormat,
    src: PathBuf,
    dest: PathBuf,
    skip_ai_import: bool,
    year: Option<i32>,
    stage: Option<LioStage>,
    authors: Option<String>,
) -> Result<()> {
    check_import(&dest, skip_ai_import)?;
    let origin = lio_origin(year, stage, authors)?;
    let dest = match format {
        ExternalFormat::Lio2024 => {
            import::lio2024(&src, &dest, origin, skip_ai_import, progress::print)?
        }
    };
    println!("ok: imported {} to {}", format, dest.display());
    Ok(())
}

fn check_import(dest: &Path, skip_ai_import: bool) -> Result<()> {
    if !dest.is_dir() {
        bail!("dest is not a directory: {}", dest.display());
    }
    if !skip_ai_import {
        assist::check_openai_api_key()?;
    }
    Ok(())
}

fn lio_origin(
    year: Option<i32>,
    stage: Option<LioStage>,
    authors: Option<String>,
) -> Result<import::LioOrigin> {
    let year = match year {
        Some(year) => year,
        None => prompt_year()?,
    };
    let stage = match stage {
        Some(stage) => stage,
        None => prompt_stage()?,
    };
    let authors = match authors {
        Some(authors) => authors,
        None => Input::new()
            .with_prompt("Authors (comma-separated, optional)")
            .allow_empty(true)
            .interact_text()
            .context("read authors")?,
    };
    Ok(import::LioOrigin {
        year,
        stage: stage.to_string(),
        authors: parse_authors(&authors),
    })
}

fn prompt_year() -> Result<i32> {
    let current = current_year();
    let input: String = Input::new()
        .with_prompt("Year (YYYY or YYYY/YYYY)")
        .default(current.to_string())
        .validate_with(|value: &String| parse_lio_year(value).map(|_| ()))
        .interact_text()
        .context("read year")?;
    parse_lio_year(&input).map_err(anyhow::Error::msg)
}

fn prompt_stage() -> Result<LioStage> {
    let stages = [
        LioStage::School,
        LioStage::Municipal,
        LioStage::National,
        LioStage::Selection,
    ];
    let selected = Select::new()
        .with_prompt("Stage")
        .items(stages)
        .default(0)
        .interact()
        .context("read stage")?;
    Ok(stages[selected])
}

fn parse_lio_year(value: &str) -> Result<i32, String> {
    let parts: Vec<_> = value.trim().split('/').collect();
    let year = match parts.as_slice() {
        [year] => parse_year_number(year)?,
        [first, second] => {
            let first = parse_year_number(first)?;
            let second = parse_year_number(second)?;
            if second != first + 1 {
                return Err("academic years must be consecutive".into());
            }
            second
        }
        _ => return Err("use YYYY or YYYY/YYYY".into()),
    };
    if !(1986..=current_year()).contains(&year) {
        return Err(format!("year must be 1986-{}", current_year()));
    }
    Ok(year)
}

fn parse_year_number(value: &str) -> Result<i32, String> {
    if value.len() != 4 || !value.bytes().all(|b| b.is_ascii_digit()) {
        return Err("use a four-digit year".into());
    }
    value.parse().map_err(|_| "invalid year".into())
}

fn current_year() -> i32 {
    Local::now().year()
}

fn parse_authors(value: &str) -> Vec<String> {
    value
        .split(',')
        .map(str::trim)
        .filter(|author| !author.is_empty())
        .map(str::to_string)
        .collect()
}
