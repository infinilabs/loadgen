---
weight: 40
title: 实战示例
---

# 实战示例

本页按使用场景分类，每个示例都可以直接复制运行。

---

## 1. 批量写入压测

**目标：** 评估写入吞吐量、集群负载与队列压力。

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

**运行命令：**

```bash
# 持续 60 秒，100 并发
loadgen -d 60 -c 100

# 固定 QPS
loadgen -d 60 -c 100 -r 2000

# 固定请求总量
loadgen -d 600 -c 100 -l 50000
```

**参考样例：** [testing/setup/loadgen/cases/bulk/create.yml](https://github.com/infinilabs/testing/blob/main/setup/loadgen/cases/bulk/create.yml)

---

## 2. 搜索性能测试

**目标：** 评估查询延迟与吞吐，观察不同查询模式的差异。

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

**运行命令：**

```bash
loadgen -d 30 -c 10
```

**参考样例：** [testing/setup/loadgen/cases/search/random_search.yml](https://github.com/infinilabs/testing/blob/main/setup/loadgen/cases/search/random_search.yml)

---

## 3. 混合请求压测

**目标：** 模拟真实流量中的 create + update + delete 混合负载。

一个 DSL 文件中写多个请求，Loadgen 会按顺序循环执行：

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

**参考样例：** [testing/setup/loadgen/cases/bulk/mixed.yml](https://github.com/infinilabs/testing/blob/main/setup/loadgen/cases/bulk/mixed.yml)

---

## 4. 功能验证与 CI 集成

**目标：** 一次性执行所有请求，断言全部通过后退出。

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

**运行命令：**

```bash
loadgen -run health_check.dsl
echo $?  # 0=通过, 1=断言失败, 2=请求错误
```

适合放入 CI 流水线，任何断言不通过即失败退出。

**参考样例：** [testing/setup/loadgen/cases/dummy/loadgen.yml](https://github.com/infinilabs/testing/blob/main/setup/loadgen/cases/dummy/loadgen.yml)

---

## 5. 动态变量注册（链路式测试）

**目标：** 从前序请求提取值，供后续请求使用。

```text
# runner: {
#   total_rounds: 1,
#   no_warm: true,
#   assert_invalid: true,
# },

# 创建文档，提取生成的 _id
POST $[[env.ES_ENDPOINT]]/my-index/_doc
{"title": "test document", "user": "loadgen"}
# assert: {
#   _ctx.response.status: 201,
# },
# register: [
#   { doc_id: "_ctx.response.body_json._id" },
# ]

# 用注册的 doc_id 读取该文档
GET $[[env.ES_ENDPOINT]]/my-index/_doc/$[[doc_id]]
# assert: {
#   _ctx.response.status: 200,
#   _ctx.response.body_json._source.user: "loadgen",
# },

# 删除该文档
DELETE $[[env.ES_ENDPOINT]]/my-index/_doc/$[[doc_id]]
# assert: {
#   _ctx.response.status: 200,
# },
```

**参考样例：** [testing/setup/easysearch/cases/crud/create_without_id/loadgen.yml](https://github.com/infinilabs/testing/blob/main/setup/easysearch/cases/crud/create_without_id/loadgen.yml)

---

## 6. JSON 响应断言

**目标：** 对 JSON 响应体中特定字段做精确校验。

```text
GET $[[env.ES_ENDPOINT]]/my-index/_search
{"query": {"match_all": {}}}
# assert: {
#   _ctx.response.status: 200,
#   _ctx.response.body_json.hits.total.value: 100,
#   _ctx.response.body_json.timed_out: false,
# },
```

**简写断言格式：**

```text
GET /api/status
# assert: (200, {status: "ok", version: "1.0"}),
```

**参考样例：** [testing/setup/gateway/cases/filters/set_response/loadgen.yml](https://github.com/infinilabs/testing/blob/main/setup/gateway/cases/filters/set_response/loadgen.yml)

---

## 7. 带鉴权的 API 测试（Token 模式）

**目标：** 先登录获取 token，后续请求通过 Header 鉴权。

```text
# runner: {
#   total_rounds: 1,
#   no_warm: true,
#   assert_invalid: true,
#   default_endpoint: "$[[env.API_HOST]]",
# },

# 登录
POST /account/login
{"email": "admin@example.com", "password": "$[[env.ADMIN_PASSWORD]]"}
# assert: (200, {}),
# register: [
#   { token: "_ctx.response.body_json.access_token" },
# ]

# 使用 token 调用业务接口
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

# 创建资源
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

# 删除资源
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

## 8. Coco AI 集成测试实践

我们在 [Coco Server](https://github.com/infinilabs/coco-server) 项目中用 Loadgen 实现了完整的集成测试：

### 项目结构

```
tests/
├── loadgen.yml              # 全局配置（环境变量、runner）
├── assets/
│   ├── run_integration_tests.py   # 自动执行所有场景
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

### 全局配置

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

### 单场景调试

```bash
cd coco
loadgen -config ./tests/loadgen.yml -run ./tests/llm/scenario3.dsl -debug
```

### 全量回归测试

```bash
cd coco
python3 ./tests/assets/run_integration_tests.py
```

该脚本自动执行：停止环境 → 重置数据 → 启动服务 → 逐个执行 DSL → 清理。

### 相关链接

- [tests/loadgen.yml](https://github.com/infinilabs/coco-server/blob/main/tests/loadgen.yml)
- [tests/assets/run_integration_tests.py](https://github.com/infinilabs/coco-server/blob/main/tests/assets/run_integration_tests.py)
- [tests/llm/scenario3.dsl](https://github.com/infinilabs/coco-server/blob/main/tests/llm/scenario3.dsl)

---

## 9. 项目组织建议

推荐将测试工程按以下方式组织：

```
project/
├── loadgen.yml              # 全局配置
├── dict/                    # 语料文件
│   ├── ip.txt
│   ├── user.txt
│   └── keywords.txt
├── cases/                   # 按功能分组的测试用例
│   ├── bulk/
│   │   ├── create.dsl
│   │   └── mixed.dsl
│   ├── search/
│   │   └── random.dsl
│   └── crud/
│       └── lifecycle.dsl
└── suites/                  # 套件配置（批量运行多用例）
    └── regression.yml
```

**语料文件示例（dict/ip.txt）：**

```
192.168.1.1
10.0.0.1
172.16.0.100
```

每行一个值，变量每次请求随机取一行。

**参考仓库：**
- [testing/setup/loadgen](https://github.com/infinilabs/testing/tree/main/setup/loadgen)
- [testing/suites](https://github.com/infinilabs/testing/tree/main/suites)

---

## 10. 常用命令速查

```bash
# 基础压测（60 秒、100 并发）
loadgen -run bench.dsl -d 60 -c 100

# 固定 QPS
loadgen -run bench.dsl -d 60 -c 100 -r 1000

# 固定总请求数
loadgen -run bench.dsl -d 600 -c 100 -l 50000

# 启用 gzip 压缩
loadgen -run bench.dsl -d 60 -c 100 -compress

# 功能验证（单次执行）
loadgen -run test.dsl

# 指定配置文件
loadgen -run test.dsl -config env/production.yml

# 调试模式
loadgen -run test.dsl -debug
```
