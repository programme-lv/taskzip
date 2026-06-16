use taskzip::check;
use taskzip::exec;
use taskzip::generate;
use taskzip::package;

use anyhow::Result;
use clap::{Parser, Subcommand};
use std::path::PathBuf;

#[derive(Parser)]
#[command(name = "taskzip", about = "TaskZip package tooling")]
struct Cli {
    #[command(subcommand)]
    cmd: Command,
}

#[derive(Subcommand)]
enum Command {
    Check {
        #[arg(default_value = ".")]
        package: PathBuf,
    },
    Generate {
        #[arg(default_value = ".")]
        package: PathBuf,
        #[arg(long)]
        write: bool,
        #[arg(long)]
        force: bool,
        #[arg(long, default_value = ".taskzip/generated")]
        out: PathBuf,
    },
    ValidateTests {
        #[arg(default_value = ".")]
        package: PathBuf,
    },
    RunSolutions {
        #[arg(default_value = ".")]
        package: PathBuf,
    },
    Verify {
        #[arg(default_value = ".")]
        package: PathBuf,
    },
}

fn main() -> Result<()> {
    let cli = Cli::parse();
    match cli.cmd {
        Command::Check { package } => {
            let pkg = package::open(&package)?;
            let warns = check::check(&pkg)?;
            for w in warns {
                eprintln!("warn: {w}");
            }
            println!("ok: {}", pkg.id);
        }
        Command::Generate {
            package,
            write,
            force,
            out,
        } => {
            let pkg = package::open(&package)?;
            check::check(&pkg)?;
            let dst = if write { pkg.root.join("tests") } else { out };
            let report = generate::generate(&pkg, &dst, force)?;
            println!(
                "ok: wrote inputs to {} (cached {}, regenerated {})",
                dst.display(),
                report.cached,
                report.regenerated
            );
        }
        Command::ValidateTests { package } => {
            let pkg = package::open(&package)?;
            check::check(&pkg)?;
            exec::validate_tests(&pkg)?;
            println!("ok: validator passed");
        }
        Command::RunSolutions { package } => {
            let pkg = package::open(&package)?;
            check::check(&pkg)?;
            let rows = exec::run_solutions(&pkg)?;
            for r in rows {
                println!("{}: {}/{}", r.fname, r.score, r.total);
            }
        }
        Command::Verify { package } => {
            let pkg = package::open(&package)?;
            let warns = check::check(&pkg)?;
            for w in warns {
                eprintln!("warn: {w}");
            }
            exec::validate_tests(&pkg)?;
            let rows = exec::run_solutions(&pkg)?;
            for r in &rows {
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
                println!("{}: {}/{}", r.fname, r.score, r.total);
            }
        }
    }
    Ok(())
}
