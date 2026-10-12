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

//! Configuration type definitions matching the Go implementation

use serde::{Deserialize, Serialize};
use std::collections::HashMap;

/// Basic authentication credentials
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct BasicAuth {
    #[serde(default)]
    pub username: String,
    #[serde(default)]
    pub password: String,
}

/// HTTP request definition
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct Request {
    /// HTTP method (GET, POST, PUT, DELETE, etc.)
    #[serde(default)]
    pub method: String,

    /// Request URL (can contain template variables)
    #[serde(default)]
    pub url: String,

    /// Request body (can contain template variables)
    #[serde(default)]
    pub body: String,

    /// Simple mode - no variable substitution
    #[serde(default)]
    pub simple_mode: bool,

    /// Number of times to repeat the body in a single request
    #[serde(default, alias = "body_repeat_times")]
    pub repeat_body_n_times: i32,

    /// HTTP headers
    #[serde(default)]
    pub headers: Vec<HashMap<String, String>>,

    /// Basic authentication for this request
    #[serde(default)]
    pub basic_auth: Option<BasicAuth>,

    /// Disable header names normalizing
    #[serde(default)]
    pub disable_header_names_normalizing: bool,

    /// Runtime variables to set before executing the request
    #[serde(default)]
    pub runtime_variables: HashMap<String, String>,

    /// Runtime variables to set per body line
    #[serde(default)]
    pub runtime_body_line_variables: HashMap<String, String>,

    /// Number of times to execute this request
    #[serde(default)]
    pub execute_repeat_times: i32,
}

/// Variable definition
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct Variable {
    /// Variable type (file, list, sequence, uuid, etc.)
    #[serde(rename = "type")]
    pub var_type: String,

    /// Variable name
    #[serde(default)]
    pub name: String,

    /// File path for file type variables
    #[serde(default)]
    pub path: String,

    /// Inline data for list type variables
    #[serde(default)]
    pub data: Vec<String>,

    /// Date format for now_with_format type
    #[serde(default)]
    pub format: String,

    /// Range start for range/sequence types
    #[serde(default)]
    pub from: u64,

    /// Range end for range/sequence types
    #[serde(default)]
    pub to: u64,

    /// Character replacement map
    #[serde(default)]
    pub replace: HashMap<String, String>,

    /// Size for random_array type
    #[serde(default)]
    pub size: usize,

    /// Variable key for random_array type
    #[serde(default)]
    pub variable_key: String,

    /// Variable type for random_array (number/string)
    #[serde(default)]
    pub variable_type: String,

    /// Whether to include square brackets for random_array
    #[serde(default)]
    pub square_bracket: bool,

    /// String bracket character for random_array
    #[serde(default)]
    pub string_bracket: String,
}

/// Assertion condition configuration
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct AssertConfig {
    /// Equals assertion: field -> expected value
    #[serde(default)]
    pub equals: HashMap<String, serde_json::Value>,

    /// Not equals assertion
    #[serde(default)]
    pub not_equals: HashMap<String, serde_json::Value>,

    /// Contains assertion
    #[serde(default)]
    pub contains: HashMap<String, String>,

    /// Not contains assertion
    #[serde(default)]
    pub not_contains: HashMap<String, String>,

    /// Regex match assertion
    #[serde(default)]
    pub regexp: HashMap<String, String>,

    /// Regex not match assertion
    #[serde(default)]
    pub not_regexp: HashMap<String, String>,

    /// Range assertion
    #[serde(default)]
    pub range: HashMap<String, RangeCondition>,

    /// In list assertion
    #[serde(default, rename = "in")]
    pub in_list: HashMap<String, Vec<serde_json::Value>>,

    /// Not in list assertion
    #[serde(default)]
    pub not_in: HashMap<String, Vec<serde_json::Value>>,
}

/// Range condition for assertions
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct RangeCondition {
    #[serde(default)]
    pub gte: Option<f64>,
    #[serde(default)]
    pub gt: Option<f64>,
    #[serde(default)]
    pub lte: Option<f64>,
    #[serde(default)]
    pub lt: Option<f64>,
}

/// Sleep action between requests
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct SleepAction {
    #[serde(default)]
    pub sleep_in_milli_seconds: i64,
}

/// Request item with optional assertions
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct RequestItem {
    /// The HTTP request
    #[serde(default)]
    pub request: Option<Request>,

    /// Assertion conditions
    #[serde(default)]
    pub assert: Option<AssertConfig>,

    /// DSL-based assertion
    #[serde(default)]
    pub assert_dsl: String,

    /// Sleep action after request
    #[serde(default)]
    pub sleep: Option<SleepAction>,

    /// Register response values to global context
    #[serde(default)]
    pub register: Vec<HashMap<String, String>>,
}

