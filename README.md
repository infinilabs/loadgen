
# INFINI Loadgen


Highlights of Loadgen:

- Robust performance
- Lightweight and dependency-free
- Random selection of template-based parameters
- High concurrency
- Balanced traffic control at the benchmark end
- Validate server responses.

Install with script:

```
curl -sSL http://get.infini.cloud | bash -s -- -p loadgen
```

> Or download from here: [http://release.infinilabs.com/loadgen/](http://release.infinilabs.com/loadgen/)

```
➜  /tmp mkdir loadgen
➜  /tmp curl -sSL http://get.infini.cloud | bash -s -- -p loadgen -d /tmp/loadgen

                                 @@@@@@@@@@@
                                @@@@@@@@@@@@
                                @@@@@@@@@@@@
                               @@@@@@@@@&@@@
                              #@@@@@@@@@@@@@
        @@@                   @@@@@@@@@@@@@
       &@@@@@@@              &@@@@@@@@@@@@@
       @&@@@@@@@&@           @@@&@@@@@@@&@
      @@@@@@@@@@@@@@@@      @@@@@@@@@@@@@@
      @@@@@@@@@@@@@@@@@@&   @@@@@@@@@@@@@
        %@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
            @@@@@@@@@@@@&@@@@@@@@@@@@@@@
    @@         ,@@@@@@@@@@@@@@@@@@@@@@@&
    @@@@@.         @@@@@&@@@@@@@@@@@@@@
   @@@@@@@@@@          @@@@@@@@@@@@@@@#
   @&@@@&@@@&@@@          &@&@@@&@@@&@
  @@@@@@@@@@@@@.              @@@@@@@*
  @@@@@@@@@@@@@                  %@@@
 @@@@@@@@@@@@@
/@@@@@@@&@@@@@
@@@@@@@@@@@@@
@@@@@@@@@@@@@
@@@@@@@@@@@@        Welcome to INFINI Labs!


Now attempting the installation...

Name: [loadgen], Version: [1.26.1-598], Path: [/tmp/loadgen]
File: [https://release.infinilabs.com/loadgen/stable/loadgen-1.26.1-598-mac-arm64.zip]
##=O#- #

Installation complete. [loadgen] is ready to use!


----------------------------------------------------------------
cd /tmp/loadgen && ./loadgen-mac-arm64
----------------------------------------------------------------


   __ _  __ ____ __ _  __ __
  / // |/ // __// // |/ // /
 / // || // _/ / // || // /
/_//_/|_//_/  /_//_/|_//_/

©INFINI.LTD, All Rights Reserved.
```

## Loadgen

Loadgen is easy to use. After the tool is downloaded and decompressed, two files are obtained: one executable program and one configuration file `loadgen.yml`. An example of the configuration file is as follows:

```
env:
  ES_USERNAME: elastic
  ES_PASSWORD: elastic
runner:
  # total_rounds: 1
  no_warm: false
  log_requests: false
  assert_invalid: false
  assert_error: false
variables:
  - name: ip
    type: file
    path: test/ip.txt
  - name: user
    type: file
    path: test/user.txt
  - name: id
    type: sequence
  - name: uuid
    type: uuid
  - name: now_local
    type: now_local
  - name: now_utc
    type: now_utc
  - name: now_unix
    type: now_unix
requests:
  - request:
      method: GET
      basic_auth:
        username: $[[env.ES_USERNAME]]
        password: $[[env.ES_PASSWORD]]
      url: http://localhost:8000/medcl/_search
      body: '{  "query": {"match": {    "name": "$[[user]]"  }}}'
```

### Runner Configurations

By default, `loadgen` will run under the benchmarking mode, repeating through all the `requests` during the specified duration (`-d`). If you only need to test the responses, setting `runner.total_rounds: 1` will let `loadgen` run for only once.

### HTTP Headers Canonization

By default, `loadgen` will canonilize the HTTP response header keys received from the server side (`user-agent: xxx` -> `User-Agent: xxx`). If you need to assert the header keys exactly, you can set `runner.disable_header_names_normalizing: true` to disable this behavior.

## Usage of Variables

In the above configuration, `variables` is used to define variable parameters and variables are identified by `name`. In a constructed request, `$[[Variable name]]` can be used to access the value of the variable. Supported variable types are as follows:

| Type              | Description                                                                                              | Parameters                                                                                                                                                                                                                                     |
| ----------------- | -------------------------------------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `file`            | Load variables from file                                                                                 | `path`: the path of the data files<br>`data`: a list of values, will get appended to the end of the data specified by `path` file                                                                                                              |
| `list`            | Defined variables inline                                                                                 | use `data` to define a string array                                                                                                                                                                                                            |
| `sequence`        | 32-bit Variable of the auto incremental numeric type                                                     | `from`: the minimum of the values<br>`to`: the maximum of the values                                                                                                                                                                           |
| `sequence64`      | 64-bit Variable of the auto incremental numeric type                                                     | `from`: the minimum of the values<br>`to`: the maximum of the values                                                                                                                                                                           |
| `range`           | Variable of the range numbers, support parameters `from` and `to` to define the range                    | `from`: the minimum of the values<br>`to`: the maximum of the values                                                                                                                                                                           |
| `random_array`    | Generate a random array from the variable specified by `variable_key`                                    | `variable_key`: the variable name for the source of array values<br>`size`: the size of array<br>`square_bracket`: `true/false`, whether to add `[]` for the outputed array<br>`string_bracket`: the string to surround the outputed elements. |
| `uuid`            | Variable of the UUID character type                                                                      |                                                                                                                                                                                                                                                |
| `now_local`       | Current time and local time zone                                                                         |                                                                                                                                                                                                                                                |
| `now_utc`         | Current time and UTC time zone                                                                           |                                                                                                                                                                                                                                                |
| `now_unix`        | Current time and Unix timestamp                                                                          |                                                                                                                                                                                                                                                |
| `now_with_format` | Current time，support parameter `format` to customize the output format， eg: `2006-01-02T15:04:05-0700` | `format`: the format of the time output ([Example](https://www.geeksforgeeks.org/time-formatting-in-golang/))                                                                                                                                  |

### Examples

Variable parameters of the `file` type are loaded from an external text file. One variable parameter occupies one line. When one variable of the file type is accessed, one variable value is taken randomly. An example of the variable format is as follows:

```
➜  loadgen git:(master) ✗ cat test/user.txt
medcl
elastic
```

Tips about how to generate a random string of fixed length, such as 1024 per line:

```
LC_CTYPE=C tr -dc A-Za-z0-9_\!\@\#\$\%\^\&\*\(\)-+= < /dev/random | head -c 1024 >> 1k.txt
```

### Environment Variables

`loadgen` supporting loading and using environment variables in `loadgen.yml`, you can specify the default values in `env` configuration. `loadgen` will overwrite the variables at runtime if they're also specified by the command-line environment.

The environment variables can be access by `$[[env.ENV_KEY]]`:

```
# Default values for the environment variables.
env:
  ES_USERNAME: elastic
  ES_PASSWORD: elastic
  ES_ENDPOINT: http://localhost:8000
requests:
  - request:
      method: GET
      basic_auth:
        username: $[[env.ES_USERNAME]] # Use environment variables
        password: $[[env.ES_PASSWORD]] # Use environment variables
      url: $[[env.ES_ENDPOINT]]/medcl/_search # Use environment variables
      body: '{  "query": {"match": {    "name": "$[[user]]"  }}}'
```

## Request Definition

The `requests` node is used to set requests to be executed by Loadgen in sequence. Loadgen supports fixed-parameter requests and requests constructed using template-based variable parameters. The following is an example of a common query request.

```
requests:
  - request:
      method: GET
      basic_auth:
        username: elastic
        password: pass
      url: http://localhost:8000/medcl/_search?q=name:$[[user]]
```

In the above query, Loadgen conducts queries based on the `medcl` index and executes one query based on the `name` field. The value of each request is from the random variable `user`.

### Simulating Bulk Ingestion

It is very easy to use Loadgen to simulate bulk ingestion. Configure one index operation in the request body and then use the `body_repeat_times` parameter to randomly replicate several parameterized requests to complete the preparation of a batch of requests. See the following example.

```
  - request:
      method: POST
      basic_auth:
        username: test
        password: testtest
      url: http://localhost:8000/_bulk
      body_repeat_times: 1000
      body: |
        { "index" : { "_index" : "medcl-y4","_type":"doc", "_id" : "$[[uuid]]" } }
        { "id" : "$[[id]]","field1" : "$[[user]]","ip" : "$[[ip]]","now_local" : "$[[now_local]]","now_unix" : "$[[now_unix]]" }
```

### Response Assertions

You can use the `assert` configuration to check the response values. `assert` now supports most of all the [condition checkers](https://docs.infinilabs.com/gateway/main/docs/references/flow/#condition-type) of INFINI Gateway.

```
requests:
  - request:
      method: GET
      basic_auth:
        username: elastic
        password: pass
      url: http://localhost:8000/medcl/_search?q=name:$[[user]]
    assert:
      equals:
        _ctx.response.status: 201
```

The response value can be accessed from the `_ctx` value, currently it contains these values:

| Parameter                 | Description                                                                                     |
| ------------------------- | ----------------------------------------------------------------------------------------------- |
| `_ctx.response.status`    | HTTP response status code                                                                       |
| `_ctx.response.header`    | HTTP response headers                                                                           |
| `_ctx.response.body`      | HTTP response body text                                                                         |
| `_ctx.response.body_json` | If the HTTP response body is a valid JSON string, you can access the JSON fields by `body_json` |
| `_ctx.elapsed`            | The time elapsed since request sent to the server (milliseconds)                                |

If the request failed (e.g. the host is not reachable), `loadgen` will record it under `Number of Errors` as part of the testing output. If you configured `runner.assert_error: true`, `loadgen` will exit as `exit(2)` when there're any requests failed.

If the assertion failed, `loadgen` will record it under `Number of Invalid` as part of the testing output and skip the subsequent requests in this round. If you configured `runner.assert_invalid: true`, `loadgen` will exit as `exit(1)` when there're any assertions failed.

### Dynamic Variable Registration

Each request can use `register` to dynamically set the variables based on the response value, a common usage is to update the parameters of the later requests based on the previous responses.

In the below example, we're registering the response value `_ctx.response.body_json.test.settings.index.uuid` of the `$[[env.ES_ENDPOINT]]/test` to the `index_id` variable, then we can access it by `$[[index_id]]`.

```
requests:
  - request:
      method: GET
      url: $[[env.ES_ENDPOINT]]/test
    assert:
      equals:
        _ctx.response.status: 200
    register:
      - index_id: _ctx.response.body_json.test.settings.index.uuid
```

### Benchmark Test

Run Loadgen to perform the benchmark test as follows:

```
➜  loadgen git:(master) ✗ ./bin/loadgen -d 30 -c 100 -compress
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

209 requests finished in 10.031365126s, 0.00bytes sent, 32.86KB received

[Loadgen Client Metrics]
Requests/sec:		20.82
Request Traffic/sec:	0.00bytes
Total Transfer/sec:	3.27KB
Fastest Request:	1ms
Slowest Request:	182.437792ms
Status 302:		209

[Latency Metrics]
209 samples of 209 events
Cumulative:	10.031365126s
HMean:		46.31664ms
Avg.:		47.996962ms
p50: 		45.712292ms
p75:		51.6065ms
p95:		53.05475ms
p99:		118.162416ms
p999:		182.437792ms
Long 5%:	87.678145ms
Short 5%:	39.11217ms
Max:		182.437792ms
Min:		38.257791ms
Range:		144.180001ms
StdDev:		14.407579ms
Rate/sec.:	20.82

[Latency Distribution]
   38.257ms - 52.675ms ------------------------------
   52.675ms - 67.093ms --
   67.093ms - 81.511ms -
   81.511ms - 95.929ms -
  95.929ms - 110.347ms -
 110.347ms - 124.765ms -


[Estimated Server Metrics]
Requests/sec:		20.83
Avg Req Time:		47.996962ms
Transfer/sec:		3.28KB
```

Loadgen executes all requests once to warm up before the formal benchmark test. If an error occurs, a prompt is displayed, asking you whether to continue.
The warm-up request results are also output to the terminal. After execution, an execution summary is output.
You can set `runner.no_warm: true` to skip the warm-up stage.

> The final results of Loadgen are the cumulative statistics after all requests are executed, and they may be inaccurate. You are advised to start the Kibana dashboard to check all operating indicators of Elasticsearch in real time.

### CLI Parameters

Loadgen cyclically executes requests defined in the configuration file. By default, Loadgen runs for `5s` and then automatically exits. If you want to prolong the running time or increase the concurrency, you can set the tool's startup parameters. The help commands are as follows:

```
➜  loadgen git:(master) ✗ ./bin/loadgen --help
Usage of ./bin/loadgen:
  -c int
    	Number of concurrent threads (default 1)
  -compress
    	Compress requests with gzip
  -config string
    	the location of config file, default: loadgen.yml (default "loadgen.yml")
  -d int
    	Duration of tests in seconds (default 5)
  -debug
    	run in debug mode, loadgen will quit with panic error
  -l int
    	Limit total requests (default -1)
  -log string
    	the log level,options:trace,debug,info,warn,error (default "info")
  -r int
    	Max requests per second (fixed QPS) (default -1)
  -v	version
```

### Limiting the Client Workload

You can use Loadgen and set the CLI parameter `-r` to restrict the number of requests that can be sent by the client per second, so as to evaluate the response time and load of Elasticsearch under fixed pressure. See the following example.

```
➜  loadgen git:(master) ✗ ./bin/loadgen -d 30 -c 100 -r 100
```

> Note: The client throughput limit may not be accurate enough in the case of massive concurrencies.

### Limiting the Total Number of Requests

You can set the `-l` parameter to control the total number of requests that can be sent by the client, so as to generate a fixed number of documents. Modify the configuration as follows:

```
requests:
  - request:
      method: POST
      basic_auth:
        username: test
        password: testtest
      url: http://localhost:8000/medcl-test/doc2/_bulk
      body_repeat_times: 1
      body: |
        { "index" : { "_index" : "medcl-test", "_id" : "$[[uuid]]" } }
        { "id" : "$[[id]]","field1" : "$[[user]]","ip" : "$[[ip]]" }
```

Configured parameters use the content of only one document for each request. Then, the system executes Loadgen.

```
./bin/loadgen -config loadgen-gw.yml -d 600 -c 100 -l 50000
```

After execution, `50000` records are added for the Elasticsearch index `medcl-test`.

### Using Auto Incremental IDs to Ensure the Document Sequence

If the IDs of generated documents need to increase regularly to facilitate comparison, you can use the auto incremental IDs of the `sequence` type as the primary key and avoid using random numbers in the content. See the following example.

```
requests:
  - request:
      method: POST
      basic_auth:
        username: test
        password: testtest
      url: http://localhost:8000/medcl-test/doc2/_bulk
      body_repeat_times: 1
      body: |
        { "index" : { "_index" : "medcl-test", "_id" : "$[[id]]" } }
        { "id" : "$[[id]]" }
```

### Reuse variables in Request Context

In a request, we might want use the same variable value, such as the `routing` parameter to control the shard destination, also store the field in the JSON document.
You can use `runtime_variables` to set request-level variables, or `runtime_body_line_variables` to define request-body-level variables.
If the request body set `body_repeat_times`, each line will be different, as shown in the following example:

```
variables:
  - name: id
    type: sequence
  - name: uuid
    type: uuid
  - name: now_local
    type: now_local
  - name: now_utc
    type: now_utc
  - name: now_unix
    type: now_unix
  - name: suffix
    type: range
    from: 10
    to: 15
requests:
  - request:
      method: POST
      runtime_variables:
        batch_no: id
      runtime_body_line_variables:
        routing_no: uuid
      basic_auth:
        username: ingest
        password: password
      #url: http://localhost:8000/_search?q=$[[id]]
      url: http://192.168.3.188:9206/_bulk
      body_repeat_times: 10
      body: |
        { "create" : { "_index" : "test-$[[suffix]]","_type":"doc", "_id" : "$[[uuid]]" , "routing" : "$[[routing_no]]" } }
        { "id" : "$[[uuid]]","routing_no" : "$[[routing_no]]","batch_number" : "$[[batch_no]]", "random_no" : "$[[suffix]]","ip" : "$[[ip]]","now_local" : "$[[now_local]]","now_unix" : "$[[now_unix]]" }
```

We defined the `batch_no` variable to represent the same batch number in a batch of documents, and the `routing_no` variable to represent the routing value at each document level.

### Customize Header

```
requests:
  - request:
      method: GET
      url: http://localhost:8000/test/_search
      headers:
        - Agent: "Loadgen-1"
      disable_header_names_normalizing: false
```

By default, `loadgen` will canonilize the HTTP header keys before sending the request (`user-agent: xxx` -> `User-Agent: xxx`), if you need to set the header keys exactly as is, set `disable_header_names_normalizing: true`.

### Work with DSL

Loadgen also support simply the requests called DSL,for example, prepare a dsl file for loadgen, save as `bulk.dsl`:

```
POST /_bulk
{"index": {"_index": "$[[env.INDEX_NAME]]", "_type": "_doc", "_id": "$[[uuid]]"}}
{"id": "$[[id]]", "routing": "$[[routing_no]]", "batch": "$[[batch_no]]", "now_local": "$[[now_local]]", "now_unix": "$[[now_unix]]"}
```
And specify the dsl file with parameter `run`:

```
$ INDEX_NAME=medcl123 ES_ENDPOINT=https://localhost:9200 ES_USERNAME=admin  ES_PASSWORD=b14612393da0d4e7a70b ./bin/loadgen -run bulk.dsl
```

Now you should ready to rock~