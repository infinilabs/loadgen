---
weight: 30
title: Reference
---

# Reference

Complete reference for all Loadgen configuration options.

---

## Command-Line Arguments

```bash
loadgen [options]
```

| Argument | Type | Default | Description |
|----------|------|---------|-------------|
| `-run` | string | `loadgen.dsl` | Path to DSL test file |
| `-config` | string | `loadgen.yml` | Path to YAML configuration file |
| `-d` | int | `5` | Test duration in seconds |
| `-c` | int | `1` | Number of concurrent threads |
| `-r` | int | `-1` (unlimited) | Max requests per second (fixed QPS) |
| `-l` | int | `-1` (unlimited) | Total request limit |
| `-compress` | bool | `false` | Compress request body with gzip |
| `-timeout` | int | `60` | Request timeout in seconds |
| `-dial-timeout` | int | `3` | Connection dial timeout in seconds |
| `-read-timeout` | int | `0` (uses -timeout) | Read timeout in seconds |
| `-write-timeout` | int | `0` (uses -timeout) | Write timeout in seconds |
| `-cpu` | int | `-1` (all) | Number of CPUs to use |
| `-mem` | int | `-1` (unlimited) | Memory soft limit in MB |
| `-log` | string | — | Log level: trace, debug, info, warn, error, off |
| `-debug` | bool | `false` | Debug mode — exit immediately on panic with full stack trace |
| `-v` | bool | — | Print version info |

---

## Environment Variables (env)

Define default environment variables in YAML or DSL. Command-line environment variables override these at runtime.

**YAML format:**

```yaml
env:
  ES_ENDPOINT: http://localhost:9200
  ES_USERNAME: admin
  ES_PASSWORD: admin
```

**DSL format:**

```text
# env: {
#   ES_ENDPOINT: "http://localhost:9200",
#   ES_USERNAME: "admin",
#   ES_PASSWORD: "admin",
# },
```

**Usage:** `$[[env.ES_ENDPOINT]]`

**IPv6 Support:**

Loadgen fully supports IPv6 addresses. Use standard bracket notation (`[ipv6addr]:port`) in URLs:

```yaml
env:
  ES_ENDPOINT: "http://[::1]:9200"
  # Or with a real IPv6 address:
  # ES_ENDPOINT: "http://[fd00::1]:9200"
  # Link-local address with zone ID (URL-encoded %25 for %):
  # ES_ENDPOINT: "http://[fe80::18df:9883:1e27:b040%25en0]:9200"
```

Or from the command line:

```bash
ES_ENDPOINT="http://[::1]:9200" loadgen -run loadgen.dsl -d 5 -c 2
# Link-local with zone ID:
ES_ENDPOINT="http://[fe81::18df:9883:1e27:b040%25en0]:9200" loadgen -run loadgen.dsl -d 5 -c 2
```

> **Note:** For link-local IPv6 addresses with a zone ID (e.g. `fe80::1%en0`), the `%` must be URL-encoded as `%25` in the URL.

---

## Runner Configuration (runner)

Global parameters controlling test execution behavior.

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `total_rounds` | int | unset | Number of execution rounds. When set, enters round mode; otherwise runs for the specified duration |
| `no_warm` | bool | `false` | Skip warmup phase |
| `log_requests` | bool | `false` | Log all request details |
| `log_status_codes` | []int | — | Only log requests with these status codes (e.g. `[0, 500]`) |
| `assert_invalid` | bool | `false` | Exit with code 1 when assertions fail |
| `assert_error` | bool | `false` | Exit with code 2 when request errors occur |
| `reset_context` | bool | `false` | Reset context variables between rounds |
| `valid_status_codes_during_warmup` | []int | — | Status codes considered valid during warmup (e.g. `[200, 201, 404]`) |
| `disable_header_names_normalizing` | bool | `false` | Disable automatic response header name formatting |
| `default_endpoint` | string | — | Default request endpoint; auto-prepended when request line omits host |
| `default_basic_auth` | object | — | Default Basic Auth credentials |

**DSL format example:**

```text
# runner: {
#   total_rounds: 1,
#   no_warm: true,
#   assert_invalid: true,
#   assert_error: true,
#   valid_status_codes_during_warmup: [200, 201],
#   log_status_codes: [0, 500],
#   default_endpoint: "$[[env.ES_ENDPOINT]]",
#   default_basic_auth: {
#     username: "$[[env.ES_USERNAME]]",
#     password: "$[[env.ES_PASSWORD]]",
#   },
# },
```

