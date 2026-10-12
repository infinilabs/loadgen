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

//! Worker task for executing requests

use std::collections::HashMap;
use std::num::NonZeroU32;
use std::sync::atomic::{AtomicBool, Ordering};
use std::sync::Arc;
use std::time::{Duration, Instant};

use anyhow::Result;
use governor::{Quota, RateLimiter};
use tokio::time::sleep;
use tracing::{debug, error, info, warn};

use super::stats::LoadStats;
use crate::assertion::conditions::AssertionChecker;
use crate::config::types::{LoaderConfig, RequestItem};
use crate::http::client::HttpClient;
use crate::http::request::RequestBuilder;
use crate::variable::provider::VariableProvider;

/// Worker configuration
pub struct WorkerConfig {
    /// Maximum duration for the worker to run
    pub duration: Duration,
    /// Maximum number of requests to send (-1 = unlimited)
    pub request_limit: i64,
    /// Rate limit (requests per second, -1 = unlimited)
    pub rate_limit: i64,
    /// Total number of rounds to run (-1 = unlimited)
    pub total_rounds: i64,
    /// Enable compression
    pub compress: bool,
    /// Debug mode
    pub debug: bool,
}

/// Worker for executing HTTP requests
pub struct Worker {
    config: LoaderConfig,
    worker_config: WorkerConfig,
    http_client: Arc<HttpClient>,
    provider: Arc<VariableProvider>,
    stats: Arc<LoadStats>,
    interrupted: Arc<AtomicBool>,
    rate_limiter: Option<Arc<RateLimiter<governor::state::NotKeyed, governor::state::InMemoryState, governor::clock::DefaultClock>>>,
}

impl Worker {
    /// Create a new worker
    pub fn new(
        config: LoaderConfig,
        worker_config: WorkerConfig,
        http_client: Arc<HttpClient>,
        provider: Arc<VariableProvider>,
        stats: Arc<LoadStats>,
        interrupted: Arc<AtomicBool>,
    ) -> Self {
        // Create rate limiter if needed
        let rate_limiter = if worker_config.rate_limit > 0 {
            let rate = NonZeroU32::new(worker_config.rate_limit as u32)
                .unwrap_or(NonZeroU32::new(1).unwrap());
            let quota = Quota::per_second(rate);
            Some(Arc::new(RateLimiter::direct(quota)))
        } else {
            None
        };

        Self {
            config,
            worker_config,
            http_client,
            provider,
            stats,
            interrupted,
            rate_limiter,
        }
    }

    /// Run the worker
    pub async fn run(&self) -> Result<()> {
        let start = Instant::now();
        let mut total_requests = 0i64;
        let mut total_rounds = 0i64;
        let mut global_ctx: HashMap<String, String> = HashMap::new();

        loop {
            // Check if interrupted
            if self.interrupted.load(Ordering::Relaxed) {
                debug!("Worker interrupted");
                break;
            }

            // Check duration limit
            if start.elapsed() >= self.worker_config.duration {
                debug!("Duration limit reached");
                break;
            }

            // Check rounds limit
            if self.worker_config.total_rounds > 0 && total_rounds >= self.worker_config.total_rounds {
                debug!("Rounds limit reached");
                break;
            }

            total_rounds += 1;

            // Execute all requests in the config
            for item in &self.config.requests {
                // Check request limit
                if self.worker_config.request_limit > 0 
                    && total_requests >= self.worker_config.request_limit 
                {
                    debug!("Request limit reached");
                    return Ok(());
                }

                // Rate limiting
                if let Some(ref limiter) = self.rate_limiter {
                    while limiter.check().is_err() {
                        if self.interrupted.load(Ordering::Relaxed) {
                            return Ok(());
                        }
                        sleep(Duration::from_millis(10)).await;
                    }
                }

                // Execute request
                let continue_next = self.execute_request(item, &mut global_ctx).await?;
                total_requests += 1;

                if !continue_next {
                    break;
                }
            }
        }

        debug!("Worker completed: {} requests in {:?}", total_requests, start.elapsed());
        Ok(())
    }

