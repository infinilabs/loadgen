---
weight: 20
title: Quick Start
---

# Quick Start

Get Loadgen running in 5 minutes — understand the core workflow and run your first test.

## Core Files

Loadgen requires only two files to run:

| File | Purpose |
|------|---------|
| `loadgen.yml` | Global configuration: environment variables, default connection info |
| `loadgen.dsl` | Test definition: variables, requests, assertions, variable registration |

## Minimal Example

**loadgen.yml**

```yaml
env:
  ES_ENDPOINT: http://localhost:9200
  ES_USERNAME: admin
  ES_PASSWORD: admin
```

**loadgen.dsl**

```text
GET $[[env.ES_ENDPOINT]]/_cluster/health
# request: {
#   basic_auth: {
#     username: "$[[env.ES_USERNAME]]",
#     password: "$[[env.ES_PASSWORD]]",
#   },
# },
# assert: {
#   _ctx.response.status: 200,
# },
```

**Run**

```bash
loadgen -run loadgen.dsl -config loadgen.yml -d 10 -c 1
```

This runs a 10-second test with 1 concurrent thread hitting the cluster health endpoint.

## DSL Syntax Structure

A DSL file is composed of the following sections (top to bottom):

```text
# env: { ... },              ← Environment variable defaults (optional)
# runner: { ... },           ← Run control parameters (optional)
# variables: [ ... ],        ← Variable definitions (optional)

METHOD /path                  ← Request line (required)
{request body}                ← Request body (optional, follows request line)
# request: { ... },          ← Per-request configuration (optional)
# assert: { ... },           ← Assertions (optional)
# register: [ ... ]          ← Dynamic variable registration (optional)
```

### Comment Block Convention

All configuration blocks in DSL are written inside `#` comments using JSON5-like syntax:

```text
# runner: {
#   total_rounds: 1,
#   no_warm: true,
# },
```

This design keeps the DSL file itself a valid HTTP request description, with configuration embedded as metadata.

### Variable References

Use `$[[variable_name]]` to reference variable values:

- `$[[env.ES_ENDPOINT]]` — Reference an environment variable
- `$[[uuid]]` — Reference a variable defined in `variables`
- `$[[token]]` — Reference a variable dynamically registered via `register`

### Multi-Request Orchestration

A single DSL file can contain multiple requests, executed sequentially:

```text
POST /account/login
{"email":"admin@mail.com","password":"$[[env.PASSWORD]]"}
# assert: (200, {}),
# register: [
#   { token: "_ctx.response.body_json.access_token" },
# ]

GET /api/data
# request: {
#   headers: [
#     {Authorization: "Bearer $[[token]]"},
#   ],
# },
# assert: {
#   _ctx.response.status: 200,
# },
```

Variables registered via `register` in a preceding request are available to subsequent requests, enabling request chaining.

## YAML Configuration (loadgen.yml)

YAML provides global settings. It is equivalent to DSL-level `runner`/`env` blocks but has lower priority (DSL definitions override same-named YAML entries).

```yaml
env:
  ES_ENDPOINT: http://localhost:9200
  ES_USERNAME: admin
  ES_PASSWORD: admin

runner:
  total_rounds: 1
  no_warm: true
  assert_invalid: true
  assert_error: true
  default_endpoint: "$[[env.ES_ENDPOINT]]"
  default_basic_auth:
    username: "$[[env.ES_USERNAME]]"
    password: "$[[env.ES_PASSWORD]]"

variables:
  - name: id
    type: sequence
  - name: uuid
    type: uuid
```

## Two Execution Modes

| Mode | Behavior | Use Case |
|------|----------|----------|
| Performance mode (default) | Loops all requests for the specified duration | Benchmarking, load testing |
| Round mode | Executes a fixed number of rounds then exits | Functional validation, integration tests, CI |

To switch: set `runner.total_rounds` to enter round mode.

## Next Steps

- [Reference]({{< relref "reference" >}}) — Complete variable, runner, request, and CLI parameter documentation
- [Examples]({{< relref "examples" >}}) — Runnable examples organized by scenario
