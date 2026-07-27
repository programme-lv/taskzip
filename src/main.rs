use taskzip::check;
use taskzip::exec;
use taskzip::generate;
use taskzip::import;
use taskzip::package;
use taskzip::progress;

use anyhow::Result;
use clap::{Parser, Subcommand, ValueEnum};
use std::path::PathBuf;
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

#[derive(Subcommand)]
enum Command {
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
        skip_statement_import: bool,
    },
    #[command(about = "Compile registered C++ solutions, run official tests, and print scores")]
    RunSolutions {
        #[arg(default_value = ".")]
        package: PathBuf,
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
        Command::Check { package } => run_check(package),
        Command::Tests { cmd } => run_tests(cmd),
        Command::RunSolutions { package } => run_solutions(package),
        Command::Verify { package } => run_verify(package),
        Command::Import {
            format,
            src,
            dest,
            skip_statement_import,
        } => run_import(format, src, dest, skip_statement_import),
    }
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
    exec::validate_tests(&pkg)?;
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

fn run_solutions(package: PathBuf) -> Result<()> {
    let pkg = package::open(&package)?;
    check::check(&pkg)?;
    let rows = exec::run_solutions(&pkg)?;
    for r in rows {
        println!("{}: {}/{}", r.fname, r.score, r.total);
    }
    Ok(())
}

fn run_verify(package: PathBuf) -> Result<()> {
    let pkg = package::open(&package)?;
    let warns = check::check(&pkg)?;
    for w in warns {
        eprintln!("warn: {w}");
    }
    exec::validate_tests(&pkg)?;
    let rows = exec::run_solutions(&pkg)?;
    for r in &rows {
        check_expected_score(r)?;
        println!("{}: {}/{}", r.fname, r.score, r.total);
    }
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
    skip_statement_import: bool,
) -> Result<()> {
    let dest = match format {
        ExternalFormat::Lio2024 => {
            import::lio2024(&src, &dest, skip_statement_import, progress::print)?
        }
    };
    println!("ok: imported {} to {}", format, dest.display());
    Ok(())
}
