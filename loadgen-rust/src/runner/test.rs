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

//! Test runner for executing test suites

use std::path::Path;
use std::sync::atomic::AtomicBool;
use std::sync::Arc;
use std::time::Instant;

use chrono::Local;
use tracing::{debug, error, info};

use crate::config::dsl::parse_dsl_file;
use crate::config::types::AppConfig;
use crate::loader::generator::{CliOptions, LoadGenerator};

/// Test result
#[derive(Debug)]
pub struct TestResult {
    pub failed: bool,
    pub duration_ms: i64,
    pub error: Option<String>,
}

/// Test message for reporting
#[derive(Debug)]
pub struct TestMsg {
    pub path: String,
    pub status: String, // ABORTED/FAILED/SUCCESS
    pub duration_ms: i64,
}

/// Test runner for executing test suites
pub struct TestRunner<'a> {
    config: &'a AppConfig,
}

impl<'a> TestRunner<'a> {
    /// Create a new test runner
    pub fn new(config: &'a AppConfig) -> Self {
        Self { config }
    }

    /// Run all tests
    pub async fn run(&self) -> bool {
        let mut results: Vec<TestMsg> = Vec::new();

        for test in &self.config.tests {
            // Wait between tests
            tokio::time::sleep(std::time::Duration::from_secs(1)).await;

            let result = self.run_test(test).await;
            let msg = TestMsg {
                path: test.path.clone(),
                status: match &result {
                    Ok(r) if r.failed => "FAILED".to_string(),
                    Ok(_) => "SUCCESS".to_string(),
                    Err(_) => "ABORTED".to_string(),
                },
                duration_ms: result.as_ref().map(|r| r.duration_ms).unwrap_or(0),
            };
            results.push(msg);
        }

        // Print summary
        let mut all_ok = true;
        for msg in &results {
            let timestamp = Local::now().format("%Y-%m-%d %H:%M:%S");
            info!(
                "[{}][TEST][{}] [{}] duration: {}ms",
                timestamp, msg.status, msg.path, msg.duration_ms
            );
            if msg.status != "SUCCESS" {
                all_ok = false;
            }
        }

        all_ok
    }

    /// Run a single test
    async fn run_test(&self, test: &crate::config::types::Test) -> anyhow::Result<TestResult> {
        let start = Instant::now();
        let mut result = TestResult {
            failed: false,
            duration_ms: 0,
            error: None,
        };

        // Find DSL file
        let dsl_path = Path::new(&test.path).join("loadgen.dsl");
        if !dsl_path.exists() {
            debug!("DSL file not found: {:?}", dsl_path);
            result.error = Some(format!("DSL file not found: {:?}", dsl_path));
            result.failed = true;
            result.duration_ms = start.elapsed().as_millis() as i64;
            return Ok(result);
        }

        // Parse DSL
        let loader_config = match parse_dsl_file(&dsl_path, self.config).await {
            Ok(c) => c,
            Err(e) => {
                error!("Failed to parse DSL: {}", e);
                result.error = Some(format!("Failed to parse DSL: {}", e));
                result.failed = true;
                result.duration_ms = start.elapsed().as_millis() as i64;
                return Ok(result);
            }
        };

        // Create generator
        let cli_options = CliOptions {
            compress: test.compress,
            ..Default::default()
        };

        let interrupted = Arc::new(AtomicBool::new(false));
        let mut generator = match LoadGenerator::new(loader_config.clone(), cli_options, interrupted) {
            Ok(g) => g,
            Err(e) => {
                error!("Failed to create generator: {}", e);
                result.error = Some(format!("Failed to create generator: {}", e));
                result.failed = true;
                result.duration_ms = start.elapsed().as_millis() as i64;
                return Ok(result);
            }
        };

        // Run generator
        let stats = match generator.run().await {
            Ok(s) => s,
            Err(e) => {
                error!("Generator failed: {}", e);
                result.error = Some(format!("Generator failed: {}", e));
                result.failed = true;
                result.duration_ms = start.elapsed().as_millis() as i64;
                return Ok(result);
            }
        };

        // Check results
        if loader_config.runner.assert_invalid && stats.num_assert_invalid > 0 {
            result.failed = true;
        }
        if loader_config.runner.assert_error && stats.num_errors > 0 {
            result.failed = true;
        }

        result.duration_ms = start.elapsed().as_millis() as i64;
        Ok(result)
    }
}
