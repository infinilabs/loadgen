---
weight: 80
title: "版本历史"
---

# 版本发布日志

这里是`loadgen`历史版本发布的相关说明。

## 1.27.0 (2024-12-13)

### Improvements

- 代码开源，统一采用 Github [仓库](https://github.com/infinilabs/loadgen) 进行开发
- 保持与 Console 相同版本
- 同步更新 Framework 修复的已知问题

### Bug fix

- 修复 API 接口测试逻辑异常问题

## 1.26.1 (2024-08-13)

### Improvements

- 与 INFINI Console 统一版本号
- 同步更新 Framework 修复的已知问题

## 1.26.0 (2024-06-07)

### Improvements

- 与 INFINI Console 统一版本号
- 同步更新 Framework 修复的已知问题

## 1.25.0 (2024-04-30)

### Improvements

- 与 INFINI Console 统一版本号
- 同步更新 Framework 修复的已知问题

## 1.24.0 (2024-04-15)

### Improvements

- 与 INFINI Console 统一版本号
- 同步更新 Framework 修复的已知问题

## 1.22.0 (2024-01-26)

### Improvements

- 与 INFINI Console 统一版本号

## 1.8.0 (2023-11-02)

### Breaking changes

- 原 Loadrun 功能并入 Loadgen
- 测试请求、断言等使用新的 Loadgen DSL 语法来配置

## 1.7.0 (2023-04-20)

### Breaking changes

- `variables` 不再允许定义相同 `name` 的变量。

### Features

- 增加 `log_status_code` 配置，支持打印特定状态码的请求日志。

## 1.6.0 (2023-04-06)

### Breaking ghanges

- `file` 类型变量默认不再转义 `"` `\` 字符，使用 `replace` 功能手动设置变量转义。

### Features

- 变量定义增加 `replace` 选项，可选跳过 `"` `\` 转义。

### Improvements

- 优化内存占用

### Bug fix

- 修复 YAML 字符串无法使用 `\n` 的问题
- 修复无效的 assert 配置被忽略的问题

## 1.5.1

### Bug fix

- 修复配置文件中无效的变量语法。

## 1.5.0

### Features

- 配置文件添加 `assert` 配置，支持验证访问数据。
- 配置文件添加 `register` 配置，支持注册动态变量。
- 配置文件添加 `env` 配置，支持加载使用环境变量。
- 支持在 `headers` 配置中使用动态变量。

### Improvements

- 启动参数添加 `-l`： 控制发送请求的数量。
- 配置文件添加 `runner.no_warm` 参数跳过预热阶段。

### Bug fix
