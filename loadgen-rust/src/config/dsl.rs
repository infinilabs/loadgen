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

//! DSL parser for loadgen
//!
//! The DSL format supports:
//! - Comments starting with #
//! - JSON configuration in comments (# runner: {...}, # variables: [...])
//! - HTTP method and path on a single line (GET /path, POST /path)
//! - Request body on following lines until next HTTP method or comment
//! - Expected status code comment (# 200)
//! - Request configuration comment (# request: {...})

use anyhow::{Context, Result};
use regex::Regex;
use std::path::Path;
use tracing::debug;

use super::types::{AppConfig, LoaderConfig, Request, RequestItem, RunnerConfig, Variable};

/// Parse a DSL file and return a LoaderConfig
pub async fn parse_dsl_file(path: &Path, app_config: &AppConfig) -> Result<LoaderConfig> {
    let contents = tokio::fs::read_to_string(path)
        .await
        .with_context(|| format!("Failed to read DSL file: {:?}", path))?;

    parse_dsl(&contents, app_config)
}

/// Parse DSL content and return a LoaderConfig
pub fn parse_dsl(content: &str, app_config: &AppConfig) -> Result<LoaderConfig> {
    let mut config = LoaderConfig {
        runner: app_config.runner.clone(),
        variables: app_config.variables.clone(),
        requests: Vec::new(),
    };

    // Regex patterns
    let http_method_re = Regex::new(r"^(GET|POST|PUT|DELETE|PATCH|HEAD|OPTIONS)\s+(.+)$").unwrap();
    let json_comment_re = Regex::new(r"^#\s*(runner|variables|request):\s*(.+)$").unwrap();
    let status_comment_re = Regex::new(r"^#\s*(\d{3})\s*$").unwrap();

    let mut current_request: Option<RequestItem> = None;
    let mut current_body = String::new();
    let mut in_body = false;
    let mut pending_json_config: Option<(String, String)> = None;
    let mut multiline_json = String::new();
    let mut in_multiline_json = false;
    let mut json_brace_count = 0;
    let mut json_bracket_count = 0;
    let mut json_type = String::new();

    for line in content.lines() {
        let trimmed = line.trim();

        // Handle multiline JSON in comments
        if in_multiline_json {
            // Continue collecting JSON
            if trimmed.starts_with('#') {
                let json_part = trimmed.trim_start_matches('#').trim();
                multiline_json.push_str(json_part);
                multiline_json.push('\n');

                // Count braces/brackets
                for c in json_part.chars() {
                    match c {
                        '{' => json_brace_count += 1,
                        '}' => json_brace_count -= 1,
                        '[' => json_bracket_count += 1,
                        ']' => json_bracket_count -= 1,
                        _ => {}
                    }
                }

                // Check if JSON is complete
                if json_brace_count == 0 && json_bracket_count == 0 {
                    in_multiline_json = false;
                    pending_json_config = Some((json_type.clone(), multiline_json.clone()));
                    multiline_json.clear();
                }
            } else {
                // Not a comment, JSON is malformed
                in_multiline_json = false;
                multiline_json.clear();
            }
            continue;
        }

        // Check for JSON configuration comments
        if let Some(caps) = json_comment_re.captures(trimmed) {
            let config_type = caps.get(1).unwrap().as_str();
            let json_str = caps.get(2).unwrap().as_str();

            // Count opening braces/brackets
            json_brace_count = 0;
            json_bracket_count = 0;
            for c in json_str.chars() {
                match c {
                    '{' => json_brace_count += 1,
                    '}' => json_brace_count -= 1,
                    '[' => json_bracket_count += 1,
                    ']' => json_bracket_count -= 1,
                    _ => {}
                }
            }

            if json_brace_count == 0 && json_bracket_count == 0 {
                // Complete JSON on one line
                pending_json_config = Some((config_type.to_string(), json_str.to_string()));
            } else {
                // Multiline JSON
                in_multiline_json = true;
                json_type = config_type.to_string();
                multiline_json = json_str.to_string();
                multiline_json.push('\n');
            }
            continue;
        }

        // Process pending JSON config
        if let Some((config_type, json_str)) = pending_json_config.take() {
            match config_type.as_str() {
                "runner" => {
                    if let Ok(runner) = serde_json::from_str::<RunnerConfig>(&json_str) {
                        config.runner = runner;
                    }
                }
                "variables" => {
                    if let Ok(vars) = serde_json::from_str::<Vec<Variable>>(&json_str) {
                        config.variables.extend(vars);
                    }
                }
                "request" => {
                    if let Some(ref mut req) = current_request {
                        if let Ok(request) = serde_json::from_str::<Request>(&json_str) {
                            if let Some(ref mut existing) = req.request {
                                // Merge request config
                                if !request.runtime_variables.is_empty() {
                                    existing.runtime_variables = request.runtime_variables;
                                }
                                if !request.runtime_body_line_variables.is_empty() {
                                    existing.runtime_body_line_variables = request.runtime_body_line_variables;
                                }
                                if request.repeat_body_n_times > 0 {
                                    existing.repeat_body_n_times = request.repeat_body_n_times;
                                }
                                if request.basic_auth.is_some() {
                                    existing.basic_auth = request.basic_auth;
                                }
                            }
                        }
                    }
                }
                _ => {}
            }
        }

        // Skip empty lines
        if trimmed.is_empty() {
            if in_body {
                current_body.push('\n');
            }
            continue;
        }

        // Skip regular comments (not JSON config)
        if trimmed.starts_with('#') {
            // Check for status code comment
            if let Some(caps) = status_comment_re.captures(trimmed) {
                // This is an expected status code, could be used for assertions
                let _status: i32 = caps.get(1).unwrap().as_str().parse().unwrap_or(200);
                // We don't do anything with status code comments for now
            }

            // Check for //  style comments (DSL comments)
            if trimmed.starts_with("# //") || trimmed.starts_with("#//") {
                continue;
            }

            // End body collection on regular comment
            if in_body && !current_body.is_empty() {
                if let Some(ref mut req) = current_request {
                    if let Some(ref mut request) = req.request {
                        request.body = current_body.trim_end().to_string();
                    }
                }
                current_body.clear();
                in_body = false;
            }
            continue;
        }

        // Check for HTTP method line
        if let Some(caps) = http_method_re.captures(trimmed) {
            // Save previous request
            if let Some(mut req) = current_request.take() {
                if in_body && !current_body.is_empty() {
                    if let Some(ref mut request) = req.request {
                        request.body = current_body.trim_end().to_string();
                    }
                }
                config.requests.push(req);
            }
            current_body.clear();

            let method = caps.get(1).unwrap().as_str();
            let path = caps.get(2).unwrap().as_str();

            current_request = Some(RequestItem {
                request: Some(Request {
                    method: method.to_string(),
                    url: path.to_string(),
                    ..Default::default()
                }),
                ..Default::default()
            });
            in_body = true;
            continue;
        }

        // Collect body lines
        if in_body {
            if !current_body.is_empty() {
                current_body.push('\n');
            }
            current_body.push_str(line);
        }
    }

    // Save last request
    if let Some(mut req) = current_request.take() {
        if in_body && !current_body.is_empty() {
            if let Some(ref mut request) = req.request {
                request.body = current_body.trim_end().to_string();
            }
        }
        config.requests.push(req);
    }

    debug!("Parsed {} requests from DSL", config.requests.len());
    Ok(config)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_parse_simple_dsl() {
        let dsl = r#"
DELETE /medcl

PUT /medcl

POST /medcl/_doc/1
{
 "name": "medcl"
}

GET /medcl/_search
"#;

        let app_config = AppConfig::default();
        let config = parse_dsl(dsl, &app_config).unwrap();

        assert_eq!(config.requests.len(), 4);
        
        let req0 = config.requests[0].request.as_ref().unwrap();
        assert_eq!(req0.method, "DELETE");
        assert_eq!(req0.url, "/medcl");

        let req2 = config.requests[2].request.as_ref().unwrap();
        assert_eq!(req2.method, "POST");
        assert_eq!(req2.url, "/medcl/_doc/1");
        assert!(req2.body.contains("medcl"));
    }

    #[test]
    fn test_parse_dsl_with_runner_config() {
        let dsl = r#"
# runner: {"total_rounds": 5, "no_warm": true}

GET /test
"#;

        let app_config = AppConfig::default();
        let config = parse_dsl(dsl, &app_config).unwrap();

        assert_eq!(config.runner.total_rounds, 5);
        assert!(config.runner.no_warm);
    }

    #[test]
    fn test_parse_dsl_with_variables() {
        let dsl = r#"
# variables: [{"name": "id", "type": "sequence"}]

POST /test/$[[id]]
"#;

        let app_config = AppConfig::default();
        let config = parse_dsl(dsl, &app_config).unwrap();

        assert_eq!(config.variables.len(), 1);
        assert_eq!(config.variables[0].name, "id");
        assert_eq!(config.variables[0].var_type, "sequence");
    }
}
