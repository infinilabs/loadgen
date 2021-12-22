pipeline {

    agent none

    environment { 
        CI = 'true'
    }
    stages {
        
        stage('Build Linux Packages') {

            agent {
                label 'linux'
            }

            steps {
                catchError(buildResult: 'SUCCESS', stageResult: 'FAILURE'){
                    sh 'cd /home/jenkins/go/src/infini.sh/loadgen && git stash && git pull origin master && make clean config build-linux'
                    sh 'cd /home/jenkins/go/src/infini.sh/loadgen && git stash && git pull origin master && make config build-arm'
                    sh 'cd /home/jenkins/go/src/infini.sh/loadgen && git stash && git pull origin master && make config build-darwin'
                    sh 'cd /home/jenkins/go/src/infini.sh/loadgen && git stash && git pull origin master && make config build-win'
                   
                   sh label: 'package-linux-amd64', script: 'cd /home/jenkins/go/src/infini.sh/loadgen/bin && tar cfz ${WORKSPACE}/loadgen-$VERSION-$BUILD_NUMBER-linux-amd64.tar.gz loadgen-linux-amd64 loadgen.yml '
                   sh label: 'package-linux-386', script: 'cd /home/jenkins/go/src/infini.sh/loadgen/bin && tar cfz ${WORKSPACE}/loadgen-$VERSION-$BUILD_NUMBER-linux-386.tar.gz loadgen-linux-386 loadgen.yml '
                   sh label: 'package-linux-mips', script: 'cd /home/jenkins/go/src/infini.sh/loadgen/bin && tar cfz ${WORKSPACE}/loadgen-$VERSION-$BUILD_NUMBER-linux-mips.tar.gz loadgen-linux-mips loadgen.yml '
                   sh label: 'package-linux-mipsle', script: 'cd /home/jenkins/go/src/infini.sh/loadgen/bin && tar cfz ${WORKSPACE}/loadgen-$VERSION-$BUILD_NUMBER-linux-mipsle.tar.gz loadgen-linux-mipsle loadgen.yml '
                   sh label: 'package-linux-mips64', script: 'cd /home/jenkins/go/src/infini.sh/loadgen/bin && tar cfz ${WORKSPACE}/loadgen-$VERSION-$BUILD_NUMBER-linux-mips64.tar.gz loadgen-linux-mips64 loadgen.yml '
                   sh label: 'package-linux-mips64le', script: 'cd /home/jenkins/go/src/infini.sh/loadgen/bin && tar cfz ${WORKSPACE}/loadgen-$VERSION-$BUILD_NUMBER-linux-mips64le.tar.gz loadgen-linux-mips64le loadgen.yml '
                   sh label: 'package-linux-arm5', script: 'cd /home/jenkins/go/src/infini.sh/loadgen/bin && tar cfz ${WORKSPACE}/loadgen-$VERSION-$BUILD_NUMBER-linux-arm5.tar.gz loadgen-linux-armv5 loadgen.yml '
                   sh label: 'package-linux-arm6', script: 'cd /home/jenkins/go/src/infini.sh/loadgen/bin && tar cfz ${WORKSPACE}/loadgen-$VERSION-$BUILD_NUMBER-linux-arm6.tar.gz loadgen-linux-armv6 loadgen.yml '
                   sh label: 'package-linux-arm7', script: 'cd /home/jenkins/go/src/infini.sh/loadgen/bin && tar cfz ${WORKSPACE}/loadgen-$VERSION-$BUILD_NUMBER-linux-arm7.tar.gz loadgen-linux-armv7 loadgen.yml '
                   sh label: 'package-linux-arm64', script: 'cd /home/jenkins/go/src/infini.sh/loadgen/bin && tar cfz ${WORKSPACE}/loadgen-$VERSION-$BUILD_NUMBER-linux-arm64.tar.gz loadgen-linux-arm64 loadgen.yml '

                    sh label: 'package-mac-amd64', script: 'cd /home/jenkins/go/src/infini.sh/loadgen/bin && zip -r ${WORKSPACE}/loadgen-$VERSION-$BUILD_NUMBER-mac-amd64.zip loadgen-mac-amd64 loadgen.yml'
                    sh label: 'package-mac-arm64', script: 'cd /home/jenkins/go/src/infini.sh/loadgen/bin && zip -r ${WORKSPACE}/loadgen-$VERSION-$BUILD_NUMBER-mac-arm64.zip loadgen-mac-arm64 loadgen.yml'

                    sh label: 'package-win-amd64', script: 'cd /home/jenkins/go/src/infini.sh/loadgen/bin && zip -r ${WORKSPACE}/loadgen-$VERSION-$BUILD_NUMBER-windows-amd64.zip loadgen-windows-amd64.exe loadgen.yml'
                    sh label: 'package-win-386', script: 'cd /home/jenkins/go/src/infini.sh/loadgen/bin && zip -r ${WORKSPACE}/loadgen-$VERSION-$BUILD_NUMBER-windows-386.zip loadgen-windows-386.exe loadgen.yml'


                    fingerprint 'loadgen-$VERSION-$BUILD_NUMBER-*'
                    archiveArtifacts artifacts: 'loadgen-$VERSION-$BUILD_NUMBER-*', fingerprint: true, followSymlinks: true, onlyIfSuccessful: true
                }
            }
        }

    }
}
