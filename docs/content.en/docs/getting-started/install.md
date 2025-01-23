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

The above script can automatically download the latest version of the corresponding platform's loadgen and extract it to /opt/loadgen

The optional parameters for the script are as follows:

> _-v [version number]（Default to use the latest version number）_  
> _-d [installation directory] (default installation to /opt/loadgen)_

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

**Manual install**

Select a package for downloading in the following URL based on your operating system and platform:

[https://release.infinilabs.com/loadgen/](https://release.infinilabs.com/loadgen/)
