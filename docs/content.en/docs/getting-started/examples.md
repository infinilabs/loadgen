---
weight: 40
title: Examples
---

# Examples

Practical examples organized by use case. Each example can be copied and run directly.

---

## 1. Bulk Write Benchmarking

**Goal:** Evaluate write throughput, cluster load, and queue pressure.

```text
# variables: [
#   {name: "id", type: "sequence"},
#   {name: "uuid", type: "uuid"},
#   {name: "suffix", type: "range", from: 1, to: 100},
#   {name: "ip", type: "file", path: "dict/ip.txt"},
#   {name: "now_unix", type: "now_unix"},
# ],

POST $[[env.ES_ENDPOINT]]/_bulk
{"index": {"_index": "bench-$[[suffix]]", "_id": "$[[uuid]]", "routing": "$[[routing_no]]"}}
{"id": "$[[id]]", "ip": "$[[ip]]", "batch": "$[[batch_no]]", "ts": "$[[now_unix]]"}
# request: {
#   basic_auth: {
#     username: "$[[env.ES_USERNAME]]",
#     password: "$[[env.ES_PASSWORD]]",
#   },
#   runtime_variables: {
#     batch_no: "uuid",
#   },
#   runtime_body_line_variables: {
#     routing_no: "uuid",
#   },
#   body_repeat_times: 1000,
# },
```

**Run commands:**

```bash
# Sustained 60 seconds, 100 concurrent threads
loadgen -d 60 -c 100

# Fixed QPS
loadgen -d 60 -c 100 -r 2000

# Fixed total request count
loadgen -d 600 -c 100 -l 50000
```