/// Runner configuration
#[derive(Debug, Clone, Serialize, Deserialize)]
pub struct RunnerConfig {
    /// How many rounds of requests to run
    #[serde(default)]
    pub total_rounds: i64,

    /// Skip warmup round
    #[serde(default)]
    pub no_warm: bool,

    /// Valid status codes during warmup
    #[serde(default)]
    pub valid_status_codes_during_warmup: Vec<i32>,

    /// Exit(1) if any assertion failed
    #[serde(default)]
    pub assert_invalid: bool,

    /// Continue running on assertion failure
    #[serde(default)]
    pub continue_on_assert_invalid: bool,

    /// Skip invalid assertions
    #[serde(default)]
    pub skip_invalid_assert: bool,

    /// Exit(2) if any error occurred
    #[serde(default)]
    pub assert_error: bool,

    /// Log all requests
    #[serde(default)]
    pub log_requests: bool,

    /// Benchmark only mode
    #[serde(default)]
    pub benchmark_only: bool,

    /// Use microseconds for duration
    #[serde(default)]
    pub duration_in_us: bool,

    /// Disable stats collection
    #[serde(default)]
    pub no_stats: bool,

    /// Disable size stats
    #[serde(default)]
    pub no_size_stats: bool,

    /// Metric sample size for histogram
    #[serde(default = "default_metric_sample_size")]
    pub metric_sample_size: usize,

    /// Log requests with these status codes
    #[serde(default)]
    pub log_status_codes: Vec<i32>,

    /// Disable header name normalizing
    #[serde(default)]
    pub disable_header_names_normalizing: bool,

    /// Reset context before test run
    #[serde(default)]
    pub reset_context: bool,

    /// Default endpoint for requests
    #[serde(default)]
    pub default_endpoint: String,

    /// Default basic auth for requests
    #[serde(default)]
    pub default_basic_auth: Option<BasicAuth>,
}

fn default_metric_sample_size() -> usize {
    10000
}

impl Default for RunnerConfig {
    fn default() -> Self {
        Self {
            total_rounds: -1,
            no_warm: false,
            valid_status_codes_during_warmup: vec![],
            assert_invalid: false,
            continue_on_assert_invalid: false,
            skip_invalid_assert: false,
            assert_error: false,
            log_requests: false,
            benchmark_only: false,
            duration_in_us: false,
            no_stats: false,
            no_size_stats: false,
            metric_sample_size: 10000,
            log_status_codes: vec![],
            disable_header_names_normalizing: false,
            reset_context: false,
            default_endpoint: String::new(),
            default_basic_auth: None,
        }
    }
}

/// Test case configuration
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct Test {
    /// Directory path containing test configurations
    #[serde(default)]
    pub path: String,

    /// Whether to use compression
    #[serde(default)]
    pub compress: bool,
}

/// Loader configuration
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct LoaderConfig {
    /// Variable definitions
    #[serde(default)]
    pub variables: Vec<Variable>,

    /// Request definitions
    #[serde(default)]
    pub requests: Vec<RequestItem>,

    /// Runner configuration
    #[serde(default)]
    pub runner: RunnerConfig,
}

/// Application configuration
#[derive(Debug, Clone, Default, Serialize, Deserialize)]
pub struct AppConfig {
    /// Environment variables
    #[serde(default, alias = "env")]
    pub environments: HashMap<String, String>,

    /// Test cases
    #[serde(default)]
    pub tests: Vec<Test>,

    /// Variable definitions
    #[serde(default)]
    pub variables: Vec<Variable>,

    /// Request definitions
    #[serde(default)]
    pub requests: Vec<RequestItem>,

    /// Runner configuration
    #[serde(default)]
    pub runner: RunnerConfig,
}

impl AppConfig {
    /// Convert to LoaderConfig
    pub fn to_loader_config(&self) -> LoaderConfig {
        LoaderConfig {
            variables: self.variables.clone(),
            requests: self.requests.clone(),
            runner: self.runner.clone(),
        }
    }

    /// Check if environment variables exist
    pub fn test_env(&self, env_vars: &[&str]) -> bool {
        for env_var in env_vars {
            match self.environments.get(*env_var) {
                Some(v) if !v.is_empty() => continue,
                _ => return false,
            }
        }
        true
    }
}

/// Request result for statistics
#[derive(Debug, Clone, Default)]
pub struct RequestResult {
    pub request_count: i32,
    pub request_size: i64,
    pub response_size: i64,
    pub status: i32,
    pub error: bool,
    pub invalid: bool,
    pub duration: std::time::Duration,
}

impl RequestResult {
    pub fn reset(&mut self) {
        self.request_count = 0;
        self.request_size = 0;
        self.response_size = 0;
        self.status = 0;
        self.error = false;
        self.invalid = false;
        self.duration = std::time::Duration::ZERO;
    }
}
