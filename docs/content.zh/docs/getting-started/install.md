---
weight: 10
title: 下载安装
asciinema: true
---

# 安装 INFINI Loadgen

INFINI Loadgen 支持主流的操作系统和平台，程序包很小，没有任何额外的外部依赖，安装起来应该是很快的 ：）

## 下载安装

**自动安装**

```bash
curl -sSL http://get.infini.cloud | bash -s -- -p loadgen
```

通过以上脚本可自动下载相应平台的 loadgen 最新版本并解压到/opt/loadgen

脚本的可选参数如下：

> _-v [版本号]（默认采用最新版本号）_  
> _-d [安装目录]（默认安装到/opt/loadgen）_

```bash
➜  /tmp mkdir loadgen
➜  /tmp curl -sSL http://get.infini.cloud | bash -s -- -p loadgen -d /tmp/loadgen

                                 @@@@@@@@@@@
                                @@@@@@@@@@@@
                                @@@@@@@@@@@@
                               @@@@@@@@@&@@@
                              #@@@@@@@@@@@@@
        @@@                   @@@@@@@@@@@@@
       &@@@@@@@              &@@@@@@@@@@@@@
       @&@@@@@@@&@           @@@&@@@@@@@&@
      @@@@@@@@@@@@@@@@      @@@@@@@@@@@@@@
      @@@@@@@@@@@@@@@@@@&   @@@@@@@@@@@@@
        %@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
            @@@@@@@@@@@@&@@@@@@@@@@@@@@@
    @@         ,@@@@@@@@@@@@@@@@@@@@@@@&
    @@@@@.         @@@@@&@@@@@@@@@@@@@@
   @@@@@@@@@@          @@@@@@@@@@@@@@@#
   @&@@@&@@@&@@@          &@&@@@&@@@&@
  @@@@@@@@@@@@@.              @@@@@@@*
  @@@@@@@@@@@@@                  %@@@
 @@@@@@@@@@@@@
/@@@@@@@&@@@@@
@@@@@@@@@@@@@
@@@@@@@@@@@@@
@@@@@@@@@@@@        Welcome to INFINI Labs!


Now attempting the installation...

Name: [loadgen], Version: [1.26.1-598], Path: [/tmp/loadgen]
File: [https://release.infinilabs.com/loadgen/stable/loadgen-1.26.1-598-mac-arm64.zip]
##=O#- #

Installation complete. [loadgen] is ready to use!


----------------------------------------------------------------
cd /tmp/loadgen && ./loadgen-mac-arm64
----------------------------------------------------------------


   __ _  __ ____ __ _  __ __
  / // |/ // __// // |/ // /
 / // || // _/ / // || // /
/_//_/|_//_/  /_//_/|_//_/

©INFINI.LTD, All Rights Reserved.
```

**手动安装**

根据您所在的操作系统和平台选择下面相应的下载地址：

[https://release.infinilabs.com/loadgen/](https://release.infinilabs.com/loadgen/)
