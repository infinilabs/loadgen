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

//! YAML configuration loader

use anyhow::{Context, Result};
use std::path::Path;
use tracing::debug;

use super::types::AppConfig;

/// Load configuration from a YAML file
pub async fn load_config(path: &Path) -> Result<AppConfig> {
    debug!("Loading config from {:?}", path);

    // Check if file exists
    if !path.exists() {
        // Return default config if file doesn't exist
        debug!("Config file not found, using defaults");
        return Ok(AppConfig::default());
    }

    // Read file contents
    let contents = tokio::fs::read_to_string(path)
        .await
        .with_context(|| format!("Failed to read config file: {:?}", path))?;

    // Parse YAML
    let mut config: AppConfig = serde_yaml::from_str(&contents)
        .with_context(|| format!("Failed to parse config file: {:?}", path))?;

    // Merge system environment variables
    for (key, value) in std::env::vars() {
        if config.environments.contains_key(&key) {
            config.environments.insert(key, value);
        }
    }

    debug!("Loaded config with {} requests", config.requests.len());
    Ok(config)
}

#[cfg(test)]
mod tests {
    use super::*;
    use std::io::Write;
    use tempfile::NamedTempFile;

    #[tokio::test]
    async fn test_load_config_not_found() {
        let result = load_config(Path::new("nonexistent.yml")).await;
        assert!(result.is_ok());
    }

    #[tokio::test]
    async fn test_load_config_basic() {
        let mut file = NamedTempFile::new().unwrap();
        writeln!(
            file,
            r#"
env:
  TEST_VAR: test_value

runner:
  no_warm: true
  total_rounds: 5

variables:
  - name: test_id
    type: sequence

requests:
  - request:
      method: GET
      url: http://localhost:8080/test
"#
        )
        .unwrap();

        let result = load_config(file.path()).await;
        assert!(result.is_ok());

        let config = result.unwrap();
        assert_eq!(config.environments.get("TEST_VAR"), Some(&"test_value".to_string()));
        assert!(config.runner.no_warm);
        assert_eq!(config.runner.total_rounds, 5);
        assert_eq!(config.variables.len(), 1);
        assert_eq!(config.requests.len(), 1);
    }

    #[tokio::test]
    async fn test_load_config_with_basic_auth() {
        let mut file = NamedTempFile::new().unwrap();
        writeln!(
            file,
            r#"
runner:
  default_basic_auth:
    username: admin
    password: secret

requests:
  - request:
      method: POST
      url: http://localhost:8080/api
      basic_auth:
        username: user
        password: pass
"#
        )
        .unwrap();

        let result = load_config(file.path()).await;
        assert!(result.is_ok());

        let config = result.unwrap();
        assert!(config.runner.default_basic_auth.is_some());
        let auth = config.runner.default_basic_auth.unwrap();
        assert_eq!(auth.username, "admin");
        assert_eq!(auth.password, "secret");
    }
}
