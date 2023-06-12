 #!/bin/bash

#init
WORKBASE=/home/jenkins/go/src/infini.sh
WORKDIR=$WORKBASE/$FNAME/cmd/$PNAME
DEST=/infini/Sync/Release/$PNAME/stable

if [[ $VERSION =~ NIGHTLY ]]; then
  BUILD_NUMBER=$BUILD_DAY
  DEST=/infini/Sync/Release/$PNAME/snapshot
fi
export DOCKER_CLI_EXPERIMENTAL=enabled

#clean all
cd $WORKSPACE && git clean -fxd

#pull code
cd $WORKDIR && git clean -fxd
git stash && git pull origin master

 #build
make clean config build-linux
make config build-arm
make config build-darwin
make config build-win
GOROOT="/infini/go-pkgs/go-loongarch" PATH=$GOROOT/bin:$PATH make build-linux-loong64
#GOROOT="/infini/go-pkgs/go-swarch" PATH=$GOROOT/bin:$PATH make build-linux-sw64

#copy-configs
cp -rf $WORKBASE/framework/LICENSE $WORKDIR/bin && cat $WORKBASE/framework/NOTICE $WORKDIR/NOTICE > $WORKDIR/bin/NOTICE

cd $WORKDIR/bin
for t in 386 amd64 arm64 armv5 armv6 armv7 loong64 mips mips64 mips64le mipsle riscv64 ; do
  tar zcf ${WORKSPACE}/$PNAME-$VERSION-$BUILD_NUMBER-linux-$t.tar.gz "${PNAME}-linux-$t" $PNAME.yml LICENSE NOTICE 
done

for t in mac-amd64 mac-arm64 ; do
  zip -qr ${WORKSPACE}/$PNAME-$VERSION-$BUILD_NUMBER-$t.zip $PNAME-$t $PNAME.yml LICENSE NOTICE
done

for t in windows-amd64 windows-386 ; do
  zip -qr ${WORKSPACE}/$PNAME-$VERSION-$BUILD_NUMBER-$t.zip $PNAME-$t.exe $PNAME.yml LICENSE NOTICE config
done

#publish
for t in 386 amd64 arm64 armv5 armv6 armv7 loong64 mips mips64 mips64le mipsle riscv64 ; do
  [ -f ${WORKSPACE}/$PNAME-$VERSION-$BUILD_NUMBER-linux-$t.tar.gz ] && cp -rf ${WORKSPACE}/$PNAME-$VERSION-$BUILD_NUMBER-linux-$t.tar.gz $DEST
done

for t in mac-amd64 mac-arm64 windows-amd64 windows-386 ; do
  [ -f ${WORKSPACE}/$PNAME-$VERSION-$BUILD_NUMBER-$t.zip ] && cp -rf ${WORKSPACE}/$PNAME-$VERSION-$BUILD_NUMBER-$t.zip $DEST
done

#git reset
cd $WORKSPACE && git reset --hard
cd $WORKDIR && git reset --hard
