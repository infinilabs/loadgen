# INFINI Loadgen (Rust)

A high-performance HTTP load generator and testing suite written in Rust, compatible with the original Go implementation's configuration files.

## Features

- **High Performance**: Built with Tokio async runtime for maximum throughput
- **Compatible Configuration**: Supports the same YAML and DSL configuration formats as the Go version
- **Variable System**: Full support for all variable types including:
  - `file` - Load from external files
  - `list` - Inline defined lists
  - `sequence` / `sequence64` - Auto-incrementing sequences
  - `range` - Random numbers in range
  - `uuid` - UUID v4 generation
  - `now_local`, `now_utc`, `now_unix` - Time variables
  - `now_with_format` - Custom formatted time
  - `random_array` - Random arrays from other variables
  - `int_array_bitmap` - Roaring bitmap encoded arrays
- **Template Engine**: `$[[variable]]` syntax for dynamic request generation
- **Response Assertions**: Validate responses with conditions (equals, contains, regexp, range, etc.)
- **Rate Limiting**: Control request rate with `-r` flag
- **Metrics**: Detailed latency histograms and percentiles

## Building

```bash
cd loadgen-rust
cargo build --release
```

The binary will be at `target/release/loadgen`.

## Usage

```bash
# Run with default config (loadgen.yml)
./loadgen

# Run with specific config
./loadgen -C myconfig.yml

# Run DSL file
./loadgen --run test.dsl

# Load test with concurrency
./loadgen -c 10 -d 30 -r 1000

# Mixed mode (run both DSL and YAML)
./loadgen --run test.dsl -C config.yml --mixed
```

## CLI Options

| Flag | Description | Default |
|------|-------------|---------|
| `-c, --concurrency` | Number of concurrent threads | 1 |
| `-d, --duration` | Test duration in seconds | 5 |
| `-r, --rate` | Max requests per second (-1 = unlimited) | -1 |
| `-l, --limit` | Total request limit (-1 = unlimited) | -1 |
| `--timeout` | Request timeout in seconds (0 = no timeout) | 0 |
| `--read-timeout` | Read timeout in seconds | 0 |
| `--write-timeout` | Write timeout in seconds | 0 |
| `--dial-timeout` | Connection dial timeout in seconds | 3 |
| `--compress` | Enable gzip compression | false |
| `--mixed` | Enable mixed YAML/DSL mode | false |
| `--total-rounds` | Number of request rounds (-1 = unlimited) | -1 |
| `--run` | Path to DSL file | - |
| `-C, --config` | Path to YAML config | loadgen.yml |
| `--log` | Log level (trace, debug, info, warn, error) | info |
| `--debug` | Enable debug mode | false |

## Configuration Format

### YAML Configuration (`loadgen.yml`)

```yaml
env:
  ES_USERNAME: elastic
  ES_PASSWORD: password
  ES_ENDPOINT: http://localhost:9200

runner:
  total_rounds: 1
  no_warm: true
  log_requests: false
  assert_invalid: false
  assert_error: false
  default_endpoint: $[[env.ES_ENDPOINT]]
  default_basic_auth:
    username: $[[env.ES_USERNAME]]
    password: $[[env.ES_PASSWORD]]

variables:
  - name: id
    type: sequence
  - name: uuid
    type: uuid
  - name: user
    type: list
    data:
      - alice
      - bob
      - charlie

requests:
  - request:
      method: POST
      url: /_bulk
      body: |
        {"index": {"_index": "test", "_id": "$[[uuid]]"}}
        {"id": "$[[id]]", "user": "$[[user]]"}
```

### DSL Configuration

```dsl
# runner: {"total_rounds": 1, "no_warm": true}
# variables: [{"name": "id", "type": "sequence"}]

DELETE /test

PUT /test

POST /test/_doc/$[[id]]
{
  "name": "test document"
}

GET /test/_search
# 200
```

## Response Assertions

```yaml
requests:
  - request:
      method: GET
      url: /test/_search
    assert:
      equals:
        _ctx.response.status: 200
      contains:
        _ctx.response.body: "hits"
      range:
        _ctx.elapsed:
          lte: 1000
```

## License

AGPL-3.0 - See [LICENSE](../LICENSE) for details.
