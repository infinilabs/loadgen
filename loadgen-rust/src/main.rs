// Copyright (C) INFINI Labs & INFINI LIMITED.
//
// The INFINI Loadgen is offered under the GNU Affero General Public License v3.0
// and as commercial software.
//
// For commercial licensing, contact us at:
//   - Website: infinilabs.com
//   - Email: hello@infini.ltd
//
// Open Source licensed under AGPL V3:
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <http://www.gnu.org/licenses/>.

//! INFINI Loadgen - Main entry point
//!
//! A high-performance HTTP load generator and testing suite.

use anyhow::Result;
use clap::Parser;
use std::path::{Path, PathBuf};
use std::process::ExitCode;
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::Arc;
use tracing::{error, info};

use loadgen::config::yaml::load_config;
use loadgen::loader::generator::LoadGenerator;
use loadgen::runner::test::TestRunner;

/// CLI argument parser
#[derive(Parser, Debug)]
#[command(
    name = "loadgen",
    about = "A high-performance HTTP load generator and testing suite",
    version,
    author = "INFINI Labs <hello@infini.ltd>",
    before_help = r#"
   __   ___  _      ___  ___   __    __
  / /  /___\/_\    /   \/ _ \ /__\/\ \ \
 / /  //  ///_\\  / /\ / /_\//_\ /  \/ /
/ /__/ \_//  _  \/ /_// /_\\//__/ /\  /
\____|___/\_/ \_/___,'\____/\__/\_\ \/

HOME: https://github.com/infinilabs/loadgen/
"#
)]
struct Cli {
    /// Number of concurrent threads to use
    #[arg(short = 'c', long = "concurrency", default_value = "1")]
    concurrency: usize,

    /// Duration of the test in seconds
    #[arg(short = 'd', long = "duration", default_value = "5")]
    duration: u64,

    /// Maximum requests per second (fixed QPS), -1 for unlimited
    #[arg(short = 'r', long = "rate", default_value = "-1")]
    rate_limit: i64,

    /// Total number of requests to send, -1 for unlimited
    #[arg(short = 'l', long = "limit", default_value = "-1")]
    request_limit: i64,

    /// Request timeout in seconds, 0 for no timeout
    #[arg(long = "timeout", default_value = "0")]
    timeout: u64,

    /// Connection read timeout in seconds, 0 inherits from timeout
    #[arg(long = "read-timeout", default_value = "0")]
    read_timeout: u64,

    /// Connection write timeout in seconds, 0 inherits from timeout
    #[arg(long = "write-timeout", default_value = "0")]
    write_timeout: u64,

    /// Connection dial timeout in seconds
    #[arg(long = "dial-timeout", default_value = "3")]
    dial_timeout: u64,

    /// Enable gzip compression for requests
    #[arg(long = "compress", default_value = "false")]
    compress: bool,

    /// Enable mixed requests from YAML/DSL
    #[arg(long = "mixed", default_value = "false")]
    mixed: bool,

    /// Number of rounds for each request configuration, -1 for unlimited
    #[arg(long = "total-rounds", default_value = "-1")]
    total_rounds: i64,

    /// Path to a DSL-based request file to execute
    #[arg(long = "run")]
    dsl_file: Option<PathBuf>,

    /// Path to YAML config file
    #[arg(short = 'C', long = "config", default_value = "loadgen.yml")]
    config: PathBuf,

    /// Log level (trace, debug, info, warn, error)
    #[arg(long = "log", default_value = "info")]
    log_level: String,

    /// Run in debug mode
    #[arg(long = "debug", default_value = "false")]
    debug: bool,
}

fn print_header() {
    println!(
        r#"
   __   ___  _      ___  ___   __    __
  / /  /___\/_\    /   \/ _ \ /__\/\ \ \
 / /  //  ///_\\  / /\ / /_\//_\ /  \/ /
/ /__/ \_//  _  \/ /_// /_\\//__/ /\  /
\____|___/\_/ \_/___,'\____/\__/\_\ \/

HOME: https://github.com/infinilabs/loadgen/
"#
    );
}

