---
weight: 30
title: 参数参考
---

# 参数参考

本页是 Loadgen 所有配置项的完整参考。

---

## 命令行参数

```bash
loadgen [选项]
```

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `-run` | string | `loadgen.dsl` | DSL 测试文件路径 |
| `-config` | string | `loadgen.yml` | YAML 配置文件路径 |
| `-d` | int | `5` | 测试持续时间（秒） |
| `-c` | int | `1` | 并发线程数 |
| `-r` | int | `-1`（不限） | 每秒最大请求数（固定 QPS） |
| `-l` | int | `-1`（不限） | 请求总数上限 |
| `-compress` | bool | `false` | 使用 gzip 压缩请求体 |
| `-timeout` | int | `60` | 请求超时时间（秒） |
| `-dial-timeout` | int | `3` | 连接超时时间（秒） |
| `-read-timeout` | int | `0`（使用 -timeout） | 读超时时间（秒） |
| `-write-timeout` | int | `0`（使用 -timeout） | 写超时时间（秒） |
| `-cpu` | int | `-1`（全部） | 使用的 CPU 核数 |
| `-mem` | int | `-1`（不限） | 内存软上限（MB） |
| `-log` | string | — | 日志级别：trace, debug, info, warn, error, off |
| `-debug` | bool | `false` | 调试模式，panic 时立即退出并打印完整栈 |
| `-v` | bool | — | 输出版本信息 |

---

## 环境变量（env）

在 YAML 或 DSL 中定义环境变量默认值，运行时可通过命令行环境变量覆盖。

**YAML 格式：**

```yaml
env:
  ES_ENDPOINT: http://localhost:9200
  ES_USERNAME: admin
  ES_PASSWORD: admin
```

**DSL 格式：**

```text
# env: {
#   ES_ENDPOINT: "http://localhost:9200",
#   ES_USERNAME: "admin",
#   ES_PASSWORD: "admin",
# },
```

**引用方式：** `$[[env.ES_ENDPOINT]]`

---

## 运行控制（runner）

控制测试运行行为的全局参数。

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `total_rounds` | int | 不设置 | 执行轮次。设置后进入轮次模式，不设置则按时间运行 |
| `no_warm` | bool | `false` | 跳过预热阶段 |
| `log_requests` | bool | `false` | 记录所有请求详情到日志 |
| `log_status_codes` | []int | — | 仅记录指定状态码的请求（如 `[0, 500]`） |
| `assert_invalid` | bool | `false` | 断言失败时 Loadgen 退出码为 1 |
| `assert_error` | bool | `false` | 请求错误时 Loadgen 退出码为 2 |
| `reset_context` | bool | `false` | 每轮次重置上下文变量 |
| `valid_status_codes_during_warmup` | []int | — | 预热期间视为有效的状态码（如 `[200, 201, 404]`） |
| `disable_header_names_normalizing` | bool | `false` | 禁止自动格式化响应头名称 |
| `default_endpoint` | string | — | 默认请求地址，请求行中省略 host 时自动补全 |
| `default_basic_auth` | object | — | 默认 Basic Auth 认证信息 |

**DSL 格式示例：**

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

## 变量定义（variables）

变量在请求中通过 `$[[变量名]]` 引用，每次请求自动取值。

### 变量类型一览

| 类型 | 说明 | 必要参数 |
|------|------|----------|
| `file` | 从文本文件随机取一行 | `path` |
| `list` | 从内联列表随机取一个 | `data` |
| `sequence` | 32 位自增整数 | — |
| `sequence64` | 64 位自增整数 | — |
| `range` | 指定范围内随机整数 | `from`, `to` |
| `random_array` | 由其他变量组合生成随机数组 | `variable_key`, `size` |
| `uuid` | UUID v4 字符串 | — |
| `now_local` | 当前时间（本地时区） | — |
| `now_utc` | 当前时间（UTC，完整格式） | — |
| `now_utc_lite` | 当前时间（UTC，ISO 短格式） | — |
| `now_unix` | 当前 Unix 时间戳 | — |
| `now_with_format` | 当前时间（自定义格式） | `format` |

### 变量参数详解

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

| 参数 | 说明 |
|------|------|
| `path` | 外部文本文件路径，每行一个值 |
| `data` | 追加到文件内容之后的额外数据列表 |
| `replace` | 对取到的值做字符替换（用于转义特殊字符） |

**NDJSON 语料支持：**