**Reference:** [testing/setup/loadgen/cases/bulk/create.yml](https://github.com/infinilabs/testing/blob/main/setup/loadgen/cases/bulk/create.yml)

---

## 2. Search Performance Testing

**Goal:** Evaluate query latency and throughput across different query patterns.

```text
# variables: [
#   {name: "user", type: "file", path: "dict/user.txt"},
#   {name: "ip", type: "file", path: "dict/ip.txt"},
# ],

GET $[[env.ES_ENDPOINT]]/my-index/_search?q=user:$[[user]]
# request: {
#   basic_auth: {
#     username: "$[[env.ES_USERNAME]]",
#     password: "$[[env.ES_PASSWORD]]",
#   },
# },
```

**Run command:**

```bash
loadgen -d 30 -c 10
```

**Reference:** [testing/setup/loadgen/cases/search/random_search.yml](https://github.com/infinilabs/testing/blob/main/setup/loadgen/cases/search/random_search.yml)

---

## 3. Mixed Request Benchmarking

**Goal:** Simulate real-world traffic with create + update + delete mixed workload.

A single DSL file with multiple requests — Loadgen executes them sequentially in a loop:

```text
# variables: [
#   {name: "id", type: "sequence"},
#   {name: "uuid", type: "uuid"},
#   {name: "user", type: "list", data: ["alice", "bob", "test"]},
# ],

POST $[[env.ES_ENDPOINT]]/_bulk
{"index": {"_index": "mixed-test", "_id": "$[[uuid]]"}}
{"id": "$[[id]]", "user": "$[[user]]"}
# request: {
#   body_repeat_times: 10,
# },

POST $[[env.ES_ENDPOINT]]/_bulk
{"delete": {"_index": "mixed-test", "_id": "$[[uuid]]"}}
# request: {
#   body_repeat_times: 5,
# },
```

**Reference:** [testing/setup/loadgen/cases/bulk/mixed.yml](https://github.com/infinilabs/testing/blob/main/setup/loadgen/cases/bulk/mixed.yml)

---

## 4. Functional Validation & CI Integration

**Goal:** Execute all requests once, assert everything passes, then exit.

```text
# runner: {
#   total_rounds: 1,
#   no_warm: true,
#   assert_invalid: true,
#   assert_error: true,
# },

GET $[[env.ES_ENDPOINT]]/_cluster/health
# assert: {
#   _ctx.response.status: 200,
# },

GET $[[env.ES_ENDPOINT]]/_cat/indices
# assert: {
#   _ctx.response.status: 200,
# },
```

**Run command:**

```bash
loadgen -run health_check.dsl
echo $?  # 0=pass, 1=assertion failure, 2=request error
```

Ideal for CI pipelines — any assertion failure causes a non-zero exit.

**Reference:** [testing/setup/loadgen/cases/dummy/loadgen.yml](https://github.com/infinilabs/testing/blob/main/setup/loadgen/cases/dummy/loadgen.yml)

---

## 5. Dynamic Variable Registration (Chain Testing)

**Goal:** Extract values from preceding requests for use in subsequent ones.

```text
# runner: {
#   total_rounds: 1,
#   no_warm: true,
#   assert_invalid: true,
# },

# Create a document, extract the generated _id
POST $[[env.ES_ENDPOINT]]/my-index/_doc
{"title": "test document", "user": "loadgen"}
# assert: {
#   _ctx.response.status: 201,
# },
# register: [
#   { doc_id: "_ctx.response.body_json._id" },
# ]

# Read the document using the registered doc_id
GET $[[env.ES_ENDPOINT]]/my-index/_doc/$[[doc_id]]
# assert: {
#   _ctx.response.status: 200,
#   _ctx.response.body_json._source.user: "loadgen",
# },

# Delete the document
DELETE $[[env.ES_ENDPOINT]]/my-index/_doc/$[[doc_id]]
# assert: {
#   _ctx.response.status: 200,
# },
```

**Reference:** [testing/setup/easysearch/cases/crud/create_without_id/loadgen.yml](https://github.com/infinilabs/testing/blob/main/setup/easysearch/cases/crud/create_without_id/loadgen.yml)

---

## 6. JSON Response Assertions

**Goal:** Validate specific fields in JSON response bodies.

```text
GET $[[env.ES_ENDPOINT]]/my-index/_search
{"query": {"match_all": {}}}
# assert: {
#   _ctx.response.status: 200,
#   _ctx.response.body_json.hits.total.value: 100,
#   _ctx.response.body_json.timed_out: false,
# },
```

**Shorthand assertion format:**

```text
GET /api/status
# assert: (200, {status: "ok", version: "1.0"}),
```

**Reference:** [testing/setup/gateway/cases/filters/set_response/loadgen.yml](https://github.com/infinilabs/testing/blob/main/setup/gateway/cases/filters/set_response/loadgen.yml)

---

## 7. Authenticated API Testing (Token Mode)

**Goal:** Login to get a token, then use it in subsequent requests via headers.

```text
# runner: {
#   total_rounds: 1,
#   no_warm: true,
#   assert_invalid: true,
#   default_endpoint: "$[[env.API_HOST]]",
# },

# Login
POST /account/login
{"email": "admin@example.com", "password": "$[[env.ADMIN_PASSWORD]]"}
# assert: (200, {}),
# register: [
#   { token: "_ctx.response.body_json.access_token" },
# ]

# Use token to call a business API
GET /api/resources?size=10
# request: {
#   headers: [
#     {Authorization: "Bearer $[[token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
# assert: {
#   _ctx.response.status: 200,
# },

# Create a resource
POST /api/resources
{"name": "test-resource", "type": "demo"}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
# assert: (200, {}),
# register: [
#   { resource_id: "_ctx.response.body_json.id" },
# ]

# Delete the resource
DELETE /api/resources/$[[resource_id]]
# request: {
#   headers: [
#     {Authorization: "Bearer $[[token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
# assert: {
#   _ctx.response.status: 200,
# },
```

---

## 8. Coco AI Integration Testing

We use Loadgen for full integration testing in the [Coco Server](https://github.com/infinilabs/coco-server) project:

### Project Structure

```
tests/
├── loadgen.yml              # Global config (env vars, runner)
├── assets/
│   ├── run_integration_tests.py   # Auto-execute all scenarios
│   ├── start_coco.sh
│   ├── stop_coco.sh
│   └── reset_coco_indices.sh
├── llm/
│   ├── scenario1.dsl
│   ├── scenario2.dsl
│   └── scenario3.dsl
├── assistant/
│   └── scenario1.dsl
└── datasource/
    └── scenario1.dsl
```

### Global Configuration

```yaml
# tests/loadgen.yml
runner:
  total_rounds: 1
  no_warm: true
  reset_context: true
  assert_invalid: true
  default_endpoint: "$[[env.COCO_SERVER]]"

env:
  COCO_SERVER: http://127.0.0.1:9000
  ADMIN_PASSWORD: PASSword_123
```

### Single Scenario Debugging

```bash
cd coco
loadgen -config ./tests/loadgen.yml -run ./tests/llm/scenario3.dsl -debug
```

### Full Regression Testing

```bash
cd coco
python3 ./tests/assets/run_integration_tests.py
```

The script automatically: stops environment → resets data → starts service → executes each DSL → cleans up.

### Links

- [tests/loadgen.yml](https://github.com/infinilabs/coco-server/blob/main/tests/loadgen.yml)
- [tests/assets/run_integration_tests.py](https://github.com/infinilabs/coco-server/blob/main/tests/assets/run_integration_tests.py)
- [tests/llm/scenario3.dsl](https://github.com/infinilabs/coco-server/blob/main/tests/llm/scenario3.dsl)

---

## 9. Project Organization

Recommended test project structure:

```
project/
├── loadgen.yml              # Global configuration
├── dict/                    # Corpus files
│   ├── ip.txt
│   ├── user.txt
│   └── keywords.txt
├── cases/                   # Test cases grouped by function
│   ├── bulk/
│   │   ├── create.dsl
│   │   └── mixed.dsl
│   ├── search/
│   │   └── random.dsl
│   └── crud/
│       └── lifecycle.dsl
└── suites/                  # Suite configs (batch run multiple cases)
    └── regression.yml
```

**Corpus file example (dict/ip.txt):**

```
192.168.1.1
10.0.0.1
172.16.0.100
```

One value per line; a random line is picked for each request.

**Reference repositories:**
- [testing/setup/loadgen](https://github.com/infinilabs/testing/tree/main/setup/loadgen)
- [testing/suites](https://github.com/infinilabs/testing/tree/main/suites)

---

## 10. Command Quick Reference

```bash
# Basic benchmark (60 seconds, 100 concurrent)
loadgen -run bench.dsl -d 60 -c 100

# Fixed QPS
loadgen -run bench.dsl -d 60 -c 100 -r 1000

# Fixed total requests
loadgen -run bench.dsl -d 600 -c 100 -l 50000

# Enable gzip compression
loadgen -run bench.dsl -d 60 -c 100 -compress

# Functional validation (single execution)
loadgen -run test.dsl

# Specify config file
loadgen -run test.dsl -config env/production.yml

# Debug mode
loadgen -run test.dsl -debug
```
