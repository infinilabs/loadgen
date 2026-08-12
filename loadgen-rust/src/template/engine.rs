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

//! Template engine for variable substitution
//!
//! Supports the $[[variable_name]] syntax used by loadgen

use std::collections::HashMap;

/// Template engine for processing variable substitutions
pub struct TemplateEngine;

/// Start delimiter for variables
const START_DELIM: &str = "$[[";
/// End delimiter for variables
const END_DELIM: &str = "]]";

impl TemplateEngine {
    /// Check if a string contains template variables
    pub fn has_template(input: &str) -> bool {
        input.contains(START_DELIM)
    }

    /// Execute template substitution with a callback function for variable resolution
    pub fn execute<F>(input: &str, resolver: F) -> String
    where
        F: Fn(&str) -> String,
    {
        if !Self::has_template(input) {
            return input.to_string();
        }

        let mut result = String::with_capacity(input.len());
        let mut remaining = input;

        while let Some(start_idx) = remaining.find(START_DELIM) {
            // Append text before the variable
            result.push_str(&remaining[..start_idx]);

            // Find the end of the variable
            let after_start = &remaining[start_idx + START_DELIM.len()..];
            if let Some(end_idx) = after_start.find(END_DELIM) {
                let var_name = &after_start[..end_idx];
                let value = resolver(var_name);
                result.push_str(&value);
                remaining = &after_start[end_idx + END_DELIM.len()..];
            } else {
                // No closing delimiter found, append the rest and break
                result.push_str(&remaining[start_idx..]);
                remaining = "";
                break;
            }
        }

        // Append remaining text
        result.push_str(remaining);
        result
    }

    /// Execute template substitution with a HashMap of variables
    pub fn execute_with_map(input: &str, variables: &HashMap<String, String>) -> String {
        Self::execute(input, |name| {
            variables
                .get(name)
                .cloned()
                .unwrap_or_else(|| format!("$[[{}]]", name))
        })
    }

    /// Extract all variable names from a template string
    pub fn extract_variables(input: &str) -> Vec<String> {
        let mut variables = Vec::new();
        let mut remaining = input;

        while let Some(start_idx) = remaining.find(START_DELIM) {
            let after_start = &remaining[start_idx + START_DELIM.len()..];
            if let Some(end_idx) = after_start.find(END_DELIM) {
                let var_name = &after_start[..end_idx];
                variables.push(var_name.to_string());
                remaining = &after_start[end_idx + END_DELIM.len()..];
            } else {
                break;
            }
        }

        variables
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_has_template() {
        assert!(TemplateEngine::has_template("Hello $[[name]]!"));
        assert!(TemplateEngine::has_template("$[[a]] $[[b]]"));
        assert!(!TemplateEngine::has_template("Hello world!"));
        assert!(!TemplateEngine::has_template("$[name]"));
    }

    #[test]
    fn test_execute_simple() {
        let result = TemplateEngine::execute("Hello $[[name]]!", |var| {
            if var == "name" {
                "World".to_string()
            } else {
                format!("unknown:{}", var)
            }
        });
        assert_eq!(result, "Hello World!");
    }

    #[test]
    fn test_execute_multiple() {
        let result = TemplateEngine::execute("$[[a]] and $[[b]]", |var| {
            match var {
                "a" => "first".to_string(),
                "b" => "second".to_string(),
                _ => "unknown".to_string(),
            }
        });
        assert_eq!(result, "first and second");
    }

    #[test]
    fn test_execute_no_template() {
        let result = TemplateEngine::execute("No template here", |_| "unused".to_string());
        assert_eq!(result, "No template here");
    }

    #[test]
    fn test_execute_with_map() {
        let mut vars = HashMap::new();
        vars.insert("name".to_string(), "Alice".to_string());
        vars.insert("greeting".to_string(), "Hello".to_string());

        let result = TemplateEngine::execute_with_map("$[[greeting]] $[[name]]!", &vars);
        assert_eq!(result, "Hello Alice!");
    }

    #[test]
    fn test_execute_missing_variable() {
        let vars = HashMap::new();
        let result = TemplateEngine::execute_with_map("Hello $[[name]]!", &vars);
        assert_eq!(result, "Hello $[[name]]!");
    }

    #[test]
    fn test_extract_variables() {
        let vars = TemplateEngine::extract_variables("$[[a]] and $[[b.c]] = $[[d]]");
        assert_eq!(vars, vec!["a", "b.c", "d"]);
    }

    #[test]
    fn test_execute_nested_json() {
        let result = TemplateEngine::execute(
            r#"{"id": "$[[id]]", "name": "$[[name]]"}"#,
            |var| match var {
                "id" => "123".to_string(),
                "name" => "test".to_string(),
                _ => "".to_string(),
            },
        );
        assert_eq!(result, r#"{"id": "123", "name": "test"}"#);
    }

    #[test]
    fn test_execute_env_variable() {
        let result = TemplateEngine::execute(
            "http://$[[env.ES_HOST]]:$[[env.ES_PORT]]/index",
            |var| match var {
                "env.ES_HOST" => "localhost".to_string(),
                "env.ES_PORT" => "9200".to_string(),
                _ => "".to_string(),
            },
        );
        assert_eq!(result, "http://localhost:9200/index");
    }

    #[test]
    fn test_execute_unclosed_delimiter() {
        let result = TemplateEngine::execute("Hello $[[name", |_| "World".to_string());
        assert_eq!(result, "Hello $[[name");
    }
}