`file` 类型变量同样支持 NDJSON 格式的语料文件（每行一个完整 JSON 对象）。这在批量写入场景中非常实用——可以直接将语料文件作为 bulk 请求体的数据源：

```text
# dict/docs.json（每行一个 JSON 文档）
{"title": "Elasticsearch 入门", "author": "alice", "views": 100}
{"title": "性能调优实践", "author": "bob", "views": 350}
{"title": "集群运维指南", "author": "coco", "views": 220}
```

变量定义：

```text
{
  name: "doc",
  type: "file",
  path: "dict/docs.json",
}
```

在 bulk 请求中引用：

```text
POST $[[env.ES_ENDPOINT]]/_bulk
{"index": {"_index": "articles", "_id": "$[[uuid]]"}}
$[[doc]]
# request: {
#   body_repeat_times: 1000,
# },
```

每次重复时 `$[[doc]]` 会从语料文件中随机取一行 JSON，直接作为文档内容写入，无需在 DSL 中手写 JSON 结构。这种方式适合：

- 用真实数据做写入压测
- 批量导入已有数据集
- 保持文档结构多样性

#### list

```text
{
  name: "user",
  type: "list",
  data: ["alice", "bob", "coco"],
}
```

| 参数 | 说明 |
|------|------|
| `data` | 字符串数组，每次请求随机取一个 |

#### sequence / sequence64

```text
{
  name: "id",
  type: "sequence",
  from: 1,
  to: 999999,
}
```

| 参数 | 说明 |
|------|------|
| `from` | 起始值（默认 0） |
| `to` | 最大值（到达后回绕） |

#### range

```text
{
  name: "suffix",
  type: "range",
  from: 10,
  to: 1000,
}
```

| 参数 | 说明 |
|------|------|
| `from` | 范围下限 |
| `to` | 范围上限 |

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

| 参数 | 说明 |
|------|------|
| `variable_type` | 元素类型：`number` 或 `string` |
| `variable_key` | 数据源变量名 |
| `size` | 数组长度 |
| `square_bracket` | 是否用 `[` `]` 包裹输出 |
| `string_bracket` | 每个元素前后包裹的字符串 |
| `replace` | 对整个输出做字符替换 |

#### now_with_format

```text
{
  name: "ts",
  type: "now_with_format",
  format: "2006-01-02T15:04:05-0700",
}
```

