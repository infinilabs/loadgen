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

//! Load statistics collection

use std::collections::HashMap;
use std::sync::atomic::{AtomicI64, AtomicU64, Ordering};
use std::time::Duration;

use bytesize::ByteSize;
use hdrhistogram::Histogram;
use parking_lot::Mutex;

/// Statistics collected during load generation
#[derive(Debug)]
pub struct LoadStats {
    /// Total request size in bytes
    pub total_request_size: AtomicI64,
    /// Total response size in bytes
    pub total_response_size: AtomicI64,
    /// Total duration of all requests
    pub total_duration: AtomicU64,
    /// Minimum request time
    pub min_request_time: AtomicU64,
    /// Maximum request time
    pub max_request_time: AtomicU64,
    /// Number of requests completed
    pub num_requests: AtomicU64,
    /// Number of errors
    pub num_errors: AtomicU64,
    /// Number of invalid assertions
    pub num_assert_invalid: AtomicU64,
    /// Number of skipped assertions
    pub num_assert_skipped: AtomicU64,
    /// Status code counts
    pub status_codes: Mutex<HashMap<i32, u64>>,
    /// Latency histogram
    pub histogram: Mutex<Histogram<u64>>,
}

impl Default for LoadStats {
    fn default() -> Self {
        Self::new()
    }
}

impl LoadStats {
    /// Create new empty statistics
    pub fn new() -> Self {
        Self {
            total_request_size: AtomicI64::new(0),
            total_response_size: AtomicI64::new(0),
            total_duration: AtomicU64::new(0),
            min_request_time: AtomicU64::new(u64::MAX),
            max_request_time: AtomicU64::new(0),
            num_requests: AtomicU64::new(0),
            num_errors: AtomicU64::new(0),
            num_assert_invalid: AtomicU64::new(0),
            num_assert_skipped: AtomicU64::new(0),
            status_codes: Mutex::new(HashMap::new()),
            histogram: Mutex::new(Histogram::new(3).unwrap()),
        }
    }

    /// Record a request result
    pub fn record(
        &self,
        duration: Duration,
        request_size: i64,
        response_size: i64,
        status: i32,
        is_error: bool,
    ) {
        let duration_us = duration.as_micros() as u64;

        self.num_requests.fetch_add(1, Ordering::Relaxed);
        self.total_duration.fetch_add(duration_us, Ordering::Relaxed);
        self.total_request_size.fetch_add(request_size, Ordering::Relaxed);
        self.total_response_size.fetch_add(response_size, Ordering::Relaxed);

        // Update min/max atomically
        let mut current_min = self.min_request_time.load(Ordering::Relaxed);
        while duration_us < current_min {
            match self.min_request_time.compare_exchange_weak(
                current_min,
                duration_us,
                Ordering::SeqCst,
                Ordering::Relaxed,
            ) {
                Ok(_) => break,
                Err(x) => current_min = x,
            }
        }

        let mut current_max = self.max_request_time.load(Ordering::Relaxed);
        while duration_us > current_max {
            match self.max_request_time.compare_exchange_weak(
                current_max,
                duration_us,
                Ordering::SeqCst,
                Ordering::Relaxed,
            ) {
                Ok(_) => break,
                Err(x) => current_max = x,
            }
        }

        // Update status codes
        {
            let mut codes = self.status_codes.lock();
            *codes.entry(status).or_insert(0) += 1;
        }

        // Update histogram
        {
            let mut hist = self.histogram.lock();
            let _ = hist.record(duration_us);
        }

        if is_error {
            self.num_errors.fetch_add(1, Ordering::Relaxed);
        }
    }

    /// Record an assertion failure
    pub fn record_assert_invalid(&self) {
        self.num_assert_invalid.fetch_add(1, Ordering::Relaxed);
    }

    /// Record a skipped assertion
    pub fn record_assert_skipped(&self) {
        self.num_assert_skipped.fetch_add(1, Ordering::Relaxed);
    }

