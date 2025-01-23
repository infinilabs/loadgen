---
weight: 50
title: "Benchmark Testing"
---

# Benchmark Testing

INFINI Loadgen is a lightweight performance testing tool specifically designed for Easysearch, Elasticsearch, and OpenSearch.

Features of Loadgen:

- Robust performance
- Lightweight and dependency-free
- Supports template-based parameter randomization
- Supports high concurrency
- Supports balanced traffic control at the benchmark end
- Supports server response validation

> Download link: <https://release.infinilabs.com/loadgen/>

## Loadgen

Loadgen is easy to use. After the tool is downloaded and decompressed, you will get three files: an executable program, a configuration file `loadgen.yml`, and a test file `loadgen.dsl`. The configuration file example is as follows:

```yaml
env:
  ES_USERNAME: elastic
  ES_PASSWORD: elastic
  ES_ENDPOINT: http://localhost:8000
```

The test file example is as follows:

```text
# runner: {
#   // total_rounds: 1
#   no_warm: false,
#   // Whether to log all requests
#   log_requests: false,
#   // Whether to log all requests with the specified response status
#   log_status_codes: [0, 500],
#   assert_invalid: false,
#   assert_error: false,
# },
# variables: [
#   {
#     name: "ip",
#     type: "file",
#     path: "dict/ip.txt",
#     // Replace special characters in the value
#     replace: {
#       '"': '\\"',
#       '\\': '\\\\',
#     },
#   },
#   {
#     name: "id",
#     type: "sequence",
#   },
#   {
#     name: "id64",
#     type: "sequence64",
#   },
#   {
#     name: "uuid",
#     type: "uuid",
#   },
#   {
#     name: "now_local",
#     type: "now_local",
#   },
#   {
#     name: "now_utc",
#     type: "now_utc",
#   },
#   {
#     name: "now_utc_lite",
#     type: "now_utc_lite",
#   },
#   {
#     name: "now_unix",
#     type: "now_unix",
#   },
#   {
#     name: "now_with_format",
#     type: "now_with_format",
#     // https://programming.guide/go/format-parse-string-time-date-example.html
#     format: "2006-01-02T15:04:05-0700",
#   },
#   {
#     name: "suffix",
#     type: "range",
#     from: 10,
#     to: 1000,
#   },
#   {
#     name: "bool",
#     type: "range",
#     from: 0,
#     to: 1,
#   },
#   {
#     name: "list",
#     type: "list",
#     data: ["medcl", "abc", "efg", "xyz"],
#   },
#   {
#     name: "id_list",
#     type: "random_array",
#     variable_type: "number", // number/string
#     variable_key: "suffix", // variable key to get array items
#     square_bracket: false,
#     size: 10, // how many items for array
#   },
#   {
#     name: "str_list",
#     type: "random_array",
#     variable_type: "number", // number/string
#     variable_key: "suffix", // variable key to get array items
#     square_bracket: true,
#     size: 10, // how many items for array
#     replace: {
#       // Use ' instead of " for string quotes
#       '"': "'",
#       // Use {} instead of [] as array brackets
#       "[": "{",
#       "]": "}",
#     },
#   },
# ],

POST $[[env.ES_ENDPOINT]]/medcl/_search
{ "track_total_hits": true, "size": 0, "query": { "terms": { "patent_id": [ $[[id_list]] ] } } }
# request: {
#   runtime_variables: {batch_no: "uuid"},
#   runtime_body_line_variables: {routing_no: "uuid"},
#   basic_auth: {
#     username: "$[[env.ES_USERNAME]]",
#     password: "$[[env.ES_PASSWORD]]",
#   },
# },
```

### Running Mode Settings

By default, Loadgen runs in performance testing mode, repeating all requests in `requests` for the specified duration (`-d`). If you only need to check the test results once, you can set the number of executions of `requests` by `runner.total_rounds`.

### HTTP Header Handling

By default, Loadgen will automatically format the HTTP response headers (`user-agent: xxx` -> `User-Agent: xxx`). If you need to precisely determine the response headers returned by the server, you can disable this behavior by setting `runner.disable_header_names_normalizing`.

## Usage of Variables

In the above configuration, `variables` is used to define variable parameters, identified by `name`. In a constructed request, `$[[Variable name]]` can be used to access the value of the variable. The currently supported variable types are:

