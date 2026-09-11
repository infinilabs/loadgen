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

//! Assertion conditions for response validation

use std::collections::HashMap;

use regex::Regex;
use tracing::debug;

use crate::config::types::AssertConfig;

/// Assertion checker for validating responses
pub struct AssertionChecker<'a> {
    ctx: &'a HashMap<String, serde_json::Value>,
}

impl<'a> AssertionChecker<'a> {
    /// Create a new assertion checker
    pub fn new(ctx: &'a HashMap<String, serde_json::Value>) -> Self {
        Self { ctx }
    }

    /// Check all assertions in the config
    pub fn check(&self, config: &AssertConfig) -> bool {
        // Check equals
        for (field, expected) in &config.equals {
            if !self.check_equals(field, expected) {
                debug!("equals assertion failed for {}: expected {:?}", field, expected);
                return false;
            }
        }

        // Check not_equals
        for (field, expected) in &config.not_equals {
            if self.check_equals(field, expected) {
                debug!("not_equals assertion failed for {}", field);
                return false;
            }
        }

        // Check contains
        for (field, expected) in &config.contains {
            if !self.check_contains(field, expected) {
                debug!("contains assertion failed for {}: expected to contain {:?}", field, expected);
                return false;
            }
        }

        // Check not_contains
        for (field, expected) in &config.not_contains {
            if self.check_contains(field, expected) {
                debug!("not_contains assertion failed for {}", field);
                return false;
            }
        }

        // Check regexp
        for (field, pattern) in &config.regexp {
            if !self.check_regexp(field, pattern) {
                debug!("regexp assertion failed for {}: pattern {:?}", field, pattern);
                return false;
            }
        }

        // Check not_regexp
        for (field, pattern) in &config.not_regexp {
            if self.check_regexp(field, pattern) {
                debug!("not_regexp assertion failed for {}", field);
                return false;
            }
        }

        // Check range
        for (field, range) in &config.range {
            if !self.check_range(field, range) {
                debug!("range assertion failed for {}", field);
                return false;
            }
        }

        // Check in
        for (field, values) in &config.in_list {
            if !self.check_in(field, values) {
                debug!("in assertion failed for {}", field);
                return false;
            }
        }

        // Check not_in
        for (field, values) in &config.not_in {
            if self.check_in(field, values) {
                debug!("not_in assertion failed for {}", field);
                return false;
            }
        }

        true
    }

    /// Get a value from the context
    fn get_value(&self, field: &str) -> Option<&serde_json::Value> {
        // Direct lookup
        if let Some(value) = self.ctx.get(field) {
            return Some(value);
        }

        // Try nested lookup for body_json paths
        if field.starts_with("_ctx.response.body_json.") {
            if let Some(body_json) = self.ctx.get("_ctx.response.body_json") {
                let path = field.strip_prefix("_ctx.response.body_json.")?;
                return self.get_nested_value(body_json, path);
            }
        }

        None
    }