---

## Variable Definitions (variables)

Variables are referenced in requests using `$[[variable_name]]` and produce a new value on each request.

### Variable Types

| Type | Description | Required Parameters |
|------|-------------|---------------------|
| `file` | Random line from a text file | `path` |
| `list` | Random item from an inline list | `data` |
| `sequence` | 32-bit auto-incrementing integer | — |
| `sequence64` | 64-bit auto-incrementing integer | — |
| `range` | Random integer within a range | `from`, `to` |
| `random_array` | Random array composed from another variable | `variable_key`, `size` |
| `uuid` | UUID v4 string | — |
| `now_local` | Current time (local timezone) | — |
| `now_utc` | Current time (UTC, full format) | — |
| `now_utc_lite` | Current time (UTC, ISO short format) | — |
| `now_unix` | Current Unix timestamp | — |
| `now_with_format` | Current time (custom format) | `format` |

### Variable Parameter Details

#### file

```text
{
  name: "ip",
  type: "file",
  path: "dict/ip.txt",
  data: ["10.0.0.1", "10.0.0.2"],
  replace: {
    '"': '\\"',
    '\\': '\\\\',
  },
}
```

| Parameter | Description |
|-----------|-------------|
| `path` | Path to external text file (one value per line) |
| `data` | Additional values appended after file contents |
| `replace` | Character replacements applied to the value (for escaping) |

**NDJSON Corpus Support:**

The `file` variable type also supports NDJSON-formatted corpus files (one complete JSON object per line). This is especially useful for bulk write scenarios — you can use the corpus file directly as the data source for bulk request bodies:

```text
# dict/docs.json (one JSON document per line)
{"title": "Getting Started with Elasticsearch", "author": "alice", "views": 100}
{"title": "Performance Tuning Best Practices", "author": "bob", "views": 350}
{"title": "Cluster Operations Guide", "author": "coco", "views": 220}
```

Variable definition:

```text
{
  name: "doc",
  type: "file",
  path: "dict/docs.json",
}
```

Usage in a bulk request:

```text
POST $[[env.ES_ENDPOINT]]/_bulk
{"index": {"_index": "articles", "_id": "$[[uuid]]"}}
$[[doc]]
# request: {
#   body_repeat_times: 1000,
# },
```

On each repetition, `$[[doc]]` picks a random line from the corpus file and uses it directly as the document body — no need to manually construct JSON in the DSL. This approach is ideal for:

- Benchmarking writes with realistic data
- Bulk-importing existing datasets
- Maintaining document structure diversity

#### list

```text
{
  name: "user",
  type: "list",
  data: ["alice", "bob", "coco"],
}
```

| Parameter | Description |
|-----------|-------------|
| `data` | String array; one item picked randomly per request |

#### sequence / sequence64

```text
{
  name: "id",
  type: "sequence",
  from: 1,
  to: 999999,
}
```

| Parameter | Description |
|-----------|-------------|
| `from` | Start value (default 0) |
| `to` | Max value (wraps around when reached) |

#### range

```text
{
  name: "suffix",
  type: "range",
  from: 10,
  to: 1000,
}
```

| Parameter | Description |
|-----------|-------------|
| `from` | Lower bound |
| `to` | Upper bound |

#### random_array

```text
{
  name: "id_list",
  type: "random_array",
  variable_type: "number",
  variable_key: "suffix",
  square_bracket: false,
  string_bracket: "",
  size: 10,
  replace: { ... },
}
```

| Parameter | Description |
|-----------|-------------|
| `variable_type` | Element type: `number` or `string` |
| `variable_key` | Source variable name |
| `size` | Array length |
| `square_bracket` | Whether to wrap output with `[` `]` |
| `string_bracket` | String to wrap around each element |
| `replace` | Character replacements on the entire output |

#### now_with_format

```text
{
  name: "ts",
  type: "now_with_format",
  format: "2006-01-02T15:04:05-0700",
}
```

