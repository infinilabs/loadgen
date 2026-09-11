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

//! HTTP request building for loadgen

use std::collections::HashMap;
use std::io::Write;

use anyhow::{Context, Result};
use flate2::write::GzEncoder;
use flate2::Compression;
use reqwest::header::{HeaderMap, HeaderName, HeaderValue, ACCEPT_ENCODING, CONTENT_ENCODING};
use reqwest::{Body, Method, Url};
use tracing::debug;

use crate::config::types::{Request, RequestItem, RunnerConfig};
use crate::template::TemplateEngine;
use crate::variable::provider::VariableProvider;

/// Request builder for loadgen
pub struct RequestBuilder<'a> {
    provider: &'a VariableProvider,
    runner_config: &'a RunnerConfig,
    compress: bool,
}

impl<'a> RequestBuilder<'a> {
    /// Create a new request builder
    pub fn new(
        provider: &'a VariableProvider,
        runner_config: &'a RunnerConfig,
        compress: bool,
    ) -> Self {
        Self {
            provider,
            runner_config,
            compress,
        }
    }

    /// Build a reqwest Request from a RequestItem
    pub fn build(
        &self,
        item: &RequestItem,
        runtime_vars: &mut HashMap<String, String>,
        client: &reqwest::Client,
    ) -> Result<reqwest::Request> {
        let request = item.request.as_ref().context("Request is None")?;

        // Process runtime variables
        for (key, var_name) in &request.runtime_variables {
            let value = self.provider.get_value(var_name, Some(runtime_vars));
            runtime_vars.insert(key.clone(), value);
        }

        // Build URL
        let url = self.build_url(request, runtime_vars)?;
        debug!("Request URL: {}", url);

        // Get method
        let method = self.parse_method(&request.method)?;

        // Start building request
        let mut builder = client.request(method, url);

        // Set basic auth
        builder = self.set_basic_auth(builder, request);

        // Build headers
        builder = self.set_headers(builder, request, runtime_vars)?;

        // Build body
        if !request.body.is_empty() {
            let body = self.build_body(request, runtime_vars)?;
            
            if self.compress && !body.is_empty() {
                // Compress body
                let mut encoder = GzEncoder::new(Vec::new(), Compression::best());
                encoder.write_all(body.as_bytes())?;
                let compressed = encoder.finish()?;
                
                builder = builder
                    .header(CONTENT_ENCODING, "gzip")
                    .header(ACCEPT_ENCODING, "gzip")
                    .header("X-PayLoad-Compressed", "true")
                    .body(Body::from(compressed));
            } else {
                builder = builder.body(Body::from(body.clone()));
            }
            
            builder = builder.header("X-PayLoad-Size", body.len().to_string());
        }

        builder.build().context("Failed to build request")
    }

    /// Build the URL with template substitution
    fn build_url(
        &self,
        request: &Request,
        runtime_vars: &HashMap<String, String>,
    ) -> Result<Url> {
        let url_str = if TemplateEngine::has_template(&request.url) {
            TemplateEngine::execute(&request.url, |var| {
                self.provider.get_value(var, Some(runtime_vars))
            })
        } else {
            request.url.clone()
        };

        // If URL doesn't have a host, use default endpoint
        if url_str.starts_with('/') {
            if self.runner_config.default_endpoint.is_empty() {
                anyhow::bail!("URL is relative but no default_endpoint configured: {}", url_str);
            }
            let full_url = format!("{}{}", self.runner_config.default_endpoint.trim_end_matches('/'), url_str);
            Url::parse(&full_url).context("Failed to parse URL with default endpoint")
        } else {
            Url::parse(&url_str).context("Failed to parse URL")
        }
    }

    /// Parse HTTP method
    fn parse_method(&self, method: &str) -> Result<Method> {
        match method.to_uppercase().as_str() {
            "GET" => Ok(Method::GET),
            "POST" => Ok(Method::POST),
            "PUT" => Ok(Method::PUT),
            "DELETE" => Ok(Method::DELETE),
            "PATCH" => Ok(Method::PATCH),
            "HEAD" => Ok(Method::HEAD),
            "OPTIONS" => Ok(Method::OPTIONS),
            _ => anyhow::bail!("Unsupported HTTP method: {}", method),
        }
    }