    /// Merge another stats instance into this one
    pub fn merge(&self, other: &LoadStats) {
        self.total_request_size.fetch_add(
            other.total_request_size.load(Ordering::Relaxed),
            Ordering::Relaxed,
        );
        self.total_response_size.fetch_add(
            other.total_response_size.load(Ordering::Relaxed),
            Ordering::Relaxed,
        );
        self.total_duration.fetch_add(
            other.total_duration.load(Ordering::Relaxed),
            Ordering::Relaxed,
        );
        self.num_requests.fetch_add(
            other.num_requests.load(Ordering::Relaxed),
            Ordering::Relaxed,
        );
        self.num_errors.fetch_add(
            other.num_errors.load(Ordering::Relaxed),
            Ordering::Relaxed,
        );
        self.num_assert_invalid.fetch_add(
            other.num_assert_invalid.load(Ordering::Relaxed),
            Ordering::Relaxed,
        );
        self.num_assert_skipped.fetch_add(
            other.num_assert_skipped.load(Ordering::Relaxed),
            Ordering::Relaxed,
        );

        // Merge min
        let other_min = other.min_request_time.load(Ordering::Relaxed);
        let mut current_min = self.min_request_time.load(Ordering::Relaxed);
        while other_min < current_min {
            match self.min_request_time.compare_exchange_weak(
                current_min,
                other_min,
                Ordering::SeqCst,
                Ordering::Relaxed,
            ) {
                Ok(_) => break,
                Err(x) => current_min = x,
            }
        }

        // Merge max
        let other_max = other.max_request_time.load(Ordering::Relaxed);
        let mut current_max = self.max_request_time.load(Ordering::Relaxed);
        while other_max > current_max {
            match self.max_request_time.compare_exchange_weak(
                current_max,
                other_max,
                Ordering::SeqCst,
                Ordering::Relaxed,
            ) {
                Ok(_) => break,
                Err(x) => current_max = x,
            }
        }

        // Merge status codes
        {
            let other_codes = other.status_codes.lock();
            let mut self_codes = self.status_codes.lock();
            for (k, v) in other_codes.iter() {
                *self_codes.entry(*k).or_insert(0) += v;
            }
        }

        // Merge histogram
        {
            let other_hist = other.histogram.lock();
            let mut self_hist = self.histogram.lock();
            let _ = self_hist.add(&*other_hist);
        }
    }
}

/// Final aggregated statistics for display
pub struct AggregatedStats {
    pub num_requests: u64,
    pub num_errors: u64,
    pub num_assert_invalid: u64,
    pub num_assert_skipped: u64,
    pub total_request_size: i64,
    pub total_response_size: i64,
    pub min_request_time: Duration,
    pub max_request_time: Duration,
    pub avg_request_time: Duration,
    pub total_duration: Duration,
    pub status_codes: HashMap<i32, u64>,
    pub p50: Duration,
    pub p75: Duration,
    pub p95: Duration,
    pub p99: Duration,
    pub p999: Duration,
}

impl From<&LoadStats> for AggregatedStats {
    fn from(stats: &LoadStats) -> Self {
        let num_requests = stats.num_requests.load(Ordering::Relaxed);
        let total_duration_us = stats.total_duration.load(Ordering::Relaxed);
        let min_us = stats.min_request_time.load(Ordering::Relaxed);
        let max_us = stats.max_request_time.load(Ordering::Relaxed);

        let histogram = stats.histogram.lock();

        AggregatedStats {
            num_requests,
            num_errors: stats.num_errors.load(Ordering::Relaxed),
            num_assert_invalid: stats.num_assert_invalid.load(Ordering::Relaxed),
            num_assert_skipped: stats.num_assert_skipped.load(Ordering::Relaxed),
            total_request_size: stats.total_request_size.load(Ordering::Relaxed),
            total_response_size: stats.total_response_size.load(Ordering::Relaxed),
            min_request_time: Duration::from_micros(if min_us == u64::MAX { 0 } else { min_us }),
            max_request_time: Duration::from_micros(max_us),
            avg_request_time: if num_requests > 0 {
                Duration::from_micros(total_duration_us / num_requests)
            } else {
                Duration::ZERO
            },
            total_duration: Duration::from_micros(total_duration_us),
            status_codes: stats.status_codes.lock().clone(),
            p50: Duration::from_micros(histogram.value_at_quantile(0.50)),
            p75: Duration::from_micros(histogram.value_at_quantile(0.75)),
            p95: Duration::from_micros(histogram.value_at_quantile(0.95)),
            p99: Duration::from_micros(histogram.value_at_quantile(0.99)),
            p999: Duration::from_micros(histogram.value_at_quantile(0.999)),
        }
    }
}

