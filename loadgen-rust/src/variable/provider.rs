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

//! Variable provider - generates values for variables

use std::collections::HashMap;
use std::fs::File;
use std::io::{BufRead, BufReader};
use std::sync::Arc;

use base64::{engine::general_purpose::STANDARD as BASE64, Engine};
use chrono::{Local, Utc};
use rand::Rng;
use roaring::RoaringBitmap;
use uuid::Uuid;

use super::types::{SequenceCounter32, SequenceCounter64, VariableType};
use crate::config::types::Variable;

/// Timestamp format for now_utc_lite
const TS_LAYOUT: &str = "%Y-%m-%dT%H:%M:%S%.3f";

/// Variable provider for generating variable values
pub struct VariableProvider {
    /// Variable definitions
    variables: HashMap<String, Variable>,
    /// Data loaded from files or inline definitions
    dict: HashMap<String, Vec<String>>,
    /// 32-bit sequence counters
    seq32: HashMap<String, Arc<SequenceCounter32>>,
    /// 64-bit sequence counters
    seq64: HashMap<String, Arc<SequenceCounter64>>,
    /// Character replacers
    replacers: HashMap<String, Vec<(String, String)>>,
    /// Environment variables
    env_vars: HashMap<String, String>,
}

impl VariableProvider {
    /// Create a new variable provider
    pub fn new(variables: &[Variable], env_vars: HashMap<String, String>) -> Self {
        let mut provider = VariableProvider {
            variables: HashMap::new(),
            dict: HashMap::new(),
            seq32: HashMap::new(),
            seq64: HashMap::new(),
            replacers: HashMap::new(),
            env_vars,
        };

        for var in variables {
            provider.register_variable(var.clone());
        }

        provider
    }

    /// Register a variable
    pub fn register_variable(&mut self, var: Variable) {
        let name = var.name.trim().to_string();

        // Load data from file if specified
        let mut lines: Vec<String> = Vec::new();

        if !var.path.is_empty() {
            if let Ok(file) = File::open(&var.path) {
                let reader = BufReader::new(file);
                for line in reader.lines().map_while(Result::ok) {
                    let trimmed = line.trim().to_string();
                    if !trimmed.is_empty() {
                        lines.push(trimmed);
                    }
                }
            }
        }

        // Add inline data
        for item in &var.data {
            let trimmed = item.trim().to_string();
            if !trimmed.is_empty() {
                lines.push(trimmed);
            }
        }

        // Store replacer if specified
        if !var.replace.is_empty() {
            let replacements: Vec<(String, String)> = var
                .replace
                .iter()
                .map(|(k, v)| (k.clone(), v.clone()))
                .collect();
            self.replacers.insert(name.clone(), replacements);
        }

        // Create sequence counters
        let var_type = VariableType::from(var.var_type.as_str());
        match var_type {
            VariableType::Sequence => {
                let counter = Arc::new(SequenceCounter32::new(var.from as u32, var.to as u32));
                self.seq32.insert(name.clone(), counter);
            }
            VariableType::Sequence64 => {
                let counter = Arc::new(SequenceCounter64::new(var.from, var.to));
                self.seq64.insert(name.clone(), counter);
            }
            _ => {}
        }

        self.dict.insert(name.clone(), lines);
        self.variables.insert(name, var);
    }

    /// Get a variable value
    pub fn get_value(&self, key: &str, runtime_vars: Option<&HashMap<String, String>>) -> String {
        // Check runtime variables first
        if let Some(vars) = runtime_vars {
            if let Some(value) = vars.get(key) {
                return value.clone();
            }
        }

        // Check for environment variable reference
        if let Some(env_key) = key.strip_prefix("env.") {
            return self.env_vars.get(env_key).cloned().unwrap_or_default();
        }

        // Get variable definition
        let var = match self.variables.get(key) {
            Some(v) => v,
            None => return "not_found".to_string(),
        };

        let raw_value = self.build_variable_value(var);

        // Apply replacer if exists
        if let Some(replacements) = self.replacers.get(key) {
            let mut result = raw_value;
            for (from, to) in replacements {
                result = result.replace(from, to);
            }
            return result;
        }

        raw_value
    }

