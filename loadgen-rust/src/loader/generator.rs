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

//! Load generator - orchestrates workers and aggregates statistics

use std::collections::HashMap;
use std::sync::atomic::AtomicBool;
use std::sync::Arc;
use std::time::{Duration, Instant};

use anyhow::Result;
use tokio::task::JoinSet;
use tracing::{error, info, warn};

use super::stats::{AggregatedStats, LoadStats};
use super::worker::{Worker, WorkerConfig};
use crate::config::types::LoaderConfig;
use crate::http::client::{HttpClient, HttpClientConfig};
use crate::http::request::RequestBuilder;
use crate::variable::provider::VariableProvider;

/// CLI options passed to the generator
#[derive(Debug, Clone)]
pub struct CliOptions {
    /// Number of concurrent workers
    pub concurrency: usize,
    /// Duration in seconds
    pub duration: u64,
    /// Rate limit (requests per second)
    pub rate_limit: i64,
    /// Total request limit
    pub request_limit: i64,
    /// Request timeout
    pub timeout: u64,
    /// Read timeout
    pub read_timeout: u64,
    /// Write timeout
    pub write_timeout: u64,
    /// Dial timeout
    pub dial_timeout: u64,
    /// Enable compression
    pub compress: bool,
    /// Total rounds
    pub total_rounds: i64,
    /// Debug mode
    pub debug: bool,
}

impl Default for CliOptions {
    fn default() -> Self {
        Self {
            concurrency: 1,
            duration: 5,
            rate_limit: -1,
            request_limit: -1,
            timeout: 0,
            read_timeout: 0,
            write_timeout: 0,
            dial_timeout: 3,
            compress: false,
            total_rounds: -1,
            debug: false,
        }
    }
}

/// Load generator
pub struct LoadGenerator {
    config: LoaderConfig,
    options: CliOptions,
    interrupted: Arc<AtomicBool>,
    http_client: Arc<HttpClient>,
    provider: Arc<VariableProvider>,
}

impl LoadGenerator {
    /// Create a new load generator
    pub fn new(
        config: LoaderConfig,
        options: CliOptions,
        interrupted: Arc<AtomicBool>,
    ) -> Result<Self> {
        // Create HTTP client
        let http_config = HttpClientConfig {
            timeout: options.timeout,
            read_timeout: options.read_timeout,
            write_timeout: options.write_timeout,
            dial_timeout: options.dial_timeout,
            compress: options.compress,
            disable_header_normalizing: config.runner.disable_header_names_normalizing,
            max_connections_per_host: options.concurrency,
            ..Default::default()
        };
        let http_client = Arc::new(HttpClient::new(http_config)?);

        // Create variable provider with environment variables
        let env_vars: HashMap<String, String> = std::env::vars().collect();
        let provider = Arc::new(VariableProvider::new(&config.variables, env_vars));

        Ok(Self {
            config,
            options,
            interrupted,
            http_client,
            provider,
        })
    }

    /// Run the load generator
    pub async fn run(&mut self) -> Result<AggregatedStats> {
        // Override total_rounds from CLI if specified
        let total_rounds = if self.options.total_rounds > 0 {
            self.options.total_rounds
        } else {
            self.config.runner.total_rounds
        };

        // Warmup phase
        if !self.config.runner.no_warm {
            self.warmup().await?;
        }

        // Calculate request limit per worker
        let request_limit = self.options.request_limit;
        let concurrency = if request_limit > 0 && (request_limit as usize) < self.options.concurrency {
            request_limit as usize
        } else {
            self.options.concurrency
        };

        let requests_per_worker = if request_limit > 0 {
            (request_limit + concurrency as i64 - 1) / concurrency as i64
        } else {
            -1
        };

        // Create shared stats
        let stats = Arc::new(LoadStats::new());

        // Start wall clock
        let wall_start = Instant::now();

        // Spawn workers
        let mut tasks = JoinSet::new();

        for i in 0..concurrency {
            let worker_request_limit = if requests_per_worker > 0 {
                let remaining = request_limit - (i as i64 * requests_per_worker);
                remaining.min(requests_per_worker)
            } else {
                -1
            };

            let worker_config = WorkerConfig {
                duration: Duration::from_secs(self.options.duration),
                request_limit: worker_request_limit,
                rate_limit: self.options.rate_limit,
                total_rounds,
                compress: self.options.compress,
                debug: self.options.debug,
            };

            let worker = Worker::new(
                self.config.clone(),
                worker_config,
                Arc::clone(&self.http_client),
                Arc::clone(&self.provider),
                Arc::clone(&stats),
                Arc::clone(&self.interrupted),
            );

            tasks.spawn(async move {
                if let Err(e) = worker.run().await {
                    error!("Worker error: {}", e);
                }
            });
        }

        // Wait for all workers to complete
        while let Some(result) = tasks.join_next().await {
            if let Err(e) = result {
                error!("Worker task panicked: {}", e);
            }
        }

        let wall_time = wall_start.elapsed();

        // Aggregate and print stats
        let aggregated = AggregatedStats::from(stats.as_ref());
        
        if aggregated.num_requests == 0 {
            error!("Error: No statistics collected / no requests found");
        } else {
            aggregated.print(wall_time, &self.config.runner);
        }

        Ok(aggregated)
    }

    /// Run warmup phase
    async fn warmup(&self) -> Result<()> {
        info!("warmup started");

        let global_ctx: HashMap<String, String> = HashMap::new();

        for item in &self.config.requests {
            let _request = match &item.request {
                Some(r) => r,
                None => continue,
            };

            // Build request
            let mut runtime_vars = global_ctx.clone();
            let request_builder = RequestBuilder::new(
                &self.provider,
                &self.config.runner,
                self.options.compress,
            );

            let req = match request_builder.build(item, &mut runtime_vars, self.http_client.client()) {
                Ok(r) => r,
                Err(e) => {
                    error!("Failed to build warmup request: {}", e);
                    continue;
                }
            };

            info!("[{}] {}", req.method(), req.url());

            // Execute request
            let start = Instant::now();
            let response = self.http_client.execute(req).await;
            let duration = start.elapsed();

            match response {
                Ok(resp) => {
                    let status = resp.status().as_u16() as i32;
                    let body = resp.text().await.unwrap_or_default();
                    
                    info!(
                        "status: {}, duration: {:?}, response: {}",
                        status,
                        duration,
                        if body.len() > 512 { &body[..512] } else { &body }
                    );

                    // Check if status indicates failure
                    let is_valid = self.config.runner.valid_status_codes_during_warmup.is_empty()
                        || self.config.runner.valid_status_codes_during_warmup.contains(&status);

                    if !is_valid && (status >= 400 || status == 0) {
                        warn!(
                            "Warmup request returned status {}, are you sure to continue?",
                            status
                        );
                    }
                }
                Err(e) => {
                    error!("Warmup request failed: {}", e);
                    warn!("Warmup request failed, are you sure to continue?");
                }
            }
        }

        info!("warmup finished");
        Ok(())
    }
}