impl AggregatedStats {
    /// Print statistics in the format matching the Go implementation
    pub fn print(&self, wall_time: Duration, config: &crate::config::types::RunnerConfig) {
        let req_rate = self.num_requests as f64 / wall_time.as_secs_f64();
        let req_bytes_rate = self.total_request_size as f64 / wall_time.as_secs_f64();
        let total_bytes_rate = (self.total_request_size + self.total_response_size) as f64 
            / wall_time.as_secs_f64();

        // Summary line
        if config.no_size_stats {
            println!(
                "\n{} requests finished in {:?}",
                self.num_requests, wall_time
            );
        } else {
            println!(
                "\n{} requests finished in {:?}, {} sent, {} received",
                self.num_requests,
                wall_time,
                ByteSize::b(self.total_request_size as u64),
                ByteSize::b(self.total_response_size as u64)
            );
        }

        // Client metrics
        println!("\n[Loadgen Client Metrics]");
        println!("Requests/sec:\t\t{:.2}", req_rate);

        if !config.benchmark_only && !config.no_size_stats {
            println!("Request Traffic/sec:\t{}/s", ByteSize::b(req_bytes_rate as u64));
            println!("Total Transfer/sec:\t{}/s", ByteSize::b(total_bytes_rate as u64));
        }

        println!("Fastest Request:\t{:?}", self.min_request_time);
        println!("Slowest Request:\t{:?}", self.max_request_time);

        if config.assert_error {
            println!("Number of Errors:\t{}", self.num_errors);
        }

        if config.assert_invalid {
            println!("Assert Invalid:\t\t{}", self.num_assert_invalid);
            println!("Assert Skipped:\t\t{}", self.num_assert_skipped);
        }

        // Status codes
        let mut sorted_codes: Vec<_> = self.status_codes.iter().collect();
        sorted_codes.sort_by_key(|(k, _)| *k);
        for (code, count) in sorted_codes {
            println!("Status {}:\t\t{}", code, count);
        }

        // Latency metrics
        if !config.benchmark_only && !config.no_stats {
            println!("\n[Latency Metrics]");
            println!(
                "{} samples of {} events",
                self.num_requests, self.num_requests
            );
            println!("Avg:\t\t{:?}", self.avg_request_time);
            println!("p50:\t\t{:?}", self.p50);
            println!("p75:\t\t{:?}", self.p75);
            println!("p95:\t\t{:?}", self.p95);
            println!("p99:\t\t{:?}", self.p99);
            println!("p999:\t\t{:?}", self.p999);
            println!("Max:\t\t{:?}", self.max_request_time);
            println!("Min:\t\t{:?}", self.min_request_time);
        }

        // Estimated server metrics
        println!("\n[Estimated Server Metrics]");
        let server_req_rate = if self.total_duration.as_secs_f64() > 0.0 {
            self.num_requests as f64 / self.total_duration.as_secs_f64()
        } else {
            0.0
        };
        println!("Requests/sec:\t\t{:.2}", server_req_rate);
        println!("Avg Req Time:\t\t{:?}", self.avg_request_time);

        if !config.benchmark_only && !config.no_size_stats {
            let server_bytes_rate = (self.total_request_size + self.total_response_size) as f64
                / self.total_duration.as_secs_f64();
            println!("Transfer/sec:\t\t{}/s", ByteSize::b(server_bytes_rate as u64));
        }

        println!();
    }
}