| 参数 | 说明 |
|------|------|
| `format` | Go 时间格式字符串（[格式参考](https://www.geeksforgeeks.org/time-formatting-in-golang/)） |

---

## 请求配置（request）

每个请求可以通过 `# request: { ... },` 注释块配置以下参数：

| 参数 | 类型 | 说明 |
|------|------|------|
| `basic_auth` | object | 请求级 Basic Auth（覆盖 runner 默认值） |
| `headers` | []object | 自定义请求头列表 |
| `disable_header_names_normalizing` | bool | 禁止自动格式化请求头名称 |
| `runtime_variables` | object | 请求级运行时变量（同一请求内共享） |
| `runtime_body_line_variables` | object | 请求体行级变量（每行独立生成） |
| `body_repeat_times` | int | 请求体重复次数（用于 bulk 批量构造） |

**完整示例：**

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

| 类型 | 生成时机 | 典型用途 |
|------|----------|----------|
| `runtime_variables` | 每次请求生成一次，整个请求体共享同一个值 | 批次号、追踪 ID |
| `runtime_body_line_variables` | 请求体中每一"行对"独立生成一个新值 | routing、文档级唯一 ID |

> **何为"行对"？** 对于 `_bulk` 请求，每条文档由 action 行 + source 行组成。当 `body_repeat_times: N` 复制 N 份时，`runtime_body_line_variables` 确保每份拥有独立值，而 `runtime_variables` 在整个请求中保持不变。

**图示：**

```
请求体（body_repeat_times: 3）:
  行对 1: routing_no = aaa   ← runtime_body_line_variables 每行对独立
  行对 2: routing_no = bbb
  行对 3: routing_no = ccc
  所有行: batch_no   = xxx   ← runtime_variables 整个请求共享
```

### DSL 格式 vs YAML 格式

Loadgen 支持两种配置格式。**YAML 格式解析更快**，在高吞吐场景下建议优先使用：

**YAML 格式（推荐用于压测）：**

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

上面的示例中，body 包含 2 个 action+source 行对，配合 `body_repeat_times: 5000` 最终每次请求发送 10000 条文档。每个行对的 `routing_no` 独立生成，而 `batch_no` 整个请求共享一个值。

**等价 DSL 格式：**

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

> **性能提示：** YAML 格式省去了运行时 DSL 解析开销，在极端吞吐场景下可观测到更高的客户端发送速率。功能验证和集成测试场景两种格式性能差异可忽略。

---

## 断言（assert）

断言用于校验响应内容，不满足时当前轮次停止后续请求。

### 简写格式

```text
# assert: (200, {status: "ok"}),
```

等价于状态码为 200 且响应体包含 `{"status": "ok"}`。

### 标准格式

```text
# assert: {
#   _ctx.response.status: 200,
#   _ctx.response.body_json.hits.total.value: 1,
# },
```

### 可断言的上下文变量

| 路径 | 说明 |
|------|------|
| `_ctx.response.status` | HTTP 状态码 |
| `_ctx.response.header` | 响应头 |
| `_ctx.response.body` | 响应体原文 |
| `_ctx.response.body_json.*` | 响应体 JSON 字段（支持嵌套路径） |
| `_ctx.elapsed` | 请求耗时（毫秒） |

### 退出码

| 退出码 | 条件 |
|--------|------|
| 0 | 全部通过 |
| 1 | 存在断言失败且 `runner.assert_invalid: true` |
| 2 | 存在请求错误且 `runner.assert_error: true` |

---

## 动态变量注册（register）

从响应中提取值并注册为变量，供后续请求使用。

```text
# register: [
#   { token: "_ctx.response.body_json.access_token" },
#   { task_id: "_ctx.response.body_json.task" },
# ]
```

注册后可通过 `$[[token]]`、`$[[task_id]]` 引用。

典型场景：登录获取 token → 后续请求使用 token。

---

## 预热行为

默认情况下，Loadgen 在正式测试前会将所有请求执行一次作为预热：

- 预热结果输出到终端
- 如果预热失败会提示是否继续
- 通过 `runner.no_warm: true` 跳过预热
- 通过 `runner.valid_status_codes_during_warmup` 指定预热期间视为成功的状态码

---

## 输出指标

测试完成后 Loadgen 输出以下信息：

```
75 requests finished in 13.261499ms, 139.34MB sent, 0.00bytes received

[Loadgen Client Metrics]
Requests/sec:        14.85           ← 客户端实际 QPS
Request Traffic/sec: 27.58MB         ← 发送流量/秒
Total Transfer/sec:  27.58MB         ← 总流量/秒（含响应）
Fastest Request:     99.375µs        ← 最快请求
Slowest Request:     1.898458ms      ← 最慢请求
Status 0:            75              ← 各状态码计数

[Latency Metrics]
75 samples of 75 events
Cumulative:  13.261499ms             ← 累计耗时
HMean:       148.19µs               ← 调和平均
Avg.:        176.819µs              ← 算术平均
p50:         146.75µs               ← 中位数
p75:         168µs                  ← 75 分位
p95:         202.583µs              ← 95 分位
p99:         409.458µs              ← 99 分位
p999:        1.898458ms             ← 99.9 分位
Long 5%:     690.854µs              ← 最慢 5% 平均
Short 5%:    106.5µs                ← 最快 5% 平均
Max:         1.898458ms
Min:         99.375µs
Range:       1.799083ms
StdDev:      204.268µs
Rate/sec.:   14.85

[Latency Distribution]
          99µs - 279µs ----------    ← 延迟分布直方图
         279µs - 459µs -
         459µs - 639µs -

[Estimated Server Metrics]
Requests/sec:        5655.47         ← 估算服务端 QPS
Avg Req Time:        176.819µs       ← 估算服务端处理时间
Transfer/sec:        10.26GB         ← 估算服务端吞吐
```

### 指标说明

| 区块 | 说明 |
|------|------|
| **Loadgen Client Metrics** | 从客户端视角统计的吞吐与极值 |
| **Latency Metrics** | 详细延迟分布，包含百分位数 |
| **Latency Distribution** | 可视化延迟直方图 |
| **Estimated Server Metrics** | 剔除网络开销后的服务端估算值 |

### 状态码与错误

- `Status 200: N` — 正常响应
- `Status 0: N` — 连接失败或超时（未收到 HTTP 响应）
- `Number of Errors` — 网络/连接错误总数（启用 `assert_error` 时显示）
- `Number of Invalid` — 断言失败总数（启用 `assert_invalid` 时显示）

> 建议配合 INFINI Console 监控或 APM 查看服务端真实指标，客户端统计仅供参考。
