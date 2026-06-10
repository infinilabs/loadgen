---
weight: 10
title: 下载安装
---

# 下载安装

INFINI Loadgen 是单文件可执行程序，无外部依赖，支持 Linux / macOS / Windows。

## 自动安装（推荐）

```bash
curl -sSL http://get.infini.cloud | bash -s -- -p loadgen
```

默认安装到 `/opt/loadgen`。可选参数：

| 参数 | 说明 |
|------|------|
| `-v <版本号>` | 指定版本（默认最新） |
| `-d <目录>` | 自定义安装路径 |

示例：安装到指定目录

```bash
curl -sSL http://get.infini.cloud | bash -s -- -p loadgen -d /usr/local/bin
```

## 手动安装

根据操作系统和架构下载对应包：

> https://release.infinilabs.com/loadgen/

解压后得到：

```
loadgen              # 可执行文件
loadgen.yml          # 配置文件模板
loadgen.dsl          # DSL 测试文件模板
```

## 验证安装

```bash
loadgen -v
```

输出版本号即安装成功。

## 下一步

前往 [快速上手]({{< relref "concepts" >}}) 运行你的第一个测试。
