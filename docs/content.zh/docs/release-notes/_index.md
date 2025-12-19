---
weight: 80
title: "版本历史"
---

# 版本发布日志

这里是`loadgen`历史版本发布的相关说明。

## Latest (In development)  
### ❌ Breaking changes  
### 🚀 Features  
### 🐛 Bug fix  
### ✈️ Improvements  

## 1.30.1 (2025-12-19)
### ❌ Breaking changes  
### 🚀 Features  
### 🐛 Bug fix  
### ✈️ Improvements  
- 此版本包含了底层 [Framework v1.4.0](https://docs.infinilabs.com/framework/v1.4.0) 的更新，解决了一些常见问题，并增强了整体稳定性和性能。虽然 Loadgen 本身没有直接的变更，但从 Framework 中继承的改进间接地使 Loadgen 受益。

## 1.30.0 (2025-11-19)
### ❌ Breaking changes  
### 🚀 Features  
- feat: support array for _ctx.response.body_json #41
### 🐛 Bug fix  
### ✈️ Improvements  
- 此版本包含了底层 [Framework v1.3.0](https://docs.infinilabs.com/framework/v1.3.0) 的更新，解决了一些常见问题，并增强了整体稳定性和性能。虽然 Loadgen 本身没有直接的变更，但从 Framework 中继承的改进间接地使 Loadgen 受益。

## 1.29.8 (2025-07-25)
### ❌ Breaking changes  
### 🚀 Features  
### 🐛 Bug fix  
### ✈️ Improvements  
- 此版本包含了底层 [Framework v1.2.0](https://docs.infinilabs.com/framework/v1.2.0) 的更新，解决了一些常见问题，并增强了整体稳定性和性能。虽然 Loadgen 本身没有直接的变更，但从 Framework 中继承的改进间接地使 Loadgen 受益。

## 1.29.7 (2025-06-29)
### ❌ Breaking changes  
### 🚀 Features  
### 🐛 Bug fix  
### ✈️ Improvements  
- 此版本包含了底层 [Framework v1.1.9](https://docs.infinilabs.com/framework/v1.1.9) 的更新，解决了一些常见问题，并增强了整体稳定性和性能。虽然 LOADGEN 本身没有直接的变更，但从 Framework 中继承的改进间接地使 LOADGEN 受益。

## 1.29.6 (2025-06-13)
### ❌ Breaking changes  
### 🚀 Features  
### 🐛 Bug fix  
### ✈️ Improvements  

## 1.29.4 (2025-05-16)
### ❌ Breaking changes  
### 🚀 Features  
### 🐛 Bug fix  
### ✈️ Improvements  
- 同步更新 [Framework v1.1.7](https://docs.infinilabs.com/framework/v1.1.7/docs/references/http_client/) 修复的一些已知问题

## 1.29.3 (2025-04-27)
- 同步更新 [Framework v1.1.6](https://docs.infinilabs.com/framework/v1.1.6/docs/references/http_client/) 修复的一些已知问题

## 1.29.2 (2025-03-31)
- 同步更新 [Framework v1.1.5](https://docs.infinilabs.com/framework/v1.1.5/docs/references/http_client/) 修复的一些已知问题


## 1.29.1 (2025-03-14)
- 同步更新 [Framework v1.1.4](https://docs.infinilabs.com/framework/v1.1.4/docs/references/http_client/) 修复的一些已知问题


## 1.29.0 (2025-02-28)

- 同步更新 Framework 修复的一些已知问题

## 1.28.2 (2025-02-15)

- 同步更新 Framework 修复的一些已知问题

## 1.28.1 (2025-01-24)

- 同步更新 Framework 修复的一些已知问题

## 1.28.0 (2025-01-11)

- 同步更新 Framework 修复的一些已知问题

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