| Parameter | Description |
|-----------|-------------|
| `format` | Go time format string ([reference](https://www.geeksforgeeks.org/time-formatting-in-golang/)) |

---

## Request Configuration (request)

Each request can be configured via a `# request: { ... },` comment block with the following parameters:

| Parameter | Type | Description |
|-----------|------|-------------|
| `basic_auth` | object | Per-request Basic Auth (overrides runner default) |
| `headers` | []object | Custom request headers |
| `disable_header_names_normalizing` | bool | Disable automatic request header name formatting |
| `runtime_variables` | object | Request-level runtime variables (shared within the request) |
| `runtime_body_line_variables` | object | Body line-level variables (independently generated per line pair) |
| `body_repeat_times` | int | Number of times to repeat the request body (for bulk construction) |

**Full example:**

```text
POST /_bulk
{"index": {"_index": "demo-$[[suffix]]", "_id": "$[[uuid]]", "routing": "$[[routing_no]]"}}
{"id":"$[[id]]","user":"$[[user]]","batch":"$[[batch_no]]","ts":"$[[ts]]"}
# request: {
#   basic_auth: {
#     username: "$[[env.ES_USERNAME]]",
#     password: "$[[env.ES_PASSWORD]]",
#   },
#   headers: [
#     {Authorization: "Bearer $[[token]]"},
#     {X-Trace-Id: "$[[uuid]]"},
#   ],
#   disable_header_names_normalizing: true,
#   runtime_variables: {
#     batch_no: "uuid",
#   },
#   runtime_body_line_variables: {
#     routing_no: "uuid",
#   },
#   body_repeat_times: 100,
# },
```

### runtime_variables vs runtime_body_line_variables

| Type | When Generated | Typical Use |
|------|---------------|-------------|
| `runtime_variables` | Once per request — the entire request body shares the same value | Batch number, trace ID |
| `runtime_body_line_variables` | Independently per body "line pair" — each pair gets a unique value | Routing key, per-document ID |

> **What's a "line pair"?** For `_bulk` requests, each document consists of an action line + source line. When `body_repeat_times: N` replicates the body N times, `runtime_body_line_variables` ensures each copy gets an independent value, while `runtime_variables` remains constant across the entire request.

**Illustration:**

```
Request body (body_repeat_times: 3):
  Line pair 1: routing_no = aaa   ← runtime_body_line_variables: unique per pair
  Line pair 2: routing_no = bbb
  Line pair 3: routing_no = ccc
  All pairs:   batch_no   = xxx   ← runtime_variables: shared across request
```

### DSL Format vs YAML Format

Loadgen supports two configuration formats. **YAML format parses faster** and is recommended for high-throughput scenarios:

**YAML format (recommended for benchmarking):**

```yaml
requests:
  - request:
      method: POST
      url: /_bulk
      runtime_variables:
        batch_no: uuid
      runtime_body_line_variables:
        routing_no: uuid
      body_repeat_times: 5000
      body: |
        {"index": {"_index": "medcl", "_type": "_doc", "_id": "$[[uuid]]"}}
        {"id": "$[[id]]", "field1": "$[[list]]", "now_local": "$[[now_local]]", "now_unix": "$[[now_unix]]"}
        {"index": {"_index": "infinilabs", "_type": "_doc", "_id": "$[[uuid]]"}}
        {"id": "$[[id]]", "field1": "$[[list]]", "now_local": "$[[now_local]]", "now_unix": "$[[now_unix]]"}
```

The above example has 2 action+source line pairs in the body. With `body_repeat_times: 5000`, each request sends 10,000 documents. Each line pair's `routing_no` is independently generated, while `batch_no` is shared across the entire request.

**Equivalent DSL format:**

```text
POST /_bulk
{"index": {"_index": "medcl", "_type": "_doc", "_id": "$[[uuid]]"}}
{"id": "$[[id]]", "field1": "$[[list]]", "now_local": "$[[now_local]]", "now_unix": "$[[now_unix]]"}
{"index": {"_index": "infinilabs", "_type": "_doc", "_id": "$[[uuid]]"}}
{"id": "$[[id]]", "field1": "$[[list]]", "now_local": "$[[now_local]]", "now_unix": "$[[now_unix]]"}
# request: {
#   runtime_variables: {
#     batch_no: "uuid",
#   },
#   runtime_body_line_variables: {
#     routing_no: "uuid",
#   },
#   body_repeat_times: 5000,
# },
```

> **Performance tip:** YAML format eliminates runtime DSL parsing overhead. In extreme throughput scenarios, you'll observe higher client send rates with YAML. For functional validation and integration tests, the performance difference is negligible.

---

## Assertions (assert)

Assertions validate response content. When an assertion fails, remaining requests in the current round are skipped.

### Shorthand Format

```text
# assert: (200, {status: "ok"}),
```

Equivalent to: status code is 200 AND response body contains `{"status": "ok"}`.

### Standard Format

```text
# assert: {
#   _ctx.response.status: 200,
#   _ctx.response.body_json.hits.total.value: 1,
# },
```

### Available Context Variables

| Path | Description |
|------|-------------|
| `_ctx.response.status` | HTTP status code |
| `_ctx.response.header` | Response headers |
| `_ctx.response.body` | Raw response body |
| `_ctx.response.body_json.*` | JSON response fields (supports nested paths) |
| `_ctx.elapsed` | Request elapsed time (milliseconds) |

### Exit Codes

| Code | Condition |
|------|-----------|
| 0 | All assertions passed |
| 1 | Assertion failures exist and `runner.assert_invalid: true` |
| 2 | Request errors exist and `runner.assert_error: true` |

---

## Dynamic Variable Registration (register)

Extract values from responses and register them as variables for subsequent requests.

```text
# register: [
#   { token: "_ctx.response.body_json.access_token" },
#   { task_id: "_ctx.response.body_json.task" },
# ]
```

After registration, use `$[[token]]` or `$[[task_id]]` in later requests.

Typical scenario: login to get a token → use token in subsequent requests.

---

## Warmup Behavior

By default, Loadgen executes all requests once as a warmup before the actual test:

- Warmup results are printed to the terminal
- If warmup fails, you're prompted whether to continue
- Set `runner.no_warm: true` to skip warmup
- Set `runner.valid_status_codes_during_warmup` to define which status codes are acceptable during warmup

---

## Output Metrics

After test completion, Loadgen prints the following:

```
75 requests finished in 13.261499ms, 139.34MB sent, 0.00bytes received

[Loadgen Client Metrics]
Requests/sec:        14.85           ← Actual client QPS
Request Traffic/sec: 27.58MB         ← Send traffic per second
Total Transfer/sec:  27.58MB         ← Total traffic per second (including response)
Fastest Request:     99.375µs        ← Fastest request
Slowest Request:     1.898458ms      ← Slowest request
Status 0:            75              ← Status code counts

[Latency Metrics]
75 samples of 75 events
Cumulative:  13.261499ms             ← Total elapsed time
HMean:       148.19µs               ← Harmonic mean
Avg.:        176.819µs              ← Arithmetic mean
p50:         146.75µs               ← Median
p75:         168µs                  ← 75th percentile
p95:         202.583µs              ← 95th percentile
p99:         409.458µs              ← 99th percentile
p999:        1.898458ms             ← 99.9th percentile
Long 5%:     690.854µs              ← Slowest 5% average
Short 5%:    106.5µs                ← Fastest 5% average
Max:         1.898458ms
Min:         99.375µs
Range:       1.799083ms
StdDev:      204.268µs
Rate/sec.:   14.85

[Latency Distribution]
          99µs - 279µs ----------    ← Latency distribution histogram
         279µs - 459µs -
         459µs - 639µs -

[Estimated Server Metrics]
Requests/sec:        5655.47         ← Estimated server-side QPS
Avg Req Time:        176.819µs       ← Estimated server processing time
Transfer/sec:        10.26GB         ← Estimated server throughput
```

### Metric Sections

| Section | Description |
|---------|-------------|
| **Loadgen Client Metrics** | Throughput and extremes from the client perspective |
| **Latency Metrics** | Detailed latency distribution with percentiles |
| **Latency Distribution** | Visual latency histogram |
| **Estimated Server Metrics** | Server-side estimates (network overhead excluded) |

### Status Codes and Errors

- `Status 200: N` — Successful responses
- `Status 0: N` — Connection failures or timeouts (no HTTP response received)
- `Number of Errors` — Total network/connection errors (shown when `assert_error` is enabled)
- `Number of Invalid` — Total assertion failures (shown when `assert_invalid` is enabled)

> Use INFINI Console for monitoring or APM for accurate server-side metrics; client-side statistics are approximations.