    /// Get a nested value from JSON
    fn get_nested_value<'b>(&self, json: &'b serde_json::Value, path: &str) -> Option<&'b serde_json::Value> {
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

        Some(current)
    }

    /// Check equals assertion
    fn check_equals(&self, field: &str, expected: &serde_json::Value) -> bool {
        match self.get_value(field) {
            Some(actual) => actual == expected,
            None => false,
        }
    }

    /// Check contains assertion
    fn check_contains(&self, field: &str, expected: &str) -> bool {
        match self.get_value(field) {
            Some(serde_json::Value::String(s)) => s.contains(expected),
            Some(v) => v.to_string().contains(expected),
            None => false,
        }
    }

    /// Check regexp assertion
    fn check_regexp(&self, field: &str, pattern: &str) -> bool {
        let regex = match Regex::new(pattern) {
            Ok(r) => r,
            Err(_) => return false,
        };

        match self.get_value(field) {
            Some(serde_json::Value::String(s)) => regex.is_match(s),
            Some(v) => regex.is_match(&v.to_string()),
            None => false,
        }
    }

    /// Check range assertion
    fn check_range(&self, field: &str, range: &crate::config::types::RangeCondition) -> bool {
        let value = match self.get_value(field) {
            Some(serde_json::Value::Number(n)) => n.as_f64(),
            _ => None,
        };

        let value = match value {
            Some(v) => v,
            None => return false,
        };

        if let Some(gte) = range.gte {
            if value < gte {
                return false;
            }
        }

        if let Some(gt) = range.gt {
            if value <= gt {
                return false;
            }
        }

        if let Some(lte) = range.lte {
            if value > lte {
                return false;
            }
        }

        if let Some(lt) = range.lt {
            if value >= lt {
                return false;
            }
        }

        true
    }

    /// Check in assertion
    fn check_in(&self, field: &str, values: &[serde_json::Value]) -> bool {
        match self.get_value(field) {
            Some(actual) => values.contains(actual),
            None => false,
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    fn make_ctx(pairs: Vec<(&str, serde_json::Value)>) -> HashMap<String, serde_json::Value> {
        pairs.into_iter().map(|(k, v)| (k.to_string(), v)).collect()
    }

    #[test]
    fn test_check_equals() {
        let ctx = make_ctx(vec![
            ("_ctx.response.status", serde_json::json!(200)),
            ("_ctx.response.body", serde_json::json!("hello")),
        ]);

        let checker = AssertionChecker::new(&ctx);

        assert!(checker.check_equals("_ctx.response.status", &serde_json::json!(200)));
        assert!(!checker.check_equals("_ctx.response.status", &serde_json::json!(201)));
        assert!(checker.check_equals("_ctx.response.body", &serde_json::json!("hello")));
    }

    #[test]
    fn test_check_contains() {
        let ctx = make_ctx(vec![
            ("_ctx.response.body", serde_json::json!("hello world")),
        ]);

        let checker = AssertionChecker::new(&ctx);

        assert!(checker.check_contains("_ctx.response.body", "world"));
        assert!(!checker.check_contains("_ctx.response.body", "foo"));
    }

    #[test]
    fn test_check_regexp() {
        let ctx = make_ctx(vec![
            ("_ctx.response.body", serde_json::json!("hello123world")),
        ]);

        let checker = AssertionChecker::new(&ctx);

        assert!(checker.check_regexp("_ctx.response.body", r"\d+"));
        assert!(!checker.check_regexp("_ctx.response.body", r"^\d+$"));
    }

    #[test]
    fn test_check_range() {
        let ctx = make_ctx(vec![
            ("_ctx.elapsed", serde_json::json!(150)),
        ]);

        let checker = AssertionChecker::new(&ctx);

        let range = crate::config::types::RangeCondition {
            gte: Some(100.0),
            lte: Some(200.0),
            ..Default::default()
        };

        assert!(checker.check_range("_ctx.elapsed", &range));

        let range2 = crate::config::types::RangeCondition {
            gt: Some(150.0),
            ..Default::default()
        };

        assert!(!checker.check_range("_ctx.elapsed", &range2));
    }

    #[test]
    fn test_check_in() {
        let ctx = make_ctx(vec![
            ("_ctx.response.status", serde_json::json!(200)),
        ]);

        let checker = AssertionChecker::new(&ctx);

        assert!(checker.check_in(
            "_ctx.response.status",
            &[serde_json::json!(200), serde_json::json!(201)]
        ));
        assert!(!checker.check_in(
            "_ctx.response.status",
            &[serde_json::json!(400), serde_json::json!(500)]
        ));
    }

    #[test]
    fn test_check_config() {
        let ctx = make_ctx(vec![
            ("_ctx.response.status", serde_json::json!(200)),
            ("_ctx.response.body", serde_json::json!("success")),
        ]);

        let checker = AssertionChecker::new(&ctx);

        let mut config = AssertConfig::default();
        config.equals.insert("_ctx.response.status".to_string(), serde_json::json!(200));
        config.contains.insert("_ctx.response.body".to_string(), "success".to_string());

        assert!(checker.check(&config));

        config.equals.insert("_ctx.response.status".to_string(), serde_json::json!(201));
        assert!(!checker.check(&config));
    }
}