    /// Set basic auth on request
    fn set_basic_auth(
        &self,
        mut builder: reqwest::RequestBuilder,
        request: &Request,
    ) -> reqwest::RequestBuilder {
        // Check request-level auth first
        if let Some(ref auth) = request.basic_auth {
            if !auth.username.is_empty() {
                builder = builder.basic_auth(&auth.username, Some(&auth.password));
                return builder;
            }
        }

        // Fall back to default auth
        if let Some(ref auth) = self.runner_config.default_basic_auth {
            if !auth.username.is_empty() {
                builder = builder.basic_auth(&auth.username, Some(&auth.password));
            }
        }

        builder
    }

    /// Set headers on request
    fn set_headers(
        &self,
        mut builder: reqwest::RequestBuilder,
        request: &Request,
        runtime_vars: &HashMap<String, String>,
    ) -> Result<reqwest::RequestBuilder> {
        let mut headers = HeaderMap::new();

        for header_map in &request.headers {
            for (key, value) in header_map {
                let header_value = if TemplateEngine::has_template(value) {
                    TemplateEngine::execute(value, |var| {
                        self.provider.get_value(var, Some(runtime_vars))
                    })
                } else {
                    value.clone()
                };

                let name = HeaderName::try_from(key.as_str())
                    .context(format!("Invalid header name: {}", key))?;
                let value = HeaderValue::from_str(&header_value)
                    .context(format!("Invalid header value: {}", header_value))?;
                headers.insert(name, value);
            }
        }

        builder = builder.headers(headers);
        Ok(builder)
    }

    /// Build request body with template substitution and body repetition
    fn build_body(
        &self,
        request: &Request,
        runtime_vars: &mut HashMap<String, String>,
    ) -> Result<String> {
        let repeat_times = if request.repeat_body_n_times > 0 {
            request.repeat_body_n_times as usize
        } else {
            1
        };

        let mut result = String::new();

        for _ in 0..repeat_times {
            // Process runtime body line variables for each repetition
            for (key, var_name) in &request.runtime_body_line_variables {
                let value = self.provider.get_value(var_name, Some(runtime_vars));
                runtime_vars.insert(key.clone(), value);
            }

            let body = if TemplateEngine::has_template(&request.body) {
                TemplateEngine::execute(&request.body, |var| {
                    self.provider.get_value(var, Some(runtime_vars))
                })
            } else {
                request.body.clone()
            };

            result.push_str(&body);
        }

        Ok(result)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_parse_method() {
        let provider = VariableProvider::new(&[], HashMap::new());
        let runner_config = RunnerConfig::default();
        let builder = RequestBuilder::new(&provider, &runner_config, false);

        assert_eq!(builder.parse_method("GET").unwrap(), Method::GET);
        assert_eq!(builder.parse_method("post").unwrap(), Method::POST);
        assert_eq!(builder.parse_method("Put").unwrap(), Method::PUT);
        assert!(builder.parse_method("INVALID").is_err());
    }

    #[test]
    fn test_build_url_absolute() {
        let provider = VariableProvider::new(&[], HashMap::new());
        let runner_config = RunnerConfig::default();
        let builder = RequestBuilder::new(&provider, &runner_config, false);

        let request = Request {
            url: "http://localhost:9200/test".to_string(),
            ..Default::default()
        };

        let url = builder.build_url(&request, &HashMap::new()).unwrap();
        assert_eq!(url.to_string(), "http://localhost:9200/test");
    }

    #[test]
    fn test_build_url_relative() {
        let provider = VariableProvider::new(&[], HashMap::new());
        let runner_config = RunnerConfig {
            default_endpoint: "http://localhost:9200".to_string(),
            ..Default::default()
        };
        let builder = RequestBuilder::new(&provider, &runner_config, false);

        let request = Request {
            url: "/test/_search".to_string(),
            ..Default::default()
        };

        let url = builder.build_url(&request, &HashMap::new()).unwrap();
        assert_eq!(url.to_string(), "http://localhost:9200/test/_search");
    }

    #[test]
    fn test_build_url_with_template() {
        let mut env_vars = HashMap::new();
        env_vars.insert("ES_ENDPOINT".to_string(), "http://localhost:9200".to_string());
        
        let provider = VariableProvider::new(&[], env_vars);
        let runner_config = RunnerConfig::default();
        let builder = RequestBuilder::new(&provider, &runner_config, false);

        let request = Request {
            url: "$[[env.ES_ENDPOINT]]/test".to_string(),
            ..Default::default()
        };

        let url = builder.build_url(&request, &HashMap::new()).unwrap();
        assert_eq!(url.to_string(), "http://localhost:9200/test");
    }
}
