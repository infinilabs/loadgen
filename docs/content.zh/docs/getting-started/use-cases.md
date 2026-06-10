---
weight: 60
title: 使用场景与实战示例
---

# 使用场景与实战示例

本文基于现有测试样例整理，重点覆盖以下场景：

- 性能压测（批量写入、混合请求、搜索）
- 易用模式（一次性验证、快速回归）
- 变量与上下文复用（字典、序列、运行时变量）
- 基于 JSON 返回值断言与动态变量注册
- 如何用语料库数据驱动 JSON 请求与响应验证

如果你是第一次接触 Loadgen，建议先阅读安装与基础说明，再回到本文按场景复制示例。

## 样例来源

以下示例来自仓库内已有用例，便于直接复用：

- [testing/setup/loadgen/cases/bulk/create.yml](https://github.com/infinilabs/testing/blob/main/setup/loadgen/cases/bulk/create.yml)
- [testing/setup/loadgen/cases/bulk/mixed.yml](https://github.com/infinilabs/testing/blob/main/setup/loadgen/cases/bulk/mixed.yml)
- [testing/setup/loadgen/cases/search/random_search.yml](https://github.com/infinilabs/testing/blob/main/setup/loadgen/cases/search/random_search.yml)
- [testing/setup/loadgen/cases/dummy/loadgen.yml](https://github.com/infinilabs/testing/blob/main/setup/loadgen/cases/dummy/loadgen.yml)
- [testing/setup/easysearch/cases/crud/create_without_id/loadgen.yml](https://github.com/infinilabs/testing/blob/main/setup/easysearch/cases/crud/create_without_id/loadgen.yml)
- [testing/setup/gateway/cases/filters/set_response/gateway.yml](https://github.com/infinilabs/testing/blob/main/setup/gateway/cases/filters/set_response/gateway.yml)
- [testing/setup/gateway/cases/filters/set_response/loadgen.yml](https://github.com/infinilabs/testing/blob/main/setup/gateway/cases/filters/set_response/loadgen.yml)
- [testing/setup/gateway/cases/echo/echo_with_context/gateway.yml](https://github.com/infinilabs/testing/blob/main/setup/gateway/cases/echo/echo_with_context/gateway.yml)
- [testing/setup/gateway/cases/echo/echo_with_context/loadgen.yml](https://github.com/infinilabs/testing/blob/main/setup/gateway/cases/echo/echo_with_context/loadgen.yml)
- [testing/setup/loadgen/assets/dict/ip.txt](https://github.com/infinilabs/testing/blob/main/setup/loadgen/assets/dict/ip.txt)
- [testing/setup/loadgen/assets/dict/user.txt](https://github.com/infinilabs/testing/blob/main/setup/loadgen/assets/dict/user.txt)
- [testing/setup/loadgen/assets/dict/type.txt](https://github.com/infinilabs/testing/blob/main/setup/loadgen/assets/dict/type.txt)

## 场景 1：批量写入压测（高吞吐）

适用目标：验证写入吞吐、节点负载、队列压力。

可直接参考样例：

- [testing/setup/loadgen/cases/bulk/create.yml](https://github.com/infinilabs/testing/blob/main/setup/loadgen/cases/bulk/create.yml)

关键点：

- 使用 body_repeat_times 构造单请求多文档
- 使用 runtime_variables 固定批次号
- 使用 runtime_body_line_variables 为每条文档生成不同 routing

可直接复用的核心片段（来自 [testing/setup/loadgen/cases/bulk/create.yml](https://github.com/infinilabs/testing/blob/main/setup/loadgen/cases/bulk/create.yml)）：

```yaml
requests:
  - request:
      method: POST
      runtime_variables:
        batch_no: uuid
      runtime_body_line_variables:
        routing_no: uuid
      url: http://127.0.0.1:8000/_bulk
      body_repeat_times: 1000
      body: |
        { "create" : { "_index" : "ab-test-$[[suffix]]","_type":"_doc", "_id" : "$[[id]]" , "routing" : "$[[routing_no]]" } }
        { "id" : "$[[uuid]]","routing_no" : "$[[routing_no]]","batch_number" : "$[[batch_no]]", "random_no" : "$[[suffix]]","ip" : "$[[ip]]","now_local" : "$[[now_local]]","now_with_format" : "$[[now_with_format]]","now_unix" : "$[[now_unix]]" }
```

推荐启动参数：

- 基准压测：loadgen -run testing/setup/loadgen/cases/bulk/create.yml -d 60 -c 100
- 固定 QPS：loadgen -run testing/setup/loadgen/cases/bulk/create.yml -d 60 -c 100 -r 2000
- 固定请求总量：loadgen -run testing/setup/loadgen/cases/bulk/create.yml -d 600 -c 100 -l 50000

## 场景 2：混合请求压测（贴近真实流量）

适用目标：模拟 create、update、delete 混合流量，观察集群稳定性。

可直接参考样例：

- [testing/setup/loadgen/cases/bulk/mixed.yml](https://github.com/infinilabs/testing/blob/main/setup/loadgen/cases/bulk/mixed.yml)

建议做法：

- 先用小并发验证语义正确性（-c 1 -d 5）
- 再逐步提升并发和持续时间
- 结合监控看失败率、慢请求、磁盘 IO 与队列积压

## 场景 3：搜索压测（查询延迟与吞吐）

适用目标：评估查询性能和过滤条件命中差异。

可直接参考样例：

- [testing/setup/loadgen/cases/search/random_search.yml](https://github.com/infinilabs/testing/blob/main/setup/loadgen/cases/search/random_search.yml)

关键点：

- 使用 file 变量从语料文件中随机取值
- 请求中用变量拼接查询参数
- 可结合 assert 校验结果结构

变量片段（来自 [testing/setup/loadgen/cases/search/random_search.yml](https://github.com/infinilabs/testing/blob/main/setup/loadgen/cases/search/random_search.yml)）：

```yaml
variables:
  - name: ip
    type: file
    path: dict/ip.txt
  - name: id
    type: sequence
  - name: uuid
    type: uuid
```

## 场景 4：快速可用性检查（一次性验证）

适用目标：部署后健康检查、CI 中快速回归。

可直接参考样例：

- [testing/setup/loadgen/cases/dummy/loadgen.yml](https://github.com/infinilabs/testing/blob/main/setup/loadgen/cases/dummy/loadgen.yml)

关键点：

- runner.total_rounds: 1
- runner.no_warm: true
- runner.assert_invalid: true
- runner.assert_error: true

这组配置适合在流水线中“快速失败”。

## 场景 5：变量与上下文复用

### 5.1 字典变量（语料库）

语料文件示例：

- [testing/setup/loadgen/assets/dict/ip.txt](https://github.com/infinilabs/testing/blob/main/setup/loadgen/assets/dict/ip.txt)
- [testing/setup/loadgen/assets/dict/user.txt](https://github.com/infinilabs/testing/blob/main/setup/loadgen/assets/dict/user.txt)

变量定义示例：

```yaml
variables:
  - name: ip
    type: file
    path: testing/setup/loadgen/assets/dict/ip.txt
  - name: user
    type: file
    path: testing/setup/loadgen/assets/dict/user.txt
```

说明：

- file 变量每次请求会从文件中随机取一行
- 适合模拟真实语料分布（用户、IP、关键字、标签）

### 5.2 运行时变量复用

见样例：

- [testing/setup/loadgen/cases/bulk/create.yml](https://github.com/infinilabs/testing/blob/main/setup/loadgen/cases/bulk/create.yml)

这个模式用于“一批请求共享一个值，每行再拥有独立值”：

- runtime_variables：请求级变量（例如 batch_no）
- runtime_body_line_variables：请求体每行变量（例如 routing_no）

## 场景 6：JSON 返回值断言与动态变量注册

### 6.1 基于 body_json 断言

见样例：

- [testing/setup/gateway/cases/filters/set_response/loadgen.yml](https://github.com/infinilabs/testing/blob/main/setup/gateway/cases/filters/set_response/loadgen.yml)
- [testing/setup/easysearch/cases/crud/create_without_id/loadgen.yml](https://github.com/infinilabs/testing/blob/main/setup/easysearch/cases/crud/create_without_id/loadgen.yml)

典型断言：

- _ctx.response.status
- _ctx.response.body_json.message
- _ctx.response.body_json._source.a

断言片段（来自 [testing/setup/gateway/cases/filters/set_response/loadgen.yml](https://github.com/infinilabs/testing/blob/main/setup/gateway/cases/filters/set_response/loadgen.yml)）：

```yaml
assert:
  equals:
    _ctx.response.status: 200
    _ctx.response.body_json.message: hello world
```

### 6.2 从 JSON 返回值注册变量（register）

见样例：

- [testing/setup/easysearch/cases/crud/create_without_id/loadgen.yml](https://github.com/infinilabs/testing/blob/main/setup/easysearch/cases/crud/create_without_id/loadgen.yml)

先注册：

- doc_id: _ctx.response.body_json._id

注册片段（来自 [testing/setup/easysearch/cases/crud/create_without_id/loadgen.yml](https://github.com/infinilabs/testing/blob/main/setup/easysearch/cases/crud/create_without_id/loadgen.yml)）：

```yaml
register:
  - doc_id: _ctx.response.body_json._id
```

后续请求复用：

- URL 里使用 $[[doc_id]] 进行读取、更新、删除

这是一种非常实用的“链路式测试”写法。

## 场景 7：如何基于语料库做 JSON 请求与 JSON 响应验证

这个场景通常有两层：

- 用语料库驱动请求内容（Loadgen 侧）
- 对 JSON 响应按字段断言（Loadgen 侧）

### 7.1 语料驱动 JSON 请求

推荐方式：

1. 把语料整理成文本行（例如用户、IP、类型）
2. 通过 file 变量读取
3. 在 JSON body 中插入变量

现成语料可参考：

- [testing/setup/loadgen/assets/dict/user.txt](https://github.com/infinilabs/testing/blob/main/setup/loadgen/assets/dict/user.txt)
- [testing/setup/loadgen/assets/dict/ip.txt](https://github.com/infinilabs/testing/blob/main/setup/loadgen/assets/dict/ip.txt)
- [testing/setup/loadgen/assets/dict/type.txt](https://github.com/infinilabs/testing/blob/main/setup/loadgen/assets/dict/type.txt)

### 7.2 JSON 响应字段断言

参考：

- [testing/setup/gateway/cases/filters/set_response/gateway.yml](https://github.com/infinilabs/testing/blob/main/setup/gateway/cases/filters/set_response/gateway.yml)
- [testing/setup/gateway/cases/filters/set_response/loadgen.yml](https://github.com/infinilabs/testing/blob/main/setup/gateway/cases/filters/set_response/loadgen.yml)

该用例中 Gateway 固定返回 JSON，Loadgen 使用 _ctx.response.body_json.message 做字段级断言。

响应构造片段（来自 [testing/setup/gateway/cases/filters/set_response/gateway.yml](https://github.com/infinilabs/testing/blob/main/setup/gateway/cases/filters/set_response/gateway.yml)）：

```yaml
flow:
  - name: test
    filter:
      - set_response:
          status: 200
          content_type: application/json
          body: '{"message":"hello world"}'
```

### 7.3 上下文拼装响应内容

参考：

- [testing/setup/gateway/cases/echo/echo_with_context/gateway.yml](https://github.com/infinilabs/testing/blob/main/setup/gateway/cases/echo/echo_with_context/gateway.yml)
- [testing/setup/gateway/cases/echo/echo_with_context/loadgen.yml](https://github.com/infinilabs/testing/blob/main/setup/gateway/cases/echo/echo_with_context/loadgen.yml)

这种模式通过上下文变量拼接响应，再由 Loadgen 校验响应片段，适合做“动态响应内容”测试。

## 场景 8：建议的目录组织（便于团队协作）

建议把压测工程拆成三层：

- suites：环境与套件编排
- setup/.../cases：按功能分组的用例
- assets/dict：语料与字典

当前仓库可参考：

- [testing/suites](https://github.com/infinilabs/testing/tree/main/suites)
- [testing/setup/loadgen/cases](https://github.com/infinilabs/testing/tree/main/setup/loadgen/cases)
- [testing/setup/loadgen/assets/dict](https://github.com/infinilabs/testing/tree/main/setup/loadgen/assets/dict)

## 场景 9：性能测试实践建议

- 先验证正确性，再加压
- 保持单次变更可对比（只改一个变量）
- 固定语料与随机语料分开测试
- 关注 Number of Errors 与 Number of Invalid
- 对服务端性能结论，优先看监控平台而不是只看客户端汇总

## 场景 10：Loadgen 做 Coco AI 集成测试

我们已经在 Coco AI 项目里用 Loadgen 落地了集成测试，核心模式是：

- 用一份公共 loadgen.yml 提供环境变量与全局 runner
- 用多份 DSL 场景文件表达权限、资源、行为测试
- 用脚本自动执行全部 DSL，失败即中断

可直接参考（公开仓库）：

- [coco/tests/loadgen.yml](https://github.com/infinilabs/coco-server/blob/main/tests/loadgen.yml)
- [coco/tests/assets/run_integration_tests.py](https://github.com/infinilabs/coco-server/blob/main/tests/assets/run_integration_tests.py)
- [coco/tests/llm/scenario3.dsl](https://github.com/infinilabs/coco-server/blob/main/tests/llm/scenario3.dsl)
- [coco/tests/assistant/scenario1.dsl](https://github.com/infinilabs/coco-server/blob/main/tests/assistant/scenario1.dsl)
- [coco/tests/datasource/scenario1.dsl](https://github.com/infinilabs/coco-server/blob/main/tests/datasource/scenario1.dsl)

### 10.0 DSL 怎么写（可直接复制）

下面给出 3 组最常用 DSL 片段，覆盖“登录拿 token -> 带鉴权调用 -> 断言与链路变量”。

示例 A：登录并注册 token（register）

```text
POST /account/login
{
  "email": "admin@mail.com",
  "password": "$[[env.ADMIN_PASSWORD]]"
}
# assert: (200, {status: "ok"}),
# register: [
#   { admin_token: "_ctx.response.body_json.access_token" },
# ]
```

示例 B：使用 token 调用业务接口（headers + 断言）

```text
GET /model_provider/_search?from=0&size=10&query=llm_a
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
# assert: (200, {
#   "hits.total.value": 1
# })
```

示例 C：授权写入 + JSON 字段断言

```text
POST /resources/llm-provider/$[[env.LLM_A_ID]]/share
{
  "shares": [
    {
      "resource_type": "llm-provider",
      "resource_id": "$[[env.LLM_A_ID]]",
      "principal_type": "user",
      "principal_id": "$[[env.A_ID]]",
      "permission": $[[env.EDIT_ACCESS]]
    }
  ],
  "revokes": []
}
# request: {
#   headers: [
#     {Authorization: "Bearer $[[admin_token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
# assert: (200, {
#   "created": [
#     {
#       "resource_id": "$[[env.LLM_A_ID]]",
#       "principal_id": "$[[env.A_ID]]",
#       "permission": $[[env.EDIT_ACCESS]]
#     }
#   ]
# })
```

### 10.1 单场景调试

当你只想调试某一类能力（例如 LLM 权限共享）时，可以直接运行单个 DSL：

```bash
cd coco
loadgen -config ./tests/loadgen.yml -run ./tests/llm/scenario3.dsl -debug
```

对应的现成脚本：

- [coco/tests/llm/test.sh](https://github.com/infinilabs/coco-server/blob/main/tests/llm/test.sh)

最小可运行配置（loadgen.yml）示例：

```yaml
runner:
  total_rounds: 1
  no_warm: true
  reset_context: true
  assert_invalid: true
  default_endpoint: "$[[env.COCO_SERVER]]"

env:
  COCO_SERVER: http://127.0.0.1:9000
  ADMIN_PASSWORD: PASSword_123
  LLM_A_ID: d4cnisa8sig62phqk9ug
  A_ID: dd3626ce78553287cd9216f4e4974d26
  EDIT_ACCESS: 4
```

### 10.2 全量集成回归

如果要做完整回归，使用 Python 脚本自动发现 tests 目录下所有 DSL 并串行执行：

```bash
cd coco
python3 ./tests/assets/run_integration_tests.py
```

这个脚本会自动执行以下动作：

1. 停止并清理环境
2. 启动 Coco 服务
3. 调用 loadgen 运行 DSL
4. 再次清理并进入下一个场景

相关脚本：

- [coco/tests/assets/start_coco.sh](https://github.com/infinilabs/coco-server/blob/main/tests/assets/start_coco.sh)
- [coco/tests/assets/stop_coco.sh](https://github.com/infinilabs/coco-server/blob/main/tests/assets/stop_coco.sh)
- [coco/tests/assets/reset_coco_indices.sh](https://github.com/infinilabs/coco-server/blob/main/tests/assets/reset_coco_indices.sh)

### 10.3 适用收益

这个模式对集成测试非常实用：

- 同时验证 API 协议、鉴权、资源可见性与权限语义
- 用 register 从登录响应里提取 token，实现跨步骤编排
- 场景 DSL 可读性高，便于团队协作与评审
- 可以平滑接入 CI，作为发布前强校验

### 10.4 DSL 编写速查（避免只看链接）

1. 请求行：`METHOD /path` 或 `METHOD http://host/path`
2. 请求体：紧跟在请求行后，支持 JSON
3. 请求配置：放在 `# request: { ... },` 注释块
4. 断言：放在 `# assert: (...)` 注释块
5. 注册变量：放在 `# register: [ ... ]` 注释块

示例（完整链路最小版）：

```text
POST /account/login
{"email":"admin@mail.com","password":"$[[env.ADMIN_PASSWORD]]"}
# assert: (200, {status: "ok"}),
# register: [
#   { token: "_ctx.response.body_json.access_token" },
# ]

GET /account/profile
# request: {
#   headers: [
#     {Authorization: "Bearer $[[token]]"},
#   ],
#   disable_header_names_normalizing: true,
# },
# assert: (200, {
#   "status": "ok"
# })
```

## 场景 11：全量变量与参数模板（建议复制后按需删减）

这一节给一份“尽量完整”的模板，覆盖常见变量类型与参数。

### 11.1 变量类型全量示例

```text
# variables: [
#   {
#     name: "ip",
#     type: "file",
#     path: "dict/ip.txt",
#     data: ["10.0.0.1", "10.0.0.2"],
#     replace: {
#       '"': '\\"',
#       '\\': '\\\\',
#     },
#   },
#   {
#     name: "user",
#     type: "list",
#     data: ["alice", "bob", "coco"],
#   },
#   {
#     name: "id",
#     type: "sequence",
#     from: 1,
#     to: 999999,
#   },
#   {
#     name: "id64",
#     type: "sequence64",
#     from: 1,
#     to: 999999999999,
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
#     name: "now_fmt",
#     type: "now_with_format",
#     format: "2006-01-02T15:04:05-0700",
#   },
#   {
#     name: "suffix",
#     type: "range",
#     from: 10,
#     to: 1000,
#   },
#   {
#     name: "id_list",
#     type: "random_array",
#     variable_type: "number",
#     variable_key: "suffix",
#     square_bracket: false,
#     string_bracket: "",
#     size: 10,
#   },
#   {
#     name: "str_list",
#     type: "random_array",
#     variable_type: "string",
#     variable_key: "user",
#     square_bracket: true,
#     string_bracket: "\"",
#     size: 5,
#     replace: {
#       '"': "'",
#       "[": "{",
#       "]": "}",
#     },
#   },
# ]
```

变量类型速览：

- file
- list
- sequence
- sequence64
- range
- random_array
- uuid
- now_local
- now_utc
- now_utc_lite
- now_unix
- now_with_format

### 11.2 runner 参数全量示例

```text
# runner: {
#   total_rounds: 1,
#   no_warm: false,
#   valid_status_codes_during_warmup: [200, 201, 404],
#   log_requests: false,
#   log_status_codes: [0, 400, 500],
#   assert_invalid: false,
#   assert_error: false,
#   reset_context: false,
#   disable_header_names_normalizing: false,
#   default_endpoint: "$[[env.ES_ENDPOINT]]",
#   default_basic_auth: {
#     username: "$[[env.ES_USERNAME]]",
#     password: "$[[env.ES_PASSWORD]]",
#   },
# },
```

### 11.3 request 参数全量示例

```text
POST /_bulk
{"index": {"_index": "demo-$[[suffix]]", "_id": "$[[uuid]]", "routing": "$[[routing_no]]"}}
{"id":"$[[id]]","user":"$[[user]]","ip":"$[[ip]]","batch":"$[[batch_no]]","ts":"$[[now_fmt]]"}
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
# assert: {
#   _ctx.response.status: 200,
# },
# register: [
#   { task_id: "_ctx.response.body_json.task" },
# ]
```

## 常用命令清单

单用例运行：

```bash
loadgen -run testing/setup/loadgen/cases/search/random_search.yml -d 30 -c 10
```

固定 QPS：

```bash
loadgen -run testing/setup/loadgen/cases/bulk/create.yml -d 60 -c 100 -r 1000
```

固定总请求数：

```bash
loadgen -run testing/setup/loadgen/cases/bulk/create.yml -d 600 -c 100 -l 50000
```

套件运行：

```bash
loadgen -config testing/suites/gateway-sample.yml
```

Coco AI 全量集成测试：

```bash
cd coco
python3 ./tests/assets/run_integration_tests.py
```

