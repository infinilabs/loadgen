---
weight: 10
title: Installation
---

# Installation

INFINI Loadgen is a single-binary executable with no external dependencies. Supports Linux / macOS / Windows.

## Automatic Install (Recommended)

```bash
curl -sSL http://get.infini.cloud | bash -s -- -p loadgen
```

Installs to `/opt/loadgen` by default. Optional parameters:

| Parameter | Description |
|-----------|-------------|
| `-v <version>` | Specify version (defaults to latest) |
| `-d <directory>` | Custom install path |

Example: install to a custom directory

```bash
curl -sSL http://get.infini.cloud | bash -s -- -p loadgen -d /usr/local/bin
```

## Manual Install

Download the package for your OS and architecture:

> https://release.infinilabs.com/loadgen/

After extraction you'll find:

```
loadgen              # Executable
loadgen.yml          # Configuration template
loadgen.dsl          # DSL test file template
```

## Verify Installation

```bash
loadgen -v
```

If the version number is printed, installation is successful.

## Next Steps

Head to [Quick Start]({{< relref "concepts" >}}) to run your first test.
