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

//! INFINI Loadgen - A high-performance HTTP load generator and testing suite
//!
//! This is a Rust implementation of the loadgen tool, supporting the same
//! YAML configuration and DSL formats as the original Go implementation.

pub mod assertion;
pub mod config;
pub mod http;
pub mod loader;
pub mod runner;
pub mod template;
pub mod variable;

pub use config::types::{AppConfig, LoaderConfig, RunnerConfig};
pub use loader::generator::LoadGenerator;
pub use loader::stats::LoadStats;
