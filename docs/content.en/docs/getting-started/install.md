---
weight: 10
title: Installing the Loadgen
asciinema: true
---

# Installing the Loadgen

INFINI Loadgen supports mainstream operating systems and platforms. The program package is small, with no extra external dependency. So, the loadgen can be installed very rapidly.

## Downloading

**Automatic install**

```bash
curl -sSL http://get.infini.cloud | bash -s -- -p loadgen
```

> The above script can automatically download the latest version of the corresponding platform's loadgen and extract it to /opt/loadgen

> The optional parameters for the script are as follows:

> &nbsp;&nbsp;&nbsp;&nbsp;_-v [version number]（Default to use the latest version number）_

> &nbsp;&nbsp;&nbsp;&nbsp;_-d [installation directory] (default installation to /opt/loadgen)_

**Manual install**

Select a package for downloading in the following URL based on your operating system and platform:

[https://release.infinilabs.com/loadgen/](https://release.infinilabs.com/loadgen/)
