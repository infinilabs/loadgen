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

//! HTTP client wrapper for loadgen

use std::time::Duration;

use anyhow::Result;
use reqwest::{Client, Response};
use tracing::debug;

/// HTTP client configuration
#[derive(Debug, Clone)]
pub struct HttpClientConfig {
    /// Request timeout in seconds (0 = no timeout)
    pub timeout: u64,
    /// Read timeout in seconds
    pub read_timeout: u64,
    /// Write timeout in seconds
    pub write_timeout: u64,
    /// Dial/connect timeout in seconds
    pub dial_timeout: u64,
    /// Enable gzip compression
    pub compress: bool,
    /// Disable header name normalization
    pub disable_header_normalizing: bool,
    /// Maximum connections per host
    pub max_connections_per_host: usize,
    /// User agent string
    pub user_agent: String,
}

impl Default for HttpClientConfig {
    fn default() -> Self {
        Self {
            timeout: 0,
            read_timeout: 0,
            write_timeout: 0,
            dial_timeout: 3,
            compress: false,
            disable_header_normalizing: false,
            max_connections_per_host: 100,
            user_agent: format!("loadgen-rust/{}", env!("CARGO_PKG_VERSION")),
        }
    }
}

/// HTTP client wrapper
pub struct HttpClient {
    client: Client,
    config: HttpClientConfig,
}

impl HttpClient {
    /// Create a new HTTP client
    pub fn new(config: HttpClientConfig) -> Result<Self> {
        let mut builder = Client::builder()
            .user_agent(&config.user_agent)
            .pool_max_idle_per_host(config.max_connections_per_host)
            .danger_accept_invalid_certs(true) // Match Go's InsecureSkipVerify
            .use_rustls_tls();

        // Set connect timeout
        if config.dial_timeout > 0 {
            builder = builder.connect_timeout(Duration::from_secs(config.dial_timeout));
        }

        // Set overall timeout
        let effective_timeout = if config.timeout > 0 {
            config.timeout
        } else if config.read_timeout > 0 {
            config.read_timeout
        } else {
            0
        };

        if effective_timeout > 0 {
            builder = builder.timeout(Duration::from_secs(effective_timeout));
        }

        // Enable gzip
        if config.compress {
            builder = builder.gzip(true);
        }

        let client = builder.build()?;
        debug!("HTTP client created with config: {:?}", config);

        Ok(Self { client, config })
    }

    /// Get the internal reqwest client
    pub fn client(&self) -> &Client {
        &self.client
    }

    /// Get the client configuration
    pub fn config(&self) -> &HttpClientConfig {
        &self.config
    }

    /// Execute a request and return the response
    pub async fn execute(&self, request: reqwest::Request) -> Result<Response> {
        let response = self.client.execute(request).await?;
        Ok(response)
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_default_config() {
        let config = HttpClientConfig::default();
        assert_eq!(config.timeout, 0);
        assert_eq!(config.dial_timeout, 3);
        assert!(!config.compress);
    }

    #[tokio::test]
    async fn test_create_client() {
        let config = HttpClientConfig::default();
        let client = HttpClient::new(config);
        assert!(client.is_ok());
    }

    #[tokio::test]
    async fn test_create_client_with_timeout() {
        let config = HttpClientConfig {
            timeout: 30,
            dial_timeout: 5,
            ..Default::default()
        };
        let client = HttpClient::new(config);
        assert!(client.is_ok());
    }
}