    /// Build a variable value based on its type
    fn build_variable_value(&self, var: &Variable) -> String {
        let var_type = VariableType::from(var.var_type.as_str());

        match var_type {
            VariableType::Sequence => {
                if let Some(counter) = self.seq32.get(&var.name) {
                    return counter.next().to_string();
                }
                "0".to_string()
            }
            VariableType::Sequence64 => {
                if let Some(counter) = self.seq64.get(&var.name) {
                    return counter.next().to_string();
                }
                "0".to_string()
            }
            VariableType::Uuid => Uuid::new_v4().to_string(),
            VariableType::NowLocal => Local::now().to_string(),
            VariableType::NowUtc => Utc::now().to_string(),
            VariableType::NowUtcLite => Utc::now().format(TS_LAYOUT).to_string(),
            VariableType::NowUnix => Local::now().timestamp().to_string(),
            VariableType::NowUnixInMs => Local::now().timestamp_millis().to_string(),
            VariableType::NowUnixInMicro => Local::now().timestamp_micros().to_string(),
            VariableType::NowUnixInNano => {
                Local::now().timestamp_nanos_opt().unwrap_or(0).to_string()
            }
            VariableType::NowWithFormat => {
                if var.format.is_empty() {
                    panic!("Date format is not set for variable: {}", var.name);
                }
                // Convert Go date format to chrono format
                let format = convert_go_time_format(&var.format);
                Local::now().format(&format).to_string()
            }
            VariableType::Range => {
                let mut rng = rand::thread_rng();
                let range = var.to - var.from + 1;
                let value = rng.gen_range(0..range) + var.from;
                value.to_string()
            }
            VariableType::RandomArray => self.build_random_array(var),
            VariableType::IntArrayBitmap => self.build_int_array_bitmap(var),
            VariableType::File | VariableType::List => {
                if let Some(data) = self.dict.get(&var.name) {
                    if data.is_empty() {
                        return String::new();
                    }
                    if data.len() == 1 {
                        return data[0].clone();
                    }
                    let mut rng = rand::thread_rng();
                    let idx = rng.gen_range(0..data.len());
                    return data[idx].clone();
                }
                String::new()
            }
            VariableType::Env => {
                self.env_vars.get(&var.name).cloned().unwrap_or_default()
            }
        }
    }

    /// Build a random array value
    fn build_random_array(&self, var: &Variable) -> String {
        let mut result = String::new();

        if var.square_bracket {
            result.push('[');
        }

        if !var.variable_key.is_empty() && var.size > 0 {
            for i in 0..var.size {
                if i > 0 {
                    result.push(',');
                }

                // Get value from referenced variable
                let value = self.get_value(&var.variable_key, None);

                // Add string brackets if needed
                if var.variable_type == "string" {
                    let bracket = if var.string_bracket.is_empty() {
                        "\""
                    } else {
                        &var.string_bracket
                    };
                    result.push_str(bracket);
                    result.push_str(&value);
                    result.push_str(bracket);
                } else {
                    result.push_str(&value);
                }
            }
        }

        if var.square_bracket {
            result.push(']');
        }

        result
    }

    /// Build a roaring bitmap encoded integer array
    fn build_int_array_bitmap(&self, var: &Variable) -> String {
        let mut bitmap = RoaringBitmap::new();
        let mut rng = rand::thread_rng();

        if var.size > 0 {
            for _ in 0..var.size {
                let range = var.to - var.from + 1;
                let value = rng.gen_range(0..range) + var.from;
                bitmap.insert(value as u32);
            }
        }

        let mut bytes = Vec::new();
        bitmap.serialize_into(&mut bytes).unwrap_or_default();
        BASE64.encode(&bytes)
    }

