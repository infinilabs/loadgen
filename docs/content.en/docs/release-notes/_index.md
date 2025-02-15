---
weight: 80
title: "Release Notes"
---

# Release Notes

Information about release notes of INFINI Loadgen is provided here.

## 1.28.2 (2025-02-15)

### Improvements

- Synchronize updates for known issues fixed in the Framework.

## 1.28.1 (2025-01-24)

### Improvements

- Synchronize updates for known issues fixed in the Framework.

## 1.28.0 (2025-01-11)

### Improvements

- Synchronize updates for known issues fixed in the Framework.

## 1.27.0 (2024-12-13)

### Improvements

- The code is open source, and Github [repository](https://github.com/infinilabs/loadgen) is used for development.
- Keep the same version number as INFINI Console.
- Synchronize updates for known issues fixed in the Framework.

### Bug fix

- Fix the abnormal problem of the API interface testing logic.

## 1.26.1 (2024-08-13)

### Improvements

- Keep the same version number as INFINI Console.
- Synchronize updates for known issues fixed in the Framework.

## 1.26.0 (2024-06-07)

### Improvements

- Keep the same version number as INFINI Console.
- Synchronize updates for known issues fixed in the Framework.

## 1.25.0 (2024-04-30)

### Improvements

- Keep the same version number as INFINI Console.
- Synchronize updates for known issues fixed in the Framework.

## 1.24.0 (2024-04-15)

### Improvements

- Keep the same version number as INFINI Console.
- Synchronize updates for known issues fixed in the Framework.

## 1.22.0 (2024-01-26)

### Improvements

- Unified version number with INFINI Console

## 1.8.0 (2023-11-02)

### Breaking changes

- The original Loadrun function is incorporated into Loadgen.
- Test the requests, assertions, etc. that is configured using the new Loadgen DSL syntax.

## 1.7.0 (2023-04-20)

### Breaking changes

- The variables with the same `name` are no longer allowed to be defined in `variables`.

### Features

- Add the `log_status_code` configuration to support printing request logs of specific status codes.

## 1.6.0 (2023-04-06)

### Breaking ghanges

- The `file` type variable by default no longer escapes the `"` and `\` characters. Use the `replace` function to manually set variable escaping.

### Features

- The variable definition adds an optional `replace` option, which is used to escape characters such as `"` and `\`.

### Improvements

- Optimize memory usage.

### Bug fix

- Fix the problem that the `\n` cannot be used in the YAML strings.
- Fix the problem that invalid assert configurations are ignored.

## 1.5.1

### Bug fix

- [DOC] Fix invalid variable grammar in `loadgen.yml`.

## 1.5.0

### Features

- Added `assert` configuration, support testing response data.
- Added `register` configuration, support registering dynamic variables.
- Added `env` configuration, support loading and using environment variables in `loadgen.yml`.
- Support using dynamic variables in the `headers` configuration.

### Improvements

- `-l` option: precisely control the number of requests to send.
- Added `runner.no_warm` to skip warm-up stage.

### Bug fix
