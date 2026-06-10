---
weight: 20
title: 快速上手
---

# 快速上手

本节帮助你在 5 分钟内理解 Loadgen 的工作方式并运行第一个测试。

## 核心文件

Loadgen 运行只需要两个文件：

| 文件 | 作用 |
|------|------|
| `loadgen.yml` | 全局配置：环境变量、默认连接信息 |
| `loadgen.dsl` | 测试定义：变量、请求、断言、变量注册 |

## 最小示例

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

**运行**

```bash
loadgen -run loadgen.dsl -config loadgen.yml -d 10 -c 1
```

这条命令会以 1 并发持续 10 秒请求集群健康接口。

## DSL 语法结构

一个 DSL 文件由以下部分组成（从上到下）：

```text
# env: { ... },              ← 环境变量默认值（可选）
# runner: { ... },           ← 运行控制参数（可选）
# variables: [ ... ],        ← 变量定义（可选）

METHOD /path                  ← 请求行（必须）
{request body}                ← 请求体（可选，紧跟请求行）
# request: { ... },          ← 请求级配置（可选）
# assert: { ... },           ← 断言（可选）
# register: [ ... ]          ← 动态变量注册（可选）
```

### 注释块约定

DSL 中所有配置块均写在 `#` 注释内，使用类 JSON5 语法：

```text
# runner: {
#   total_rounds: 1,
#   no_warm: true,
# },
```

这种设计使 DSL 文件本身就是合法的 HTTP 请求描述，配置信息作为元数据嵌入。

### 变量引用

使用 `$[[变量名]]` 引用变量值：

- `$[[env.ES_ENDPOINT]]` — 引用环境变量
- `$[[uuid]]` — 引用 variables 中定义的变量
- `$[[token]]` — 引用通过 register 动态注册的变量

### 多请求编排

一个 DSL 文件可以包含多个请求，按顺序执行：

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

前一个请求通过 `register` 注册的变量可被后续请求使用，实现请求编排。

## YAML 配置（loadgen.yml）

YAML 配置提供全局设定，和 DSL 中的 runner/env 等价但优先级更低（DSL 中的定义会覆盖 YAML 中的同名项）。

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

## 两种运行模式

| 模式 | 行为 | 适用场景 |
|------|------|----------|
| 性能测试模式（默认） | 在指定时间内循环执行所有请求 | 压测、基准测试 |
| 轮次模式 | 执行固定轮次后退出 | 功能验证、集成测试、CI |

切换方式：设置 `runner.total_rounds` 即进入轮次模式。

## 下一步

- [参数参考]({{< relref "reference" >}}) — 完整变量、runner、request、CLI 参数说明
- [实战示例]({{< relref "examples" >}}) — 按场景分类的可运行示例