    /// Clear all data (for reset_context)
    pub fn clear(&mut self) {
        self.dict.clear();
        self.seq32.clear();
        self.seq64.clear();
        self.variables.clear();
        self.replacers.clear();
    }
}

/// Convert Go time format to chrono strftime format
fn convert_go_time_format(go_format: &str) -> String {
    // Go uses a reference time: Mon Jan 2 15:04:05 MST 2006
    // This maps Go format specifiers to chrono format specifiers
    // Order matters - replace longer patterns first to avoid partial matches
    go_format
        // Year - must be first to avoid issues with other numbers
        .replace("2006", "%Y")
        .replace("06", "%y")
        // Month patterns - order by length (longer first)
        .replace("January", "%B")
        .replace("Jan", "%b")
        .replace("01", "%m")
        // Day patterns
        .replace("Monday", "%A")
        .replace("Mon", "%a")
        .replace("02", "%d")
        // Hour patterns - must be before minute patterns
        .replace("15", "%H")  // 24-hour
        .replace("03", "%I")  // 12-hour with leading zero
        // Minute patterns - must be after hour patterns
        .replace("04", "%M")
        // Second patterns
        .replace("05", "%S")
        // Timezone patterns
        .replace("-0700", "%z")
        .replace("-07:00", "%:z")
        .replace("-07", "%z")
        .replace("MST", "%Z")
        // AM/PM
        .replace("PM", "%p")
        .replace("pm", "%P")
        // Subseconds
        .replace(".000000000", "%.9f")
        .replace(".000000", "%.6f")
        .replace(".000", "%.3f")
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_sequence_variable() {
        let vars = vec![Variable {
            name: "id".to_string(),
            var_type: "sequence".to_string(),
            from: 1,
            to: 100,
            ..Default::default()
        }];

        let provider = VariableProvider::new(&vars, HashMap::new());
        
        assert_eq!(provider.get_value("id", None), "1");
        assert_eq!(provider.get_value("id", None), "2");
        assert_eq!(provider.get_value("id", None), "3");
    }

    #[test]
    fn test_uuid_variable() {
        let vars = vec![Variable {
            name: "uuid".to_string(),
            var_type: "uuid".to_string(),
            ..Default::default()
        }];

        let provider = VariableProvider::new(&vars, HashMap::new());
        
        let value = provider.get_value("uuid", None);
        assert!(Uuid::parse_str(&value).is_ok());
    }

    #[test]
    fn test_range_variable() {
        let vars = vec![Variable {
            name: "num".to_string(),
            var_type: "range".to_string(),
            from: 10,
            to: 20,
            ..Default::default()
        }];

        let provider = VariableProvider::new(&vars, HashMap::new());
        
        for _ in 0..100 {
            let value: u64 = provider.get_value("num", None).parse().unwrap();
            assert!(value >= 10 && value <= 20);
        }
    }

    #[test]
    fn test_list_variable() {
        let vars = vec![Variable {
            name: "user".to_string(),
            var_type: "list".to_string(),
            data: vec!["alice".to_string(), "bob".to_string(), "charlie".to_string()],
            ..Default::default()
        }];

        let provider = VariableProvider::new(&vars, HashMap::new());
        
        let value = provider.get_value("user", None);
        assert!(["alice", "bob", "charlie"].contains(&value.as_str()));
    }

    #[test]
    fn test_env_variable() {
        let mut env_vars = HashMap::new();
        env_vars.insert("TEST_VAR".to_string(), "test_value".to_string());

        let provider = VariableProvider::new(&[], env_vars);
        
        assert_eq!(provider.get_value("env.TEST_VAR", None), "test_value");
    }

    #[test]
    fn test_convert_go_time_format() {
        assert_eq!(convert_go_time_format("2006-01-02"), "%Y-%m-%d");
        assert_eq!(convert_go_time_format("15:04:05"), "%H:%M:%S");
        assert_eq!(convert_go_time_format("2006-01-02T15:04:05-0700"), "%Y-%m-%dT%H:%M:%S%z");
    }
}
