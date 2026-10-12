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

//! Variable type definitions

use std::sync::atomic::{AtomicU32, AtomicU64};

/// Variable types supported by loadgen
#[derive(Debug, Clone, PartialEq)]
pub enum VariableType {
    /// Load values from a file
    File,
    /// Inline list of values
    List,
    /// 32-bit auto-incrementing sequence
    Sequence,
    /// 64-bit auto-incrementing sequence
    Sequence64,
    /// Random number in range
    Range,
    /// UUID v4
    Uuid,
    /// Current local time
    NowLocal,
    /// Current UTC time
    NowUtc,
    /// Current UTC time in compact format
    NowUtcLite,
    /// Unix timestamp in seconds
    NowUnix,
    /// Unix timestamp in milliseconds
    NowUnixInMs,
    /// Unix timestamp in microseconds
    NowUnixInMicro,
    /// Unix timestamp in nanoseconds
    NowUnixInNano,
    /// Custom formatted time
    NowWithFormat,
    /// Random array from another variable
    RandomArray,
    /// Roaring bitmap encoded integer array
    IntArrayBitmap,
    /// Environment variable
    Env,
}

impl From<&str> for VariableType {
    fn from(s: &str) -> Self {
        match s.to_lowercase().as_str() {
            "file" => VariableType::File,
            "list" => VariableType::List,
            "sequence" => VariableType::Sequence,
            "sequence64" => VariableType::Sequence64,
            "range" => VariableType::Range,
            "uuid" => VariableType::Uuid,
            "now_local" => VariableType::NowLocal,
            "now_utc" => VariableType::NowUtc,
            "now_utc_lite" => VariableType::NowUtcLite,
            "now_unix" => VariableType::NowUnix,
            "now_unix_in_ms" => VariableType::NowUnixInMs,
            "now_unix_in_micro" => VariableType::NowUnixInMicro,
            "now_unix_in_nano" => VariableType::NowUnixInNano,
            "now_with_format" => VariableType::NowWithFormat,
            "random_array" => VariableType::RandomArray,
            "int_array_bitmap" => VariableType::IntArrayBitmap,
            "env" => VariableType::Env,
            _ => VariableType::List, // Default to list
        }
    }
}

/// A sequence counter for auto-incrementing variables
#[derive(Debug)]
pub struct SequenceCounter32 {
    current: AtomicU32,
    from: u32,
    to: u32,
}

impl SequenceCounter32 {
    pub fn new(from: u32, to: u32) -> Self {
        Self {
            current: AtomicU32::new(from),
            from,
            to,
        }
    }

    pub fn next(&self) -> u32 {
        let current = self.current.fetch_add(1, std::sync::atomic::Ordering::SeqCst);
        if self.to > 0 && current >= self.to {
            self.current.store(self.from, std::sync::atomic::Ordering::SeqCst);
            return self.from;
        }
        current
    }
}

/// A sequence counter for 64-bit auto-incrementing variables
#[derive(Debug)]
pub struct SequenceCounter64 {
    current: AtomicU64,
    from: u64,
    to: u64,
}

impl SequenceCounter64 {
    pub fn new(from: u64, to: u64) -> Self {
        Self {
            current: AtomicU64::new(from),
            from,
            to,
        }
    }

    pub fn next(&self) -> u64 {
        let current = self.current.fetch_add(1, std::sync::atomic::Ordering::SeqCst);
        if self.to > 0 && current >= self.to {
            self.current.store(self.from, std::sync::atomic::Ordering::SeqCst);
            return self.from;
        }
        current
    }
}