| Type              | Description                                                                                                     | Parameters                                                                                                                                                                                                                                                      |
| ----------------- | --------------------------------------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `file`            | Load variables from file                                                                                        | `path`: the path of the data files<br>`data`: a list of values, will get appended to the end of the data specified by `path` file                                                                                                                               |
| `list`            | Defined variables inline                                                                                        | use `data` to define a string array                                                                                                                                                                                                                             |
| `sequence`        | 32-bit Variable of the auto incremental numeric type                                                            | `from`: the minimum of the values<br>`to`: the maximum of the values                                                                                                                                                                                            |
| `sequence64`      | 64-bit Variable of the auto incremental numeric type                                                            | `from`: the minimum of the values<br>`to`: the maximum of the values                                                                                                                                                                                            |
| `range`           | Variable of the range numbers, support parameters `from` and `to` to define the range                           | `from`: the minimum of the values<br>`to`: the maximum of the values                                                                                                                                                                                            |
| `random_array`    | Generate a random array, data elements come from the variable specified by `variable_key`                       | `variable_key`: data source variable<br>`size`: length of the output array<br>`square_bracket`: `true/false`, whether the output value needs `[` and `]`<br>`string_bracket`: string, the specified string will be attached before and after the output element |
| `uuid`            | UUID string type variable                                                                                       |                                                                                                                                                                                                                                                                 |
| `now_local`       | Current time, local time zone                                                                                   |                                                                                                                                                                                                                                                                 |
| `now_utc`         | Current time, UTC time zone. Output format: `2006-01-02 15:04:05.999999999 -0700 MST`                           |                                                                                                                                                                                                                                                                 |
| `now_utc_lite`    | Current time, UTC time zone. Output format: `2006-01-02T15:04:05.000`                                           |                                                                                                                                                                                                                                                                 |
| `now_unix`        | Current time, Unix timestamp                                                                                    |                                                                                                                                                                                                                                                                 |
| `now_with_format` | Current time, supports custom `format` parameter to format the time string, such as: `2006-01-02T15:04:05-0700` | `format`: output time format ([example](https://www.geeksforgeeks.org/time-formatting-in-golang/))                                                                                                                                                              |

### Variable Usage Example

Variable parameters of the `file` type are loaded from an external text file. One variable parameter occupies one line. When one variable of the file type is accessed, one variable value is taken randomly. An example of the variable format is as follows:

```text
# test/user.txt
medcl
elastic
```

Tips about how to generate a random string of fixed length, such as 1024 per line:

```bash
LC_CTYPE=C tr -dc A-Za-z0-9_\!\@\#\$\%\^\&\*\(\)-+= < /dev/random | head -c 1024 >> 1k.txt
```

### Environment Variables

Loadgen supports loading and using environment variables. You can specify the default values in the `loadgen.dsl` configuration. Loadgen will overwrite the variables at runtime if they are also specified by the command-line environment.

The environment variables can be accessed by `$[[env.ENV_KEY]]`:

```text
#// Configure default values for environment variables
# env: {
#   ES_USERNAME: "elastic",
#   ES_PASSWORD: "elastic",
#   ES_ENDPOINT: "http://localhost:8000",
# },

#// Use runtime variables
GET $[[env.ES_ENDPOINT]]/medcl/_search
{"query": {"match": {"name": "$[[user]]"}}}
# request: {
#   // Use runtime variables
#   basic_auth: {
#     username: "$[[env.ES_USERNAME]]",
#     password: "$[[env.ES_PASSWORD]]",
#   },
# },
```

## Request Definition

The `requests` node is used to set requests to be executed by Loadgen in sequence. Loadgen supports fixed-parameter requests and requests constructed using template-based variable parameters. The following is an example of a common query request:

```text
GET http://localhost:8000/medcl/_search?q=name:$[[user]]
# request: {
#   username: elastic,
#   password: pass,
# },
```

In the above query, Loadgen conducts queries based on the `medcl` index and executes one query based on the `name` field. The value of each request is from the random variable `user`.

### Simulating Bulk Ingestion

It is very easy to use Loadgen to simulate bulk ingestion. Configure one index operation in the request body and then use the `body_repeat_times` parameter to randomly replicate several parameterized requests to complete the preparation of a batch of requests. See the following example.

```text
POST http://localhost:8000/_bulk
{"index": {"_index": "medcl-y4", "_type": "doc", "_id": "$[[uuid]]"}}
{"id": "$[[id]]", "field1": "$[[user]]", "ip": "$[[ip]]", "now_local": "$[[now_local]]", "now_unix": "$[[now_unix]]"}
# request: {
#   basic_auth: {
#     username: "test",
#     password: "testtest",
#  },
#  body_repeat_times: 1000,
# },
```

### Response Assertions

You can use the `assert` configuration to check the response values. `assert` now supports most of all the [condition checkers](https://docs.infinilabs.com/gateway/main/docs/references/flow/#condition-type) of INFINI Gateway.

```text
GET http://localhost:8000/medcl/_search?q=name:$[[user]]
# request: {
#   basic_auth: {
#     username: "test",
#     password: "testtest",
#  },
# },
# assert: {
#   _ctx.response.status: 201,
# },
```

The

response value can be accessed from the `_ctx` value, currently it contains these values:

| Parameter                 | Description                                                                                     |
| ------------------------- | ----------------------------------------------------------------------------------------------- |
| `_ctx.response.status`    | HTTP response status code                                                                       |
| `_ctx.response.header`    | HTTP response headers                                                                           |
| `_ctx.response.body`      | HTTP response body text                                                                         |
| `_ctx.response.body_json` | If the HTTP response body is a valid JSON string, you can access the JSON fields by `body_json` |
| `_ctx.elapsed`            | The time elapsed since request sent to the server (milliseconds)                                |

If the request failed (e.g. the host is not reachable), Loadgen will record it under `Number of Errors` as part of the testing output. If you configured `runner.assert_error: true`, Loadgen will exit as `exit(2)` when there're any requests failed.

If the assertion failed, Loadgen will record it under `Number of Invalid` as part of the testing output and skip the subsequent requests in this round. If you configured `runner.assert_invalid: true`, Loadgen will exit as `exit(1)` when there're any assertions failed.

### Dynamic Variable Registration

Each request can use `register` to dynamically set the variables based on the response value, a common usage is to update the parameters of the later requests based on the previous responses.

In the below example, we're registering the response value `_ctx.response.body_json.test.settings.index.uuid` of the `$[[env.ES_ENDPOINT]]/test` to the `index_id` variable, then we can access it by `$[[index_id]]`.

```text
GET $[[env.ES_ENDPOINT]]/test
# register: [
#   {index_id: "_ctx.response.body_json.test.settings.index.uuid"},
# ],
# assert: (200, {}),
```

## Running the Benchmark

Run the Loadgen program to perform the benchmark test as follows:

```text
$ loadgen -d 30 -c 100 -compress -run

loadgen.dsl


   __   ___  _      ___  ___   __    __
  / /  /___\/_\    /   \/ _ \ /__\/\ \ \
 / /  //  ///_\\  / /\ / /_\//_\ /  \/ /
/ /__/ \_//  _  \/ /_// /_\\//__/ /\  /
\____|___/\_/ \_/___,'\____/\__/\_\ \/

[LOADGEN] A http load generator and testing suit.
[LOADGEN] 1.0.0_SNAPSHOT, 83f2cb9, Sun Jul 4 13:52:42 2021 +0800, medcl, support single item in dict files
[07-19 16:15:00] [INF] [instance.go:24] workspace: data/loadgen/nodes/0
[07-19 16:15:00] [INF] [loader.go:312] warmup started
[07-19 16:15:00] [INF] [app.go:306] loadgen now started.
[07-19 16:15:00] [INF] [loader.go:316] [GET] http://localhost:8000/medcl/_search
[07-19 16:15:00] [INF] [loader.go:317] status: 200,<nil>,{"took":1,"timed_out":false,"_shards":{"total":1,"successful":1,"skipped":0,"failed":0},"hits":{"total":{"value":0,"relation":"eq"},"max_score":null,"hits":[]}}
[07-19 16:15:00] [INF] [loader.go:316] [GET] http://localhost:8000/medcl/_search?q=name:medcl
[07-19 16:15:00] [INF] [loader.go:317] status: 200,<nil>,{"took":1,"timed_out":false,"_shards":{"total":1,"successful":1,"skipped":0,"failed":0},"hits":{"total":{"value":0,"relation":"eq"},"max_score":null,"hits":[]}}
[07-19 16:15:01] [INF] [loader.go:316] [POST] http://localhost:8000/_bulk
[07-19 16:15:01] [INF] [loader.go:317] status: 200,<nil>,{"took":120,"errors":false,"items":[{"index":{"_index":"medcl-y4","_type":"doc","_id":"c3qj9123r0okahraiej0","_version":1,"result":"created","_shards":{"total":2,"successful":1,"failed":0},"_seq_no":5735852,"_primary_term":3,"status":201}}]}
[07-19 16:15:01] [INF] [loader.go:325] warmup finished

5253 requests in 32.756483336s, 524.61KB sent, 2.49MB received

[Loadgen Client Metrics]
Requests/sec:		175.10
Request Traffic/sec:	17.49KB
Total Transfer/sec:	102.34KB
Avg Req Time:		5.711022ms
Fastest Request:	440.448µs
Slowest Request:	3.624302658s
Number of Errors:	0
Number of Invalid:	0
Status 200:		5253

[Estimated Server Metrics]
Requests/sec:		160.37
Transfer/sec:		93.73KB
Avg Req Time:		623.576686ms
```

Before the formal benchmark, Loadgen will execute all requests once for warm-up. If an error occurs, it will prompt whether to continue. The warm-up request results will also be output to the terminal. After execution, a summary of the execution will be output. You can skip this check phase by setting `runner.no_warm`.

> Since the final result of Loadgen is the cumulative statistics after all requests are completed, there may be inaccuracies. It is recommended to monitor Elasticsearch's various operating indicators in real-time through the Kibana monitoring dashboard.

### Command Line Parameters

Loadgen will loop through the requests defined in the configuration file. By default, Loadgen will only run for `5s` and then automatically exit. If you want to extend the runtime or increase concurrency, you can control it by setting parameters at startup. Check the help command as follows:

```text
$ loadgen -help
Usage of loadgen:
  -c int
      Number of concurrent threads (default 1)
  -compress
      Compress requests with gzip
  -config string
      the location of config file (default "loadgen.yml")
  -cpu int
      the number of CPUs to use (default -1)
  -d int
      Duration of tests in seconds (default 5)
  -debug
      run in debug mode, loadgen will quit on panic immediately with full stack trace
  -dial-timeout int
      Connection dial timeout in seconds, default 3s (default 3)
  -gateway-log string
      Log level of Gateway (default "debug")
  -l int
      Limit total requests (default -1)
  -log string
      the log level, options: trace,debug,info,warn,error,off
  -mem int
      the max size of Memory to use, soft limit in megabyte (default -1)
  -plugin value
      load additional plugins
  -r int
      Max requests per second (fixed QPS) (default -1)
  -read-timeout int
      Connection read timeout in seconds, default 0s (use -timeout)
  -run string
      DSL config to run tests (default "loadgen.dsl")
  -service string
      service management, options: install,uninstall,start,stop
  -timeout int
      Request timeout in seconds, default 60s (default 60)
  -v  version
  -write-timeout int
      Connection write timeout in seconds, default 0s (use -timeout)
```

### Limiting Client Workload

Using Loadgen and setting the command line parameter `-r` can limit the number of requests sent by the client per second, thereby evaluating the response time and load of Elasticsearch under fixed pressure, as follows:

```bash
loadgen -d 30 -c 100 -r 100
```

> Note: The client throughput limit may not be accurate enough in the case of massive concurrencies.

### Limiting the Total Number of Requests

By setting the parameter `-l`, you can control the total number of requests sent by the client to generate fixed documents. Modify the configuration as follows:

```text
#// loadgen-gw.dsl
POST http://localhost:8000/medcl-test/doc2/_bulk
{"index": {"_index": "medcl-test", "_id": "$[[uuid]]"}}
{"id": "$[[id]]", "field1": "$[[user]]", "ip": "$[[ip]]"}
# request: {
#   basic_auth: {
#     username: "test",
#     password: "testtest",
#   },
#   body_repeat_times: 1,
# },
```

Each request contains only one document, then execute Loadgen

```bash
loadgen -run loadgen-gw.dsl -d 600 -c 100 -l 50000
```

After execution, the Elasticsearch index `medcl-test` will have `50000` more records.

### Using Auto Incremental IDs to Ensure the Document Sequence

If you want the generated document IDs to increase regularly for easy comparison, you can use the `sequence` type auto incremental ID as the primary key and avoid using random numbers in the content, as follows:

```text
POST http://localhost:8000/medcl-test/doc2/_bulk
{"index": {"_index": "medcl-test", "_id": "$[[id]]"}}
{"id": "$[[id]]"}
# request: {
#   basic_auth: {
#     username: "test",
#     password: "testtest",
#   },
#   body_repeat_times: 1,
# },
```

### Reuse Variables in Request Context

In a request, we might want to use the same variable value, such as the `routing` parameter to control the shard destination, also store the field in the JSON document. You can use `runtime_variables` to set request-level variables, or `runtime_body_line_variables` to define request-body-level variables. If the request body is replicated N times, each line will be different, as shown in the following example:

```text
# variables: [
#   {name: "id", type: "sequence"},
#   {name: "uuid", type: "uuid"},
#   {name: "now_local", type: "now_local"},
#   {name: "now_utc", type: "now_utc"},
#   {name: "now_unix", type: "now_unix"},
#   {name: "suffix", type: "range", from: 10, to 15},
# ],

POST http://192.168.3.188:9206/_bulk
{"create": {"_index": "test-$[[suffix]]", "_type": "doc", "_id": "$[[uuid]]", "routing": "$[[routing_no]]"}}
{"id": "$[[uuid]]", "routing_no": "$[[routing_no]]", "batch_number": "$[[batch_no]]", "random_no": "$[[suffix]]", "ip": "$[[ip]]", "now_local": "$[[now_local]]", "now_unix": "$[[now_unix]]"}
# request: {
#   runtime_variables: {
#     batch_no: "id",
#   },
#   runtime_body_line_variables: {
#     routing_no: "uuid",
#   },
#   basic_auth: {
#     username: "ingest",
#     password: "password",
#   },
#   body_repeat_times: 10,
# },
```

We defined the `batch_no` variable to represent the same batch number in a batch of documents, and the `routing_no` variable to represent the routing value at each document level.

### Customize Header

```text
GET http://localhost:8000/test/_search
# request: {
#   headers: [
#     {Agent: "Loadgen-1"},
#   ],
#   disable_header_names_normalizing: false,
# },
```

By default, Loadgen will canonilize the HTTP header keys in the configuration (`user-agent: xxx` -> `User-Agent: xxx`). If you need to set the HTTP header keys exactly, you can disable this behavior by setting `disable_header_names_normalizing: true`.

## Running Test Suites

Loadgen supports running test cases in batches without writing test cases repeatedly. You can quickly test different environment configurations by switching suite configurations:

```yaml
# loadgen.yml
env:
  # Set up environments to run test suite
  LR_TEST_DIR: ./testing # The path to the test cases.
  # If you want to start gateway dynamically and automatically:
  LR_GATEWAY_CMD: ./bin/gateway # The path to the executable of INFINI Gateway
  LR_GATEWAY_HOST: 0.0.0.0:18000 # The binding host of the INFINI Gateway
  LR_GATEWAY_API_HOST: 0.0.0.0:19000 # The binding host of the INFINI Gateway API server
  # Set up other environments for the gateway and loadgen
  LR_ELASTICSEARCH_ENDPOINT: http://localhost:19201
  CUSTOM_ENV: myenv
tests:
  # The relative path of test cases under `LR_TEST_DIR`
  #
  # - gateway.yml: (Optional) the configuration to start the INFINI Gateway dynamically.
  # - loadgen.dsl: the configuration to run the loadgen tool.
  #
  # The environments set in `env` section will be passed to the INFINI Gateway and loadgen.
  - path: cases/gateway/echo/echo_with_context
```

### Environment Variables Configuration

Loadgen dynamically configures INFINI Gateway through environment variables specified in `env`. The following environment variables are required:

| Variable Name | Description             |
| ------------- | ----------------------- |
| `LR_TEST_DIR` | Directory of test cases |

If you need `loadgen` to dynamically start INFINI Gateway based on the configuration, you need to set the following environment variables:

| Variable Name         | Description                              |
| --------------------- | ---------------------------------------- |
| `LR_GATEWAY_CMD`      | Path to the executable of INFINI Gateway |
| `LR_GATEWAY_HOST`     | Binding host:port of INFINI Gateway      |
| `LR_GATEWAY_API_HOST` | Binding host:port of INFINI Gateway API  |

### Test Case Configuration

Test cases are configured in `tests`, each path points to a directory of a test case. Each test case needs to configure a `gateway.yml` (optional) and a `loadgen.dsl`. Configuration files can use environment variables configured in `env` (`$[[env.ENV_KEY]]`).

Example `gateway.yml` configuration:

```yaml
path.data: data
path.logs: log

entry:
  - name: my_es_entry
    enabled: true
    router: my_router
    max_concurrency: 200000
    network:
      binding: $[[env.LR_GATEWAY_HOST]]

flow:
  - name: hello_world
    filter:
      - echo:
          message: "hello world"
router:
  - name: my_router
    default_flow: hello_world
```

Example `loadgen.dsl` configuration:

```
# runner: {
#   total_rounds: 1,
#   no_warm: true,
#   log_requests: true,
#   assert_invalid: true,
#   assert_error: true,
# },

GET http://$[[env.LR_GATEWAY_HOST]]/
# assert: {
#   _ctx.response: {
#     status: 200,
#     body: "hello world",
#   },
# },
```

### Running Test Suites

After configuring `loadgen.yml`, you can run Loadgen with the following command:

```bash
loadgen -config loadgen.yml
```

Loadgen will run all the test cases specified in the configuration and output the test results:

```text
$ loadgen -config loadgen.yml
   __   ___  _      ___  ___   __    __
  / /  /___\/_\    /   \/ _ \ /__\/\ \ \
 / /  //  ///_\\  / /\ / /_\//_\ /  \/ /
/ /__/ \_//  _  \/ /_// /_\\//__/ /\  /
\____|___/\_/ \_/___,'\____/\__/\_\ \/

[LOADGEN] A http load generator and testing suit.
[LOADGEN] 1.0.0_SNAPSHOT, 83f2cb9, Sun Jul 4 13:52:42 2021 +0800, medcl, support single item in dict files
[02-21 10:50:05] [INF] [app.go:192] initializing loadgen
[02-21 10:50:05] [INF] [app.go:193] using config: /Users/kassian/Workspace/infini/src/infini.sh/testing/suites/dev.yml
[02-21 10:50:05] [INF] [instance.go:78] workspace: /Users/kassian/Workspace/infini/src/infini.sh/testing/data/loadgen/nodes/cfpihf15k34iqhpd4d00
[02-21 10:50:05] [INF] [app.go:399] loadgen is up and running now.
[2023-02-21 10:50:05][TEST][SUCCESS] [setup/loadgen/cases/dummy] duration: 105(ms)

1 requests in 68.373875ms, 0.00bytes sent, 0.00bytes received

[Loadgen Client Metrics]
Requests/sec:   0.20
Request Traffic/sec:  0.00bytes
Total Transfer/sec: 0.00bytes
Avg Req Time:   5s
Fastest Request:  68.373875ms
Slowest Request:  68.373875ms
Number of Errors: 0
Number of Invalid:  0
Status 200:   1

[Estimated Server Metrics]
Requests/sec:   14.63
Transfer/sec:   0.00bytes
Avg Req Time:   68.373875ms


[2023-02-21 10:50:06][TEST][FAILED] [setup/gateway/cases/echo/echo_with_context/] duration: 1274(ms)
#0 request, GET http://$[[env.LR_GATEWAY_HOST]]/any/, assertion failed, skiping subsequent requests
1 requests in 1.255678s, 0.00bytes sent, 0.00bytes received

[Loadgen Client Metrics]
Requests/sec:   0.20
Request Traffic/sec:  0.00bytes
Total Transfer/sec: 0.00bytes
Avg Req Time:   5s
Fastest Request:  1.255678s
Slowest Request:  1.255678s
Number of Errors: 1
Number of Invalid:  1
Status 0:   1

[Estimated Server Metrics]
Requests/sec:   0.80
Transfer/sec:   0.00bytes
Avg Req Time:   1.255678s

```