    /// Execute a single request
    async fn execute_request(
        &self,
        item: &RequestItem,
        global_ctx: &mut HashMap<String, String>,
    ) -> Result<bool> {
        let request = match &item.request {
            Some(r) => r,
            None => return Ok(true),
        };

        let repeat_times = if request.execute_repeat_times > 0 {
            request.execute_repeat_times as usize
        } else {
            1
        };

        for _ in 0..repeat_times {
            let mut runtime_vars = global_ctx.clone();
            
            // Build request
            let request_builder = RequestBuilder::new(
                &self.provider,
                &self.config.runner,
                self.worker_config.compress,
            );

            let req = match request_builder.build(item, &mut runtime_vars, self.http_client.client()) {
                Ok(r) => r,
                Err(e) => {
                    error!("Failed to build request: {}", e);
                    self.stats.num_errors.fetch_add(1, Ordering::Relaxed);
                    continue;
                }
            };

            // Get request size
            let request_size = req.body().map(|b| b.as_bytes().map(|b| b.len()).unwrap_or(0)).unwrap_or(0) as i64;

            // Log request if configured
            if self.config.runner.log_requests {
                info!(
                    "[{}] {}, {:?}",
                    req.method(),
                    req.url(),
                    req.headers()
                );
            }

            // Execute request
            let start = Instant::now();
            let response = self.http_client.execute(req).await;
            let duration = start.elapsed();

            match response {
                Ok(resp) => {
                    let status = resp.status().as_u16() as i32;
                    let response_size = resp.content_length().unwrap_or(0) as i64;

                    // Log response if configured
                    if self.config.runner.log_requests 
                        || self.config.runner.log_status_codes.contains(&status) 
                    {
                        info!("status: {}, duration: {:?}", status, duration);
                    }

                    // Check for error status
                    let is_error = status >= 400 || status == 0;
                    self.stats.record(duration, request_size, response_size, status, is_error);

                    // Handle assertions
                    if let Some(ref assert_config) = item.assert {
                        // Get response body for assertions
                        let body = resp.text().await.unwrap_or_default();
                        
                        // Build context for assertions
                        let ctx = build_assert_context(status, &body, duration);
                        
                        // Check assertions
                        let checker = AssertionChecker::new(&ctx);
                        if !checker.check(assert_config) {
                            self.stats.record_assert_invalid();
                            
                            if !self.config.runner.continue_on_assert_invalid {
                                warn!(
                                    "{} {}, assertion failed, skipping subsequent requests",
                                    request.method, request.url
                                );
                                return Ok(false);
                            }
                        }
                    }

                    // Handle register
                    for register_map in &item.register {
                        for (dest, src) in register_map {
                            if let Some(value) = get_ctx_value(&build_assert_context(status, "", duration), src) {
                                global_ctx.insert(dest.clone(), value);
                            }
                        }
                    }
                }
                Err(e) => {
                    error!("Request failed: {}", e);
                    self.stats.record(duration, request_size, 0, 0, true);
                    self.stats.record_assert_invalid();
                }
            }

            // Handle sleep
            if let Some(ref sleep_action) = item.sleep {
                if sleep_action.sleep_in_milli_seconds > 0 {
                    sleep(Duration::from_millis(sleep_action.sleep_in_milli_seconds as u64)).await;
                }
            }
        }

        Ok(true)
    }
}

/// Build assertion context from response
fn build_assert_context(status: i32, body: &str, duration: Duration) -> HashMap<String, serde_json::Value> {
    let mut ctx = HashMap::new();
    
    // Response context
    ctx.insert("_ctx.response.status".to_string(), serde_json::json!(status));
    ctx.insert("_ctx.response.body".to_string(), serde_json::json!(body));
    ctx.insert("_ctx.response.body_length".to_string(), serde_json::json!(body.len()));
    ctx.insert("_ctx.elapsed".to_string(), serde_json::json!(duration.as_millis() as i64));

    // Try to parse body as JSON
    if let Ok(json) = serde_json::from_str::<serde_json::Value>(body) {
        ctx.insert("_ctx.response.body_json".to_string(), json);
    }

    ctx
}

/// Get a value from the context by path
fn get_ctx_value(ctx: &HashMap<String, serde_json::Value>, path: &str) -> Option<String> {
    // Direct lookup first
    if let Some(value) = ctx.get(path) {
        return Some(value.to_string());
    }

    // Try JSON path lookup for body_json
    if path.starts_with("_ctx.response.body_json.") {
        if let Some(body_json) = ctx.get("_ctx.response.body_json") {
            let json_path = path.strip_prefix("_ctx.response.body_json.")?;
            return get_json_value(body_json, json_path);
        }
    }

    None
}

/// Get a value from JSON by path (e.g., "foo.bar.baz")
fn get_json_value(json: &serde_json::Value, path: &str) -> Option<String> {
    let parts: Vec<&str> = path.split('.').collect();
    let mut current = json;

    for part in parts {
        match current {
            serde_json::Value::Object(map) => {
                current = map.get(part)?;
            }
            serde_json::Value::Array(arr) => {
                let idx: usize = part.parse().ok()?;
                current = arr.get(idx)?;
            }
            _ => return None,
        }
    }

    match current {
        serde_json::Value::String(s) => Some(s.clone()),
        serde_json::Value::Number(n) => Some(n.to_string()),
        serde_json::Value::Bool(b) => Some(b.to_string()),
        _ => Some(current.to_string()),
    }
}