fn setup_logging(log_level: &str, debug: bool) {
    let level = if debug {
        "debug"
    } else {
        log_level
    };

    let filter = tracing_subscriber::EnvFilter::try_from_default_env()
        .unwrap_or_else(|_| tracing_subscriber::EnvFilter::new(level));

    tracing_subscriber::fmt()
        .with_env_filter(filter)
        .with_target(false)
        .with_thread_ids(false)
        .init();
}

#[tokio::main]
async fn main() -> ExitCode {
    let cli = Cli::parse();

    print_header();
    setup_logging(&cli.log_level, cli.debug);

    // Setup Ctrl+C handler
    let interrupted = Arc::new(AtomicBool::new(false));
    let interrupted_clone = Arc::clone(&interrupted);
    
    ctrlc::set_handler(move || {
        info!("Received interrupt signal, shutting down...");
        interrupted_clone.store(true, Ordering::SeqCst);
    })
    .expect("Failed to set Ctrl+C handler");

    // Load configuration
    let config = match load_config(&cli.config).await {
        Ok(config) => config,
        Err(e) => {
            error!("Failed to load config: {}", e);
            return ExitCode::from(1);
        }
    };

    // Create CLI options
    let cli_options = loadgen::loader::generator::CliOptions {
        concurrency: cli.concurrency,
        duration: cli.duration,
        rate_limit: cli.rate_limit,
        request_limit: cli.request_limit,
        timeout: cli.timeout,
        read_timeout: cli.read_timeout,
        write_timeout: cli.write_timeout,
        dial_timeout: cli.dial_timeout,
        compress: cli.compress,
        total_rounds: cli.total_rounds,
        debug: cli.debug,
    };

    let mut exit_code = 0;

    // Run DSL file if specified
    if let Some(dsl_file) = &cli.dsl_file {
        info!("Running DSL based requests from {:?}", dsl_file);
        match run_dsl_file(&config, dsl_file, &cli_options, Arc::clone(&interrupted)).await {
            Ok(status) => {
                if status != 0 {
                    exit_code = status;
                }
            }
            Err(e) => {
                error!("Failed to run DSL file: {}", e);
                return ExitCode::from(1);
            }
        }
        if !cli.mixed {
            return ExitCode::from(exit_code as u8);
        }
    }

    // Run YAML-based requests
    if !config.requests.is_empty() {
        info!("Running YAML based requests");
        match run_loader(&config, &cli_options, Arc::clone(&interrupted)).await {
            Ok(status) => {
                if status != 0 {
                    exit_code = status;
                }
            }
            Err(e) => {
                error!("Failed to run loader: {}", e);
                return ExitCode::from(2);
            }
        }
        if !cli.mixed {
            return ExitCode::from(exit_code as u8);
        }
    }

    // Run test suite if configured
    if !config.tests.is_empty() {
        info!("Running test suite");
        let runner = TestRunner::new(&config);
        if !runner.run().await {
            exit_code = 1;
        }
    }

    ExitCode::from(exit_code as u8)
}

async fn run_dsl_file(
    config: &loadgen::AppConfig,
    path: &Path,
    cli_options: &loadgen::loader::generator::CliOptions,
    interrupted: Arc<AtomicBool>,
) -> Result<i32> {
    use loadgen::config::dsl::parse_dsl_file;

    let loader_config = parse_dsl_file(path, config).await?;
    run_loader_config(&loader_config, cli_options, interrupted).await
}

async fn run_loader(
    config: &loadgen::AppConfig,
    cli_options: &loadgen::loader::generator::CliOptions,
    interrupted: Arc<AtomicBool>,
) -> Result<i32> {
    let loader_config = config.to_loader_config();
    run_loader_config(&loader_config, cli_options, interrupted).await
}

async fn run_loader_config(
    config: &loadgen::LoaderConfig,
    cli_options: &loadgen::loader::generator::CliOptions,
    interrupted: Arc<AtomicBool>,
) -> Result<i32> {
    let mut generator = LoadGenerator::new(config.clone(), cli_options.clone(), interrupted)?;
    let stats = generator.run().await?;

    // Check assertions
    if config.runner.assert_invalid && stats.num_assert_invalid > 0 {
        return Ok(1);
    }
    if config.runner.assert_error && stats.num_errors > 0 {
        return Ok(2);
    }

    Ok(0)
}
